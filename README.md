

# 🚀 HostPalace VPS Manager

A professional-grade, terminal-based VPS management platform written in Go.
It automates the lifecycle of KVM virtual machines, handling disk provisioning (QCOW2), Cloud-Init configuration, network bridging, and VNC remote access.

## ✨ Features
* **Multi-OS Support:** Automatically scrapes upstream mirrors for the latest Ubuntu, Debian, AlmaLinux, Rocky, Fedora, OpenSUSE, Alpine, and Arch Linux.
* **Smart Caching:** Caches ISO lists locally to prevent slow startups; auto-updates every 24 hours.
* **Full Lifecycle:** Create, Delete, Stop, Start, and Scale (RAM/CPU) VMs.
* **Networking:** Supports both **NAT** (default) and **Bridge** (public LAN IP) modes.
* **Remote Access:** Built-in VNC port detection and automatic SSH key injection.
* **Zero-Config:** Uses Cloud-Init to pre-configure users, passwords, and hostnames.

---

## 🛠️ Prerequisites

This tool is designed for **RHEL-based systems** (AlmaLinux 8/9, Rocky Linux, CentOS Stream, Fedora).

### 1. Install System Dependencies
Run these commands as root to install KVM, Go, and required utilities:

```bash
# 1. Enable Virtualization Tools
dnf groupinstall -y "Virtualization Host"
dnf install -y epel-release

# 2. Install Core Tools
dnf install -y virt-install qemu-img bridge-utils wget git

# 3. Install Cloud-Init Utilities (Critical for ISO generation)
dnf install -y genisoimage
# Manually install cloud-localds if missing from repo
curl -L -o /usr/local/bin/cloud-localds [https://raw.githubusercontent.com/canonical/cloud-utils/main/bin/cloud-localds](https://raw.githubusercontent.com/canonical/cloud-utils/main/bin/cloud-localds)
chmod +x /usr/local/bin/cloud-localds

# 4. Install Go (1.23+)
wget [https://go.dev/dl/go1.23.4.linux-amd64.tar.gz](https://go.dev/dl/go1.23.4.linux-amd64.tar.gz)
rm -rf /usr/local/go && tar -C /usr/local -xzf go1.23.4.linux-amd64.tar.gz
echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
source ~/.bashrc

```

### 2. Configure Firewall (For VNC)

To access the VNC remote desktop console from your PC, you must open the VNC port range (5900-5910).

```bash
firewall-cmd --permanent --add-port=5900-5910/tcp
firewall-cmd --reload

```

### 3. Start KVM Services

Ensure Libvirt is running:

```bash
systemctl enable --now libvirtd

```

---

## 🌐 Network Setup (Bridge Mode)

**Optional but Recommended.**
By default, VMs use NAT (hidden IP). To make VMs appear on your main network (LAN) with their own real IP addresses, set up a Bridge.

**Warning:** Run this on the host server console. It might briefly disconnect SSH.

```bash
# 1. Create a bridge interface named 'br0'
nmcli con add type bridge ifname br0 con-name br0

# 2. Attach your physical interface (e.g., eth0) to the bridge
# REPLACE 'eth0' with your actual interface name (check 'ip a')
nmcli con add type bridge-slave ifname eth0 master br0

# 3. Activate the bridge
nmcli con up br0

```

*Now, when creating a VPS, type `br0` when asked for the Bridge Interface.*

---

## 📦 Installation & Build

Clone the repo and build the binary:

```bash
# 1. Clone repository
git clone [https://github.com/shaman786/vps-manager.git](https://github.com/shaman786/vps-manager.git)
cd vps-manager

# 2. Download Go modules
go mod tidy

# 3. Build the executable
go build -o vps-manager cmd/vps-manager/main.go

# 4. Install globally (Optional)
mv vps-manager /usr/local/bin/

```

---

## 🚀 Usage

Run the tool as root (required for KVM access):

```bash
vps-manager

```

### Menu Options

* **[1] Create New VPS:**
* Checks the Image Catalog (auto-updates if old).
* Downloads the OS image (if not cached).
* Asks for Hostname, Username, and Password.
* Generates a Cloud-Init ISO and boots the VM.


* **[2] List All VPS:**
* Shows VM Name, State (Running/Shutoff), and **Real IP Address**.


* **[3] Manage VPS:**
* **Delete:** Instantly destroys VM and deletes all `.qcow2` disks and configs.
* **Scale:** Change RAM and CPU allocation (requires reboot).
* **VNC Port:** Displays the connection port (e.g., `5900`) for remote desktop.


* **[4] Network Tools:**
* Helper to create a bridge network XML (Advanced).


* **[5] Refresh Image Catalog:**
* Forces a re-scrape of upstream mirrors (Ubuntu, Fedora, etc.) to find new versions.



---

## 🔧 Troubleshooting

| Error | Fix |
| --- | --- |
| `exec: "cloud-localds": executable file not found` | You missed step 3 in Prerequisites. Install `genisoimage` and the `cloud-localds` script. |
| `Permission denied` / `libvirt error` | You must run the tool as `root` (sudo). |
| `VNC Connection Refused` | Check your firewall settings (Step 2). Ensure ports 5900+ are open. |
| `No IP Address found` | Wait 30 seconds for the VM to boot. If using Bridge, ensure your router has DHCP enabled. |

---

**Author:** [Shaman786](https://www.google.com/search?q=https://github.com/shaman786)

```

```
