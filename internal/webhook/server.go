package webhook

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/Shaman786/vps-manager/internal/vm"
)

type ImageRelease struct {
	Distro  string `json:"distro"`
	Version string `json:"version"`
	URL     string `json:"url"`
}

func Start(mgr *vm.Manager, port string) {
	http.HandleFunc("/webhook",
		func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				http.Error(w,
					"Only POST allowed",
					http.StatusMethodNotAllowed)
				return
			}
			var release ImageRelease
			if err := json.NewDecoder(r.Body).Decode(&release); err != nil {
				http.Error(w, "Invalid JSON", http.StatusBadRequest)
				return
			}
			// FIXED: Added URL logging and ensured syntax is correct
			fmt.Printf("\n[webhook] New Image Alert: %s %s (URL: %s )\n",
				release.Distro,
				release.Version,
				release.URL)
			// Auto-Download logic
			filename := fmt.Sprintf("%s-%s.img", release.Distro, release.Version)
			if err := mgr.EnsureBaseImage(release.Distro, release.URL, filename); err != nil {
				fmt.Printf("❌ Download failed: %v\n", err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("Image download initiated"))
		})
	fmt.Printf("📡 VPS Webhook Listener running on %s\n", port)
	log.Fatal(http.ListenAndServe(port, nil))
}
