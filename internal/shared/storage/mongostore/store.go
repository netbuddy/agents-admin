// Package mongostore 实现基于 MongoDB 的 PersistentStore
//
// 使用 mongo-go-driver v2，通过 bson tag 实现 model 结构体的序列化/反序列化。
// 所有 Collection 名称和索引在 ensureIndexes 中统一管理。
package mongostore

import (
	"context"
	"fmt"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// Collection 名称常量
const (
	ColTasks             = "tasks"
	ColTaskTemplates     = "task_templates"
	ColRuns              = "runs"
	ColEvents            = "events"
	ColNodes             = "nodes"
	ColNodeProvisions    = "node_provisions"
	ColAccounts          = "accounts"
	ColAuthTasks         = "auth_tasks"
	ColOperations        = "operations"
	ColActions           = "actions"
	ColProxies           = "proxies"
	ColAgents            = "agents"
	ColTerminalSessions  = "terminal_sessions"
	ColApprovalRequests  = "approval_requests"
	ColApprovalDecisions = "approval_decisions"
	ColFeedbacks         = "feedbacks"
	ColInterventions     = "interventions"
	ColConfirmations     = "confirmations"
	ColAgentTemplates    = "agent_templates"
	ColSkills            = "skills"
	ColMCPServers        = "mcp_servers"
	ColSecurityPolicies  = "security_policies"
	ColUsers             = "users"
	ColPromptTemplates   = "prompt_templates"
	ColArtifacts         = "artifacts"
	ColMemories          = "memories"
)

// Store 实现 storage.PersistentStore 接口的 MongoDB 驱动
type Store struct {
	client *mongo.Client
	db     *mongo.Database
}

// NewStore 创建 MongoDB 存储实例
//
// uri: MongoDB 连接 URI，如 "mongodb://localhost:27017"
// dbName: 数据库名称，如 "agents_admin"
func NewStore(uri, dbName string) (*Store, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(options.Client().ApplyURI(uri))
	if err != nil {
		return nil, fmt.Errorf("mongostore: connect failed: %w", err)
	}

	// 验证连接
	if err := client.Ping(ctx, nil); err != nil {
		return nil, fmt.Errorf("mongostore: ping failed: %w", err)
	}

	db := client.Database(dbName)
	s := &Store{client: client, db: db}

	// 创建索引
	if err := s.ensureIndexes(ctx); err != nil {
		log.Printf("WARNING: mongostore: ensure indexes failed: %v", err)
	}

	return s, nil
}

// Close 关闭 MongoDB 连接
func (s *Store) Close() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return s.client.Disconnect(ctx)
}

// col 获取指定 Collection
func (s *Store) col(name string) *mongo.Collection {
	return s.db.Collection(name)
}

// ensureIndexes 创建所有必要的索引
func (s *Store) ensureIndexes(ctx context.Context) error {
	type idx struct {
		col    string
		keys   bson.D
		unique bool
	}

	indexes := []idx{
		// tasks
		{ColTasks, bson.D{{Key: "status", Value: 1}}, false},
		{ColTasks, bson.D{{Key: "parent_id", Value: 1}}, false},
		{ColTasks, bson.D{{Key: "template_id", Value: 1}}, false},
		{ColTasks, bson.D{{Key: "created_at", Value: -1}}, false},

		// runs
		{ColRuns, bson.D{{Key: "task_id", Value: 1}}, false},
		{ColRuns, bson.D{{Key: "node_id", Value: 1}}, false},
		{ColRuns, bson.D{{Key: "status", Value: 1}}, false},
		{ColRuns, bson.D{{Key: "created_at", Value: -1}}, false},

		// events
		{ColEvents, bson.D{{Key: "run_id", Value: 1}, {Key: "seq", Value: 1}}, false},

		// nodes
		{ColNodes, bson.D{{Key: "status", Value: 1}}, false},

		// accounts
		{ColAccounts, bson.D{{Key: "node_id", Value: 1}}, false},

		// auth_tasks
		{ColAuthTasks, bson.D{{Key: "account_id", Value: 1}}, false},
		{ColAuthTasks, bson.D{{Key: "node_id", Value: 1}}, false},
		{ColAuthTasks, bson.D{{Key: "status", Value: 1}}, false},

		// operations
		{ColOperations, bson.D{{Key: "type", Value: 1}}, false},
		{ColOperations, bson.D{{Key: "status", Value: 1}}, false},

		// actions
		{ColActions, bson.D{{Key: "operation_id", Value: 1}}, false},
		{ColActions, bson.D{{Key: "status", Value: 1}}, false},

		// proxies
		{ColProxies, bson.D{{Key: "is_default", Value: 1}}, false},

		// agents
		{ColAgents, bson.D{{Key: "node_id", Value: 1}}, false},

		// terminal_sessions
		{ColTerminalSessions, bson.D{{Key: "node_id", Value: 1}}, false},
		{ColTerminalSessions, bson.D{{Key: "status", Value: 1}}, false},

		// approval_requests
		{ColApprovalRequests, bson.D{{Key: "run_id", Value: 1}}, false},

		// feedbacks
		{ColFeedbacks, bson.D{{Key: "run_id", Value: 1}}, false},

		// interventions
		{ColInterventions, bson.D{{Key: "run_id", Value: 1}}, false},

		// confirmations
		{ColConfirmations, bson.D{{Key: "run_id", Value: 1}}, false},

		// users
		{ColUsers, bson.D{{Key: "email", Value: 1}}, true},
	}

	for _, i := range indexes {
		model := mongo.IndexModel{Keys: i.keys}
		if i.unique {
			model.Options = options.Index().SetUnique(true)
		}
		if _, err := s.col(i.col).Indexes().CreateOne(ctx, model); err != nil {
			return fmt.Errorf("create index on %s: %w", i.col, err)
		}
	}

	return nil
}
