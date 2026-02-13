# Remote libvirt host
libvirt_uri = "${LIBVIRT_URI}"

# VM configuration
vm_name      = "${VM_NAME}"
vm_vcpu      = ${VM_VCPU}
vm_memory    = ${VM_MEMORY}
vm_disk_size = ${VM_DISK_SIZE}

# Base image (full path on remote host)
base_image_path = "${BASE_IMAGE_PATH}"
storage_pool    = "${STORAGE_POOL}"

# Network
bridge_name = "${BRIDGE_NAME}"

# Static IP
vm_ip      = "${VM_IP}"
vm_gateway = "${VM_GATEWAY}"
vm_dns     = [${VM_DNS}]

# VM user
vm_username = "${VM_USERNAME}"
