package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"syscall"

	"github.com/Shaman786/vps-manager/internal/cloudinit"
	"github.com/Shaman786/vps-manager/internal/images" // <--- Import Images
	"github.com/Shaman786/vps-manager/internal/plans"
	"github.com/Shaman786/vps-manager/internal/utils"
	"github.com/Shaman786/vps-manager/internal/vm"
	"golang.org/x/term"
)

func main() {
	// 1. Initialize VM Manager
	mgr := vm.NewManager(vm.ManagerConfig{
		BaseImageDir: "/var/lib/libvirt/images/base",
		VMDiskDir:    "/host-data/vms",
		ConfigDir:    "/host-data/configs",
	})

	reader := bufio.NewReader(os.Stdin)
	fmt.Println("=== VPS MANAGER v1.4 (Multi-OS Support) ===")

	// --- STEP 1: SELECT OS ---
	fmt.Println("\nAvailable Operating Systems:")
	for i, img := range images.Available {
		fmt.Printf("[%d] %s\n", i+1, img.Name)
	}
	osIdxStr := readInput(reader, "Select OS [Enter Number]: ")
	osIdx, _ := strconv.Atoi(osIdxStr)
	if osIdx < 1 || osIdx > len(images.Available) {
		log.Fatal("Invalid OS selection.")
	}
	selectedOS := images.Available[osIdx-1]

	// --- STEP 2: SELECT PLAN ---
	fmt.Println("\nAvailable Plans:")
	for i, p := range plans.Available {
		fmt.Printf("[%d] %s: %dMB RAM | %d vCPU | %s Disk\n", i+1, p.Name, p.RAM, p.CPUs, p.Disk)
	}
	planIdxStr := readInput(reader, "Select Plan [Enter Number]: ")
	planIdx, _ := strconv.Atoi(planIdxStr)
	if planIdx < 1 || planIdx > len(plans.Available) {
		log.Fatal("Invalid plan selection.")
	}
	selectedPlan := plans.Available[planIdx-1]

	// --- STEP 3: USER DETAILS ---
	vmName := readInput(reader, "\nEnter VM Name: ")
	username := readInput(reader, "Enter Customer Username: ")
	userPass := readPassword("Enter User Password: ")
	rootPass := readPassword("Enter Root Password: ")

	// --- EXECUTION ---

	// 1. Ensure Base Image (Downloads if missing)
	// 1. Ensure Base Image (Downloads if missing)
	fmt.Printf("\n[1/4] Checking Image: %s...\n", selectedOS.Name)

	// --- NEW: DYNAMIC FEDORA CHECK ---
	if selectedOS.URL == "DYNAMIC_FEDORA" {
		// Run the scraper
		realURL, realFilename, err := utils.GetLatestFedora()
		if err != nil {
			log.Fatalf("Failed to resolve Fedora version: %v", err)
		}
		// Update the struct with the real data found on the website
		selectedOS.URL = realURL
		selectedOS.Filename = realFilename
	}
	// ---------------------------------

	err := mgr.EnsureBaseImage(
		selectedOS.Name,
		selectedOS.URL,
		selectedOS.Filename,
	)
	if err != nil {
		log.Fatalf("Failed to download image: %v", err)
	}

	// 2. Create Disk (Using selected OS as backing file)
	fmt.Printf("[2/4] Provisioning %s Disk...\n", selectedPlan.Disk)
	diskPath, err := mgr.CreateDisk(vmName, selectedOS.Filename, selectedPlan.Disk)
	if err != nil {
		log.Fatalf("Failed to create disk: %v", err)
	}

	// 3. Generate Cloud-Init
	fmt.Println("[3/4] Generating Configuration...")
	cfg, err := cloudinit.Generate(cloudinit.ConfigData{
		Hostname: vmName,
		Username: username,
		UserPass: userPass,
		RootPass: rootPass,
	})
	if err != nil {
		log.Fatalf("Failed to generate config: %v", err)
	}
	isoPath, err := mgr.CreateISO(vmName, cfg)
	if err != nil {
		log.Fatalf("Failed to create ISO: %v", err)
	}

	// 4. Launch VM
	fmt.Printf("[4/4] Launching VM (%dMB RAM, %d vCPU)...\n", selectedPlan.RAM, selectedPlan.CPUs)
	if err := mgr.Launch(vmName, selectedPlan.RAM, selectedPlan.CPUs, diskPath, isoPath); err != nil {
		log.Fatalf("Failed to launch VM: %v", err)
	}

	fmt.Printf("\n✅ Success! VM '%s' is running.\n", vmName)
	fmt.Printf("   OS:   %s\n", selectedOS.Name)
	fmt.Printf("   Plan: %s\n", selectedPlan.Name)
	fmt.Printf("Connect via: virsh console %s\n", vmName)
}

func readInput(r *bufio.Reader, prompt string) string {
	fmt.Print(prompt)
	input, _ := r.ReadString('\n')
	return strings.TrimSpace(input)
}

func readPassword(prompt string) string {
	fmt.Print(prompt)
	bytePassword, err := term.ReadPassword(int(syscall.Stdin))
	if err != nil {
		return ""
	}
	fmt.Println()
	return string(bytePassword)
}
