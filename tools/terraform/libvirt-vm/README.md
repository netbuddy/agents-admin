# Terraform Libvirt VM 自动化创建

使用 Terraform 和 [dmacvicar/libvirt](https://registry.terraform.io/providers/dmacvicar/libvirt/latest/docs) provider (v0.9.2)，通过 `qemu+ssh` 在远程 KVM 宿主机上自动创建虚拟机。

## 架构概述

```
┌──────────────┐  qemu+ssh   ┌──────────────────────────────────┐
│  本地主机      │ ──────────► │  远程 KVM 宿主机                   │
│  (Terraform)  │             │  ┌──────────────────────────────┐ │
│               │             │  │ libvirtd                     │ │
│               │             │  │  ├─ volume: COW overlay      │ │
│               │             │  │  ├─ volume: cloudinit ISO    │ │
│               │             │  │  └─ domain: VM               │ │
│               │             │  └──────────────────────────────┘ │
└──────────────┘              └──────────────────────────────────┘
```

**资源创建流程：**

1. `libvirt_volume.vm_disk` — 基于 cloud image 创建 COW overlay 磁盘，`capacity` 属性直接指定目标大小
2. `libvirt_cloudinit_disk` — 在本地生成 cloud-init ISO（用户、网络、SSH 配置）
3. `libvirt_volume.cloudinit` — 将 ISO 上传到远程存储池
4. `libvirt_domain` — 创建并启动虚拟机
5. VM 内部 cloud-init — `growpart` 自动扩展分区，`resize_rootfs` 扩展文件系统

## 文件说明

### Terraform 配置文件（需提交到仓库）

| 文件 | 说明 |
|------|------|
| `versions.tf` | Terraform 版本约束和 provider 声明（dmacvicar/libvirt v0.9.2） |
| `variables.tf` | 所有可配置变量定义（VM 规格、网络、磁盘、SSH 等） |
| `main.tf` | 核心资源定义：volume、cloud-init、domain |
| `outputs.tf` | 输出值：VM 名称、UUID、IP、磁盘路径、SSH 命令 |
| `cloud_init.cfg` | cloud-init 用户数据模板（hostname、用户、SSH key、growpart） |
| `network_config.cfg` | cloud-init 网络配置模板（静态 IP、网关、DNS） |
| `terraform.tfvars.tpl` | 变量文件模板（由渲染脚本填充） |
| `scripts/render_tfvars.sh` | 模板渲染脚本，输出到 `/tmp/libvirt-vm/terraform.tfvars` |

### 中间过程文件和临时文件（不提交到仓库）

| 文件/目录 | 说明 | 生成方式 |
|-----------|------|----------|
| `.terraform/` | Provider 插件缓存目录 | `terraform init` |
| `.terraform.lock.hcl` | Provider 版本锁定文件 | `terraform init` |
| `terraform.tfstate` | 当前基础设施状态（含敏感信息） | `terraform apply` |
| `terraform.tfstate.backup` | 状态备份文件 | `terraform apply` |
| `/tmp/libvirt-vm/terraform.tfvars` | 渲染后的变量文件 | `scripts/render_tfvars.sh` |

> **安全提示**：`terraform.tfstate` 可能包含 SSH 密钥、IP 地址等敏感信息，**严禁**提交到 Git 仓库。

## 前置条件

### 本地环境

- **Terraform** >= 1.4
- **genisoimage**（生成 cloud-init ISO）：`sudo apt install genisoimage`
- **SSH 密钥对**：默认读取 `~/.ssh/id_ed25519.pub`
- **envsubst**（模板渲染）：通常已预装（`gettext` 包）

### 远程 KVM 宿主机

- **libvirtd** 服务运行中
- **QEMU/KVM** 已安装
- **SSH 免密登录**已配置（本地公钥已添加到远程主机）
- **存储池**已创建（默认名称 `images`，路径 `/var/lib/libvirt/images`）
- **Ubuntu cloud image** 已下载到存储池中：
  ```bash
  sudo wget -O /var/lib/libvirt/images/ubuntu-24.04-cloudimg.qcow2 \
    https://cloud-images.ubuntu.com/releases/24.04/release/ubuntu-24.04-server-cloudimg-amd64.img
  sudo virsh pool-refresh images
  ```
- **网桥**已配置（bridge 模式需要，如 `cloudbr0`）

## 使用方法

### 快速开始

```bash
# 1. 进入目录
cd tools/terraform/libvirt-vm

# 2. 渲染变量文件（使用默认值）
./scripts/render_tfvars.sh

# 3. 初始化
terraform init

# 4. 创建 VM
terraform apply -var-file=/tmp/libvirt-vm/terraform.tfvars

# 5. SSH 登录
ssh ubuntu@192.168.213.100
```

### 自定义配置

通过环境变量覆盖默认值：

```bash
VM_NAME=dev-server VM_VCPU=4 VM_MEMORY=4096 VM_DISK_SIZE=40 \
  VM_IP="192.168.213.101/24" \
  ./scripts/render_tfvars.sh
```

或创建 `values.env` 文件：

```bash
cat > values.env << 'EOF'
LIBVIRT_URI=qemu+ssh://admin@10.0.0.1/system
VM_NAME=prod-server
VM_VCPU=8
VM_MEMORY=16384
VM_DISK_SIZE=100
BASE_IMAGE_PATH=/var/lib/libvirt/images/ubuntu-24.04-cloudimg.qcow2
BRIDGE_NAME=br0
VM_IP=10.0.0.50/24
VM_GATEWAY=10.0.0.1
VM_DNS_LIST=8.8.8.8,1.1.1.1
EOF

./scripts/render_tfvars.sh -f values.env
```

### 变量说明

| 环境变量 | 默认值 | 说明 |
|----------|--------|------|
| `LIBVIRT_URI` | `qemu+ssh://public@192.168.213.10/system` | libvirt 远程连接 URI |
| `VM_NAME` | `ubuntu-vm` | 虚拟机名称（宿主机唯一） |
| `VM_VCPU` | `2` | vCPU 数量 |
| `VM_MEMORY` | `2048` | 内存大小（MiB） |
| `VM_DISK_SIZE` | `20` | 磁盘大小（GB） |
| `BASE_IMAGE_PATH` | `/var/lib/libvirt/images/ubuntu-24.04-cloudimg.qcow2` | 基础云镜像完整路径 |
| `STORAGE_POOL` | `images` | libvirt 存储池名称 |
| `BRIDGE_NAME` | `cloudbr0` | 网桥名称 |
| `VM_IP` | `192.168.213.100/24` | 静态 IP（CIDR 格式） |
| `VM_GATEWAY` | `192.168.213.1` | 默认网关 |
| `VM_DNS_LIST` | `223.5.5.5,8.8.8.8` | DNS 服务器（逗号分隔） |
| `VM_USERNAME` | `ubuntu` | VM 内用户名 |

### 常用操作

```bash
# 查看当前状态
terraform show

# 销毁 VM
terraform destroy -var-file=/tmp/libvirt-vm/terraform.tfvars

# 使用自定义网桥
BRIDGE_NAME=br0 ./scripts/render_tfvars.sh
terraform apply -var-file=/tmp/libvirt-vm/terraform.tfvars
```

## 技术细节

### 磁盘大小扩展

v0.9.2 provider 完全重写了 volume 资源，使用 `capacity`（int64，字节）+ `backing_store` 替代旧的 `size` + `base_volume_name`，彻底解决了 v0.8.x 中的 int32 溢出问题。磁盘扩展流程：

1. **Terraform 层**：`libvirt_volume` 的 `capacity` 属性直接指定目标磁盘大小（如 20GB = 21474836480 字节）
2. **VM 内部**：cloud-init 的 `growpart` 模块自动扩展分区，`resize_rootfs` 扩展文件系统

### Cloud Image vs 安装 ISO

本配置使用 **Ubuntu cloud image**（`*.img` / `*.qcow2`，~600MB），而非安装 ISO：

- Cloud image 预装了 `cloud-init`，支持自动化配置
- 安装 ISO 需要交互式安装流程，不适合 Terraform 自动化
- Cloud image 虚拟大小通常较小（~3.5GB），通过 `capacity` 属性在 overlay 层扩展

### COW Overlay

VM 磁盘采用 QEMU Copy-On-Write（COW）overlay 机制：

- 基础镜像（cloud image）保持只读，可被多个 VM 共享
- 每个 VM 创建独立的 overlay 文件，仅存储与基础镜像的差异
- 节省存储空间，加速 VM 创建

## 故障排查

### cloud-init 报错 `package_update_upgrade_install`

**原因**：VM 内 `apt-get update` 失败（DNS 未就绪或网络环境有透明代理干扰）。

**影响**：仅 `qemu-guest-agent` 未安装，不影响 VM 核心功能。

**修复**：SSH 登录 VM 后手动安装：
```bash
sudo apt-get update && sudo apt-get install -y qemu-guest-agent
sudo systemctl enable --now qemu-guest-agent
```

### SSH 连接 `Host key changed` 警告

**原因**：销毁并重建 VM 后，新 VM 生成了不同的 host key。

**修复**：
```bash
ssh-keygen -R 192.168.213.100
```

### 基础镜像被锁定

**原因**：另一个 VM 正在使用该镜像作为 backing store。

**修复**：复制一份独立的基础镜像：
```bash
ssh user@host "sudo cp /var/lib/libvirt/images/original.qcow2 /var/lib/libvirt/images/copy.qcow2"
ssh user@host "sudo virsh pool-refresh images"
```
然后更新渲染脚本中的 `BASE_IMAGE_PATH`。
