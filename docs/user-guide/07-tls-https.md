# TLS/HTTPS 安全通信配置

> 本文介绍如何为 API Server 启用 HTTPS，以及配置 Node Manager 通过 TLS 安全连接 API Server。

## 概述

Agents Admin 支持在 API Server 与 Node Manager 之间启用 TLS 加密通信：

- **API Server**：作为 HTTPS 服务端，使用服务端证书 + 私钥
- **Node Manager**：作为客户端，使用 CA 证书验证 API Server 的身份

```
┌───────────────┐     HTTPS (TLS)      ┌───────────────┐
│  Node Manager │ ◄──────────────────► │  API Server   │
│  (TLS Client) │     验证服务端证书     │  (TLS Server) │
│               │                      │               │
│  ca.pem       │                      │  server.pem   │
│               │                      │  server-key.pem│
│               │                      │  ca.pem (可选) │
└───────────────┘                      └───────────────┘
```

## 快速启用（零配置，推荐内网环境）

如果你只是需要在内网环境中加密通信，无需手动生成证书，只需在 `api-server.yaml` 中添加两行：

```yaml
tls:
  enabled: true
  auto_generate: true
```

API Server 启动时会自动：
1. 生成自签名 CA 和服务端证书（保存到 `/etc/agents-admin/certs/`）
2. 证书 SANs 自动包含 `localhost`、`127.0.0.1`、本机 hostname 和所有网卡 IP
3. 已有证书不会被覆盖（安全幂等）

如需指定额外的域名或 IP：

```yaml
tls:
  enabled: true
  auto_generate: true
  hosts: "api.example.internal,10.0.1.100"
```

Node Manager 配置：将自动生成的 `/etc/agents-admin/certs/ca.pem` 复制到节点机器，然后：

```yaml
api_server_url: https://YOUR_API_SERVER_IP:8080
tls:
  enabled: true
  ca_file: /etc/agents-admin/certs/ca.pem
```

> 如需更细粒度的证书控制（自有 CA、Let's Encrypt 等），请参考下方手动配置章节。

---

## 手动配置（高级）

### 前置条件

- 已安装 `openssl` 或 `cfssl` 等证书工具
- 已部署 API Server 和 Node Manager（参考 [快速入门](./01-quick-start.md)）

### 1. 生成证书

### 方式一：使用 openssl 自签名（开发/测试）

```bash
# 创建证书目录
mkdir -p /etc/agents-admin/certs
cd /etc/agents-admin/certs

# 1) 生成 CA 私钥和证书
openssl genrsa -out ca-key.pem 4096
openssl req -new -x509 -days 3650 -key ca-key.pem -out ca.pem \
  -subj "/CN=Agents Admin CA"

# 2) 生成 API Server 私钥和证书签名请求
openssl genrsa -out server-key.pem 2048
openssl req -new -key server-key.pem -out server.csr \
  -subj "/CN=agents-admin-api"

# 3) 创建 SAN 扩展文件（支持多域名/IP）
cat > server-ext.cnf <<EOF
[v3_req]
subjectAltName = @alt_names
[alt_names]
DNS.1 = localhost
DNS.2 = api.example.com
IP.1 = 127.0.0.1
IP.2 = 192.168.1.100
EOF
# ⚠️ 请将 IP.2 替换为 API Server 的实际 IP 地址

# 4) 用 CA 签发服务端证书
openssl x509 -req -days 365 -in server.csr \
  -CA ca.pem -CAkey ca-key.pem -CAcreateserial \
  -out server.pem -extfile server-ext.cnf -extensions v3_req

# 5) 验证证书
openssl verify -CAfile ca.pem server.pem
# 应输出: server.pem: OK

# 6) 设置文件权限
chmod 600 *-key.pem
chmod 644 ca.pem server.pem
```

### 方式二：使用 cfssl（推荐用于生产环境）

```bash
# 安装 cfssl
go install github.com/cloudflare/cfssl/cmd/cfssl@latest
go install github.com/cloudflare/cfssl/cmd/cfssljson@latest

# CA 配置
cat > ca-csr.json <<EOF
{
  "CN": "Agents Admin CA",
  "key": { "algo": "rsa", "size": 4096 },
  "names": [{ "O": "Agents Admin" }]
}
EOF

cfssl gencert -initca ca-csr.json | cfssljson -bare ca

# Server 证书配置
cat > server-csr.json <<EOF
{
  "CN": "agents-admin-api",
  "hosts": [
    "localhost",
    "127.0.0.1",
    "api.example.com",
    "192.168.1.100"
  ],
  "key": { "algo": "rsa", "size": 2048 }
}
EOF

cat > ca-config.json <<EOF
{
  "signing": {
    "default": { "expiry": "8760h" },
    "profiles": {
      "server": { "expiry": "8760h", "usages": ["signing","key encipherment","server auth"] }
    }
  }
}
EOF

cfssl gencert -ca=ca.pem -ca-key=ca-key.pem \
  -config=ca-config.json -profile=server \
  server-csr.json | cfssljson -bare server
```

## 2. 配置 API Server（HTTPS 服务端）

编辑 `/etc/agents-admin/api-server.yaml`：

```yaml
server:
  port: "8080"

# 启用 TLS
tls:
  enabled: true
  cert_file: /etc/agents-admin/certs/server.pem
  key_file: /etc/agents-admin/certs/server-key.pem
  ca_file: /etc/agents-admin/certs/ca.pem   # 可选，用于未来的 mTLS
```

启动后日志应显示：

```
API Server listening on :8080 (TLS)
```

### 验证 HTTPS

```bash
# 使用 curl 验证（指定 CA 证书）
curl --cacert /etc/agents-admin/certs/ca.pem \
  https://localhost:8080/api/v1/nodes

# 或跳过证书验证（仅测试用）
curl -k https://localhost:8080/api/v1/nodes
```

## 3. 配置 Node Manager（TLS 客户端）

编辑 `/etc/agents-admin/nodemanager.yaml`：

```yaml
node:
  id: node-01
  api_server_url: https://192.168.1.100:8080   # ⚠️ 注意使用 https:// 前缀
  workspace_dir: /var/lib/agents-admin/workspaces
  labels:
    os: linux

# 启用 TLS（配置 CA 证书以验证 API Server）
tls:
  enabled: true
  ca_file: /etc/agents-admin/certs/ca.pem
```

### 工作原理

当 `tls.enabled=true` 且提供了 `ca_file` 时，Node Manager 会：

1. 读取 CA 证书文件
2. 创建带自定义 CA 根证书池的 `http.Client`
3. 所有到 API Server 的 HTTP 请求（心跳、任务拉取等）都通过此客户端发送
4. 客户端验证 API Server 证书是否由该 CA 签发

## 4. 远程节点部署时的 TLS 配置

使用「添加节点」功能远程部署时，如果 API Server 已启用 TLS：

1. **API Server URL** 填写 `https://` 前缀的地址
2. 部署流程会自动在目标节点上：
   - 创建 `/etc/agents-admin/certs/` 目录
   - 将 CA 证书上传到目标节点
   - 在生成的 `nodemanager.yaml` 中启用 TLS 配置

## 5. 证书文件位置约定

| 文件 | 路径 | 说明 |
|------|------|------|
| CA 证书 | `/etc/agents-admin/certs/ca.pem` | 根证书，API Server 和 Node Manager 共用 |
| CA 私钥 | `/etc/agents-admin/certs/ca-key.pem` | **仅保留在签发机器上**，Node Manager 不需要 |
| 服务端证书 | `/etc/agents-admin/certs/server.pem` | API Server 使用 |
| 服务端私钥 | `/etc/agents-admin/certs/server-key.pem` | API Server 使用，权限 `600` |

## 6. 安全最佳实践

- **私钥权限**：确保 `*-key.pem` 文件权限为 `600`，仅 root/服务用户可读
- **证书轮换**：建议每年更换服务端证书，CA 证书可设置较长有效期
- **SAN 配置**：服务端证书的 SAN 需包含所有 Node Manager 用于连接的 IP/域名
- **CA 私钥保护**：CA 私钥仅在签发证书时使用，签发完成后可离线保存
- **生产环境**：建议使用正规 CA 签发的证书，或使用 Let's Encrypt 等免费方案

## 7. 故障排查

### Node Manager 连接失败

```
x509: certificate signed by unknown authority
```
→ 检查 `nodemanager.yaml` 中 `tls.ca_file` 路径是否正确，CA 证书是否与签发服务端证书的 CA 一致。

### 证书域名不匹配

```
x509: certificate is valid for localhost, not api.example.com
```
→ 重新签发服务端证书，在 SAN 中添加正确的域名/IP。

### 证书过期

```
x509: certificate has expired or is not yet valid
```
→ 检查证书有效期：`openssl x509 -in server.pem -noout -dates`

## API 参考

TLS 配置不影响 API 路径，仅将 `http://` 替换为 `https://`。

| 组件 | 配置文件 | TLS 相关字段 |
|------|---------|-------------|
| API Server | `api-server.yaml` | `tls.enabled`, `tls.cert_file`, `tls.key_file`, `tls.ca_file` |
| Node Manager | `nodemanager.yaml` | `tls.enabled`, `tls.ca_file` |
