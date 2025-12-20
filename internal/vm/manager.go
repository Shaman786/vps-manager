// Package vm handles the creation, storage management, and lifecycle
// of KVM virtual machines using QEMU and Libvirt tools.
package vm

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/Shaman786/vps-manager/internal/utils"
)

// ManagerConfig holds the storage paths.
// Using a struct here makes the code extensible (e.g., adding network paths later).
type ManagerConfig struct {
	BaseImageDir string
	VMDiskDir    string
	ConfigDir    string
}

// Manager is the main controller for VM operations.
type Manager struct {
	Config ManagerConfig
}

// NewManager creates a new instance and ensures storage directories exist.
func NewManager(cfg ManagerConfig) *Manager {
	// Automatically create the necessary folders if they don't exist
	dirs := []string{cfg.BaseImageDir, cfg.VMDiskDir, cfg.ConfigDir}
	for _, d := range dirs {
		// 0755 means: Owner can read/write/execute, everyone else can read/execute.
		os.MkdirAll(d, 0o755)
	}
	return &Manager{Config: cfg}
}

// EnsureBaseImage checks if the OS image exists; if not, it downloads it.
func (m *Manager) EnsureBaseImage(name, url, filename string) error {
	path := filepath.Join(m.Config.BaseImageDir, filename)

	// Check if file exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		fmt.Printf("Base image missing. Downloading %s...\n", name)
		// We use the Utils package you just wrote!
		return utils.DownloadFile(url, path)
	}

	// If it exists, we do nothing.
	return nil
}

// CreateDisk creates a QCOW2 disk linked to the base image (Copy-on-Write).
func (m *Manager) CreateDisk(vmName, baseFilename, size string) (string, error) {
	basePath := filepath.Join(m.Config.BaseImageDir, baseFilename)
	diskPath := filepath.Join(m.Config.VMDiskDir, vmName+".qcow2")

	// qemu-img create -f qcow2 -F qcow2 -b <BACKING_FILE> <NEW_DISK> <SIZE>
	cmd := exec.Command("qemu-img", "create",
		"-f", "qcow2",
		"-F", "qcow2",
		"-b", basePath,
		diskPath,
		size,
	)

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to create disk: %w", err)
	}
	return diskPath, nil
}

// CreateISO writes the Cloud-Init YAML to disk and converts it to an ISO.
func (m *Manager) CreateISO(vmName, yamlContent string) (string, error) {
	yamlPath := filepath.Join(m.Config.ConfigDir, vmName+"-user-data.yaml")
	isoPath := filepath.Join(m.Config.ConfigDir, vmName+"-cidata.iso")

	// 1. Write the YAML file
	if err := os.WriteFile(yamlPath, []byte(yamlContent), 0o644); err != nil {
		return "", fmt.Errorf("failed to write config file: %w", err)
	}

	// 2. Convert to ISO using cloud-localds
	cmd := exec.Command("cloud-localds", isoPath, yamlPath)
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to generate ISO: %w", err)
	}

	return isoPath, nil
}

// Launch triggers the actual KVM instance via virt-install.
func (m *Manager) Launch(vmName string, ram, cpu int, diskPath, isoPath string) error {
	cmd := exec.Command("virt-install",
		"--name", vmName,
		"--memory", fmt.Sprintf("%d", ram),
		"--vcpus", fmt.Sprintf("%d", cpu),
		"--disk", "path="+diskPath+",device=disk,bus=virtio",
		"--disk", "path="+isoPath+",device=cdrom", // Attach the Cloud-Init ISO
		"--os-variant", "ubuntu22.04", // Safe default for Linux
		"--network", "network=default,model=virtio",
		"--graphics", "none",
		"--import",        // Skip OS install, just boot
		"--noautoconsole", // Don't attach terminal immediately
	)

	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("launch failed: %s: %w", string(output), err)
	}

	return nil
}
