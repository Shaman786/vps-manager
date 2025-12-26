package main

import (
	"fmt"
	"os"

	"github.com/Shaman786/vps-manager/internal/cli"
	"github.com/Shaman786/vps-manager/internal/drivers/kvm"
	"github.com/Shaman786/vps-manager/internal/images"
	"github.com/Shaman786/vps-manager/internal/vm"
	"github.com/Shaman786/vps-manager/internal/webhook"
)

func main() {
	// 1. Initialize Image Store (The Database of OS Images)
	// This handles downloading and caching "ubuntu-24.04", etc.
	imgStore, err := images.NewStore("/host-data/images/registry.json", "/host-data/images/cache")
	if err != nil {
		panic(fmt.Sprintf("Failed to init image store: %v", err))
	}

	// 2. Initialize KVM Driver (The Engine)
	// This talks to Libvirt/QEMU
	driver := kvm.NewKVMDriver(
		imgStore,
		"/host-data/vms",     // DiskDir
		"/host-data/configs", // ConfigDir
	)

	// 3. Initialize Manager (The Brain)
	// The manager doesn't know about files anymore, it just talks to the driver
	mgr := vm.NewManager(driver)

	// 4. Check Mode: Webhook Listener?
	if len(os.Args) > 1 && os.Args[1] == "listen" {
		// Pass the image store to the webhook so it can register new images
		webhook.Start(mgr, imgStore, ":8080")
		return
	}

	// 5. Default Mode: Interactive CLI
	// Pass both manager and store to the CLI
	app := cli.NewApp(mgr, imgStore)
	app.ShowMainMenu()
}
