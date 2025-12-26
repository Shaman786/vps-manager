package webhook

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/Shaman786/vps-manager/internal/images" // Import the new Store
	"github.com/Shaman786/vps-manager/internal/vm"
)

// Matches the JSON your 'watcher/main.go' is ALREADY sending
type LegacyImageRelease struct {
	Distro  string `json:"distro"`  // "Ubuntu"
	Version string `json:"version"` // "24.04"
	URL     string `json:"url"`
}

// Start now accepts the ImageStore instead of just Manager
func Start(mgr *vm.Manager, store *images.Store, port string) {
	http.HandleFunc("/webhook", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "POST only", 405)
			return
		}

		var req LegacyImageRelease
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid JSON", 400)
			return
		}

		// 1. ADAPT: Create the Logical Name (e.g., "ubuntu-24.04")
		// We normalize to lowercase to match our registry standards
		logicalName := fmt.Sprintf("%s-%s", req.Distro, req.Version)
		logicalName = toLowerCase(logicalName) // helper function or strings.ToLower

		fmt.Printf("🔔 Beacon Alert: Update found for '%s'\n", logicalName)

		// 2. REGISTER: Update the JSON Registry immediately
		// We pass "" for checksum for now (Watcher doesn't send it yet)
		if err := store.Register(logicalName, req.URL, ""); err != nil {
			http.Error(w, "Registry update failed", 500)
			return
		}

		// 3. OPTIONAL: Trigger background download (Cache Warming)
		// This keeps the webhook fast but starts the work
		go func() {
			fmt.Printf("   -> Triggering background pull for %s...\n", logicalName)
			store.Resolve(logicalName)
		}()

		w.WriteHeader(200)
		w.Write([]byte("Image registered and pull started"))
	})

	fmt.Printf("📡 VPS Webhook Listener running on %s\n", port)
	log.Fatal(http.ListenAndServe(port, nil))
}

func toLowerCase(s string) string {
	return strings.ToLower(s)
}
