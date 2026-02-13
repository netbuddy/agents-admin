#!/usr/bin/env bash
# cleanup.sh — 清理技能包执行后生成的临时目录和文件
#
# 清理范围:
#   1. 技能包目录内: .terraform/ .terraform.lock.hcl terraform.tfstate terraform.tfstate.backup
#   2. /tmp/libvirt-vm/ 渲染脚本生成的临时变量文件
#
# 用法:
#   ./scripts/cleanup.sh          # 清理所有临时文件
#   ./scripts/cleanup.sh --dry-run  # 仅显示将删除的文件，不实际删除

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"
TMP_DIR="/tmp/libvirt-vm"

DRY_RUN=false
if [[ "${1:-}" == "--dry-run" ]]; then
  DRY_RUN=true
  echo "[dry-run] 仅显示将删除的文件:"
  echo ""
fi

removed=0

remove_item() {
  local path="$1"
  if [[ -e "$path" || -L "$path" ]]; then
    if $DRY_RUN; then
      echo "  [将删除] $path"
    else
      rm -rf "$path"
      echo "  [已删除] $path"
    fi
    removed=$((removed + 1))
  fi
}

echo "清理技能包目录: ${PROJECT_DIR}"
remove_item "${PROJECT_DIR}/.terraform"
remove_item "${PROJECT_DIR}/.terraform.lock.hcl"
remove_item "${PROJECT_DIR}/terraform.tfstate"
remove_item "${PROJECT_DIR}/terraform.tfstate.backup"

echo ""
echo "清理临时目录: ${TMP_DIR}"
remove_item "${TMP_DIR}"

echo ""
if [[ $removed -eq 0 ]]; then
  echo "没有需要清理的文件。"
else
  if $DRY_RUN; then
    echo "共 ${removed} 个项目将被删除。运行不带 --dry-run 参数以实际执行。"
  else
    echo "清理完成，共删除 ${removed} 个项目。"
  fi
fi
