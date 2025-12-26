package cli

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

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

	// In the new architecture, Image/Size logic is inside Manager or Driver defaults for now
	// You can expand CreateServer arguments later
	fmt.Printf("Creating VM '%s' (Standard Plan)...\n", name)

	if err := a.mgr.CreateServer(name, "local"); err != nil {
		fmt.Printf("❌ Failed: %v\n", err)
	} else {
		fmt.Println("✅ VM Created Successfully!")
	}
}

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
	// This replaces the old "EnsureBaseImage" logic
	reader := bufio.NewReader(os.Stdin)

	fmt.Print("Logical Name (e.g., ubuntu-24.04): ")
	name, _ := reader.ReadString('\n')
	name = strings.TrimSpace(name)

	fmt.Print("Download URL: ")
	url, _ := reader.ReadString('\n')
	url = strings.TrimSpace(url)

	fmt.Println("Registering and Fetching...")

	// 1. Register
	if err := a.store.Register(name, url, ""); err != nil {
		fmt.Printf("❌ Registration Failed: %v\n", err)
		return
	}

	// 2. Trigger Download
	if _, err := a.store.Resolve(name); err != nil {
		fmt.Printf("❌ Download Failed: %v\n", err)
	} else {
		fmt.Println("✅ Image Ready!")
	}
}
