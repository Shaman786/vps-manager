#!/bin/bash
set -e # Exit immediately if any command fails

# Colors
GREEN='\033[0;32m'
BLUE='\033[0;34m'
RED='\033[0;31m'
NC='\033[0m'

echo -e "${BLUE}"
echo "============================================"
echo "   🚀 HostPalace VPS Manager Installer      "
echo "============================================"
echo -e "${NC}"

# 1. Check Root
if [ "$EUID" -ne 0 ]; then
  echo -e "${RED}[ERROR] Please run as root.${NC}"
  echo "Try: curl -fsSL ... | sudo bash"
  exit 1
fi

# 2. Detect OS & Install System Deps
if [ -f /etc/os-release ]; then
    . /etc/os-release
    OS=$ID
else
    echo -e "${RED}[ERROR] Cannot detect OS.${NC}"
    exit 1
fi

echo -e "${GREEN}[+] Detected OS: $OS${NC}"

case $OS in
    "almalinux"|"rocky"|"centos"|"fedora"|"rhel")
        echo -e "${BLUE}[INFO] Installing RHEL dependencies...${NC}"
        dnf install -y epel-release
        dnf groupinstall -y "Virtualization Host"
        dnf install -y virt-install qemu-img bridge-utils wget git genisoimage tar

        # Cloud-localds (Manual fetch for RHEL)
        if [ ! -f /usr/local/bin/cloud-localds ]; then
            curl -L -o /usr/local/bin/cloud-localds https://raw.githubusercontent.com/canonical/cloud-utils/main/bin/cloud-localds
            chmod +x /usr/local/bin/cloud-localds
        fi

        # Firewall
        echo -e "${BLUE}[INFO] Opening Firewall Ports (5900-5910)...${NC}"
        firewall-cmd --permanent --add-port=5900-5910/tcp >/dev/null 2>&1 || true
        firewall-cmd --reload >/dev/null 2>&1 || true
        systemctl enable --now libvirtd
        ;;
    "ubuntu"|"debian")
        echo -e "${BLUE}[INFO] Installing Debian/Ubuntu dependencies...${NC}"
        apt-get update -qq
        apt-get install -y qemu-kvm libvirt-daemon-system libvirt-clients bridge-utils virtinst cloud-image-utils genisoimage git wget tar

        # Firewall
        ufw allow 5900:5910/tcp >/dev/null 2>&1 || true
        ;;
    *)
        echo -e "${RED}[ERROR] Unsupported OS: $OS${NC}"
        exit 1
        ;;
esac

# 3. Install Go (if missing)
if ! command -v go &> /dev/null; then
    echo -e "${BLUE}[INFO] Installing Go 1.23.4...${NC}"
    wget -q https://go.dev/dl/go1.23.4.linux-amd64.tar.gz
    rm -rf /usr/local/go && tar -C /usr/local -xzf go1.23.4.linux-amd64.tar.gz
    export PATH=$PATH:/usr/local/go/bin
else
    echo -e "${GREEN}[+] Go is already installed.${NC}"
fi

# 4. Clone, Build & Install
INSTALL_DIR="/usr/local/src/vps-manager"
BIN_DIR="/usr/local/bin"

echo -e "${BLUE}[INFO] Downloading Source Code...${NC}"
rm -rf "$INSTALL_DIR"
git clone https://github.com/shaman786/vps-manager.git "$INSTALL_DIR"

echo -e "${BLUE}[INFO] Building Binary...${NC}"
cd "$INSTALL_DIR"
/usr/local/go/bin/go mod tidy
/usr/local/go/bin/go build -o vps-manager cmd/vps-manager/main.go

echo -e "${BLUE}[INFO] Installing to $BIN_DIR...${NC}"
mv vps-manager "$BIN_DIR/"
chmod +x "$BIN_DIR/vps-manager"

# 5. Cleanup
echo -e "${GREEN}[+] Cleanup complete.${NC}"
cd /root
rm -rf "$INSTALL_DIR"

echo -e "${GREEN}"
echo "============================================"
echo "   ✅ Installation Complete! "
echo "============================================"
echo -e "${NC}"
echo "Type 'vps-manager' to start."
