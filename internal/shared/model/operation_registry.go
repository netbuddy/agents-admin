// Package model 定义核心数据模型
//
// operation_registry.go 包含操作类型注册表：
//   - OperationMeta：操作类型元信息（Config/Result 类型声明、有效阶段、验证）
//   - 全局注册表：集中管理所有 OperationType 的元数据
//
// 设计参考 Google AIP-151 的 operation_info 注解模式：
//   每个返回 Operation 的 RPC 都必须声明 response_type 和 metadata_type。
//   本注册表在 Go 层面实现同等的类型声明与校验。
package model

import (
	"encoding/json"
	"fmt"
)

// ============================================================================
// OperationMeta - 操作类型元信息
// ============================================================================

// OperationMeta 声明一种 OperationType 的完整元信息
//
// 类比 Google AIP-151 中的 google.longrunning.OperationInfo：
//   - ConfigValidator：校验 Config（对应 request 校验）
//   - ValidPhases：该类型下 Action 允许经历的语义阶段序列
//   - Description：人类可读描述
//   - Sync：是否同步完成（如 api_key 无需异步 Action）
type OperationMeta struct {
	Type            OperationType    // 操作类型
	Description     string           // 人类可读描述
	Sync            bool             // 是否同步完成（无需 Action 轮询）
	ValidPhases     []ActionPhase    // 该类型允许的语义阶段
	ConfigValidator func(json.RawMessage) error // Config 校验函数（nil = 不校验）
}

// ============================================================================
// 全局注册表
// ============================================================================

// operationRegistry 操作类型注册表（包级私有，通过函数访问）
var operationRegistry = map[OperationType]*OperationMeta{
	// --- 认证操作 ---
	OperationTypeOAuth: {
		Type:        OperationTypeOAuth,
		Description: "OAuth 浏览器授权认证",
		ValidPhases: []ActionPhase{
			PhaseInitializing, PhaseLaunchingContainer, PhaseAuthenticating,
			PhaseWaitingOAuth, PhaseVerifyingCredentials, PhaseExtractingToken,
			PhaseSavingCredentials, PhaseFinalizing,
		},
		ConfigValidator: validateAuthConfig,
	},
	OperationTypeDeviceCode: {
		Type:        OperationTypeDeviceCode,
		Description: "Device Code 授权认证",
		ValidPhases: []ActionPhase{
			PhaseInitializing, PhaseLaunchingContainer, PhaseRequestingDeviceCode,
			PhaseWaitingDeviceCode, PhasePollingToken, PhaseExtractingToken,
			PhaseSavingCredentials, PhaseFinalizing,
		},
		ConfigValidator: validateAuthConfig,
	},
	OperationTypeAPIKey: {
		Type:        OperationTypeAPIKey,
		Description: "API Key 直接验证",
		Sync:        true,
		ValidPhases: []ActionPhase{
			PhaseInitializing, PhaseValidatingAPIKey, PhaseSavingCredentials, PhaseFinalizing,
		},
		ConfigValidator: validateAPIKeyConfig,
	},

	// --- 运行时操作 ---
	OperationTypeRuntimeCreate: {
		Type:        OperationTypeRuntimeCreate,
		Description: "创建运行时环境",
		ValidPhases: []ActionPhase{
			PhaseInitializing, PhasePullingImage, PhaseCreatingContainer,
			PhaseConfiguringRuntime, PhaseHealthChecking, PhaseFinalizing,
		},
		ConfigValidator: validateRuntimeConfig,
	},
	OperationTypeRuntimeStart: {
		Type:        OperationTypeRuntimeStart,
		Description: "启动运行时",
		ValidPhases: []ActionPhase{
			PhaseInitializing, PhaseStartingRuntime, PhaseHealthChecking, PhaseFinalizing,
		},
		ConfigValidator: validateRuntimeConfig,
	},
	OperationTypeRuntimeStop: {
		Type:        OperationTypeRuntimeStop,
		Description: "停止运行时",
		ValidPhases: []ActionPhase{
			PhaseInitializing, PhaseStoppingRuntime, PhaseFinalizing,
		},
		ConfigValidator: validateRuntimeConfig,
	},
	OperationTypeRuntimeDestroy: {
		Type:        OperationTypeRuntimeDestroy,
		Description: "销毁运行时",
		ValidPhases: []ActionPhase{
			PhaseInitializing, PhaseStoppingRuntime, PhaseRemovingContainer,
			PhaseCleaningVolumes, PhaseFinalizing,
		},
		ConfigValidator: validateRuntimeConfig,
	},
}

// ============================================================================
// 注册表访问函数
// ============================================================================

// GetOperationMeta 获取操作类型的元信息
func GetOperationMeta(opType OperationType) (*OperationMeta, bool) {
	meta, ok := operationRegistry[opType]
	return meta, ok
}

// ListOperationMetas 列出所有已注册的操作类型元信息
func ListOperationMetas() []*OperationMeta {
	metas := make([]*OperationMeta, 0, len(operationRegistry))
	for _, m := range operationRegistry {
		metas = append(metas, m)
	}
	return metas
}

// ValidateOperationConfig 根据类型校验 Config
func ValidateOperationConfig(opType OperationType, config json.RawMessage) error {
	meta, ok := operationRegistry[opType]
	if !ok {
		return fmt.Errorf("unknown operation type: %s", opType)
	}
	if meta.ConfigValidator == nil {
		return nil
	}
	return meta.ConfigValidator(config)
}

// IsValidPhase 检查指定阶段对该操作类型是否有效
func IsValidPhase(opType OperationType, phase ActionPhase) bool {
	meta, ok := operationRegistry[opType]
	if !ok {
		return false
	}
	for _, p := range meta.ValidPhases {
		if p == phase {
			return true
		}
	}
	return false
}

// ============================================================================
// Config 校验函数
// ============================================================================

func validateAuthConfig(raw json.RawMessage) error {
	var cfg OAuthConfig
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return fmt.Errorf("invalid auth config: %w", err)
	}
	if cfg.Name == "" {
		return fmt.Errorf("config.name is required")
	}
	if cfg.AgentType == "" {
		return fmt.Errorf("config.agent_type is required")
	}
	return nil
}

func validateAPIKeyConfig(raw json.RawMessage) error {
	var cfg APIKeyConfig
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return fmt.Errorf("invalid api_key config: %w", err)
	}
	if cfg.Name == "" {
		return fmt.Errorf("config.name is required")
	}
	if cfg.AgentType == "" {
		return fmt.Errorf("config.agent_type is required")
	}
	if cfg.APIKey == "" {
		return fmt.Errorf("config.api_key is required")
	}
	return nil
}

func validateRuntimeConfig(raw json.RawMessage) error {
	var cfg RuntimeConfig
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return fmt.Errorf("invalid runtime config: %w", err)
	}
	return nil
}
