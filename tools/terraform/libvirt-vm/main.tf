# Auto-detect SSH public key if not provided
locals {
  ssh_public_key = var.ssh_public_key != "" ? var.ssh_public_key : chomp(file(pathexpand("~/.ssh/id_ed25519.pub")))
}

# VM disk: COW overlay on existing base cloud image
# v0.9.2 uses capacity + backing_store (int64, no overflow)
resource "libvirt_volume" "vm_disk" {
  name     = "${var.vm_name}.qcow2"
  pool     = var.storage_pool
  capacity = var.vm_disk_size * 1073741824

  target = {
    format = { type = "qcow2" }
  }

  backing_store = {
    path   = var.base_image_path
    format = { type = "qcow2" }
  }
}

# Cloud-init ISO generation (local)
resource "libvirt_cloudinit_disk" "init" {
  name = "${var.vm_name}-cloudinit"

  user_data = templatefile("${path.module}/cloud_init.cfg", {
    hostname       = var.vm_name
    username       = var.vm_username
    ssh_public_key = local.ssh_public_key
  })

  meta_data = yamlencode({
    instance-id    = var.vm_name
    local-hostname = var.vm_name
  })

  network_config = templatefile("${path.module}/network_config.cfg", {
    vm_ip      = var.vm_ip
    vm_gateway = var.vm_gateway
    vm_dns     = var.vm_dns
  })
}

# Upload cloud-init ISO to remote storage pool
resource "libvirt_volume" "cloudinit" {
  name = "${var.vm_name}-cloudinit.iso"
  pool = var.storage_pool

  create = {
    content = {
      url = libvirt_cloudinit_disk.init.path
    }
  }
}

# Virtual machine domain (v0.9.2 schema)
resource "libvirt_domain" "vm" {
  name        = var.vm_name
  type        = "kvm"
  memory      = var.vm_memory
  memory_unit = "MiB"
  vcpu        = var.vm_vcpu
  autostart   = true
  running     = true

  os = {
    type     = "hvm"
    type_arch = "x86_64"
  }

  devices = {
    disks = [
      {
        driver = { name = "qemu", type = "qcow2" }
        source = {
          volume = {
            pool   = var.storage_pool
            volume = libvirt_volume.vm_disk.name
          }
        }
        target = {
          dev = "vda"
          bus = "virtio"
        }
      },
      {
        driver = { name = "qemu", type = "raw" }
        source = {
          volume = {
            pool   = var.storage_pool
            volume = libvirt_volume.cloudinit.name
          }
        }
        target = {
          dev = "vdb"
          bus = "virtio"
        }
      },
    ]

    interfaces = [
      {
        model = { type = "virtio" }
        source = {
          bridge = { bridge = var.bridge_name }
        }
      },
    ]
  }
}
