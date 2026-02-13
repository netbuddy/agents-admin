// Package model 定义核心数据模型
package model

import (
"fmt"
"net/url"
"time"
)

// ProxyType 代理类型
type ProxyType string

const (
ProxyTypeHTTP   ProxyType = "http"
ProxyTypeHTTPS  ProxyType = "https"
ProxyTypeSOCKS5 ProxyType = "socks5"
)

// ProxyStatus 代理状态
type ProxyStatus string

const (
ProxyStatusActive   ProxyStatus = "active"
ProxyStatusInactive ProxyStatus = "inactive"
)

// Proxy 代理配置
type Proxy struct {
ID        string      `json:"id" bson:"_id" db:"id"`
Name      string      `json:"name" bson:"name" db:"name"`
Type      ProxyType   `json:"type" bson:"type" db:"type"`
Host      string      `json:"host" bson:"host" db:"host"`
Port      int         `json:"port" bson:"port" db:"port"`
Username  *string     `json:"username,omitempty" bson:"username,omitempty" db:"username"`
Password  *string     `json:"-" bson:"password" db:"password"`
NoProxy   *string     `json:"no_proxy,omitempty" bson:"no_proxy,omitempty" db:"no_proxy"`
IsDefault bool        `json:"is_default" bson:"is_default" db:"is_default"`
Status    ProxyStatus `json:"status" bson:"status" db:"status"`
CreatedAt time.Time   `json:"created_at" bson:"created_at" db:"created_at"`
UpdatedAt time.Time   `json:"updated_at" bson:"updated_at" db:"updated_at"`
}

// GetURL 获取代理URL
func (p *Proxy) GetURL() string {
if p == nil {
return ""
}
var scheme string
switch p.Type {
case ProxyTypeSOCKS5:
scheme = "socks5"
default:
scheme = "http"
}
u := &url.URL{
Scheme: scheme,
Host:   fmt.Sprintf("%s:%d", p.Host, p.Port),
}
if p.Username != nil && *p.Username != "" {
if p.Password != nil && *p.Password != "" {
u.User = url.UserPassword(*p.Username, *p.Password)
} else {
u.User = url.User(*p.Username)
}
}
return u.String()
}

// ToEnvVars 生成代理环境变量
func (p *Proxy) ToEnvVars() []string {
if p == nil {
return nil
}
proxyURL := p.GetURL()
if proxyURL == "" {
return nil
}
envs := []string{
fmt.Sprintf("HTTP_PROXY=%s", proxyURL),
fmt.Sprintf("HTTPS_PROXY=%s", proxyURL),
fmt.Sprintf("http_proxy=%s", proxyURL),
fmt.Sprintf("https_proxy=%s", proxyURL),
}
if p.NoProxy != nil && *p.NoProxy != "" {
envs = append(envs,
fmt.Sprintf("NO_PROXY=%s", *p.NoProxy),
fmt.Sprintf("no_proxy=%s", *p.NoProxy),
)
}
return envs
}

// HasAuth 是否有认证信息
func (p *Proxy) HasAuth() bool {
return p != nil && p.Username != nil && *p.Username != ""
}

// Validate 验证代理配置
func (p *Proxy) Validate() error {
if p.Name == "" {
return fmt.Errorf("proxy name is required")
}
if p.Host == "" {
return fmt.Errorf("proxy host is required")
}
if p.Port <= 0 || p.Port > 65535 {
return fmt.Errorf("proxy port must be between 1 and 65535")
}
if p.Type == "" {
p.Type = ProxyTypeHTTP
}
if p.Status == "" {
p.Status = ProxyStatusActive
}
return nil
}
