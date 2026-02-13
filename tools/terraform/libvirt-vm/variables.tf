variable "libvirt_uri" {
  description = "Libvirt connection URI"
  type        = string
  default     = "qemu+ssh://public@192.168.213.10/system"
}

variable "vm_name" {
  description = "Name of the virtual machine"
  type        = string
  default     = "ubuntu-vm"
}

variable "vm_vcpu" {
  description = "Number of vCPUs"
  type        = number
  default     = 2
}

variable "vm_memory" {
  description = "Memory in MiB"
  type        = number
  default     = 2048
}

variable "vm_disk_size" {
  description = "Disk size in GB (must be >= base image virtual size)"
  type        = number
  default     = 20
}

variable "base_image_path" {
  description = "Full path to the base cloud image on the remote host"
  type        = string
  default     = "/var/lib/libvirt/images/ubuntu-24.04-cloudimg.qcow2"
}

variable "storage_pool" {
  description = "Libvirt storage pool name"
  type        = string
  default     = "images"
}

variable "network_mode" {
  description = "Network mode: bridge or nat"
  type        = string
  default     = "bridge"

  validation {
    condition     = contains(["bridge", "nat"], var.network_mode)
    error_message = "network_mode must be 'bridge' or 'nat'."
  }
}

variable "bridge_name" {
  description = "Bridge device name (used when network_mode is 'bridge')"
  type        = string
  default     = "cloudbr0"
}

variable "nat_network" {
  description = "Libvirt NAT network name (used when network_mode is 'nat')"
  type        = string
  default     = "default"
}

variable "vm_ip" {
  description = "Static IP address for the VM (CIDR notation, e.g. 192.168.213.100/24)"
  type        = string
  default     = "192.168.213.100/24"
}

variable "vm_gateway" {
  description = "Default gateway for the VM"
  type        = string
  default     = "192.168.213.1"
}

variable "vm_dns" {
  description = "DNS servers for the VM"
  type        = list(string)
  default     = ["223.5.5.5", "8.8.8.8"]
}

variable "ssh_public_key" {
  description = "SSH public key for VM access (auto-detected from ~/.ssh/ if empty)"
  type        = string
  default     = ""
}

variable "vm_username" {
  description = "Default user in the VM"
  type        = string
  default     = "ubuntu"
}
