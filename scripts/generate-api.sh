#!/bin/bash
# generate-api.sh — 从 bundled OpenAPI 规范生成 Go 类型代码
#
# 生成两个文件：
#   models.gen.go  — 所有类型定义（struct / enum / request body / params）
#   spec.gen.go    — 嵌入的 OpenAPI JSON 规范
#
# 注：oapi-codegen 的 per-file 生成（self-mapping）存在 enum 常量命名冲突的已知限制
# （issue #549），单文件是唯一能保证正确 enum 前缀命名的方案。
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

OAPI_CODEGEN="${OAPI_CODEGEN:-${HOME}/go/bin/oapi-codegen}"
OPENAPI_DIR="${PROJECT_ROOT}/api/openapi"
OUTPUT_DIR="${PROJECT_ROOT}/api/generated/go"
CODEGEN_DIR="${PROJECT_ROOT}/api/codegen"
BUNDLED="${OPENAPI_DIR}/bundled.yaml"

# 检查 oapi-codegen
if ! command -v "$OAPI_CODEGEN" &>/dev/null; then
    echo "ERROR: oapi-codegen not found at $OAPI_CODEGEN"
    echo "Install: go install github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@latest"
    exit 1
fi

echo "Using oapi-codegen: $($OAPI_CODEGEN -version 2>&1 | tail -1)"
echo ""

mkdir -p "$OUTPUT_DIR"

TMP_DIR=$(mktemp -d)
trap "rm -rf $TMP_DIR" EXIT

# ──────────────────────────────────────────────────────
# Step 1: Bundle OpenAPI specs
# ──────────────────────────────────────────────────────
if [ ! -f "$BUNDLED" ]; then
    echo "Step 1: Bundling OpenAPI specs..."
    npx @redocly/cli bundle "${OPENAPI_DIR}/openapi.yaml" -o "$BUNDLED"
else
    echo "Step 1: Using existing bundled.yaml"
fi
echo ""

# ──────────────────────────────────────────────────────
# Step 2: Generate models.gen.go
# ──────────────────────────────────────────────────────
MODELS_CONFIG="${CODEGEN_DIR}/models.yaml"
echo -n "Step 2: Generating models.gen.go ... "
"$OAPI_CODEGEN" --config "$MODELS_CONFIG" "$BUNDLED" 2>"${TMP_DIR}/models.err"
types=$(grep -c "^type " "${OUTPUT_DIR}/models.gen.go" 2>/dev/null | tr -d '[:space:]' || true)
echo "OK (${types:-0} types)"

# ──────────────────────────────────────────────────────
# Step 3: Generate spec.gen.go
# ──────────────────────────────────────────────────────
SPEC_CONFIG="${CODEGEN_DIR}/spec.yaml"
echo -n "Step 3: Generating spec.gen.go ... "
if "$OAPI_CODEGEN" --config "$SPEC_CONFIG" "$BUNDLED" 2>"${TMP_DIR}/spec.err"; then
    echo "OK"
else
    echo "FAILED"
    cat "${TMP_DIR}/spec.err" >&2
fi

echo ""
echo "Done!"
ls -la "$OUTPUT_DIR"/*.gen.go 2>/dev/null | awk '{printf "  %-50s %s bytes\n", $NF, $5}'
