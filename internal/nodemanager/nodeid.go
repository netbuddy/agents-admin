// Package nodemanager 节点管理器核心逻辑
package nodemanager

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"net"
	"os"
	"strings"
)

// GenerateNodeID 生成确定性 Node ID
//
// 基于 /etc/machine-id（systemd 标准）的 HMAC-SHA256 哈希，确保：
//   - 同一台机器始终生成相同的 Node ID
//   - 不同机器生成不同的 Node ID
//   - 不直接暴露 machine-id（遵循 machine-id(5) 安全建议）
//
// 回退策略：
//  1. /etc/machine-id（首选，systemd 标准）
//  2. /var/lib/dbus/machine-id（D-Bus 兼容）
//  3. hostname + 第一个非回环 MAC 地址
//  4. 随机 UUID v4（最终回退）
func GenerateNodeID() string {
	const appKey = "agents-admin-node-id-v1"

	machineID := ""
	for _, path := range []string{"/etc/machine-id", "/var/lib/dbus/machine-id"} {
		if data, err := os.ReadFile(path); err == nil {
			machineID = strings.TrimSpace(string(data))
			if machineID != "" {
				break
			}
		}
	}

	if machineID == "" {
		hostname, _ := os.Hostname()
		mac := getFirstMACAddress()
		if hostname != "" || mac != "" {
			machineID = hostname + ":" + mac
		}
	}

	if machineID == "" {
		return generateRandomUUID()
	}

	h := hmac.New(sha256.New, []byte(appKey))
	h.Write([]byte(machineID))
	sum := h.Sum(nil)

	sum[6] = (sum[6] & 0x0f) | 0x50
	sum[8] = (sum[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", sum[0:4], sum[4:6], sum[6:8], sum[8:10], sum[10:16])
}

func getFirstMACAddress() string {
	ifaces, err := net.Interfaces()
	if err != nil {
		return ""
	}
	for _, iface := range ifaces {
		if iface.Flags&net.FlagLoopback != 0 || len(iface.HardwareAddr) == 0 {
			continue
		}
		return iface.HardwareAddr.String()
	}
	return ""
}

func generateRandomUUID() string {
	b := make([]byte, 16)
	rand.Read(b)
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}
