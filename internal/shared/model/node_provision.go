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
	ID           string              `json:"id" bson:"_id" db:"id"`
	NodeID       string              `json:"node_id" bson:"node_id" db:"node_id"`
	DisplayName  string              `json:"display_name,omitempty" bson:"display_name,omitempty" db:"display_name"`
	Host         string              `json:"host" bson:"host" db:"host"`
	Port         int                 `json:"port" bson:"port" db:"port"`
	SSHUser      string              `json:"ssh_user" bson:"ssh_user" db:"ssh_user"`
	AuthMethod   string              `json:"auth_method" bson:"auth_method" db:"auth_method"`
	Status       NodeProvisionStatus `json:"status" bson:"status" db:"status"`
	ErrorMessage string              `json:"error_message,omitempty" bson:"error_message,omitempty" db:"error_message"`
	Version      string              `json:"version" bson:"version" db:"version"`
	GithubRepo   string              `json:"github_repo" bson:"github_repo" db:"github_repo"`
	APIServerURL string              `json:"api_server_url" bson:"api_server_url" db:"api_server_url"`
	CreatedAt    time.Time           `json:"created_at" bson:"created_at" db:"created_at"`
	UpdatedAt    time.Time           `json:"updated_at" bson:"updated_at" db:"updated_at"`
}
