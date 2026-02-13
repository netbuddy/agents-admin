output "vm_name" {
  description = "Name of the created VM"
  value       = libvirt_domain.vm.name
}

output "vm_uuid" {
  description = "UUID of the created VM"
  value       = libvirt_domain.vm.uuid
}

output "vm_ip" {
  description = "Configured static IP of the VM"
  value       = var.vm_ip
}

output "disk_path" {
  description = "Path to the VM disk on the remote host"
  value       = libvirt_volume.vm_disk.path
}

output "ssh_command" {
  description = "SSH command to connect to the VM"
  value       = "ssh ${var.vm_username}@${split("/", var.vm_ip)[0]}"
}
