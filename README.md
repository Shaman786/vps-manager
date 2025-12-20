# VPS Manager

A lightweight infrastructure-as-code tool written in Go. It automates the lifecycle of KVM virtual machines, handling disk provisioning (QCOW2), Cloud-Init configuration, and network bootstrapping.

## Features
- **Automated Image Management:** Downloads and caches OS images (Ubuntu, Debian, Rocky).
- **Copy-on-Write Storage:** Uses QCOW2 backing files for instant provisioning.
- **Cloud-Init Integration:** dynamically generates user-data for SSH keys and user management.
- **Modular Architecture:** Clean separation of concerns (VM logic vs Config logic).

## Tech Stack
- **Language:** Go (Golang)
- **Hypervisor:** KVM / QEMU / Libvirt
- **Config:** Cloud-Init (YAML)

## Usage
```bash
go run cmd/vps-manager/main.go