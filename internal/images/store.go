package images

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sync"
)

// ImageInfo represents a single entry in our local registry
type ImageInfo struct {
	Name      string `json:"name"`       // Logical: "ubuntu-24.04"
	URL       string `json:"url"`        // Remote: "https://cloud-images..."
	LocalPath string `json:"local_path"` // Physical: "/host-data/images/cache/..."
	Checksum  string `json:"checksum"`   // Integrity check
	Status    string `json:"status"`     // READY, DOWNLOADING, ERROR
}

// Store handles the logic of mapping Names -> Files
type Store struct {
	RegistryPath string // Path to images.json
	CacheDir     string // Path to store .qcow2 files
	images       map[string]ImageInfo
	mu           sync.RWMutex
}

func NewStore(registryPath, cacheDir string) (*Store, error) {
	s := &Store{
		RegistryPath: registryPath,
		CacheDir:     cacheDir,
		images:       make(map[string]ImageInfo),
	}

	// Ensure dirs exist
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return nil, err
	}

	// Load existing DB
	s.loadRegistry()
	return s, nil
}

// Resolve finds the image. If missing locally but URL is known, it triggers download.
func (s *Store) Resolve(name string) (ImageInfo, error) {
	s.mu.RLock()
	img, exists := s.images[name]
	s.mu.RUnlock()

	if !exists {
		return ImageInfo{}, fmt.Errorf("image '%s' not registered", name)
	}

	// Check if file physically exists
	if _, err := os.Stat(img.LocalPath); os.IsNotExist(err) {
		// Auto-Heal: Download if missing
		if err := s.downloadImage(&img); err != nil {
			return ImageInfo{}, fmt.Errorf("failed to download image: %v", err)
		}
	}

	return img, nil
}

// Register adds/updates an image in the DB
func (s *Store) Register(name, url, checksum string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	filename := fmt.Sprintf("%s.qcow2", name)
	localPath := filepath.Join(s.CacheDir, filename)

	s.images[name] = ImageInfo{
		Name:      name,
		URL:       url,
		LocalPath: localPath,
		Checksum:  checksum,
		Status:    "PENDING",
	}

	return s.saveRegistry()
}

// Internal: Downloads the file
func (s *Store) downloadImage(img *ImageInfo) error {
	fmt.Printf("⬇️  Pulling Image: %s from %s...\n", img.Name, img.URL)

	resp, err := http.Get(img.URL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	out, err := os.Create(img.LocalPath)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return err
	}

	// Update Status
	s.mu.Lock()
	temp := s.images[img.Name]
	temp.Status = "READY"
	s.images[img.Name] = temp
	s.saveRegistry()
	s.mu.Unlock()

	fmt.Printf("✅ Image Ready: %s\n", img.LocalPath)
	return nil
}

func (s *Store) loadRegistry() {
	data, err := os.ReadFile(s.RegistryPath)
	if err == nil {
		_ = json.Unmarshal(data, &s.images)
	}
}

func (s *Store) saveRegistry() error {
	data, _ := json.MarshalIndent(s.images, "", "  ")
	return os.WriteFile(s.RegistryPath, data, 0644)
}
