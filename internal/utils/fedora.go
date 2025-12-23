package utils

import (
	"fmt"
	"io"
	"net/http"
	"regexp"
	"sort"
	"strconv"
)

const fedoraBaseURL = "https://download.fedoraproject.org/pub/fedora/linux/releases/"

// GetLatestFedora dynamically finds the newest Fedora Cloud image
func GetLatestFedora() (string, string, error) {
	fmt.Println("   -> [Dynamic] Probing Fedora mirrors for latest version...")

	// 1. Scrape the main releases page to find version numbers
	resp, err := http.Get(fedoraBaseURL)
	if err != nil {
		return "", "", fmt.Errorf("mirror unreachable: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	// Regex to match directory links like "41/", "42/", "43/"
	reVer := regexp.MustCompile(`href="([0-9]+)/"`)
	matches := reVer.FindAllStringSubmatch(string(body), -1)

	var versions []int
	for _, m := range matches {
		if v, err := strconv.Atoi(m[1]); err == nil {
			versions = append(versions, v)
		}
	}
	if len(versions) == 0 {
		return "", "", fmt.Errorf("no version directories found at %s", fedoraBaseURL)
	}

	// 2. Sort to get the highest version (e.g., 43)
	sort.Ints(versions)
	latestVer := versions[len(versions)-1]

	// 3. Construct the path to the Cloud images folder
	// URL Structure: .../releases/43/Cloud/x86_64/images/
	imagePageURL := fmt.Sprintf("%s%d/Cloud/x86_64/images/", fedoraBaseURL, latestVer)

	// 4. Scrape the image folder to find the exact filename
	resp2, err := http.Get(imagePageURL)
	if err != nil {
		return "", "", fmt.Errorf("failed to open image dir: %w", err)
	}
	defer resp2.Body.Close()
	body2, _ := io.ReadAll(resp2.Body)

	// Regex to match "Fedora-Cloud-Base-Generic-43-1.6.x86_64.qcow2"
	reImg := regexp.MustCompile(`href="(Fedora-Cloud-Base-Generic-[^"]+\.qcow2)"`)
	imgMatches := reImg.FindStringSubmatch(string(body2))

	if len(imgMatches) < 2 {
		return "", "", fmt.Errorf("qcow2 image not found in %s", imagePageURL)
	}

	filename := imgMatches[1]
	fullURL := imagePageURL + filename

	fmt.Printf("   -> [Dynamic] Found Fedora %d (%s)\n", latestVer, filename)
	return fullURL, filename, nil
}
