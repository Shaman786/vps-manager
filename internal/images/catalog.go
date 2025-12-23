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

// Catalog holds the dynamic list of available images
var Catalog []OSImage

// Mutex to safely update catalog from multiple goroutines
var mu sync.Mutex

// Config: How often to force a web scrape? (e.g., 24 Hours)
const CacheDuration = 24 * time.Hour

// GetCachePath returns the location of the cache file (e.g., ~/.vps-manager/catalog.json)
func GetCachePath() string {
	home, _ := os.UserHomeDir()
	dir := filepath.Join(home, ".vps-manager")
	os.MkdirAll(dir, 0755)
	return filepath.Join(dir, "catalog.json")
}

// RefreshCatalog handles the "Check Cache vs Web" logic
func RefreshCatalog() error {
	cacheFile := GetCachePath()
	info, err := os.Stat(cacheFile)

	// 1. Try to Load from Cache First
	if err == nil {
		// If cache file exists and is fresh (less than 24 hours old)
		if time.Since(info.ModTime()) < CacheDuration {
			// Optional: fmt.Println("⚡ Loading Image Catalog from Cache...") 
			file, _ := os.ReadFile(cacheFile)
			if json.Unmarshal(file, &Catalog) == nil && len(Catalog) > 0 {
				return nil // Success! We skipped the slow scraping.
			}
		}
	}

	// 2. If Cache is missing or old, Scrape the Web
	fmt.Println("🔄 Catalog expired or missing. Scraping mirrors for latest versions...")
	if err := scrapeAll(); err != nil {
		return err
	}

	// 3. Save the new list to Cache
	data, _ := json.MarshalIndent(Catalog, "", "  ")
	os.WriteFile(cacheFile, data, 0644)
	fmt.Printf("✅ Catalog updated and cached to %s\n", cacheFile)
	return nil
}

// scrapeAll runs all the scrapers in parallel
func scrapeAll() error {
	Catalog = []OSImage{}
	var wg sync.WaitGroup

	// List of scrapers to run
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
		go func(scraper func()) {
			defer wg.Done()
			scraper()
		}(s)
	}
	wg.Wait()

	// Sort A-Z for the menu
	sort.Slice(Catalog, func(i, j int) bool {
		return Catalog[i].Name < Catalog[j].Name
	})
	return nil
}

// ==========================================
//              THE SCRAPERS
// ==========================================

// 1. UBUNTU
func fetchUbuntu() {
	baseURL := "https://cloud-images.ubuntu.com/releases/"
	body, err := scrapeHTML(baseURL)
	if err != nil { return }
	re := regexp.MustCompile(`href="(2[2-9]\.04)/"`) // Matches 22.04, 24.04, 26.04...
	for _, m := range re.FindAllStringSubmatch(body, -1) {
		ver := m[1]
		codename := "ubuntu"
		if ver == "22.04" { codename = "jammy" }
		if ver == "24.04" { codename = "noble" }
		url := fmt.Sprintf("%s%s/current/%s-server-cloudimg-amd64.img", baseURL, ver, codename)
		// Only add if URL exists
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
			addImage(OSImage{Name: "Debian " + ver + " (" + name + ")", Distro: "debian", Version: ver, Filename: "debian-" + ver + ".qcow2", DownloadURL: url, IsLTS: true})
		}
	}
}

// 3. RHEL CLONES
func fetchRHELClones() {
	for ver := 8; ver < 12; ver++ {
		vStr := strconv.Itoa(ver)
		// Alma
		urlA := fmt.Sprintf("https://repo.almalinux.org/almalinux/%d/cloud/x86_64/images/AlmaLinux-%d-GenericCloud-latest.x86_64.qcow2", ver, ver)
		if checkURL(urlA) {
			addImage(OSImage{Name: "AlmaLinux " + vStr, Distro: "alma", Version: vStr, Filename: "alma-" + vStr + ".qcow2", DownloadURL: urlA, IsLTS: true})
		}
		// Rocky
		urlR := fmt.Sprintf("https://dl.rockylinux.org/pub/rocky/%d/images/x86_64/Rocky-%d-GenericCloud.latest.x86_64.qcow2", ver, ver)
		if checkURL(urlR) {
			addImage(OSImage{Name: "Rocky Linux " + vStr, Distro: "rocky", Version: vStr, Filename: "rocky-" + vStr + ".qcow2", DownloadURL: urlR, IsLTS: true})
		}
		// CentOS Stream
		urlC := fmt.Sprintf("https://cloud.centos.org/centos/%d-stream/x86_64/images/CentOS-Stream-GenericCloud-%d-latest.x86_64.qcow2", ver, ver)
		if checkURL(urlC) {
			addImage(OSImage{Name: "CentOS Stream " + vStr, Distro: "centos", Version: vStr, Filename: "centos-" + vStr + ".qcow2", DownloadURL: urlC, IsLTS: false})
		}
	}
}

// 4. FEDORA
func fetchFedora() {
	base := "https://download.fedoraproject.org/pub/fedora/linux/releases/"
	body, err := scrapeHTML(base)
	if err != nil { return }
	re := regexp.MustCompile(`href="([0-9]+)/"`)
	highest := 0
	for _, m := range re.FindAllStringSubmatch(body, -1) {
		if v, _ := strconv.Atoi(m[1]); v > highest { highest = v }
	}
	if highest > 0 {
		imgBase := fmt.Sprintf("%s%d/Cloud/x86_64/images/", base, highest)
		body2, err := scrapeHTML(imgBase)
		if err == nil {
			reImg := regexp.MustCompile(`href="(Fedora-Cloud-Base-Generic-[^"]+\.qcow2)"`)
			m := reImg.FindStringSubmatch(body2)
			if len(m) > 1 {
				addImage(OSImage{Name: fmt.Sprintf("Fedora %d (Bleeding Edge)", highest), Distro: "fedora", Version: strconv.Itoa(highest), Filename: fmt.Sprintf("fedora-%d.qcow2", highest), DownloadURL: imgBase + m[1], IsLTS: false})
			}
		}
	}
}

// 5. OPENSUSE
func fetchOpenSUSE() {
	base := "https://download.opensuse.org/repositories/Cloud:/Images:/"
	body, err := scrapeHTML(base)
	if err != nil { return }
	re := regexp.MustCompile(`href="Leap_([0-9]+\.[0-9]+)/"`)
	for _, m := range re.FindAllStringSubmatch(body, -1) {
		ver := m[1]
		if v, _ := strconv.ParseFloat(ver, 64); v >= 15.5 {
			url := fmt.Sprintf("%sLeap_%s/images/openSUSE-Leap-%s.x86_64-NoCloud.qcow2", base, ver, ver)
			if checkURL(url) {
				addImage(OSImage{Name: "OpenSUSE Leap " + ver, Distro: "opensuse", Version: ver, Filename: "opensuse-" + ver + ".qcow2", DownloadURL: url, IsLTS: true})
			}
		}
	}
}

// 6. ALPINE
func fetchAlpine() {
	base := "https://dl-cdn.alpinelinux.org/alpine/"
	body, err := scrapeHTML(base)
	if err != nil { return }
	re := regexp.MustCompile(`href="(v3\.[0-9]+)/"`)
	for _, m := range re.FindAllStringSubmatch(body, -1) {
		ver := m[1]
		if v, _ := strconv.ParseFloat(strings.TrimPrefix(ver, "v"), 64); v >= 3.18 {
			cloudDir := fmt.Sprintf("%s%s/releases/cloud/", base, ver)
			body2, err := scrapeHTML(cloudDir)
			if err == nil {
				reImg := regexp.MustCompile(`href="(nocloud_alpine-[^"]+-bios-cloudinit-r0\.qcow2)"`)
				mImg := reImg.FindStringSubmatch(body2)
				if len(mImg) > 1 {
					addImage(OSImage{Name: "Alpine Linux " + ver, Distro: "alpine", Version: ver, Filename: "alpine-" + ver + ".qcow2", DownloadURL: cloudDir + mImg[1], IsLTS: true})
				}
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
func addImage(img OSImage) {
	mu.Lock()
	defer mu.Unlock()
	Catalog = append(Catalog, img)
}

func scrapeHTML(url string) (string, error) {
	client := http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(url)
	if err != nil { return "", err }
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	return string(body), err
}

func checkURL(url string) bool {
	client := http.Client{Timeout: 2 * time.Second}
	resp, err := client.Head(url)
	if err != nil { return false }
	defer resp.Body.Close()
	return resp.StatusCode == 200
}