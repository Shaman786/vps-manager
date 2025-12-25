package cli

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"syscall"

	"github.com/Shaman786/vps-manager/internal/images"
	"github.com/Shaman786/vps-manager/internal/vm"
	"golang.org/x/term"
)

type App struct {
	mgr    *vm.Manager
	reader *bufio.Reader
}

func NewApp(mgr *vm.Manager) *App {
	return &App{
		mgr:    mgr,
		reader: bufio.NewReader(os.Stdin),
	}
}

func (a *App) ShowMainMenu() {
	// Auto-refresh on start so the catalog isn't empty
	if err := images.RefreshCatalog(); err != nil {
		fmt.Println("⚠️ Warning: Could not update image catalog.")
		a.pause()
	}

	// 1. Define the structure of a menu item
	type MenuItem struct {
		Label  string
		Action func()
	}

	// 2. Define your menu items in a single list
	menu := []MenuItem{
		{
			Label:  "Create New VPS",
			Action: a.CreateVPS,
		},
		{
			Label:  "List All VPS",
			Action: a.ListVPS,
		},
		{
			Label:  "Manage VPS (Delete / Scale / VNC)",
			Action: a.ManageVPS,
		},
		{
			Label:  "Network Tools",
			Action: a.CreateBridge,
		},
		{
			Label: "Refresh Catalog",
			Action: func() { // Anonymous function for custom logic
				if err := images.RefreshCatalog(); err != nil {
					fmt.Printf("❌ Failed: %v\n", err)
				} else {
					fmt.Println("✅ Catalog updated.")
				}
				a.pause() // Helper to let user see the message
			},
		},
		{
			Label: "Exit",
			Action: func() {
				fmt.Println("Goodbye!")
				os.Exit(0)
			},
		},
	}

	// 3. The Main Loop
	for {
		a.clearScreen() // Makes it look like a real app
		fmt.Println("==========================================")
		fmt.Println("       🚀 HOSTPALACE VPS MANAGER         ")
		fmt.Println("==========================================")

		// Dynamic Print
		for i, item := range menu {
			fmt.Printf("%d. %s\n", i+1, item.Label)
		}

		input := a.readInput("\nSelect Option: ")

		// Convert string input to integer index
		var choice int
		if _, err := fmt.Sscanf(input, "%d", &choice); err == nil {
			// Check bounds (1 to length of menu)
			if choice >= 1 && choice <= len(menu) {
				// EXECUTE THE ACTION DIRECTLY
				menu[choice-1].Action()

				// Pause after action if it wasn't the Exit command, so user can see result
				// (Optional: You can remove this if your actions handle their own pauses)
				if menu[choice-1].Label != "Exit" && menu[choice-1].Label != "Create New VPS" {
					a.pause()
				}
				continue
			}
		}

		fmt.Println("❌ Invalid choice. Try again.")
		a.pause()
	}
}

// --- HELPERS ---

// Clears the terminal screen
func (a *App) clearScreen() {
	fmt.Print("\033[H\033[2J")
}

// Pauses so user can read output before screen clears
func (a *App) pause() {
	fmt.Println("\nPress Enter to continue...")
	_, _ = a.reader.ReadString('\n')
}

// Helper to read simple string input
func (a *App) readInput(prompt string) string {
	fmt.Print(prompt)
	input, _ := a.reader.ReadString('\n')
	return strings.TrimSpace(input)
}

// Helper to secure read password
func (a *App) readPasswordConfirm(prompt string) string {
	for {
		fmt.Print(prompt)
		p1, _ := term.ReadPassword(int(syscall.Stdin))
		fmt.Println()
		fmt.Print("Confirm Password: ")
		p2, _ := term.ReadPassword(int(syscall.Stdin))
		fmt.Println()

		if string(p1) == "" {
			continue
		}
		if string(p1) == string(p2) {
			return string(p1)
		}
		fmt.Println("❌ Passwords do not match.")
	}
}
