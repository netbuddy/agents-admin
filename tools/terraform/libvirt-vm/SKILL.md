---
name: libvirt-vm
description: 使用 Terraform 在远程 libvirt 宿主机上创建和管理 KVM/QEMU 虚拟机。当用户需要创建虚拟机、配置静态 IP、桥接网络、磁盘大小、cloud-init 或 SSH 访问时使用。触发词包括"创建虚拟机"、"配置 KVM"、"libvirt VM"等。
---

# Libvirt 虚拟机配置

通过 `qemu+ssh` 使用 `dmacvicar/libvirt` Terraform provider (v0.9.2) 在远程 KVM 宿主机上创建 Ubuntu cloud 虚拟机。

## 输入格式

用户以自然语言描述虚拟机配置，至少包含以下信息（括号内为默认值，缺失时使用默认值）：

```
远程宿主机连接地址：
<SSH 用户名@宿主机 IP，例如 public@192.168.213.10>

虚拟机名称：
<在宿主机上唯一的名称，例如 ubuntu-vm>

硬件规格：
<CPU 核数（默认 2）、内存大小 MiB（默认 2048）、磁盘大小 GB（默认 20）>

基础云镜像路径：
<宿主机上 qcow2 云镜像的完整路径，例如 /var/lib/libvirt/images/ubuntu-24.04-cloudimg.qcow2>

存储池名称：
<libvirt 存储池名（默认 images）>

网络配置：
<桥接设备名称（默认 cloudbr0）、VM 静态 IP/掩码（默认 192.168.213.100/24）、网关、DNS>

VM 用户名：
<cloud-init 创建的用户名（默认 ubuntu）>
```

## 工作流

### 1) 确认前置条件

- **本地**: Terraform >= 1.4、`genisoimage`、SSH 密钥对 (`~/.ssh/id_ed25519.pub`)
- **本地**: Provider 全局缓存已配置（见下方说明）
- **远程宿主机**: libvirtd 运行中、SSH 密钥认证已配置、存储池中有 Ubuntu 云镜像

#### Provider 全局缓存（避免重复下载）

`dmacvicar/libvirt` provider 约 20MB，每次 `terraform init` 默认都会下载。启用全局缓存后多个项目共享同一份 provider 二进制，后续 init 直接符号链接到缓存，无需重复下载。

**一次性配置**（创建缓存目录 + CLI 配置文件）：

```bash
mkdir -p ~/.terraform.d/plugin-cache
cat > ~/.terraformrc << 'EOF'
plugin_cache_dir = "$HOME/.terraform.d/plugin-cache"
EOF
```

或使用环境变量（适合 CI）：

```bash
export TF_PLUGIN_CACHE_DIR="$HOME/.terraform.d/plugin-cache"
```

> **注意**：缓存目录不会自动清理，版本积累后需手动删除旧版本。全局缓存不保证并发安全，CI 中避免多个 `terraform init` 同时写入同一目录。

如果云镜像缺失，下载：

```bash
ssh user@host "sudo wget -O /var/lib/libvirt/images/ubuntu-24.04-cloudimg.qcow2 \
  https://cloud-images.ubuntu.com/releases/24.04/release/ubuntu-24.04-server-cloudimg-amd64.img && \
  sudo virsh pool-refresh images"
```

> 必须使用**云镜像**（`*.img`，约 600MB），而非安装 ISO。云镜像内置 `cloud-init` 支持自动化配置。

### 2) 渲染变量文件

根据用户输入，设置环境变量并运行渲染脚本，生成 `/tmp/libvirt-vm/terraform.tfvars`：

```bash
VM_NAME=myvm VM_VCPU=4 VM_MEMORY=4096 VM_DISK_SIZE=40 \
  VM_IP="192.168.213.101/24" \
  ./scripts/render_tfvars.sh
```

或创建 `values.env` 文件后通过 `-f` 加载：

```bash
./scripts/render_tfvars.sh -f values.env
```

渲染后的变量文件保存在 `/tmp/libvirt-vm/terraform.tfvars`，不会污染技能包目录。

### 3) 执行 Terraform

```bash
terraform init
terraform apply -var-file=/tmp/libvirt-vm/terraform.tfvars
```

### 4) 验证虚拟机

```bash
ssh ubuntu@<VM_IP>
sudo fdisk -l /dev/vda   # 确认磁盘大小
df -h /                   # 确认文件系统已扩展
```

### 5) 配置代理（可选）

如果 VM 需要通过 HTTP 代理访问外网：

```bash
./scripts/setup_proxy.sh <user@host> <proxy_host:port> [no_proxy]
```

脚本会配置 `/etc/environment`、`/etc/apt/apt.conf.d/95proxy`、`/etc/profile.d/proxy.sh` 三处，并自动验证代理连通性。

### 6) 清理临时文件

VM 创建完成后，清理技能包目录中的 Terraform 状态和临时文件：

```bash
./scripts/cleanup.sh            # 执行清理
./scripts/cleanup.sh --dry-run  # 仅预览，不删除
```

清理范围：`.terraform/`、`.terraform.lock.hcl`、`terraform.tfstate*`、`/tmp/libvirt-vm/`。

## 文件结构

| 文件 | 作用 |
|------|------|
| [versions.tf](versions.tf) | Provider 声明 (dmacvicar/libvirt v0.9.2) |
| [variables.tf](variables.tf) | 所有可配置输入变量 |
| [main.tf](main.tf) | 资源定义：volume、cloud-init、domain |
| [outputs.tf](outputs.tf) | 输出：VM 名称、UUID、IP、SSH 命令 |
| [cloud_init.cfg](cloud_init.cfg) | Cloud-init 用户数据模板（用户、SSH、growpart） |
| [network_config.cfg](network_config.cfg) | Cloud-init 网络配置模板（静态 IP） |
| [terraform.tfvars.tpl](terraform.tfvars.tpl) | 变量文件模板（由渲染脚本填充） |
| [scripts/render_tfvars.sh](scripts/render_tfvars.sh) | 模板渲染脚本，输出到 /tmp |
| [scripts/cleanup.sh](scripts/cleanup.sh) | 清理 Terraform 状态和临时文件 |
| [scripts/setup_proxy.sh](scripts/setup_proxy.sh) | 远程 VM 全局代理配置（environment + apt + profile） |

## 资源创建顺序

```
libvirt_volume (COW 叠加层, 指定 capacity) ──────────────────► libvirt_domain (虚拟机)
                                                                     │
libvirt_cloudinit_disk (本地 ISO) ► libvirt_volume (上传 ISO) ──────┘
```

1. **`libvirt_volume.vm_disk`** — 基于云镜像的 COW 叠加层，`capacity` 属性直接指定目标磁盘大小
2. **`libvirt_cloudinit_disk`** — 在本地生成包含用户数据和网络配置的 cloud-init ISO
3. **`libvirt_volume.cloudinit`** — 将 ISO 上传到远程存储池
4. **`libvirt_domain`** — 创建并启动虚拟机；cloud-init 运行 `growpart` + `resize_rootfs` 自动扩展文件系统

## 已知问题

### cloud-init 中 apt 失败

有透明代理的网络环境可能导致 cloud-init 中 `apt-get update` 失败。这仅影响可选包安装（`qemu-guest-agent`），不影响核心 VM 功能。启动后手动修复：

```bash
sudo apt-get update && sudo apt-get install -y qemu-guest-agent
```

### 基础镜像锁定

如果其他虚拟机正在使用相同的基础镜像作为 backing store，创建新的 COW 叠加层会失败。解决方案：复制基础镜像为新文件名并更新 `base_image_path`。
