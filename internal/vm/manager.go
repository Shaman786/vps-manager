package vm

import (
	"fmt"
	"strings"

	"github.com/Shaman786/vps-manager/internal/cloudinit"
	"github.com/Shaman786/vps-manager/internal/core"
	"github.com/Shaman786/vps-manager/internal/plans"
)

type Manager struct {
	Driver core.HypervisorDriver
}

func NewManager(driver core.HypervisorDriver) *Manager {
	return &Manager{Driver: driver}
}

// CreateOptions packages all the user's desires
type CreateOptions struct {
	Name     string
	Image    string
	PlanName string // "Starter", "Professional"
	Username string // "admin"
	Password string // "secret123"
}

// CreateServer now orchestrates Plans + CloudInit + Driver
func (m *Manager) CreateServer(opts CreateOptions) error {
	// 1. FIND THE PLAN
	// We look up CPU/RAM/Disk from your plans.go file
	var selectedPlan plans.VMPlan
	found := false
	for _, p := range plans.Available {
		if strings.EqualFold(p.Name, opts.PlanName) {
			selectedPlan = p
			found = true
			break
		}
	}
	if !found {
		// Fallback to first plan if invalid
		selectedPlan = plans.Available[0]
		fmt.Printf("⚠️  Plan '%s' not found. Defaulting to %s.\n", opts.PlanName, selectedPlan.Name)
	}

	// 2. PARSE DISK SIZE
	// plans.go has "10G", core needs int(10). Simple parse:
	var diskInt int
	fmt.Sscanf(selectedPlan.Disk, "%dG", &diskInt)

	// 3. GENERATE CLOUD-INIT
	// We use your new generator.go logic
	configData := cloudinit.ConfigData{
		Hostname:       opts.Name,
		Username:       opts.Username,
		UserPass:       opts.Password,
		RootPass:       opts.Password, // Sync root pass for now
		AllowRootLogin: true,
	}

	userData, err := cloudinit.Generate(configData)
	if err != nil {
		return fmt.Errorf("failed to generate cloud-config: %w", err)
	}

	// 4. ASSEMBLE THE ORDER
	config := core.VMConfig{
		Name:     opts.Name,
		Image:    opts.Image,
		CPUCores: selectedPlan.CPUs,
		RAM:      selectedPlan.RAM,
		DiskSize: diskInt,
		Network:  "default",
		UserData: userData,
		MetaData: fmt.Sprintf("instance-id: %s\nlocal-hostname: %s", opts.Name, opts.Name),
	}

	fmt.Printf("📦 PROVISIONING: %s | %s | %s\n", opts.Name, selectedPlan.Name, opts.Image)
	return m.Driver.CreateVM(config)
}

// ... (ListServers and PerformAction remain unchanged) ...
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
