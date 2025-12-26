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
	// 1. SYSTEM PATHS (For Root Usage)
	// We use a central directory for all VPS data
	baseDir := "/host-data"

	registryPath := baseDir + "/images/registry.json"
	cacheDir := baseDir + "/images/cache"
	vmsDir := baseDir + "/vms"
	configDir := baseDir + "/configs"

	// 2. Ensure Directories Exist (Auto-Setup)
	// This prevents "no such file or directory" errors
	dirs := []string{cacheDir, vmsDir, configDir}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			panic(fmt.Sprintf("❌ Critical Error: Cannot create directory %s. (Did you run with sudo?): %v", dir, err))
		}
	}

	// 3. Initialize Image Store
	imgStore, err := images.NewStore(registryPath, cacheDir)
	if err != nil {
		panic(fmt.Sprintf("Failed to init image store: %v", err))
	}

	// 4. Initialize KVM Driver
	driver := kvm.NewKVMDriver(
		imgStore,
		vmsDir,
		configDir,
	)

	// 5. Initialize Manager
	mgr := vm.NewManager(driver)

	// 6. Check Mode: Webhook Listener?
	if len(os.Args) > 1 && os.Args[1] == "listen" {
		// Pass the image store to the webhook so it can register new images
		webhook.Start(mgr, imgStore, ":8080")
		return
	}

	// 7. Default Mode: Interactive CLI
	// Pass both manager and store to the CLI
	app := cli.NewApp(mgr, imgStore)
	app.ShowMainMenu()
}
