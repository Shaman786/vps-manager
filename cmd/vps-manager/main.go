package main

import (
	"os"

	"github.com/Shaman786/vps-manager/internal/cli"
	"github.com/Shaman786/vps-manager/internal/vm"
	"github.com/Shaman786/vps-manager/internal/webhook"
)

func main() {
	// 1. Init Core Logic
	// We set up the Manager here so we can pass it to either the CLI or the Webhook.
	mgr := vm.NewManager(vm.ManagerConfig{
		BaseImageDir: "/var/lib/libvirt/images/base",
		VMDiskDir:    "/host-data/vms",
		ConfigDir:    "/host-data/configs",
	})

	// 2. Check Mode: Webhook Listener?
	// If the user runs "./vps-manager listen", we start the HTTP server.
	if len(os.Args) > 1 && os.Args[1] == "listen" {
		webhook.Start(mgr, ":8080")
		return
	}

	// 3. Default Mode: Interactive CLI
	// Otherwise, we launch the beautiful text menu.
	app := cli.NewApp(mgr)
	app.ShowMainMenu()
}
