package vm

import (
	"fmt"

	"github.com/Shaman786/vps-manager/internal/core"
)

type Manager struct {
	Driver core.HypervisorDriver
}

// NewManager now accepts the Driver interface!
// This FIXES the error in main.go
func NewManager(driver core.HypervisorDriver) *Manager {
	return &Manager{
		Driver: driver,
	}
}

// --- HIGH LEVEL BUSINESS LOGIC ---

// CreateServer handles the "Business Logic" of creating a VM
// This FIXES "a.mgr.CreateServer undefined"
func (m *Manager) CreateServer(name, region string) error {
	// 1. Define Standard Plan (In real life, fetch this from DB)
	config := core.VMConfig{
		Name:     name,
		Image:    "ubuntu-24.04", // This triggers the Registry lookup
		CPUCores: 1,
		RAM:      1024,      // 1GB
		DiskSize: 10,        // 10GB
		Network:  "default", // or "virbr0"
		UserData: fmt.Sprintf(`#cloud-config
hostname: %s
ssh_pwauth: True
password: password
chpasswd: { expire: False }`, name),
		MetaData: fmt.Sprintf("instance-id: %s", name),
	}

	fmt.Printf("manager: requesting %s to create VM '%s'...\n", m.Driver.Name(), name)
	return m.Driver.CreateVM(config)
}

// ListServers returns the nice struct needed by the CLI
// This FIXES "a.mgr.ListServers undefined"
func (m *Manager) ListServers() ([]core.VMState, error) {
	ids, err := m.Driver.ListVMs()
	if err != nil {
		return nil, err
	}

	var list []core.VMState
	for _, id := range ids {
		// We ignore individual errors to keep the list flowing
		if info, err := m.Driver.GetVMInfo(id); err == nil {
			list = append(list, info)
		}
	}
	return list, nil
}

// PerformAction handles Start/Stop/Reboot
// This FIXES "a.mgr.PerformAction undefined"
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
