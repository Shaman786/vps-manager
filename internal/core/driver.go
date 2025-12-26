package core

// VMConfig defines the "Order" from the user
type VMConfig struct {
	Name     string
	Image    string // Logical name: "ubuntu-24.04"
	CPUCores int
	RAM      int // MB
	DiskSize int // GB
	Network  string
	UserData string // Cloud-init
	MetaData string
}

// VMState defines what a running VM looks like
type VMState struct {
	ID     string
	Name   string
	Status string // RUNNING, STOPPED
	IP     string
}

// HypervisorDriver is the Interface our Manager talks to
type HypervisorDriver interface {
	Name() string

	// Lifecycle
	CreateVM(config VMConfig) error
	DeleteVM(id string) error
	StartVM(id string) error
	StopVM(id string) error
	Reboot(id string) error

	// Info
	ListVMs() ([]string, error)
	GetVMInfo(id string) (VMState, error)
	GetMetrics(id string) (map[string]float64, error)
}
