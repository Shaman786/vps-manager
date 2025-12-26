package cli

import (
	"fmt"

	"github.com/Shaman786/vps-manager/internal/images"
	"github.com/Shaman786/vps-manager/internal/vm"
)

type App struct {
	mgr   *vm.Manager
	store *images.Store // Added Store here
}

// Update Constructor to accept store
func NewApp(mgr *vm.Manager, store *images.Store) *App {
	return &App{
		mgr:   mgr,
		store: store,
	}
}

func (a *App) ShowMainMenu() {
	for {
		fmt.Println("\n--- HOST-PALACE VPS MANAGER ---")
		fmt.Println("1. List VMs")
		fmt.Println("2. Create VM")
		fmt.Println("3. Control VM (Start/Stop/Reboot)")
		fmt.Println("4. Download/Register Image")
		fmt.Println("5. Exit")
		fmt.Print("Select: ")

		var choice string
		fmt.Scanln(&choice)

		switch choice {
		case "1":
			a.handleListVMs()
		case "2":
			a.handleCreateVM()
		case "3":
			a.handleControlVM()
		case "4":
			a.handleDownloadImage()
		case "5":
			return
		default:
			fmt.Println("Invalid choice")
		}
	}
}
