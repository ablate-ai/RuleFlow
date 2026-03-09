package database

import (
	"context"
	"encoding/json"
	"fmt"
)

// ConfigPolicy 配置策略
type ConfigPolicy struct {
	ID                int                    `json:"id"`
	Name              string                 `json:"name"`
	Description       string                 `json:"description"`
	SubscriptionNames []string               `json:"subscription_names"`
	TemplateName      string                 `json:"template_name"`
	Target            string                 `json:"target"`
	NodeFilters       map[string]interface{} `json:"node_filters"`
	Enabled           bool                   `json:"enabled"`
	Tags              []string               `json:"tags"`
	CreatedAt         string                 `json:"created_at"`
	UpdatedAt         string                 `json:"updated_at"`
}

// ConfigPolicyRepo 配置策略仓储
type ConfigPolicyRepo struct {
	db *DB
}

// NewConfigPolicyRepo 创建配置策略仓储
func NewConfigPolicyRepo(db *DB) *ConfigPolicyRepo {
	return &ConfigPolicyRepo{db: db}
}

// Create 创建配置策略
func (r *ConfigPolicyRepo) Create(ctx context.Context, policy *ConfigPolicy) error {
	nodeFiltersJSON, err := json.Marshal(policy.NodeFilters)
	if err != nil {
		return fmt.Errorf("序列化节点过滤条件失败: %w", err)
	}

	query := `
		INSERT INTO config_policies (name, description, subscription_names, template_name, target, node_filters, enabled, tags)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, created_at, updated_at
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
	).Scan(&policy.ID, &policy.CreatedAt, &policy.UpdatedAt)

	if err != nil {
		return fmt.Errorf("创建配置策略失败: %w", err)
	}

	return nil
}

// GetByName 根据名称获取配置策略
func (r *ConfigPolicyRepo) GetByName(ctx context.Context, name string) (*ConfigPolicy, error) {
	query := `
		SELECT id, name, description, subscription_names, template_name, target, node_filters, enabled, tags, created_at, updated_at
		FROM config_policies
		WHERE name = $1
	`

	policy := &ConfigPolicy{}
	var nodeFiltersJSON []byte

	err := r.db.Pool.QueryRow(ctx, query, name).Scan(
		&policy.ID,
		&policy.Name,
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
		return nil, fmt.Errorf("获取配置策略失败: %w", err)
	}

	if len(nodeFiltersJSON) > 0 {
		if err := json.Unmarshal(nodeFiltersJSON, &policy.NodeFilters); err != nil {
			return nil, fmt.Errorf("解析节点过滤条件失败: %w", err)
		}
	}

	return policy, nil
}

// List 获取所有配置策略
func (r *ConfigPolicyRepo) List(ctx context.Context) ([]*ConfigPolicy, error) {
	query := `
		SELECT id, name, description, subscription_names, template_name, target, node_filters, enabled, tags, created_at, updated_at
		FROM config_policies
		ORDER BY created_at DESC
	`

	rows, err := r.db.Pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("查询配置策略列表失败: %w", err)
	}
	defer rows.Close()

	policies := make([]*ConfigPolicy, 0)
	for rows.Next() {
		policy := &ConfigPolicy{}
		var nodeFiltersJSON []byte

		err := rows.Scan(
			&policy.ID,
			&policy.Name,
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
			return nil, fmt.Errorf("扫描配置策略行失败: %w", err)
		}

		if len(nodeFiltersJSON) > 0 {
			if err := json.Unmarshal(nodeFiltersJSON, &policy.NodeFilters); err != nil {
				return nil, fmt.Errorf("解析节点过滤条件失败: %w", err)
			}
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
	query := `
		SELECT id, name, description, subscription_names, template_name, target, node_filters, enabled, tags, created_at, updated_at
		FROM config_policies
		WHERE enabled = true
		ORDER BY created_at DESC
	`

	rows, err := r.db.Pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("查询启用的配置策略列表失败: %w", err)
	}
	defer rows.Close()

	policies := make([]*ConfigPolicy, 0)
	for rows.Next() {
		policy := &ConfigPolicy{}
		var nodeFiltersJSON []byte

		err := rows.Scan(
			&policy.ID,
			&policy.Name,
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
			return nil, fmt.Errorf("扫描配置策略行失败: %w", err)
		}

		if len(nodeFiltersJSON) > 0 {
			if err := json.Unmarshal(nodeFiltersJSON, &policy.NodeFilters); err != nil {
				return nil, fmt.Errorf("解析节点过滤条件失败: %w", err)
			}
		}

		policies = append(policies, policy)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("遍历配置策略结果失败: %w", err)
	}

	return policies, nil
}
