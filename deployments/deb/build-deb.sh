#!/usr/bin/env bash
#
# build-deb.sh — 将 api-server 和 node-manager 二进制打包为 .deb
#
# 用法:
#   ./deployments/deb/build-deb.sh [--version 0.9.0] [--arch amd64]
#
# 前提: 二进制文件已编译好放在 bin/ 目录:
#   bin/api-server        (或 bin/api-server-linux-amd64)
#   bin/nodemanager       (或 bin/nodemanager-linux-amd64)
#
# 输出: dist/ 目录下生成 .deb 文件
#
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

VERSION="${VERSION:-0.9.0}"
ARCH="${ARCH:-amd64}"
MAINTAINER="${MAINTAINER:-agents-admin team <admin@example.com>}"

# 解析命令行参数
while [[ $# -gt 0 ]]; do
  case "$1" in
    --version) VERSION="$2"; shift 2 ;;
    --arch)    ARCH="$2";    shift 2 ;;
    *)         echo "Unknown option: $1"; exit 1 ;;
  esac
done

DIST_DIR="$PROJECT_ROOT/dist"
mkdir -p "$DIST_DIR"

# ---------- 工具函数 ----------

find_binary() {
  local name="$1"
  local candidates=(
    "$PROJECT_ROOT/bin/${name}"
    "$PROJECT_ROOT/bin/${name}-linux-${ARCH}"
  )
  for c in "${candidates[@]}"; do
    if [[ -f "$c" ]]; then
      echo "$c"
      return
    fi
  done
  echo ""
}

build_deb() {
  local pkg_name="$1"      # e.g. agents-admin-api-server
  local binary_src="$2"    # 源二进制路径
  local binary_name="$3"   # 安装后的二进制名
  local service_name="$4"  # systemd service 名
  local description="$5"
  shift 5
  local config_files=("$@")  # 配置文件列表

  echo "==> Building ${pkg_name}_${VERSION}_${ARCH}.deb ..."

  local BUILD_DIR
  BUILD_DIR="$(mktemp -d)"
  trap "rm -rf '$BUILD_DIR'" RETURN

  # 目录结构
  mkdir -p "$BUILD_DIR/DEBIAN"
  mkdir -p "$BUILD_DIR/usr/bin"
  mkdir -p "$BUILD_DIR/lib/systemd/system"
  mkdir -p "$BUILD_DIR/etc/agents-admin"

  # 复制二进制
  cp "$binary_src" "$BUILD_DIR/usr/bin/$binary_name"
  chmod 755 "$BUILD_DIR/usr/bin/$binary_name"

  # 复制配置文件
  > "$BUILD_DIR/DEBIAN/conffiles"
  for cf in "${config_files[@]}"; do
    cp "$SCRIPT_DIR/config/$cf" "$BUILD_DIR/etc/agents-admin/$cf"
    echo "/etc/agents-admin/$cf" >> "$BUILD_DIR/DEBIAN/conffiles"
  done

  # 计算安装大小 (KB)
  local installed_size
  installed_size=$(du -sk "$BUILD_DIR" | cut -f1)

  # DEBIAN/control
  cat > "$BUILD_DIR/DEBIAN/control" <<EOF
Package: ${pkg_name}
Version: ${VERSION}
Section: net
Priority: optional
Architecture: ${ARCH}
Installed-Size: ${installed_size}
Maintainer: ${MAINTAINER}
Description: ${description}
EOF

  # systemd service 文件
  cp "$SCRIPT_DIR/systemd/${service_name}.service" \
     "$BUILD_DIR/lib/systemd/system/${service_name}.service"

  # DEBIAN/postinst
  cat > "$BUILD_DIR/DEBIAN/postinst" <<'POSTINST'
#!/bin/sh
set -e
SERVICE_NAME="__SERVICE__"

if [ "$1" = "configure" ]; then
  # 创建系统用户（如果不存在）
  if ! id agents-admin >/dev/null 2>&1; then
    useradd --system --no-create-home --shell /usr/sbin/nologin agents-admin || true
  fi

  # 创建必要的目录
  mkdir -p /var/log/agents-admin
  mkdir -p /var/lib/agents-admin
  mkdir -p /etc/agents-admin/certs
  chown -R agents-admin:agents-admin /var/log/agents-admin
  chown -R agents-admin:agents-admin /var/lib/agents-admin

  # 保护 .env 文件权限（仅 root 和服务用户可读）
  if [ -f /etc/agents-admin/*.env ]; then
    chmod 640 /etc/agents-admin/*.env
    chown root:agents-admin /etc/agents-admin/*.env
  fi

  # 重新加载 systemd
  systemctl daemon-reload || true
  # 启用开机自启
  systemctl enable "$SERVICE_NAME" || true
  # 首次安装时启动；升级时重启
  if systemctl is-active --quiet "$SERVICE_NAME"; then
    systemctl restart "$SERVICE_NAME" || true
  else
    systemctl start "$SERVICE_NAME" || true
  fi
fi
POSTINST
  sed -i "s/__SERVICE__/${service_name}/g" "$BUILD_DIR/DEBIAN/postinst"
  chmod 755 "$BUILD_DIR/DEBIAN/postinst"

  # DEBIAN/prerm
  cat > "$BUILD_DIR/DEBIAN/prerm" <<'PRERM'
#!/bin/sh
set -e
SERVICE_NAME="__SERVICE__"

if [ "$1" = "remove" ] || [ "$1" = "deconfigure" ]; then
  systemctl stop "$SERVICE_NAME" || true
  systemctl disable "$SERVICE_NAME" || true
fi
PRERM
  sed -i "s/__SERVICE__/${service_name}/g" "$BUILD_DIR/DEBIAN/prerm"
  chmod 755 "$BUILD_DIR/DEBIAN/prerm"

  # DEBIAN/postrm
  cat > "$BUILD_DIR/DEBIAN/postrm" <<'POSTRM'
#!/bin/sh
set -e
if [ "$1" = "purge" ]; then
  rm -rf /etc/agents-admin/__CONFIG__ || true
  systemctl daemon-reload || true
fi
POSTRM
  sed -i "s/__CONFIG__/${config_env}/g" "$BUILD_DIR/DEBIAN/postrm"
  chmod 755 "$BUILD_DIR/DEBIAN/postrm"

  # 构建 deb
  local deb_file="${DIST_DIR}/${pkg_name}_${VERSION}_${ARCH}.deb"
  dpkg-deb --build "$BUILD_DIR" "$deb_file"
  echo "    => $deb_file"
}

# ---------- 构建 api-server ----------

API_BIN="$(find_binary api-server)"
if [[ -z "$API_BIN" ]]; then
  echo "ERROR: api-server binary not found in bin/. Run 'make build' or 'make release-linux' first."
  exit 1
fi

build_deb \
  "agents-admin-api-server" \
  "$API_BIN" \
  "agents-admin-api-server" \
  "agents-admin-api-server" \
  "Agents Admin API Server - 智能体管理平台 API 服务" \
  "api-server.yaml" "api-server.env"

# ---------- 构建 node-manager ----------

NM_BIN="$(find_binary nodemanager)"
if [[ -z "$NM_BIN" ]]; then
  echo "ERROR: nodemanager binary not found in bin/. Run 'make build' first."
  exit 1
fi

build_deb \
  "agents-admin-node-manager" \
  "$NM_BIN" \
  "agents-admin-node-manager" \
  "agents-admin-node-manager" \
  "Agents Admin Node Manager - 智能体节点管理器" \
  "nodemanager.yaml" "node-manager.env"

echo ""
echo "All .deb packages built in ${DIST_DIR}/"
ls -lh "$DIST_DIR"/*.deb
