#!/bin/bash
set -e

echo "🚀 Starting VPS Manager Setup (Rocky Linux 9)..."

# 1. Install Dependencies
echo "📦 Installing System Packages..."
dnf install -y epel-release
/usr/bin/crb enable
dnf install -y git wget gcc make qemu-kvm libvirt libvirt-client virt-install virt-viewer genisoimage

# 2. Create the 'cloud-localds' Fake Script (The Fix)
echo "🔧 Patching cloud-localds..."
cat <<EOF > /usr/local/bin/cloud-localds
#!/bin/bash
OUTPUT="\$1"
USER_DATA="\$2"
META_DATA="\$3"

if [ -z "\$OUTPUT" ] || [ -z "\$USER_DATA" ]; then
  echo "Usage: cloud-localds <output.iso> <user-data> [meta-data]"
  exit 1
fi

FILES="\$USER_DATA"
if [ -n "\$META_DATA" ]; then
  FILES="\$FILES \$META_DATA"
fi

genisoimage -output "\$OUTPUT" -volid cidata -joliet -rock \$FILES
EOF
chmod +x /usr/local/bin/cloud-localds

# 3. Enable Libvirt
echo "🔌 Enabling KVM..."
systemctl enable --now libvirtd

# 4. Install Go
if ! command -v go &> /dev/null; then
    echo "🐹 Installing Go..."
    wget https://go.dev/dl/go1.22.5.linux-amd64.tar.gz -O /tmp/go.tar.gz
    rm -rf /usr/local/go && tar -C /usr/local -xzf /tmp/go.tar.gz
    rm -f /tmp/go.tar.gz
fi
export PATH=$PATH:/usr/local/go/bin

# 5. Build
echo "🔨 Building VPS Manager..."
APP_DIR="/opt/vps-manager"
rm -rf $APP_DIR
git clone https://github.com/Shaman786/vps-manager.git $APP_DIR
cd $APP_DIR
go mod tidy
go build -o vps-manager cmd/vps-manager/main.go
cp vps-manager /usr/local/bin/

# 6. Service
echo "🔥 Starting Service..."
cat <<EOF > /etc/systemd/system/vps-manager.service
[Unit]
Description=VPS Manager Webhook Listener
After=network.target libvirtd.service

[Service]
Type=simple
User=root
WorkingDirectory=$APP_DIR
ExecStart=/usr/local/bin/vps-manager listen
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
EOF

systemctl daemon-reload
systemctl enable --now vps-manager
firewall-cmd --permanent --add-port=8080/tcp || true
firewall-cmd --reload || true

echo "✅ God Mode Ready."