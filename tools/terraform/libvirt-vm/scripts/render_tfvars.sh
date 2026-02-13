#!/usr/bin/env bash
# render_tfvars.sh — 将 terraform.tfvars.tpl 模板渲染为实际变量文件
# 渲染后的文件保存到 /tmp/libvirt-vm/terraform.tfvars，避免污染技能包
#
# 用法:
#   ./scripts/render_tfvars.sh                    # 使用默认值
#   ./scripts/render_tfvars.sh -f values.env      # 从文件加载变量
#   VM_NAME=myvm VM_VCPU=4 ./scripts/render_tfvars.sh  # 环境变量覆盖
#
# 优先级: 环境变量 > values.env 文件 > 默认值

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"
TEMPLATE="${PROJECT_DIR}/terraform.tfvars.tpl"
OUTPUT_DIR="/tmp/libvirt-vm"
OUTPUT_FILE="${OUTPUT_DIR}/terraform.tfvars"

# 默认值
: "${LIBVIRT_URI:=qemu+ssh://public@192.168.213.10/system}"
: "${VM_NAME:=ubuntu-vm}"
: "${VM_VCPU:=2}"
: "${VM_MEMORY:=2048}"
: "${VM_DISK_SIZE:=20}"
: "${BASE_IMAGE_PATH:=/var/lib/libvirt/images/ubuntu-24.04-cloudimg.qcow2}"
: "${STORAGE_POOL:=images}"
: "${BRIDGE_NAME:=cloudbr0}"
: "${VM_IP:=192.168.213.100/24}"
: "${VM_GATEWAY:=192.168.213.1}"
: "${VM_DNS_LIST:=223.5.5.5,8.8.8.8}"
: "${VM_USERNAME:=ubuntu}"

# 解析命令行参数
while getopts "f:" opt; do
  case $opt in
    f)
      if [[ -f "$OPTARG" ]]; then
        # shellcheck source=/dev/null
        source "$OPTARG"
      else
        echo "错误: 文件 $OPTARG 不存在" >&2
        exit 1
      fi
      ;;
    *)
      echo "用法: $0 [-f values.env]" >&2
      exit 1
      ;;
  esac
done

# 将逗号分隔的 DNS 列表转为 Terraform 列表格式: "a", "b"
VM_DNS=""
IFS=',' read -ra DNS_ARRAY <<< "${VM_DNS_LIST}"
for i in "${!DNS_ARRAY[@]}"; do
  dns=$(echo "${DNS_ARRAY[$i]}" | xargs)  # trim whitespace
  if [[ $i -gt 0 ]]; then
    VM_DNS="${VM_DNS}, "
  fi
  VM_DNS="${VM_DNS}\"${dns}\""
done

export LIBVIRT_URI VM_NAME VM_VCPU VM_MEMORY VM_DISK_SIZE
export BASE_IMAGE_PATH STORAGE_POOL BRIDGE_NAME
export VM_IP VM_GATEWAY VM_DNS VM_USERNAME

# 创建输出目录
mkdir -p "${OUTPUT_DIR}"

# 渲染模板
envsubst < "${TEMPLATE}" > "${OUTPUT_FILE}"

echo "已生成: ${OUTPUT_FILE}"
echo ""
echo "使用方式:"
echo "  cd ${PROJECT_DIR}"
echo "  terraform plan  -var-file=${OUTPUT_FILE}"
echo "  terraform apply -var-file=${OUTPUT_FILE}"
