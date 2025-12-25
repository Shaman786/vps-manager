package cli

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"

	"github.com/Shaman786/vps-manager/internal/cloudinit"
	"github.com/Shaman786/vps-manager/internal/images"
	"github.com/Shaman786/vps-manager/internal/plans"
)

// --- ACTION 1: Create Vps ---
func (a *App) CreateVPS() {
	// 1. OS Selection
	fmt.Println("\n-- Select Operating System --")
	if len(images.Catalog) == 0 {
		fmt.Println("❌ No images found. Try 'Refresh Catalog' first.")
		return
	}
	for i, img := range images.Catalog {
		fmt.Printf("[%d] %s\n", i+1, img.Name)
	}
	idx, _ := strconv.Atoi(a.readInput("Choice: "))
	if idx < 1 || idx > len(images.Catalog) {
		fmt.Println("❌ Invalid selection.")
		return
	}
	selectedOS := images.Catalog[idx-1]

	// 2. Plan Selection
	fmt.Println("\n-- Select Resource Plan --")
	for i, p := range plans.Available {
		fmt.Printf("[%d] %s (RAM: %dMB, CPU: %d)\n", i+1, p.Name, p.RAM, p.CPUs)
	}
	idx2, _ := strconv.Atoi(a.readInput("Choice: "))
	if idx2 < 1 || idx2 > len(plans.Available) {
		fmt.Println("❌ Invalid plan.")
		return
	}
	selectedPlan := plans.Available[idx2-1]

	// 3. VM Details & Credentials
	fmt.Println("\n-- Configuration --")
	name := a.readInput("VM Name: ")

	var username, userPass, rootPass string
	var allowRoot bool

	// Logic: User vs Root-only
	if strings.ToLower(a.readInput("Create a dedicated User (e.g. admin)? [y/N]: ")) == "y" {
		username = a.readInput("Enter Username: ")
		userPass = a.readPasswordConfirm("Set User Password: ")
		rootPass = a.readPasswordConfirm("Set Root Password: ")

		// Ask about Root SSH
		if strings.ToLower(a.readInput("Allow Root SSH Login? (Not Recommended) [y/N]: ")) == "y" {
			allowRoot = true
		} else {
			allowRoot = false
		}
	} else {
		fmt.Println("⚠️  No user created. You will log in as 'root'.")
		rootPass = a.readPasswordConfirm("Set Root Password: ")
		allowRoot = true // Forced yes for root-only setup
	}

	// 4. Network Mode
	fmt.Println("\n-- Network Settings --")
	fmt.Println("[Enter] for Default NAT (Safe)")
	fmt.Println("[Type Name] to use a specific Bridge (e.g. br0)")
	bridge := a.readInput("Bridge Interface (optional): ")

	// 5. Execution
	fmt.Printf("\n[1/4] Ensuring Image: %s...\n", selectedOS.Name)
	if err := a.mgr.EnsureBaseImage(selectedOS.Name, selectedOS.DownloadURL, selectedOS.Filename); err != nil {
		fmt.Printf("❌ Image Error: %v\n", err)
		return
	}

	fmt.Println("[2/4] Creating Disk...")
	path, err := a.mgr.CreateDisk(name, selectedOS.Filename, selectedPlan.Disk)
	if err != nil {
		fmt.Printf("❌ Disk Error: %v\n", err)
		return
	}

	fmt.Println("[3/4] Generating Configuration...")
	cfg, _ := cloudinit.Generate(cloudinit.ConfigData{
		Hostname:       name,
		Username:       username,
		UserPass:       userPass,
		RootPass:       rootPass,
		AllowRootLogin: allowRoot,
	})

	metaData := fmt.Sprintf("instance-id: %s\nlocal-hostname: %s\n", name, name)
	iso, err := a.mgr.CreateISO(name, cfg, metaData)
	if err != nil {
		fmt.Printf("❌ Config Error: %v\n", err)
		return
	}

	fmt.Println("[4/4] Launching VM...")
	if err := a.mgr.Launch(name, selectedPlan.RAM, selectedPlan.CPUs, path, iso, bridge); err != nil {
		fmt.Printf("❌ Launch Error: %v\n", err)
	} else {
		fmt.Printf("\n✅ Success! VM '%s' is booting.\n", name)
		fmt.Println("   Tip: Check the list for its IP address.")
	}
}

// --- ACTION 2: LIST VPS ---
func (a *App) ListVPS() {
	cmd := exec.Command("virsh", "list", "--all", "--name")
	out, _ := cmd.Output()
	vms := strings.Split(strings.TrimSpace(string(out)), "\n")

	fmt.Printf("\n%-20s | %-15s | %-20s\n", "NAME", "STATE", "IP ADDRESS")
	fmt.Println(strings.Repeat("-", 60))

	for _, vmName := range vms {
		if vmName == "" {
			continue
		}
		ip, state := a.mgr.GetVMInfo(vmName)
		fmt.Printf("%-20s | %-15s | %-20s\n", vmName, state, ip)
	}
	fmt.Println("\n(Note: IP addresses appear once the VM finishes booting)")
}

// --- ACTION 3: MANAGE VPS ---
func (a *App) ManageVPS() {
	name := a.readInput("Enter VM Name to manage: ")

	fmt.Println("\n-- Actions --")
	fmt.Println("1. Delete VM")
	fmt.Println("2. Scale Resources")
	fmt.Println("3. Get VNC Port")

	switch a.readInput("Choice: ") {
	case "1":
		if a.readInput("Are you sure? (yes/no): ") == "yes" {
			if err := a.mgr.DeleteVM(name); err != nil {
				fmt.Printf("⚠️ Error: %v\n", err)
			} else {
				fmt.Println("✅ VM Deleted.")
			}
		}
	case "2":
		ram, _ := strconv.Atoi(a.readInput("New RAM (MB): "))
		cpu, _ := strconv.Atoi(a.readInput("New vCPUs: "))
		if err := a.mgr.EditResources(name, ram, cpu); err != nil {
			fmt.Printf("❌ Error: %v\n", err)
		} else {
			fmt.Println("✅ Resources updated. Please reboot the VM.")
		}
	case "3":
		out, _ := exec.Command("virsh", "vncdisplay", name).Output()
		portOutput := strings.TrimSpace(string(out))
		if portOutput == "" {
			fmt.Println("❌ Could not find VNC port. Is the VM running?")
		} else {
			clean := strings.ReplaceAll(portOutput, ":", "")
			displayNum, _ := strconv.Atoi(clean)
			fmt.Printf("\n🖥️  VNC Display: %s (Port %d)\n", portOutput, 5900+displayNum)
		}
	default:
		fmt.Println("❌ Invalid choice.")
	}
}

// --- ACTION 4: CREATE BRIDGE ---
func (a *App) CreateBridge() {
	fmt.Println("\n⚠️  ADVANCED: Create a Bridge Network mapped to a physical interface.")
	name := a.readInput("New Network Name (e.g., br0-net): ")
	iface := a.readInput("Physical Host Interface (e.g., eth0): ")

	if err := a.mgr.CreateBridgeNetwork(name, iface); err != nil {
		fmt.Printf("❌ Failed: %v\n", err)
	} else {
		fmt.Println("✅ Bridge Created.")
	}
}
