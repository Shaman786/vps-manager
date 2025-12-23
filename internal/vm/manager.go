// Package vm handles the creation, storage management, and lifecycle
// of KVM virtual machines using QEMU and Libvirt tools.
package vm

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/Shaman786/vps-manager/internal/utils"
)

type ManagerConfig struct {
	BaseImageDir string
	VMDiskDir    string
	ConfigDir    string
}

type Manager struct {
	Config ManagerConfig
}

func NewManager(cfg ManagerConfig) *Manager {
	dirs := []string{cfg.BaseImageDir, cfg.VMDiskDir, cfg.ConfigDir}
	for _, d := range dirs {
		os.MkdirAll(d, 0o755)
	}
	return &Manager{Config: cfg}
}

// --- 1. PROVISIONING ---

func (m *Manager) EnsureBaseImage(name, url, filename string) error {
	path := filepath.Join(m.Config.BaseImageDir, filename)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		fmt.Printf("⬇️  Downloading Base Image: %s...\n", name)
		return utils.DownloadFile(url, path)
	}
	return nil
}

func (m *Manager) CreateDisk(vmName, baseFilename, size string) (string, error) {
	basePath := filepath.Join(m.Config.BaseImageDir, baseFilename)
	diskPath := filepath.Join(m.Config.VMDiskDir, vmName+".qcow2")

	// Create a Copy-on-Write (QCOW2) disk linked to the base image
	cmd := exec.Command("qemu-img", "create", "-f", "qcow2", "-F", "qcow2", "-b", basePath, diskPath, size)
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("disk creation failed: %w", err)
	}
	return diskPath, nil
}

func (m *Manager) CreateISO(vmName, userData, metaData string) (string, error) {
	userPath := filepath.Join(m.Config.ConfigDir, vmName+"-user.yaml")
	metaPath := filepath.Join(m.Config.ConfigDir, vmName+"-meta.yaml")
	isoPath := filepath.Join(m.Config.ConfigDir, vmName+"-cidata.iso")

	// Write the config files to disk
	if err := os.WriteFile(userPath, []byte(userData), 0o644); err != nil {
		return "", err
	}
	if err := os.WriteFile(metaPath, []byte(metaData), 0o644); err != nil {
		return "", err
	}

	// Generate ISO using cloud-localds
	cmd := exec.Command("cloud-localds", isoPath, userPath, metaPath)
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("iso generation failed: %w", err)
	}
	return isoPath, nil
}

func (m *Manager) Launch(vmName string, ram, cpu int, diskPath, isoPath string, bridgeName string) error {
	// Network Selection: Default to NAT, or use Bridge if provided
	netArg := "network=default,model=virtio"
	if bridgeName != "" {
		netArg = fmt.Sprintf("bridge=%s,model=virtio", bridgeName)
	}

	cmd := exec.Command("virt-install",
		"--name", vmName,
		"--memory", fmt.Sprintf("%d", ram),
		"--vcpus", fmt.Sprintf("%d", cpu),
		"--disk", "path="+diskPath+",device=disk,bus=virtio",
		"--disk", "path="+isoPath+",device=cdrom", // Cloud-Init ISO
		"--os-variant", "ubuntu22.04", // Safe default
		"--network", netArg,
		"--graphics", "vnc,listen=0.0.0.0", // ENABLE VNC (Remote Desktop)
		"--import",        // Skip OS install (we are booting an image)
		"--noautoconsole", // Detach immediately
	)

	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("launch failed: %s: %w", string(output), err)
	}
	return nil
}

// --- 2. MANAGEMENT TOOLS (New Features) ---

// DeleteVM destroys the VM and deletes all associated files
func (m *Manager) DeleteVM(vmName string) error {
	fmt.Printf("🗑️  Destroying %s...\n", vmName)

	// Force Stop & Undefine from Libvirt
	exec.Command("virsh", "destroy", vmName).Run()
	exec.Command("virsh", "undefine", vmName).Run()

	// Delete Disk & Config Files
	os.Remove(filepath.Join(m.Config.VMDiskDir, vmName+".qcow2"))
	os.Remove(filepath.Join(m.Config.ConfigDir, vmName+"-cidata.iso"))
	os.Remove(filepath.Join(m.Config.ConfigDir, vmName+"-user.yaml"))
	os.Remove(filepath.Join(m.Config.ConfigDir, vmName+"-meta.yaml"))

	return nil
}

// EditResources scales RAM/CPU (requires reboot usually)
func (m *Manager) EditResources(vmName string, ram, cpu int) error {
	// --config writes to XML (persistent), --live tries to change running state
	// Note: RAM is in KiB for virsh
	if err := exec.Command("virsh", "setmem", vmName, fmt.Sprintf("%d", ram*1024), "--config").Run(); err != nil {
		return err
	}
	if err := exec.Command("virsh", "setvcpus", vmName, fmt.Sprintf("%d", cpu), "--config", "--maximum").Run(); err != nil {
		return err
	}
	if err := exec.Command("virsh", "setvcpus", vmName, fmt.Sprintf("%d", cpu), "--config").Run(); err != nil {
		return err
	}
	return nil
}

// GetVMInfo fetches the Real IP Address using ARP/DHCP leases
func (m *Manager) GetVMInfo(vmName string) (string, string) {
	// Get State (running, shut off, etc)
	stateOut, _ := exec.Command("virsh", "domstate", vmName).Output()
	state := strings.TrimSpace(string(stateOut))

	// Get IP Address
	// This asks the network driver "What IP did you give this VM?"
	ipOut, err := exec.Command("virsh", "domifaddr", vmName, "--source", "lease").Output()
	if err != nil {
		return "Waiting for IP...", state
	}

	// Output format is like: "ipv4  192.168.122.50/24"
	output := string(ipOut)
	if strings.Contains(output, "ipv4") {
		fields := strings.Fields(output)
		for i, f := range fields {
			if f == "ipv4" && i+1 < len(fields) {
				// Strip the CIDR mask (e.g., /24) if present
				ip := fields[i+1]
				if idx := strings.Index(ip, "/"); idx != -1 {
					return ip[:idx], state
				}
				return ip, state
			}
		}
	}
	return "Scanning...", state
}

// CreateBridgeNetwork sets up a new Libvirt network that bridges to a physical interface
func (m *Manager) CreateBridgeNetwork(netName, hostInterface string) error {
	// XML Definition for a Bridge Network
	xml := fmt.Sprintf(`
<network>
  <name>%s</name>
  <forward mode='bridge'>
    <interface dev='%s'/>
  </forward>
</network>`, netName, hostInterface)

	tmpFile := "/tmp/net-bridge.xml"
	os.WriteFile(tmpFile, []byte(xml), 0o644)

	// Define, Start, and Autostart the network
	if err := exec.Command("virsh", "net-define", tmpFile).Run(); err != nil {
		return err
	}
	if err := exec.Command("virsh", "net-start", netName).Run(); err != nil {
		return err
	}
	if err := exec.Command("virsh", "net-autostart", netName).Run(); err != nil {
		return err
	}

	return nil
}
