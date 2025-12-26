package kvm

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/Shaman786/vps-manager/internal/core"
	"github.com/Shaman786/vps-manager/internal/images"
)

type KVMDriver struct {
	ImageStore *images.Store
	DiskDir    string
	ConfigDir  string
}

func NewKVMDriver(store *images.Store, diskDir, confDir string) *KVMDriver {
	return &KVMDriver{
		ImageStore: store,
		DiskDir:    diskDir,
		ConfigDir:  confDir,
	}
}

func (k *KVMDriver) Name() string { return "KVM-Libvirt-v2" }

func (k *KVMDriver) CreateVM(cfg core.VMConfig) error {
	// 1. Resolve Image
	imgInfo, err := k.ImageStore.Resolve(cfg.Image)
	if err != nil {
		return fmt.Errorf("image resolve failed: %w", err)
	}

	// 2. Prepare Paths
	diskPath := filepath.Join(k.DiskDir, cfg.Name+".qcow2")
	sizeStr := fmt.Sprintf("%dG", cfg.DiskSize)

	// 3. Create Disk
	cmd := exec.Command("qemu-img", "create", "-f", "qcow2", "-F", "qcow2", "-b", imgInfo.LocalPath, diskPath, sizeStr)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("disk create failed: %s", string(out))
	}

	// 4. Cloud-Init
	isoPath, err := k.createCloudInitISO(cfg.Name, cfg.UserData, cfg.MetaData)
	if err != nil {
		return err
	}

	// 5. Launch
	return k.launchWithXML(cfg.Name, cfg.RAM, cfg.CPUCores, diskPath, isoPath, cfg.Network)
}

func (k *KVMDriver) DeleteVM(id string) error {
	_ = exec.Command("virsh", "destroy", id).Run()
	_ = exec.Command("virsh", "undefine", id).Run()
	_ = os.Remove(filepath.Join(k.DiskDir, id+".qcow2"))
	_ = os.Remove(filepath.Join(k.ConfigDir, id+"-cidata.iso"))
	return nil
}

func (k *KVMDriver) StartVM(id string) error {
	return exec.Command("virsh", "start", id).Run()
}

func (k *KVMDriver) StopVM(id string) error {
	return exec.Command("virsh", "destroy", id).Run()
}

func (k *KVMDriver) Reboot(id string) error {
	return exec.Command("virsh", "reboot", id).Run()
}

func (k *KVMDriver) ListVMs() ([]string, error) {
	out, err := exec.Command("virsh", "list", "--all", "--name").Output()
	if err != nil {
		return nil, err
	}
	var vms []string
	for _, line := range strings.Split(string(out), "\n") {
		if strings.TrimSpace(line) != "" {
			vms = append(vms, strings.TrimSpace(line))
		}
	}
	return vms, nil
}

func (k *KVMDriver) GetVMInfo(id string) (core.VMState, error) {
	stateOut, _ := exec.Command("virsh", "domstate", id).Output()

	// Get IP
	ip := "Unknown"
	out, _ := exec.Command("virsh", "domifaddr", id, "--source", "agent").Output()
	if strings.Contains(string(out), "ipv4") {
		fields := strings.Fields(string(out))
		if len(fields) >= 4 {
			ip = strings.Split(fields[3], "/")[0]
		}
	}

	return core.VMState{
		ID:     id,
		Name:   id,
		Status: strings.TrimSpace(string(stateOut)),
		IP:     ip,
	}, nil
}

func (k *KVMDriver) GetMetrics(id string) (map[string]float64, error) {
	return map[string]float64{}, nil
}

// --- PRIVATE HELPERS ---

func (k *KVMDriver) createCloudInitISO(name, user, meta string) (string, error) {
	isoPath := filepath.Join(k.ConfigDir, name+"-cidata.iso")
	userPath := filepath.Join(k.ConfigDir, name+"-user.yaml")
	metaPath := filepath.Join(k.ConfigDir, name+"-meta.yaml")

	_ = os.WriteFile(userPath, []byte(user), 0644)
	_ = os.WriteFile(metaPath, []byte(meta), 0644)

	cmd := exec.Command("cloud-localds", isoPath, userPath, metaPath)
	out, err := cmd.CombinedOutput()

	_ = os.Remove(userPath)
	_ = os.Remove(metaPath)

	if err != nil {
		return "", fmt.Errorf("cloud-localds failed: %s", string(out))
	}
	return isoPath, nil
}

func (k *KVMDriver) launchWithXML(name string, ramMB, cpu int, disk, iso, bridge string) error {
	ramKB := ramMB * 1024
	// netXML := fmt.Sprintf("<interface type='network'><source network='default'/><model type='virtio'/></interface>")
	netXML := "<interface type='network'><source network='default'/><model type='virtio'/></interface>"
	if bridge != "default" && bridge != "" {
		netXML = fmt.Sprintf("<interface type='bridge'><source bridge='%s'/><model type='virtio'/></interface>", bridge)
	}

	xml := fmt.Sprintf(`
<domain type='kvm'>
  <name>%s</name>
  <memory unit='KiB'>%d</memory>
  <vcpu>%d</vcpu>
  <os><type arch='x86_64'>hvm</type><boot dev='hd'/></os>
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
</domain>`, name, ramKB, cpu, disk, iso, netXML)

	cmd := exec.Command("virsh", "define", "/dev/stdin")
	cmd.Stdin = strings.NewReader(xml)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("virsh define failed: %s", string(out))
	}
	return exec.Command("virsh", "start", name).Run()
}
