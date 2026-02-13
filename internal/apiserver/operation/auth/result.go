package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"agents-admin/internal/shared/model"
)

// HandleAuthSuccess 处理认证成功，创建或更新 Account
//
// 由父包 operation.handleActionSuccess 分发调用
func (h *Handler) HandleAuthSuccess(ctx context.Context, op *model.Operation, resultJSON json.RawMessage) {
	// 解析 Operation config
	var config model.OAuthConfig
	if err := json.Unmarshal(op.Config, &config); err != nil {
		log.Printf("[auth] Failed to parse operation config: %v", err)
		return
	}

	// 解析 Action result
	var result model.AuthActionResult
	if resultJSON != nil {
		if err := json.Unmarshal(resultJSON, &result); err != nil {
			log.Printf("[auth] Failed to parse action result: %v", err)
		}
	}

	// 创建 Account
	now := time.Now()
	accountID := fmt.Sprintf("%s_%s", config.AgentType, sanitizeName(config.Name))

	account := &model.Account{
		ID:          accountID,
		Name:        config.Name,
		AgentTypeID: config.AgentType,
		Status:      model.AccountStatusAuthenticated,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if result.VolumeName != "" {
		account.VolumeName = &result.VolumeName
	}

	// 检查账号是否已存在
	existing, err := h.store.GetAccount(ctx, accountID)
	if err != nil {
		log.Printf("[auth] GetAccount error: %v", err)
		return
	}

	if existing != nil {
		// 账号已存在，更新状态
		if err := h.store.UpdateAccountStatus(ctx, accountID, model.AccountStatusAuthenticated); err != nil {
			log.Printf("[auth] UpdateAccountStatus error: %v", err)
		}
		if result.VolumeName != "" {
			if err := h.store.UpdateAccountVolume(ctx, accountID, result.VolumeName); err != nil {
				log.Printf("[auth] UpdateAccountVolume error: %v", err)
			}
		}
		log.Printf("[auth] Account %s updated to authenticated", accountID)
	} else {
		// 创建新账号
		if err := h.store.CreateAccount(ctx, account); err != nil {
			log.Printf("[auth] CreateAccount error: %v", err)
			return
		}
		log.Printf("[auth] Account %s created (authenticated)", accountID)
	}
}
