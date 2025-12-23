package images

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

// OSImage represents a downloadable cloud image
type OSImage struct {
	Name        string `json:"name"`
	Distro      string `json:"distro"`
	Version     string `json:"version"`
	Filename    string `json:"filename"`
	DownloadURL string `json:"url"`
	IsLTS       bool   `json:"is_lts"`
}

var (
	Catalog []OSImage
	mu      sync.Mutex
)

// Config: How often to force a web scrape? (e.g., 24 Hours)
const CacheDuration = 24 * time.Hour

func GetCachePath() string {
	home, _ := os.UserHomeDir()
	dir := filepath.Join(home, ".vps-manager")
	os.MkdirAll(dir, 0o755)
	return filepath.Join(dir, "catalog.json")
}

func RefreshCatalog() error {
	cacheFile := GetCachePath()
	info, err := os.Stat(cacheFile)

	// 1. Try to Load from Cache
	if err == nil {
		if time.Since(info.ModTime()) < CacheDuration {
			file, _ := os.ReadFile(cacheFile)
			if json.Unmarshal(file, &Catalog) == nil && len(Catalog) > 0 {
				return nil
			}
		}
	}

	// 2. If Cache missing or old, Scrape the Web
	fmt.Println("🔄 Catalog expired. Scraping mirrors for latest versions...")
	if err := scrapeAll(); err != nil {
		return err
	}

	// 3. Save to Cache
	data, _ := json.MarshalIndent(Catalog, "", "  ")
	os.WriteFile(cacheFile, data, 0o644)
	fmt.Printf("✅ Catalog cached to %s\n", cacheFile)
	return nil
}

func scrapeAll() error {
	Catalog = []OSImage{}
	var wg sync.WaitGroup

	scrapers := []func(){
		fetchUbuntu,
		fetchDebian,
		fetchRHELClones,
		fetchFedora,
		fetchOpenSUSE,
		fetchAlpine,
		fetchArch,
	}

	for _, s := range scrapers {
		wg.Add(1)
		go func(f func()) { defer wg.Done(); f() }(s)
	}
	wg.Wait()

	sort.Slice(Catalog, func(i, j int) bool {
		return Catalog[i].Name < Catalog[j].Name
	})
	return nil
}

// --- SCRAPERS ---

func addImage(img OSImage) {
	mu.Lock()
	defer mu.Unlock()
	Catalog = append(Catalog, img)
}

// 1. UBUNTU
func fetchUbuntu() {
	body, err := scrapeHTML("https://cloud-images.ubuntu.com/releases/")
	if err != nil {
		return
	}
	re := regexp.MustCompile(`href="(2[2-9]\.04)/"`)
	for _, m := range re.FindAllStringSubmatch(body, -1) {
		ver := m[1]
		code := "ubuntu"
		if ver == "22.04" {
			code = "jammy"
		}
		if ver == "24.04" {
			code = "noble"
		}
		url := fmt.Sprintf("https://cloud-images.ubuntu.com/releases/%s/current/%s-server-cloudimg-amd64.img", ver, code)
		if checkURL(url) {
			addImage(OSImage{Name: "Ubuntu " + ver + " LTS", Distro: "ubuntu", Version: ver, Filename: "ubuntu-" + ver + ".img", DownloadURL: url, IsLTS: true})
		}
	}
}

// 2. DEBIAN
func fetchDebian() {
	for ver, name := range map[string]string{"12": "bookworm", "11": "bullseye"} {
		url := fmt.Sprintf("https://cloud.debian.org/images/cloud/%s/latest/debian-%s-generic-amd64.qcow2", name, ver)
		if checkURL(url) {
			addImage(OSImage{Name: "Debian " + ver, Distro: "debian", Version: ver, Filename: "debian-" + ver + ".qcow2", DownloadURL: url, IsLTS: true})
		}
	}
}

// 3. RHEL CLONES (Fixed for Rocky 10)
func fetchRHELClones() {
	for ver := 8; ver < 12; ver++ {
		vStr := strconv.Itoa(ver)

		// --- AlmaLinux ---
		urlA := fmt.Sprintf("https://repo.almalinux.org/almalinux/%d/cloud/x86_64/images/AlmaLinux-%d-GenericCloud-latest.x86_64.qcow2", ver, ver)
		if checkURL(urlA) {
			addImage(OSImage{Name: "AlmaLinux " + vStr, Distro: "alma", Version: vStr, Filename: "alma-" + vStr + ".qcow2", DownloadURL: urlA, IsLTS: true})
		}

		// --- Rocky Linux ---
		// Pattern A (Standard for 8/9): Rocky-9-GenericCloud.latest.x86_64.qcow2
		urlR1 := fmt.Sprintf("https://dl.rockylinux.org/pub/rocky/%d/images/x86_64/Rocky-%d-GenericCloud.latest.x86_64.qcow2", ver, ver)

		// Pattern B (New for 10): Rocky-10-GenericCloud-Base.latest.x86_64.qcow2
		urlR2 := fmt.Sprintf("https://dl.rockylinux.org/pub/rocky/%d/images/x86_64/Rocky-%d-GenericCloud-Base.latest.x86_64.qcow2", ver, ver)

		if checkURL(urlR1) {
			addImage(OSImage{Name: "Rocky Linux " + vStr, Distro: "rocky", Version: vStr, Filename: "rocky-" + vStr + ".qcow2", DownloadURL: urlR1, IsLTS: true})
		} else if checkURL(urlR2) {
			// Found the "Base" variant (Rocky 10)
			addImage(OSImage{Name: "Rocky Linux " + vStr, Distro: "rocky", Version: vStr, Filename: "rocky-" + vStr + ".qcow2", DownloadURL: urlR2, IsLTS: true})
		}

		// --- CentOS Stream ---
		urlC := fmt.Sprintf("https://cloud.centos.org/centos/%d-stream/x86_64/images/CentOS-Stream-GenericCloud-%d-latest.x86_64.qcow2", ver, ver)
		if checkURL(urlC) {
			addImage(OSImage{Name: "CentOS Stream " + vStr, Distro: "centos", Version: vStr, Filename: "centos-" + vStr + ".qcow2", DownloadURL: urlC, IsLTS: false})
		}
	}
}

// 4. FEDORA
func fetchFedora() {
	body, err := scrapeHTML("https://download.fedoraproject.org/pub/fedora/linux/releases/")
	if err != nil {
		return
	}
	re := regexp.MustCompile(`href="([0-9]+)/"`)
	highest := 0
	for _, m := range re.FindAllStringSubmatch(body, -1) {
		if v, _ := strconv.Atoi(m[1]); v > highest {
			highest = v
		}
	}
	if highest > 0 {
		base := fmt.Sprintf("https://download.fedoraproject.org/pub/fedora/linux/releases/%d/Cloud/x86_64/images/", highest)
		body2, _ := scrapeHTML(base)
		reImg := regexp.MustCompile(`href="(Fedora-Cloud-Base-Generic-[^"]+\.qcow2)"`)
		if m := reImg.FindStringSubmatch(body2); len(m) > 1 {
			addImage(OSImage{Name: fmt.Sprintf("Fedora %d (Bleeding Edge)", highest), Distro: "fedora", Version: strconv.Itoa(highest), Filename: fmt.Sprintf("fedora-%d.qcow2", highest), DownloadURL: base + m[1], IsLTS: false})
		}
	}
}

// 5. OPENSUSE
func fetchOpenSUSE() {
	base := "https://download.opensuse.org/repositories/Cloud:/Images:/"
	body, _ := scrapeHTML(base)
	re := regexp.MustCompile(`href="Leap_([0-9]+\.[0-9]+)/"`)
	for _, m := range re.FindAllStringSubmatch(body, -1) {
		if v, _ := strconv.ParseFloat(m[1], 64); v >= 15.5 {
			url := fmt.Sprintf("%sLeap_%s/images/openSUSE-Leap-%s.x86_64-NoCloud.qcow2", base, m[1], m[1])
			if checkURL(url) {
				addImage(OSImage{Name: "OpenSUSE Leap " + m[1], Distro: "opensuse", Version: m[1], Filename: "opensuse-" + m[1] + ".qcow2", DownloadURL: url, IsLTS: true})
			}
		}
	}
}

// 6. ALPINE
func fetchAlpine() {
	base := "https://dl-cdn.alpinelinux.org/alpine/"
	body, _ := scrapeHTML(base)
	re := regexp.MustCompile(`href="(v3\.[0-9]+)/"`)
	for _, m := range re.FindAllStringSubmatch(body, -1) {
		if v, _ := strconv.ParseFloat(strings.TrimPrefix(m[1], "v"), 64); v >= 3.18 {
			cDir := fmt.Sprintf("%s%s/releases/cloud/", base, m[1])
			body2, _ := scrapeHTML(cDir)
			reImg := regexp.MustCompile(`href="(nocloud_alpine-[^"]+-bios-cloudinit-r0\.qcow2)"`)
			if mImg := reImg.FindStringSubmatch(body2); len(mImg) > 1 {
				addImage(OSImage{Name: "Alpine " + m[1], Distro: "alpine", Version: m[1], Filename: "alpine-" + m[1] + ".qcow2", DownloadURL: cDir + mImg[1], IsLTS: true})
			}
		}
	}
}

// 7. ARCH
func fetchArch() {
	url := "https://geo.mirror.pkgbuild.com/images/latest/Arch-Linux-x86_64-cloudimg.qcow2"
	if checkURL(url) {
		addImage(OSImage{Name: "Arch Linux (Rolling)", Distro: "arch", Version: "latest", Filename: "arch-linux.qcow2", DownloadURL: url, IsLTS: false})
	}
}

// Helpers

func scrapeHTML(url string) (string, error) {
	c := http.Client{Timeout: 5 * time.Second}
	r, err := c.Get(url)
	if err != nil {
		return "", err
	}
	defer r.Body.Close()
	b, err := io.ReadAll(r.Body)
	return string(b), err
}

func checkURL(url string) bool {
	c := http.Client{Timeout: 2 * time.Second}
	r, err := c.Head(url)
	return err == nil && r.StatusCode == 200
}
