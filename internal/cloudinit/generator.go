// Package cloudinit generates user-data and meta-data configuration files for VMs.
package cloudinit

import (
	"bytes"
	"fmt"
	"text/template"
)

type ConfigData struct {
	Hostname       string
	Username       string
	UserPass       string
	RootPass       string
	AllowRootLogin bool // New Field
}

// THE FREEDOM CONFIG
// 1. PermitRootLogin is DYNAMIC now.
// 2. PasswordAuthentication is HARDCODED to YES.
// 3. User creation is OPTIONAL.
// 4. FIX APPLIED: Smart SSH restart logic in runcmd.
const configTmpl = `#cloud-config
hostname: {{.Hostname}}
ssh_pwauth: true
package_update: true
package_upgrade: false

# --- 1. OPTIONAL USER CREATION ---
users:
  - default
{{- if .Username}}
  - name: {{.Username}}
    sudo: ['ALL=(ALL) NOPASSWD:ALL']
    groups: [sudo, wheel, users, admin]
    shell: /bin/bash
    lock_passwd: false
{{- end}}

# --- 2. SET PASSWORDS ---
chpasswd:
  list: |
    root:{{.RootPass}}
{{- if .Username}}
    {{.Username}}:{{.UserPass}}
{{- end}}
  expire: false

# --- 3. SSH CONFIGURATION ---
write_files:
  - path: /etc/ssh/sshd_config.d/99-custom.conf
    permissions: '0644'
    content: |
      PermitRootLogin {{if .AllowRootLogin}}yes{{else}}no{{end}}
      PasswordAuthentication yes
      KbdInteractiveAuthentication yes
      PubkeyAuthentication yes

# --- 4. APPLY CHANGES ---
runcmd:
  - [ systemctl, daemon-reload ]
  # Restart SSH to apply the PermitRootLogin change
  - [ sh, -c, "systemctl restart sshd 2>/dev/null || systemctl restart ssh 2>/dev/null || true" ]
`

func Generate(data ConfigData) (string, error) {
	tmpl, err := template.New("cloud-config").Parse(configTmpl)
	if err != nil {
		return "", fmt.Errorf("failed to parse template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}

	return buf.String(), nil
}
