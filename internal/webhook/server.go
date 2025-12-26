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
	// 1. IMAGE WEBHOOK
	http.HandleFunc("/webhook", handleImageWebhook(store))

	// 2. VM API
	http.HandleFunc("/api/vms", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if r.Method == http.MethodGet {
			vms, _ := mgr.ListServers()
			json.NewEncoder(w).Encode(vms)
			return
		}

		if r.Method == http.MethodPost {
			// Parse the new Rich Request
			var req struct {
				Name     string `json:"name"`
				Image    string `json:"image"`
				Plan     string `json:"plan"`     // "Starter"
				Username string `json:"username"` // "fh"
				Password string `json:"password"` // "securepass"
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, "Invalid JSON", 400)
				return
			}

			// Defaults
			if req.Plan == "" {
				req.Plan = "Starter"
			}
			if req.Username == "" {
				req.Username = "root"
			}
			if req.Password == "" {
				req.Password = "password"
			} // Unsafe default, but good for demo

			opts := vm.CreateOptions{
				Name:     req.Name,
				Image:    req.Image,
				PlanName: req.Plan,
				Username: req.Username,
				Password: req.Password,
			}

			if err := mgr.CreateServer(opts); err != nil {
				http.Error(w, err.Error(), 500)
				return
			}
			json.NewEncoder(w).Encode(map[string]string{"status": "created", "id": req.Name})
		}
	})

	// 3. ACTION API
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

	fmt.Printf("📡 VPS Control Plane running on %s\n", port)
	log.Fatal(http.ListenAndServe(port, nil))
}

// ... (Helper functions remain same) ...
func handleImageWebhook(store *images.Store) http.HandlerFunc {
	type LegacyImageRelease struct {
		Distro  string `json:"distro"`
		Version string `json:"version"`
		URL     string `json:"url"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		var req LegacyImageRelease
		json.NewDecoder(r.Body).Decode(&req)
		logicalName := strings.ToLower(fmt.Sprintf("%s-%s", req.Distro, req.Version))

		fmt.Printf("🔔 Beacon Alert: Update found for '%s'\n", logicalName)
		store.Register(logicalName, req.URL, "")
		go func() {
			store.Resolve(logicalName)
		}()
		w.WriteHeader(200)
	}
}
