package vm

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type Manager struct {
	Config ManagerConfig
}

type ManagerConfig struct {
	BaseImageDir string
	VMDiskDir    string
	ConfigDir    string
}

func NewManager(cfg ManagerConfig) *Manager {
	// FIX A: Panic if we can't make directories (Critical Failure)
	dirs := []string{cfg.BaseImageDir, cfg.VMDiskDir, cfg.ConfigDir}
	for _, d := range dirs {
		if err := os.MkdirAll(d, 0o755); err != nil {
			panic(fmt.Sprintf("CRITICAL: Failed to create directory %s: %v", d, err))
		}
	}
	return &Manager{Config: cfg}
}

func (m *Manager) EnsureBaseImage(name, url, filename string) error {
	destPath := filepath.Join(m.Config.BaseImageDir, filename)
	if _, err := os.Stat(destPath); os.IsNotExist(err) {
		fmt.Printf("⬇️  Downloading Base Image: %s...\n", name)
		// Assuming you have a separate download function or logic here
		// For now, calling wget/curl for simplicity or your internal utils
		cmd := exec.Command("wget", "-O", destPath, url)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return cmd.Run()
	}
	return nil
}

func (m *Manager) CreateDisk(vmName, baseImage, size string) (string, error) {
	basePath := filepath.Join(m.Config.BaseImageDir, baseImage)
	diskPath := filepath.Join(m.Config.VMDiskDir, vmName+".qcow2")

	// Create QCOW2 with backing file
	cmd := exec.Command("qemu-img", "create", "-f", "qcow2", "-F", "qcow2", "-b", basePath, diskPath, size)
	if out, err := cmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("qemu-img failed: %s: %w", string(out), err)
	}
	return diskPath, nil
}

func (m *Manager) CreateISO(vmName, userData, metaData string) (string, error) {
	isoPath := filepath.Join(m.Config.ConfigDir, vmName+"-cidata.iso")
	userPath := filepath.Join(m.Config.ConfigDir, vmName+"-user.yaml")
	metaPath := filepath.Join(m.Config.ConfigDir, vmName+"-meta.yaml")

	if err := os.WriteFile(userPath, []byte(userData), 0o644); err != nil {
		return "", err
	}
	if err := os.WriteFile(metaPath, []byte(metaData), 0o644); err != nil {
		return "", err
	}

	cmd := exec.Command("cloud-localds", isoPath, userPath, metaPath)
	if out, err := cmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("cloud-localds failed: %s: %w", string(out), err)
	}

	// Cleanup temp config files
	_ = os.Remove(userPath)
	_ = os.Remove(metaPath)

	return isoPath, nil
}

func (m *Manager) Launch(name string, ram, cpu int, diskPath, isoPath, bridge string) error {
	// Network Configuration
	netBlock := "<interface type='network'><source network='default'/><model type='virtio'/></interface>"
	if bridge != "" {
		netBlock = fmt.Sprintf("<interface type='bridge'><source bridge='%s'/><model type='virtio'/></interface>", bridge)
	}

	xml := fmt.Sprintf(`
<domain type='kvm'>
  <name>%s</name>
  <memory unit='KiB'>%d</memory>
  <vcpu>%d</vcpu>
  <os>
    <type arch='x86_64'>hvm</type>
    <boot dev='hd'/>
  </os>
  <features><acpi/><apic/></features>
  <devices>
    <emulator>/usr/bin/qemu-system-x86_64</emulator>
    <disk type='file' device='disk'>
      <driver name='qemu' type='qcow2'/>
      <source file='%s'/>
      <target dev='vda' bus='virtio'/>
    </disk>
    <disk type='file' device='cdrom'>
      <driver name='qemu' type='raw'/>
      <source file='%s'/>
      <target dev='sda' bus='sata'/>
      <readonly/>
    </disk>
    %s
    <console type='pty'><target type='serial' port='0'/></console>
    <graphics type='vnc' port='-1' autoport='yes' listen='0.0.0.0'/>
  </devices>
</domain>`, name, ram*1024, cpu, diskPath, isoPath, netBlock)

	tmpFile := filepath.Join(m.Config.ConfigDir, name+".xml")

	// FIX B: Only return error (not string, error)
	if err := os.WriteFile(tmpFile, []byte(xml), 0o644); err != nil {
		return fmt.Errorf("failed to write XML: %w", err)
	}

	if out, err := exec.Command("virsh", "define", tmpFile).CombinedOutput(); err != nil {
		return fmt.Errorf("virsh define failed: %s: %w", string(out), err)
	}
	if out, err := exec.Command("virsh", "start", name).CombinedOutput(); err != nil {
		return fmt.Errorf("virsh start failed: %s: %w", string(out), err)
	}

	_ = os.Remove(tmpFile)
	return nil
}

func (m *Manager) GetVMInfo(name string) (string, string) {
	// Get State
	out, _ := exec.Command("virsh", "domstate", name).Output()
	state := strings.TrimSpace(string(out))

	// Get IP (Parsing logic simplified)
	out2, _ := exec.Command("virsh", "domifaddr", name, "--source", "agent").Output()
	ip := "Unknown (Wait/Agent)"
	if strings.Contains(string(out2), "ipv4") {
		// Basic parsing, improving this requires regex
		parts := strings.Fields(string(out2))
		if len(parts) > 3 {
			ip = parts[len(parts)-1] // very rough heuristic
		}
	}
	return ip, state
}

func (m *Manager) EditResources(name string, ram, cpu int) error {
	ramKB := ram * 1024
	if err := exec.Command("virsh", "setmaxmem", name, fmt.Sprint(ramKB), "--config").Run(); err != nil {
		return err
	}
	if err := exec.Command("virsh", "setmem", name, fmt.Sprint(ramKB), "--config").Run(); err != nil {
		return err
	}
	if err := exec.Command("virsh", "setvcpus", name, fmt.Sprint(cpu), "--maximum", "--config").Run(); err != nil {
		return err
	}
	return exec.Command("virsh", "setvcpus", name, fmt.Sprint(cpu), "--config").Run()
}

func (m *Manager) CreateBridgeNetwork(name, iface string) error {
	xml := fmt.Sprintf(`
<network>
  <name>%s</name>
  <forward mode='bridge'>
    <interface dev='%s'/>
  </forward>
</network>`, name, iface)

	tmp := "/tmp/" + name + ".xml"
	_ = os.WriteFile(tmp, []byte(xml), 0o644)
	// exec.Command("virsh", "net-define", tmp).Run()
	// exec.Command("virsh", "net-start", name).Run()
	// return exec.Command("virsh", "net-autostart", name).Run()
	// // NEW CODE (With Error Checks):
	if err := exec.Command("virsh", "net-define", tmp).Run(); err != nil {
		return fmt.Errorf("failed to define network: %w", err)
	}
	if err := exec.Command("virsh", "net-start", name).Run(); err != nil {
		return fmt.Errorf("failed to start network: %w", err)
	}
	if err := exec.Command("virsh", "net-autostart", name).Run(); err != nil {
		return fmt.Errorf("failed to enable autostart: %w", err)
	}
	return nil
}

// Ensure DeleteVM is present here as discussed in previous steps
func (m *Manager) DeleteVM(vmName string) error {
	_ = exec.Command("virsh", "destroy", vmName).Run()
	if err := exec.Command("virsh", "undefine", vmName).Run(); err != nil {
		return fmt.Errorf("failed to undefine: %w", err)
	}

	// Clean files
	_ = os.Remove(filepath.Join(m.Config.VMDiskDir, vmName+".qcow2"))
	_ = os.Remove(filepath.Join(m.Config.ConfigDir, vmName+".xml"))
	_ = os.Remove(filepath.Join(m.Config.ConfigDir, vmName+"-cidata.iso"))
	return nil
}
