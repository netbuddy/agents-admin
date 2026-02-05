// Package model 定义核心数据模型
package model

import (
	"fmt"
	"net/url"
)

// EnvConfig 运行环境配置
// 存储在 nodes.metadata 的 JSONB 字段中
type EnvConfig struct {
	// 代理配置
	Proxy *ProxyConfig `json:"proxy,omitempty"`

	// 自定义环境变量
	EnvVars map[string]string `json:"env_vars,omitempty"`

	// 资源限制（预留扩展）
	Resources *ResourceConfig `json:"resources,omitempty"`
}

// ProxyConfig 代理配置
type ProxyConfig struct {
	Enabled  bool   `json:"enabled"`            // 是否启用代理
	Type     string `json:"type"`               // http, https, socks5
	Host     string `json:"host"`               // 代理主机
	Port     int    `json:"port"`               // 代理端口
	Username string `json:"username,omitempty"` // 用户名（可选）
	Password string `json:"password,omitempty"` // 密码（可选）
	NoProxy  string `json:"no_proxy,omitempty"` // 排除列表，逗号分隔
}

// ResourceConfig 资源限制（预留）
type ResourceConfig struct {
	MemoryLimit string `json:"memory_limit,omitempty"` // 如 "2G"
	CPULimit    string `json:"cpu_limit,omitempty"`    // 如 "1.0"
}

// GetURL 获取代理URL
func (p *ProxyConfig) GetURL() string {
	if p == nil || !p.Enabled || p.Host == "" {
		return ""
	}

	var scheme string
	switch p.Type {
	case "socks5":
		scheme = "socks5"
	case "https":
		scheme = "https"
	default:
		scheme = "http"
	}

	u := &url.URL{
		Scheme: scheme,
		Host:   fmt.Sprintf("%s:%d", p.Host, p.Port),
	}

	if p.Username != "" {
		if p.Password != "" {
			u.User = url.UserPassword(p.Username, p.Password)
		} else {
			u.User = url.User(p.Username)
		}
	}

	return u.String()
}

// ToContainerEnv 生成容器环境变量
func (c *EnvConfig) ToContainerEnv() []string {
	if c == nil {
		return nil
	}

	var envs []string

	// 代理环境变量
	if c.Proxy != nil && c.Proxy.Enabled {
		proxyURL := c.Proxy.GetURL()
		if proxyURL != "" {
			envs = append(envs,
				"HTTP_PROXY="+proxyURL,
				"HTTPS_PROXY="+proxyURL,
				"http_proxy="+proxyURL,
				"https_proxy="+proxyURL,
			)
			if c.Proxy.NoProxy != "" {
				envs = append(envs,
					"NO_PROXY="+c.Proxy.NoProxy,
					"no_proxy="+c.Proxy.NoProxy,
				)
			}
		}
	}

	// 自定义环境变量
	for k, v := range c.EnvVars {
		envs = append(envs, k+"="+v)
	}

	return envs
}

// ResolveEnvConfig 解析最终环境配置（任务配置 > 节点配置）
func ResolveEnvConfig(nodeConfig, taskConfig *EnvConfig) *EnvConfig {
	if taskConfig != nil {
		return taskConfig
	}
	if nodeConfig != nil {
		return nodeConfig
	}
	return nil
}

// Validate 验证配置
func (c *EnvConfig) Validate() error {
	if c == nil {
		return nil
	}
	if c.Proxy != nil && c.Proxy.Enabled {
		if c.Proxy.Host == "" {
			return fmt.Errorf("proxy host is required when proxy is enabled")
		}
		if c.Proxy.Port <= 0 || c.Proxy.Port > 65535 {
			return fmt.Errorf("proxy port must be between 1 and 65535")
		}
	}
	return nil
}
