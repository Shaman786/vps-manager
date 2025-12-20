package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/Shaman786/vps-manager/internal/cloudinit"
	"github.com/Shaman786/vps-manager/internal/vm"
)

func main() {
	// 1. Initialize the VM Manager
	// We inject the storage paths here. This makes it easy to change later.
	mgr := vm.NewManager(vm.ManagerConfig{
		BaseImageDir: "/var/lib/libvirt/images/base",
		VMDiskDir:    "/host-data/vms",
		ConfigDir:    "/host-data/configs",
	})

	reader := bufio.NewReader(os.Stdin)
	fmt.Println("=== VPS MANAGER v1.0 ===")

	// 2. Gather User Input
	vmName := readInput(reader, "Enter VM Name: ")
	username := readInput(reader, "Enter Username: ")
	userPass := readInput(reader, "Enter User Password: ")
	rootPass := readInput(reader, "Enter Root Password: ")

	// 3. Ensure we have the Base Image (Ubuntu 22.04)
	// In a real app, you would let the user select this from a list.
	fmt.Println("\n[1/4] Checking Base Image...")
	err := mgr.EnsureBaseImage(
		"Ubuntu 22.04 LTS",
		"https://cloud-images.ubuntu.com/jammy/current/jammy-server-cloudimg-amd64.img",
		"ubuntu-22.04.img",
	)
	if err != nil {
		log.Fatalf("Failed to download image: %v", err)
	}

	// 4. Create the Disk (Copy-on-Write)
	fmt.Println("[2/4] Provisioning Disk...")
	// We default to 10GB for this demo
	diskPath, err := mgr.CreateDisk(vmName, "ubuntu-22.04.img", "10G")
	if err != nil {
		log.Fatalf("Failed to create disk: %v", err)
	}

	// 5. Generate Cloud-Init Config
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

	// Create the ISO file from that config
	isoPath, err := mgr.CreateISO(vmName, cfg)
	if err != nil {
		log.Fatalf("Failed to create ISO: %v", err)
	}

	// 6. Launch the VM
	fmt.Println("[4/4] Launching VM...")
	// 2048MB RAM, 2 vCPUs
	if err := mgr.Launch(vmName, 2048, 2, diskPath, isoPath); err != nil {
		log.Fatalf("Failed to launch VM: %v", err)
	}

	fmt.Printf("\n✅ Success! VM '%s' is running.\n", vmName)
	fmt.Printf("Connect via: virsh console %s\n", vmName)
}

// Helper to read console input cleanly
func readInput(r *bufio.Reader, prompt string) string {
	fmt.Print(prompt)
	input, _ := r.ReadString('\n')
	return strings.TrimSpace(input)
}
