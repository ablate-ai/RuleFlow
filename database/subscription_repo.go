package database

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

// Subscription 订阅模型
type Subscription struct {
	ID              int        `json:"id"`
	Name            string     `json:"name"`
	URL             *string    `json:"url"`
	Target          string     `json:"target"`
	Enabled         bool       `json:"enabled"`
	AutoRefresh     bool       `json:"auto_refresh"`
	RefreshInterval int        `json:"refresh_interval"`
	Description     string     `json:"description"`
	Tags            []string   `json:"tags"`
	LastFetchedAt   *time.Time `json:"last_fetched_at"`
	LastFetchError  *string    `json:"last_fetch_error"`
	NodeCount       int        `json:"node_count"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

// SubscriptionRepo 订阅仓储
type SubscriptionRepo struct {
	db *DB
}

// NewSubscriptionRepo 创建订阅仓储
func NewSubscriptionRepo(db *DB) *SubscriptionRepo {
	return &SubscriptionRepo{db: db}
}

// Create 创建订阅
func (r *SubscriptionRepo) Create(ctx context.Context, sub *Subscription) error {
	query := `
		INSERT INTO subscriptions (name, url, target, enabled, auto_refresh, refresh_interval, description, tags)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, created_at, updated_at
	`

	err := r.db.Pool.QueryRow(ctx, query,
		sub.Name, sub.URL, sub.Target, sub.Enabled, sub.AutoRefresh, sub.RefreshInterval, sub.Description, sub.Tags,
	).Scan(&sub.ID, &sub.CreatedAt, &sub.UpdatedAt)

	if err != nil {
		return fmt.Errorf("创建订阅失败: %w", err)
	}

	return nil
}

// GetByName 根据名称获取订阅
func (r *SubscriptionRepo) GetByName(ctx context.Context, name string) (*Subscription, error) {
	query := `
		SELECT id, name, url, target, enabled, auto_refresh, refresh_interval, description, tags,
		       last_fetched_at, last_fetch_error, node_count, created_at, updated_at
		FROM subscriptions
		WHERE name = $1
	`

	sub := &Subscription{}
	err := r.db.Pool.QueryRow(ctx, query, name).Scan(
		&sub.ID,
		&sub.Name,
		&sub.URL,
		&sub.Target,
		&sub.Enabled,
		&sub.AutoRefresh,
		&sub.RefreshInterval,
		&sub.Description,
		&sub.Tags,
		&sub.LastFetchedAt,
		&sub.LastFetchError,
		&sub.NodeCount,
		&sub.CreatedAt,
		&sub.UpdatedAt,
	)

	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("订阅不存在: %s", name)
	}
	if err != nil {
		return nil, fmt.Errorf("查询订阅失败: %w", err)
	}

	return sub, nil
}

// List 列出所有订阅
func (r *SubscriptionRepo) List(ctx context.Context) ([]Subscription, error) {
	query := `
		SELECT id, name, url, target, enabled, auto_refresh, refresh_interval, description, tags,
		       last_fetched_at, last_fetch_error, node_count, created_at, updated_at
		FROM subscriptions
		ORDER BY created_at DESC
	`

	rows, err := r.db.Pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("查询订阅列表失败: %w", err)
	}
	defer rows.Close()

	subs := []Subscription{}
	for rows.Next() {
		sub := Subscription{}
		err := rows.Scan(
			&sub.ID,
			&sub.Name,
			&sub.URL,
			&sub.Target,
			&sub.Enabled,
			&sub.AutoRefresh,
			&sub.RefreshInterval,
			&sub.Description,
			&sub.Tags,
			&sub.LastFetchedAt,
			&sub.LastFetchError,
			&sub.NodeCount,
			&sub.CreatedAt,
			&sub.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("扫描订阅行失败: %w", err)
		}
		subs = append(subs, sub)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("遍历订阅行失败: %w", err)
	}

	return subs, nil
}

// Update 更新订阅
func (r *SubscriptionRepo) Update(ctx context.Context, sub *Subscription) error {
	query := `
		UPDATE subscriptions
		SET url = $2, target = $3, enabled = $4, auto_refresh = $5, refresh_interval = $6, description = $7, tags = $8
		WHERE name = $1
		RETURNING updated_at
	`

	err := r.db.Pool.QueryRow(ctx, query,
		sub.Name, sub.URL, sub.Target, sub.Enabled, sub.AutoRefresh, sub.RefreshInterval, sub.Description, sub.Tags,
	).Scan(&sub.UpdatedAt)

	if err == pgx.ErrNoRows {
		return fmt.Errorf("订阅不存在: %s", sub.Name)
	}
	if err != nil {
		return fmt.Errorf("更新订阅失败: %w", err)
	}

	return nil
}

// Delete 删除订阅
func (r *SubscriptionRepo) Delete(ctx context.Context, name string) error {
	query := `DELETE FROM subscriptions WHERE name = $1`

	result, err := r.db.Pool.Exec(ctx, query, name)
	if err != nil {
		return fmt.Errorf("删除订阅失败: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("订阅不存在: %s", name)
	}

	return nil
}

// UpdateFetchResult 更新订阅获取结果
func (r *SubscriptionRepo) UpdateFetchResult(ctx context.Context, name string, nodeCount int, fetchErr error) error {
	query := `
		UPDATE subscriptions
		SET last_fetched_at = CURRENT_TIMESTAMP,
		    node_count = $2,
		    last_fetch_error = $3
		WHERE name = $1
	`

	var errorMsg *string
	if fetchErr != nil {
		msg := fetchErr.Error()
		errorMsg = &msg
	}

	_, err := r.db.Pool.Exec(ctx, query, name, nodeCount, errorMsg)
	if err != nil {
		return fmt.Errorf("更新获取结果失败: %w", err)
	}

	return nil
}

// Exists 检查订阅是否存在
func (r *SubscriptionRepo) Exists(ctx context.Context, name string) (bool, error) {
	query := `SELECT 1 FROM subscriptions WHERE name = $1`

	var exists int
	err := r.db.Pool.QueryRow(ctx, query, name).Scan(&exists)
	if err == pgx.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("检查订阅存在性失败: %w", err)
	}

	return true, nil
}

// GetDB 获取数据库实例
func (r *SubscriptionRepo) GetDB() *DB {
	return r.db
}
