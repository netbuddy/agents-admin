package model

import "time"

// NodeProvisionStatus 节点部署状态
type NodeProvisionStatus string

const (
	NodeProvisionStatusPending     NodeProvisionStatus = "pending"
	NodeProvisionStatusConnecting  NodeProvisionStatus = "connecting"
	NodeProvisionStatusDownloading NodeProvisionStatus = "downloading"
	NodeProvisionStatusInstalling  NodeProvisionStatus = "installing"
	NodeProvisionStatusConfiguring NodeProvisionStatus = "configuring"
	NodeProvisionStatusCompleted   NodeProvisionStatus = "completed"
	NodeProvisionStatusFailed      NodeProvisionStatus = "failed"
)

// NodeProvision 节点远程部署记录
type NodeProvision struct {
	ID            string              `json:"id" db:"id"`
	NodeID        string              `json:"node_id" db:"node_id"`
	Host          string              `json:"host" db:"host"`
	Port          int                 `json:"port" db:"port"`
	SSHUser       string              `json:"ssh_user" db:"ssh_user"`
	AuthMethod    string              `json:"auth_method" db:"auth_method"`
	Status        NodeProvisionStatus `json:"status" db:"status"`
	ErrorMessage  string              `json:"error_message,omitempty" db:"error_message"`
	Version       string              `json:"version" db:"version"`
	GithubRepo    string              `json:"github_repo" db:"github_repo"`
	APIServerURL  string              `json:"api_server_url" db:"api_server_url"`
	CreatedAt     time.Time           `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time           `json:"updated_at" db:"updated_at"`
}
