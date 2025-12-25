package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

func main() {
	// SECURITY FIX: Read IP from Environment Variable
	serverIP := os.Getenv("VPS_IP")
	if serverIP == "" {
		log.Fatal("❌ CRITICAL ERROR: VPS_IP environment variable is not set.")
	}

	webhookURL := fmt.Sprintf("http://%s:8080/webhook", serverIP)

	fmt.Println("🕵️  WATCHER STARTED: Scraping official mirrors...")
	fmt.Printf("   Target Server: %s\n", serverIP)
	fmt.Println("---------------------------------------------------")

	// 1. Find Ubuntu 24.04 Latest Build
	if url, ver, err := scrapeUbuntu("24.04"); err != nil {
		log.Printf("❌ Ubuntu Scraping Failed: %v", err)
	} else {
		triggerWebhook(webhookURL, "Ubuntu", ver, url)
	}

	// 2. Find Rocky 9 Latest Build
	if url, ver, err := scrapeRocky("9"); err != nil {
		log.Printf("❌ Rocky Scraping Failed: %v", err)
	} else {
		triggerWebhook(webhookURL, "Rocky", ver, url)
	}
}

// --- SCRAPERS ---

func scrapeUbuntu(releaseBase string) (string, string, error) {
	baseURL := fmt.Sprintf("https://cloud-images.ubuntu.com/releases/%s/release/", releaseBase)
	doc, err := fetchHTML(baseURL)
	if err != nil {
		return "", "", err
	}

	var foundURL string
	doc.Find("a").Each(func(i int, s *goquery.Selection) {
		href, _ := s.Attr("href")
		if strings.HasSuffix(href, "-server-cloudimg-amd64.img") {
			foundURL = baseURL + href
		}
	})

	if foundURL == "" {
		return "", "", fmt.Errorf("file not found")
	}
	return foundURL, releaseBase, nil
}

func scrapeRocky(majorVer string) (string, string, error) {
	baseURL := fmt.Sprintf("https://download.rockylinux.org/pub/rocky/%s/images/x86_64/", majorVer)
	doc, err := fetchHTML(baseURL)
	if err != nil {
		return "", "", err
	}

	var foundURL string
	doc.Find("a").Each(func(i int, s *goquery.Selection) {
		href, _ := s.Attr("href")
		if strings.Contains(href, "GenericCloud-Base.latest") && strings.HasSuffix(href, ".qcow2") {
			foundURL = baseURL + href
		}
	})

	if foundURL == "" {
		return "", "", fmt.Errorf("file not found")
	}
	return foundURL, majorVer, nil
}

func fetchHTML(url string) (*goquery.Document, error) {
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return goquery.NewDocumentFromReader(resp.Body)
}

func triggerWebhook(targetURL, distro, version, imgUrl string) {
	fmt.Printf("   ✅ FOUND: %s %s\n", distro, version)

	payload := map[string]string{
		"distro":  distro,
		"version": version,
		"url":     imgUrl,
	}
	data, _ := json.Marshal(payload)

	resp, err := http.Post(targetURL, "application/json", bytes.NewBuffer(data))
	if err != nil {
		fmt.Printf("   ❌ Webhook Failed: %v\n", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == 200 {
		fmt.Println("   🎉 SUCCESS: Server accepted update.")
	} else {
		fmt.Printf("   ⚠️  Server rejected: Status %d\n", resp.StatusCode)
	}
}
