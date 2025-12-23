
```markdown
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

Choose your operating system below to install the required dependencies.

### Option A: RHEL / AlmaLinux / Rocky / Fedora
Run as **root**:

```bash
# 1. Enable Virtualization Tools
dnf groupinstall -y "Virtualization Host"
dnf install -y epel-release

# 2. Install Core Tools
dnf install -y virt-install qemu-img bridge-utils wget git

# 3. Install Cloud-Init Utilities
dnf install -y genisoimage
# Manually install cloud-localds
curl -L -o /usr/local/bin/cloud-localds [https://raw.githubusercontent.com/canonical/cloud-utils/main/bin/cloud-localds](https://raw.githubusercontent.com/canonical/cloud-utils/main/bin/cloud-localds)
chmod +x /usr/local/bin/cloud-localds

# 4. Install Go (1.23+)
wget [https://go.dev/dl/go1.23.4.linux-amd64.tar.gz](https://go.dev/dl/go1.23.4.linux-amd64.tar.gz)
rm -rf /usr/local/go && tar -C /usr/local -xzf go1.23.4.linux-amd64.tar.gz
echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
source ~/.bashrc

```

### Option B: Ubuntu / Debian

Run as **root** (sudo):

```bash
# 1. Install KVM & Tools
sudo apt update
sudo apt install -y qemu-kvm libvirt-daemon-system libvirt-clients bridge-utils virtinst cloud-image-utils genisoimage git wget

# 2. Install Go
sudo snap install go --classic

```

---

## 🛡️ Firewall Configuration (For VNC)

To access the VNC remote desktop console, you must open ports **5900-5910**.

**For RHEL / AlmaLinux:**

```bash
firewall-cmd --permanent --add-port=5900-5910/tcp
firewall-cmd --reload

```

**For Ubuntu / Debian:**

```bash
sudo ufw allow 5900:5910/tcp
sudo ufw reload

```

---

## ⚙️ Service Setup

Ensure the virtualization service is running.

```bash
systemctl enable --now libvirtd

```

---

## 🌐 Network Setup (Bridge Mode)

**Optional but Recommended.**
By default, VMs use **NAT** (hidden IP). To make VMs appear on your main LAN with their own real IPs, set up a Bridge.
*Warning: configuring networking remotely carries a risk of disconnection.*

### Option A: RHEL / AlmaLinux (using `nmcli`)

```bash
# 1. Create bridge 'br0'
nmcli con add type bridge ifname br0 con-name br0

# 2. Attach physical interface (e.g., eth0)
# REPLACE 'eth0' with your actual interface name (check 'ip a')
nmcli con add type bridge-slave ifname eth0 master br0

# 3. Activate
nmcli con up br0

```

### Option B: Ubuntu (using Netplan)

Edit `/etc/netplan/00-installer-config.yaml` (example config):

```yaml
network:
  version: 2
  renderer: networkd
  ethernets:
    eth0:
      dhcp4: no
  bridges:
    br0:
      interfaces: [eth0]
      dhcp4: yes

```

Apply changes: `sudo netplan apply`

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

Run the tool as root:

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
| `exec: "cloud-localds": executable file not found` | **RHEL:** Install `genisoimage` & manually download `cloud-localds`. <br>

<br> **Ubuntu:** Install `cloud-image-utils`. |
| `Permission denied` / `libvirt error` | You must run the tool as `root` (sudo). |
| `VNC Connection Refused` | Check your firewall settings (Step 2). Ensure ports 5900+ are open. |
| `No IP Address found` | Wait 30 seconds for the VM to boot. If using Bridge, ensure your router has DHCP enabled. |

---

**Author:** [Shaman786](https://www.google.com/search?q=https://github.com/shaman786)

```

```
