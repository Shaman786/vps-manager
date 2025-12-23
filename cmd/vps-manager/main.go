package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"

	"github.com/Shaman786/vps-manager/internal/cloudinit"
	"github.com/Shaman786/vps-manager/internal/images"
	"github.com/Shaman786/vps-manager/internal/plans"
	"github.com/Shaman786/vps-manager/internal/vm"
	"golang.org/x/term"
)

var mgr *vm.Manager

func main() {
	// 1. Initialize Manager
	mgr = vm.NewManager(vm.ManagerConfig{
		BaseImageDir: "/var/lib/libvirt/images/base",
		VMDiskDir:    "/host-data/vms",
		ConfigDir:    "/host-data/configs",
	})

	// 2. Load Image Catalog (Runs the scrapers)
	if err := images.RefreshCatalog(); err != nil {
		fmt.Println("⚠️ Warning: Could not update image catalog. Check internet connection.")
	}

	reader := bufio.NewReader(os.Stdin)

	// 3. Main Application Loop
	for {
		fmt.Println("\n==========================================")
		fmt.Println("       🚀 HOSTPALACE VPS MANAGER         ")
		fmt.Println("==========================================")
		fmt.Println("1. Create New VPS")
		fmt.Println("2. List All VPS (Status & IP)")
		fmt.Println("3. Manage VPS (Delete / Scale / VNC)")
		fmt.Println("4. Network Tools (Create Bridge)")
		fmt.Println("5. Refresh Image Catalog")
		fmt.Println("0. Exit")
		choice := readInput(reader, "\nSelect Option: ")

		switch choice {
		case "1":
			createVPS(reader)
		case "2":
			listVPS()
		case "3":
			manageVPS(reader)
		case "4":
			createBridge(reader)
		case "5":
			images.RefreshCatalog()
		case "0":
			fmt.Println("Goodbye!")
			os.Exit(0)
		default:
			fmt.Println("❌ Invalid choice, please try again.")
		}
	}
}

// --- FEATURE FUNCTIONS ---

func createVPS(r *bufio.Reader) {
	// 1. Select OS (Dynamic List)
	fmt.Println("\n-- Select Operating System --")
	if len(images.Catalog) == 0 {
		fmt.Println("❌ No images found. Try 'Refresh Catalog' first.")
		return
	}
	for i, img := range images.Catalog {
		fmt.Printf("[%d] %s\n", i+1, img.Name)
	}
	idxStr := readInput(r, "Choice: ")
	idx, _ := strconv.Atoi(idxStr)
	if idx < 1 || idx > len(images.Catalog) {
		fmt.Println("❌ Invalid selection.")
		return
	}
	selectedOS := images.Catalog[idx-1]

	// 2. Select Plan
	fmt.Println("\n-- Select Resource Plan --")
	for i, p := range plans.Available {
		fmt.Printf("[%d] %s (RAM: %dMB, CPU: %d)\n", i+1, p.Name, p.RAM, p.CPUs)
	}
	idx2, _ := strconv.Atoi(readInput(r, "Choice: "))
	selectedPlan := plans.Available[idx2-1]

	// 3. VM Details
	name := readInput(r, "VM Name: ")
	user := readInput(r, "Username: ")
	pass := readPassword("Password: ")

	// 4. Network Mode
	bridgeName := ""
	fmt.Println("\n-- Network Settings --")
	fmt.Println("[Enter] for Default NAT (Safe)")
	fmt.Println("[Type Name] to use a specific Bridge (e.g. br0)")
	bridgeName = readInput(r, "Bridge Interface (optional): ")

	// --- EXECUTION ---
	fmt.Printf("\n[1/4] Ensuring Image: %s...\n", selectedOS.Name)
	if err := mgr.EnsureBaseImage(selectedOS.Name, selectedOS.DownloadURL, selectedOS.Filename); err != nil {
		fmt.Printf("❌ Image Error: %v\n", err)
		return
	}

	fmt.Println("[2/4] Creating Disk...")
	path, err := mgr.CreateDisk(name, selectedOS.Filename, selectedPlan.Disk)
	if err != nil {
		fmt.Printf("❌ Disk Error: %v\n", err)
		return
	}

	fmt.Println("[3/4] Generating Configuration...")
	// Generate Cloud-Init Config
	cfg, _ := cloudinit.Generate(cloudinit.ConfigData{Hostname: name, Username: user, RootPass: pass})
	// Generate Meta-Data (Fixes the localhost hostname issue)
	metaData := fmt.Sprintf("instance-id: %s\nlocal-hostname: %s\n", name, name)
	iso, err := mgr.CreateISO(name, cfg, metaData)
	if err != nil {
		fmt.Printf("❌ Config Error: %v\n", err)
		return
	}

	fmt.Println("[4/4] Launching VM...")
	if err := mgr.Launch(name, selectedPlan.RAM, selectedPlan.CPUs, path, iso, bridgeName); err != nil {
		fmt.Printf("❌ Launch Error: %v\n", err)
	} else {
		fmt.Printf("\n✅ Success! VM '%s' is booting.\n", name)
		fmt.Println("   Tip: Use Option [2] to check for its IP address.")
	}
}

func listVPS() {
	// List all VMs (running and stopped)
	cmd := exec.Command("virsh", "list", "--all", "--name")
	out, _ := cmd.Output()
	vms := strings.Split(strings.TrimSpace(string(out)), "\n")

	fmt.Printf("\n%-20s | %-15s | %-20s\n", "NAME", "STATE", "IP ADDRESS")
	fmt.Println(strings.Repeat("-", 60))

	for _, vmName := range vms {
		if vmName == "" {
			continue
		}
		ip, state := mgr.GetVMInfo(vmName)
		fmt.Printf("%-20s | %-15s | %-20s\n", vmName, state, ip)
	}
	fmt.Println("\n(Note: IP addresses appear once the VM finishes booting)")
}

func manageVPS(r *bufio.Reader) {
	vmName := readInput(r, "Enter VM Name to manage: ")

	fmt.Println("\n-- Actions --")
	fmt.Println("1. Delete VM (Destroy & Remove Files)")
	fmt.Println("2. Scale Resources (RAM/CPU)")
	fmt.Println("3. Get VNC Port (Remote Desktop)")
	choice := readInput(r, "Choice: ")

	switch choice {
	case "1":
		if readInput(r, "Are you sure? This cannot be undone (yes/no): ") == "yes" {
			mgr.DeleteVM(vmName)
			fmt.Println("✅ VM Deleted.")
		}
	case "2":
		ram, _ := strconv.Atoi(readInput(r, "New RAM (MB): "))
		cpu, _ := strconv.Atoi(readInput(r, "New vCPUs: "))
		if err := mgr.EditResources(vmName, ram, cpu); err != nil {
			fmt.Printf("❌ Error: %v\n", err)
		} else {
			fmt.Println("✅ Resources updated. Please reboot the VM to apply changes.")
		}
	case "3":
		out, _ := exec.Command("virsh", "vncdisplay", vmName).Output()
		port := strings.TrimSpace(string(out))
		if port == "" {
			fmt.Println("❌ Could not find VNC port. Is the VM running?")
		} else {
			// VNC ports usually look like :0, :1. The actual port is 5900 + N.
			cleanPort := strings.ReplaceAll(port, ":", "")
			fmt.Printf("\n🖥️  VNC Display: %s\n", port)
			fmt.Printf("   Connect via VNC Viewer to: YOUR_SERVER_IP:59%s\n", cleanPort)
		}
	}
}

func createBridge(r *bufio.Reader) {
	fmt.Println("\n⚠️  ADVANCED: This creates a Bridge Network mapped to a physical interface.")
	fmt.Println("   This allows VMs to appear directly on your home LAN.")

	name := readInput(r, "New Network Name (e.g., br0-net): ")
	iface := readInput(r, "Physical Host Interface (e.g., eth0, enp3s0): ")

	if err := mgr.CreateBridgeNetwork(name, iface); err != nil {
		fmt.Printf("❌ Failed: %v\n", err)
	} else {
		fmt.Println("✅ Network created! You can now type this name when creating a VM.")
	}
}

// Helpers
func readInput(r *bufio.Reader, prompt string) string {
	fmt.Print(prompt)
	input, _ := r.ReadString('\n')
	return strings.TrimSpace(input)
}

func readPassword(prompt string) string {
	fmt.Print(prompt)
	bytePassword, _ := term.ReadPassword(int(syscall.Stdin))
	fmt.Println()
	return string(bytePassword)
}
