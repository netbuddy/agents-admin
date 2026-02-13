#!/usr/bin/env bash
# setup_proxy.sh — 在远程 VM 上配置全局 HTTP/HTTPS 代理
#
# 配置范围:
#   1. /etc/environment — 系统级环境变量（所有用户、所有进程）
#   2. /etc/apt/apt.conf.d/95proxy — APT 包管理器代理
#   3. /etc/profile.d/proxy.sh — 交互式 shell 代理（登录时加载）
#
# 用法:
#   ./scripts/setup_proxy.sh <user@host> <proxy_host:port> [no_proxy]
#
# 示例:
#   ./scripts/setup_proxy.sh yun@192.168.213.28 192.168.213.20:11801
#   ./scripts/setup_proxy.sh yun@192.168.213.28 192.168.213.20:11801 "localhost,127.0.0.1,::1,192.168.213.0/24"

set -euo pipefail

if [[ $# -lt 2 ]]; then
  echo "用法: $0 <user@host> <proxy_host:port> [no_proxy]" >&2
  echo "示例: $0 yun@192.168.213.28 192.168.213.20:11801" >&2
  exit 1
fi

SSH_TARGET="$1"
PROXY_ADDR="$2"
NO_PROXY="${3:-localhost,127.0.0.1,::1}"

PROXY_URL="http://${PROXY_ADDR}"

echo "=== 配置代理: ${SSH_TARGET} ==="
echo "  代理地址: ${PROXY_URL}"
echo "  排除列表: ${NO_PROXY}"
echo ""

# 通过 SSH 远程执行配置
ssh -o StrictHostKeyChecking=no "${SSH_TARGET}" bash -s -- "${PROXY_URL}" "${NO_PROXY}" << 'REMOTE_SCRIPT'
set -euo pipefail

PROXY_URL="$1"
NO_PROXY="$2"

echo "[1/3] 配置 /etc/environment ..."
# 先移除已有的代理行，再追加新配置
sudo sed -i '/^http_proxy=/d; /^https_proxy=/d; /^ftp_proxy=/d; /^no_proxy=/d' /etc/environment
sudo sed -i '/^HTTP_PROXY=/d; /^HTTPS_PROXY=/d; /^FTP_PROXY=/d; /^NO_PROXY=/d' /etc/environment

cat <<EOF | sudo tee -a /etc/environment > /dev/null
http_proxy=${PROXY_URL}
https_proxy=${PROXY_URL}
ftp_proxy=${PROXY_URL}
no_proxy=${NO_PROXY}
HTTP_PROXY=${PROXY_URL}
HTTPS_PROXY=${PROXY_URL}
FTP_PROXY=${PROXY_URL}
NO_PROXY=${NO_PROXY}
EOF
echo "  已写入 /etc/environment"

echo "[2/3] 配置 /etc/apt/apt.conf.d/95proxy ..."
cat <<EOF | sudo tee /etc/apt/apt.conf.d/95proxy > /dev/null
Acquire::http::Proxy "${PROXY_URL}";
Acquire::https::Proxy "${PROXY_URL}";
Acquire::ftp::Proxy "${PROXY_URL}";
EOF
echo "  已写入 /etc/apt/apt.conf.d/95proxy"

echo "[3/3] 配置 /etc/profile.d/proxy.sh ..."
cat <<EOF | sudo tee /etc/profile.d/proxy.sh > /dev/null
export http_proxy="${PROXY_URL}"
export https_proxy="${PROXY_URL}"
export ftp_proxy="${PROXY_URL}"
export no_proxy="${NO_PROXY}"
export HTTP_PROXY="${PROXY_URL}"
export HTTPS_PROXY="${PROXY_URL}"
export FTP_PROXY="${PROXY_URL}"
export NO_PROXY="${NO_PROXY}"
EOF
sudo chmod 644 /etc/profile.d/proxy.sh
echo "  已写入 /etc/profile.d/proxy.sh"

echo ""
echo "代理配置完成。验证中..."

# 立即在当前 session 中加载代理
export http_proxy="${PROXY_URL}"
export https_proxy="${PROXY_URL}"
export no_proxy="${NO_PROXY}"

# 验证: 通过代理访问外网
echo ""
echo "[验证] 通过代理测试外网连接..."
if curl -sS --connect-timeout 10 --max-time 15 -o /dev/null -w "HTTP %{http_code} (%{time_total}s)" https://www.google.com; then
  echo ""
  echo "[验证] 代理连接成功 ✅"
else
  echo ""
  echo "[验证] curl 通过代理访问失败，尝试 wget..."
  if wget -q --timeout=10 -O /dev/null https://www.google.com 2>&1; then
    echo "[验证] wget 代理连接成功 ✅"
  else
    echo "[验证] 代理连接失败 ❌ 请检查代理地址是否可达"
  fi
fi

echo ""
echo "[验证] APT 代理测试..."
if sudo apt-get update -qq 2>&1 | tail -3; then
  echo "[验证] APT 代理正常 ✅"
else
  echo "[验证] APT 更新失败 ❌"
fi
REMOTE_SCRIPT

echo ""
echo "=== 远程代理配置完成 ==="
