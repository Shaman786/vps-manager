// Package cloudinit generates user-data configurations for cloud instances.
package cloudinit

import (
	"bytes"
	"fmt"
	"text/template"
)

// ConfigData holds the dynamic values for the template.
type ConfigData struct {
	Hostname string
	Username string
	UserPass string
	RootPass string
}

// configTmpl is the raw Cloud-Init YAML template.
// We use Go's {{.Variable}} syntax to inject values.
const configTmpl = `#cloud-config
hostname: {{.Hostname}}
ssh_pwauth: true

# 1. Create the User
users:
  - default
  - name: {{.Username}}
    sudo: ['ALL=(ALL) NOPASSWD:ALL']
    groups: [sudo, wheel]
    shell: /bin/bash
    lock_passwd: false

# 2. Set Passwords
chpasswd:
  list: |
    root:{{.RootPass}}
    {{.Username}}:{{.UserPass}}
  expire: false

# 3. Configure SSH (Allow Root)
write_files:
  - path: /etc/ssh/sshd_config.d/99-custom.conf
    permissions: '0644'
    content: |
      PermitRootLogin yes
      PasswordAuthentication yes

# 4. Apply Changes
runcmd:
  - systemctl restart ssh || systemctl restart sshd
`

// Generate takes the config data and returns a fully formatted YAML string.
func Generate(data ConfigData) (string, error) {
	// Parse the template
	tmpl, err := template.New("cloud-config").Parse(configTmpl)
	if err != nil {
		return "", fmt.Errorf("failed to parse template: %w", err)
	}

	// Execute the template into a buffer
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}

	return buf.String(), nil
}
