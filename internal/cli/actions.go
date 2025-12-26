package cli

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/Shaman786/vps-manager/internal/vm"
)

// ... (ListVMs remains same) ...
func (a *App) handleListVMs() {
	vms, err := a.mgr.ListServers()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	fmt.Println("\nNAME          STATUS      IP")
	fmt.Println("--------------------------------")
	for _, v := range vms {
		fmt.Printf("%-13s %-11s %s\n", v.Name, v.Status, v.IP)
	}
}

func (a *App) handleCreateVM() {
	reader := bufio.NewReader(os.Stdin)

	fmt.Print("Enter VM Name: ")
	name, _ := reader.ReadString('\n')
	name = strings.TrimSpace(name)

	fmt.Print("Image (default: ubuntu-24.04): ")
	image, _ := reader.ReadString('\n')
	image = strings.TrimSpace(image)
	if image == "" {
		image = "ubuntu-24.04"
	}

	fmt.Print("Plan [Starter/Professional]: ")
	plan, _ := reader.ReadString('\n')
	plan = strings.TrimSpace(plan)
	if plan == "" {
		plan = "Starter"
	}

	fmt.Print("Root Password: ")
	pass, _ := reader.ReadString('\n')
	pass = strings.TrimSpace(pass)

	// Build the Options Struct
	opts := vm.CreateOptions{
		Name:     name,
		Image:    image,
		PlanName: plan,
		Username: "root", // Defaulting to root for CLI simplicity
		Password: pass,
	}

	fmt.Printf("\n🚀 Creating %s (%s) on %s...\n", name, plan, image)

	if err := a.mgr.CreateServer(opts); err != nil {
		fmt.Printf("❌ Failed: %v\n", err)
	} else {
		fmt.Println("✅ VM Created Successfully!")
	}
}

// ... (Keep handleControlVM and handleDownloadImage as is) ...
func (a *App) handleControlVM() {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("VM Name: ")
	name, _ := reader.ReadString('\n')
	name = strings.TrimSpace(name)

	fmt.Print("Action (start/stop/reboot/delete): ")
	action, _ := reader.ReadString('\n')
	action = strings.TrimSpace(action)

	if err := a.mgr.PerformAction(name, action); err != nil {
		fmt.Printf("❌ Failed: %v\n", err)
	} else {
		fmt.Println("✅ Action completed.")
	}
}

func (a *App) handleDownloadImage() {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Logical Name (e.g., ubuntu-24.04): ")
	name, _ := reader.ReadString('\n')
	name = strings.TrimSpace(name)

	fmt.Print("Download URL: ")
	url, _ := reader.ReadString('\n')
	url = strings.TrimSpace(url)

	fmt.Println("Registering and Fetching...")
	if err := a.store.Register(name, url, ""); err != nil {
		fmt.Printf("❌ Registration Failed: %v\n", err)
		return
	}
	if _, err := a.store.Resolve(name); err != nil {
		fmt.Printf("❌ Download Failed: %v\n", err)
	} else {
		fmt.Println("✅ Image Ready!")
	}
}
