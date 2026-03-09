package database

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"
)

// ConfigPolicy 配置策略
type ConfigPolicy struct {
	ID                int                    `json:"id"`
	Name              string                 `json:"name"`
	Token             string                 `json:"token"`
	Description       string                 `json:"description"`
	SubscriptionNames []string               `json:"subscription_names"`
	TemplateName      string                 `json:"template_name"`
	Target            string                 `json:"target"`
	NodeFilters       map[string]interface{} `json:"node_filters"`
	Enabled           bool                   `json:"enabled"`
	Tags              []string               `json:"tags"`
	CreatedAt         time.Time              `json:"created_at"`
	UpdatedAt         time.Time              `json:"updated_at"`
}

// ConfigPolicyRepo 配置策略仓储
type ConfigPolicyRepo struct {
	db *DB
}

// NewConfigPolicyRepo 创建配置策略仓储
func NewConfigPolicyRepo(db *DB) *ConfigPolicyRepo {
	return &ConfigPolicyRepo{db: db}
}

// generateToken 生成随机访问 token
func generateToken() (string, error) {
	b := make([]byte, 24)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// Create 创建配置策略，自动生成访问 token
func (r *ConfigPolicyRepo) Create(ctx context.Context, policy *ConfigPolicy) error {
	token, err := generateToken()
	if err != nil {
		return fmt.Errorf("生成 token 失败: %w", err)
	}
	policy.Token = token

	nodeFiltersJSON, err := json.Marshal(policy.NodeFilters)
	if err != nil {
		return fmt.Errorf("序列化节点过滤条件失败: %w", err)
	}

	query := `
		INSERT INTO config_policies (name, token, description, subscription_names, template_name, target, node_filters, enabled, tags)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, created_at, updated_at
	`

	err = r.db.Pool.QueryRow(ctx, query,
		policy.Name,
		policy.Token,
		policy.Description,
		policy.SubscriptionNames,
		policy.TemplateName,
		policy.Target,
		nodeFiltersJSON,
		policy.Enabled,
		policy.Tags,
	).Scan(&policy.ID, &policy.CreatedAt, &policy.UpdatedAt)

	if err != nil {
		return fmt.Errorf("创建配置策略失败: %w", err)
	}

	return nil
}

// scanPolicy 扫描一行策略数据
func scanPolicy(scan func(...any) error) (*ConfigPolicy, error) {
	policy := &ConfigPolicy{}
	var nodeFiltersJSON []byte

	err := scan(
		&policy.ID,
		&policy.Name,
		&policy.Token,
		&policy.Description,
		&policy.SubscriptionNames,
		&policy.TemplateName,
		&policy.Target,
		&nodeFiltersJSON,
		&policy.Enabled,
		&policy.Tags,
		&policy.CreatedAt,
		&policy.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	if len(nodeFiltersJSON) > 0 {
		if err := json.Unmarshal(nodeFiltersJSON, &policy.NodeFilters); err != nil {
			return nil, fmt.Errorf("解析节点过滤条件失败: %w", err)
		}
	}

	return policy, nil
}

const selectPolicyFields = `
	SELECT id, name, token, description, subscription_names, template_name, target, node_filters, enabled, tags, created_at, updated_at
	FROM config_policies
`

// GetByName 根据名称获取配置策略
func (r *ConfigPolicyRepo) GetByName(ctx context.Context, name string) (*ConfigPolicy, error) {
	query := selectPolicyFields + `WHERE name = $1`
	policy, err := scanPolicy(r.db.Pool.QueryRow(ctx, query, name).Scan)
	if err != nil {
		return nil, fmt.Errorf("获取配置策略失败: %w", err)
	}
	return policy, nil
}

// GetByToken 根据 token 获取配置策略
func (r *ConfigPolicyRepo) GetByToken(ctx context.Context, token string) (*ConfigPolicy, error) {
	query := selectPolicyFields + `WHERE token = $1`
	policy, err := scanPolicy(r.db.Pool.QueryRow(ctx, query, token).Scan)
	if err != nil {
		return nil, fmt.Errorf("配置策略不存在或 token 无效: %w", err)
	}
	return policy, nil
}

// List 获取所有配置策略
func (r *ConfigPolicyRepo) List(ctx context.Context) ([]*ConfigPolicy, error) {
	query := selectPolicyFields + `ORDER BY created_at DESC`

	rows, err := r.db.Pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("查询配置策略列表失败: %w", err)
	}
	defer rows.Close()

	policies := make([]*ConfigPolicy, 0)
	for rows.Next() {
		policy, err := scanPolicy(rows.Scan)
		if err != nil {
			return nil, fmt.Errorf("扫描配置策略行失败: %w", err)
		}
		policies = append(policies, policy)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("遍历配置策略结果失败: %w", err)
	}

	return policies, nil
}

// Update 更新配置策略
func (r *ConfigPolicyRepo) Update(ctx context.Context, policy *ConfigPolicy) error {
	nodeFiltersJSON, err := json.Marshal(policy.NodeFilters)
	if err != nil {
		return fmt.Errorf("序列化节点过滤条件失败: %w", err)
	}

	query := `
		UPDATE config_policies
		SET description = $2, subscription_names = $3, template_name = $4, target = $5, node_filters = $6, enabled = $7, tags = $8
		WHERE name = $1
		RETURNING updated_at
	`

	err = r.db.Pool.QueryRow(ctx, query,
		policy.Name,
		policy.Description,
		policy.SubscriptionNames,
		policy.TemplateName,
		policy.Target,
		nodeFiltersJSON,
		policy.Enabled,
		policy.Tags,
	).Scan(&policy.UpdatedAt)

	if err != nil {
		return fmt.Errorf("更新配置策略失败: %w", err)
	}

	return nil
}

// Delete 删除配置策略
func (r *ConfigPolicyRepo) Delete(ctx context.Context, name string) error {
	query := `DELETE FROM config_policies WHERE name = $1`

	result, err := r.db.Pool.Exec(ctx, query, name)
	if err != nil {
		return fmt.Errorf("删除配置策略失败: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("配置策略不存在: %s", name)
	}

	return nil
}

// GetEnabled 获取所有启用的配置策略
func (r *ConfigPolicyRepo) GetEnabled(ctx context.Context) ([]*ConfigPolicy, error) {
	query := selectPolicyFields + `WHERE enabled = true ORDER BY created_at DESC`

	rows, err := r.db.Pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("查询启用的配置策略列表失败: %w", err)
	}
	defer rows.Close()

	policies := make([]*ConfigPolicy, 0)
	for rows.Next() {
		policy, err := scanPolicy(rows.Scan)
		if err != nil {
			return nil, fmt.Errorf("扫描配置策略行失败: %w", err)
		}
		policies = append(policies, policy)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("遍历配置策略结果失败: %w", err)
	}

	return policies, nil
}
