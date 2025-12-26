package vm

import (
	"fmt"

	"github.com/Shaman786/vps-manager/internal/core"
)

type Manager struct {
	Driver core.HypervisorDriver
}

func NewManager(driver core.HypervisorDriver) *Manager {
	return &Manager{Driver: driver}
}

// Update: Now accepts 'image' argument
func (m *Manager) CreateServer(name, image, region string) error {

	// Default to Ubuntu if empty
	if image == "" {
		image = "ubuntu-24.04"
	}

	config := core.VMConfig{
		Name:     name,
		Image:    image, // <--- Uses the argument now
		CPUCores: 1,
		RAM:      1024,
		DiskSize: 10,
		Network:  "default",
		UserData: fmt.Sprintf(`#cloud-config
hostname: %s
ssh_pwauth: True
password: password
chpasswd: { expire: False }`, name),
		MetaData: fmt.Sprintf("instance-id: %s", name),
	}

	fmt.Printf("manager: requesting %s to create VM '%s' (%s)...\n", m.Driver.Name(), name, image)
	return m.Driver.CreateVM(config)
}

func (m *Manager) ListServers() ([]core.VMState, error) {
	ids, err := m.Driver.ListVMs()
	if err != nil {
		return nil, err
	}
	var list []core.VMState
	for _, id := range ids {
		if info, err := m.Driver.GetVMInfo(id); err == nil {
			list = append(list, info)
		}
	}
	return list, nil
}

func (m *Manager) PerformAction(id, action string) error {
	switch action {
	case "start":
		return m.Driver.StartVM(id)
	case "stop":
		return m.Driver.StopVM(id)
	case "reboot":
		return m.Driver.Reboot(id)
	case "delete":
		return m.Driver.DeleteVM(id)
	}
	return fmt.Errorf("unknown action: %s", action)
}
