// 12
package images

type OSImage struct {
	Name     string
	URL      string
	Filename string
}

var Available = []OSImage{
	{
		Name:     "Ubuntu 24.04 LTS (Noble)",
		URL:      "https://cloud-images.ubuntu.com/noble/current/noble-server-cloudimg-amd64.img",
		Filename: "ubuntu-24.04.img",
	},
	{
		Name:     "Ubuntu 22.04 LTS (Jammy)",
		URL:      "https://cloud-images.ubuntu.com/jammy/current/jammy-server-cloudimg-amd64.img",
		Filename: "ubuntu-22.04.img",
	},

	// --- DEBIAN (Stable & Slim) ---
	{
		Name:     "Debian 12 (Bookworm)",
		URL:      "https://cloud.debian.org/images/cloud/bookworm/latest/debian-12-generic-amd64.qcow2",
		Filename: "debian-12.qcow2",
	},
	{
		Name:     "Debian 11 (Bullseye)",
		URL:      "https://cloud.debian.org/images/cloud/bullseye/latest/debian-11-generic-amd64.qcow2",
		Filename: "debian-11.qcow2",
	},

	// --- ENTERPRISE (RHEL Clones - Free) ---
	{
		Name:     "AlmaLinux 9 (RHEL Compatible)",
		URL:      "https://repo.almalinux.org/almalinux/9/cloud/x86_64/images/AlmaLinux-9-GenericCloud-latest.x86_64.qcow2",
		Filename: "almalinux-9.qcow2",
	},
	{
		Name:     "Rocky Linux 9 (RHEL Compatible)",
		URL:      "https://dl.rockylinux.org/pub/rocky/9/images/x86_64/Rocky-9-GenericCloud.latest.x86_64.qcow2",
		Filename: "rockylinux-9.qcow2",
	},
	{
		Name:     "CentOS Stream 9",
		URL:      "https://cloud.centos.org/centos/9-stream/x86_64/images/CentOS-Stream-GenericCloud-9-latest.x86_64.qcow2",
		Filename: "centos-stream-9.qcow2",
	},

	// --- BLEEDING EDGE ---
	{
		Name: "Fedora Cloud (Bleeding Edge)",
		// Fedora updates filenames frequently, so we use a specific stable version here to prevent 404s
		URL:      "DYNAMIC_FEDORA",
		Filename: "fedora-latest.qcow2",
	},
	{
		Name:     "Arch Linux (Rolling)",
		URL:      "https://geo.mirror.pkgbuild.com/images/latest/Arch-Linux-x86_64-cloudimg.qcow2",
		Filename: "arch-linux.qcow2",
	},

	// --- LIGHTWEIGHT / SPECIALTY ---
	{
		Name:     "OpenSUSE Leap 15.6",
		URL:      "https://download.opensuse.org/repositories/Cloud:/Images:/Leap_15.6/images/openSUSE-Leap-15.6.x86_64-NoCloud.qcow2",
		Filename: "opensuse-leap-15.6.qcow2",
	},
	{
		Name: "Alpine Linux 3.21 (Virtual)",
		// Alpine "NoCloud" image is optimized for KVM and cloud-init
		URL:      "https://dl-cdn.alpinelinux.org/alpine/v3.21/releases/cloud/nocloud_alpine-3.21.0-x86_64-bios-cloudinit-r0.qcow2",
		Filename: "alpine-3.21.qcow2",
	},
}
