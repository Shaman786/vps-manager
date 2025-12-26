package webhook

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/Shaman786/vps-manager/internal/images"
	"github.com/Shaman786/vps-manager/internal/vm"
)

func Start(mgr *vm.Manager, store *images.Store, port string) {
	// 1. IMAGE WEBHOOK (Keep this!)
	http.HandleFunc("/webhook", handleImageWebhook(store))

	// 2. VM LIST & CREATE API (New!)
	http.HandleFunc("/api/vms", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if r.Method == http.MethodGet {
			// LIST
			vms, _ := mgr.ListServers()
			json.NewEncoder(w).Encode(vms)
			return
		}

		if r.Method == http.MethodPost {
			// CREATE
			var req struct {
				Name   string `json:"name"`
				Image  string `json:"image"`
				Region string `json:"region"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, "Invalid JSON", 400)
				return
			}

			// Call Manager with the specific image
			if err := mgr.CreateServer(req.Name, req.Image, req.Region); err != nil {
				http.Error(w, err.Error(), 500)
				return
			}
			json.NewEncoder(w).Encode(map[string]string{"status": "created", "id": req.Name})
		}
	})

	// 3. VM ACTIONS API (Start/Stop)
	http.HandleFunc("/api/vms/action", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "POST only", 405)
			return
		}
		var req struct {
			ID     string `json:"id"`
			Action string `json:"action"`
		}
		json.NewDecoder(r.Body).Decode(&req)

		if err := mgr.PerformAction(req.ID, req.Action); err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		w.WriteHeader(200)
	})

	fmt.Printf("📡 VPS Control Plane & Webhook running on %s\n", port)
	log.Fatal(http.ListenAndServe(port, nil))
}

// ... (Keep your existing handleImageWebhook helper and toLowerCase helper below) ...
func handleImageWebhook(store *images.Store) http.HandlerFunc {
	type LegacyImageRelease struct {
		Distro  string `json:"distro"`
		Version string `json:"version"`
		URL     string `json:"url"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		var req LegacyImageRelease
		json.NewDecoder(r.Body).Decode(&req)
		logicalName := toLowerCase(fmt.Sprintf("%s-%s", req.Distro, req.Version))

		fmt.Printf("🔔 Beacon Alert: Update found for '%s'\n", logicalName)
		store.Register(logicalName, req.URL, "")
		go func() {
			fmt.Printf("   -> Triggering background pull for %s...\n", logicalName)
			store.Resolve(logicalName)
		}()
		w.WriteHeader(200)
	}
}

func toLowerCase(s string) string { return strings.ToLower(s) }
