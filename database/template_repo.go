package database

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

// Template 模板模型
type Template struct {
	ID          int       `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Content     string    `json:"content"`
	Target      string    `json:"target"`
	Tags        []string  `json:"tags"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// TemplateRepo 模板仓储
type TemplateRepo struct {
	db *DB
}

// NewTemplateRepo 创建模板仓储
func NewTemplateRepo(db *DB) *TemplateRepo {
	return &TemplateRepo{db: db}
}

// Create 创建模板
func (r *TemplateRepo) Create(ctx context.Context, tpl *Template) error {
	query := `
		INSERT INTO templates (name, description, content, target, tags)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at, updated_at
	`

	err := r.db.Pool.QueryRow(ctx, query,
		tpl.Name, tpl.Description, tpl.Content, tpl.Target, tpl.Tags,
	).Scan(&tpl.ID, &tpl.CreatedAt, &tpl.UpdatedAt)

	if err != nil {
		return fmt.Errorf("创建模板失败: %w", err)
	}

	return nil
}

// GetByName 根据名称获取模板
func (r *TemplateRepo) GetByName(ctx context.Context, name string) (*Template, error) {
	query := `
		SELECT id, name, description, content, target, tags,
		       created_at, updated_at
		FROM templates
		WHERE name = $1
	`

	tpl := &Template{}
	err := r.db.Pool.QueryRow(ctx, query, name).Scan(
		&tpl.ID,
		&tpl.Name,
		&tpl.Description,
		&tpl.Content,
		&tpl.Target,
		&tpl.Tags,
		&tpl.CreatedAt,
		&tpl.UpdatedAt,
	)

	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("模板不存在: %s", name)
	}
	if err != nil {
		return nil, fmt.Errorf("查询模板失败: %w", err)
	}

	return tpl, nil
}

// GetByID 根据 ID 获取模板
func (r *TemplateRepo) GetByID(ctx context.Context, id int) (*Template, error) {
	query := `
		SELECT id, name, description, content, target, tags,
		       created_at, updated_at
		FROM templates
		WHERE id = $1
	`

	tpl := &Template{}
	err := r.db.Pool.QueryRow(ctx, query, id).Scan(
		&tpl.ID,
		&tpl.Name,
		&tpl.Description,
		&tpl.Content,
		&tpl.Target,
		&tpl.Tags,
		&tpl.CreatedAt,
		&tpl.UpdatedAt,
	)

	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("模板不存在: %d", id)
	}
	if err != nil {
		return nil, fmt.Errorf("查询模板失败: %w", err)
	}

	return tpl, nil
}

// List 列出所有模板
func (r *TemplateRepo) List(ctx context.Context) ([]Template, error) {
	query := `
		SELECT id, name, description, content, target, tags,
		       created_at, updated_at
		FROM templates
		ORDER BY created_at DESC
	`

	rows, err := r.db.Pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("查询模板列表失败: %w", err)
	}
	defer rows.Close()

	tpls := []Template{}
	for rows.Next() {
		tpl := Template{}
		err := rows.Scan(
			&tpl.ID,
			&tpl.Name,
			&tpl.Description,
			&tpl.Content,
			&tpl.Target,
			&tpl.Tags,
			&tpl.CreatedAt,
			&tpl.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("扫描模板行失败: %w", err)
		}
		tpls = append(tpls, tpl)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("遍历模板行失败: %w", err)
	}

	return tpls, nil
}

// Update 更新模板，支持修改模板名称，并同步更新策略引用
func (r *TemplateRepo) Update(ctx context.Context, id int, tpl *Template) error {
	tx, err := r.db.Pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("开始事务失败: %w", err)
	}
	defer tx.Rollback(ctx)

	var oldName string
	if err := tx.QueryRow(ctx, `SELECT name FROM templates WHERE id = $1`, id).Scan(&oldName); err != nil {
		if err == pgx.ErrNoRows {
			return fmt.Errorf("模板不存在: %d", id)
		}
		return fmt.Errorf("查询模板失败: %w", err)
	}

	query := `
		UPDATE templates
		SET name = $2, description = $3, content = $4, target = $5, tags = $6
		WHERE id = $1
		RETURNING updated_at
	`

	err = tx.QueryRow(ctx, query,
		id, tpl.Name, tpl.Description, tpl.Content, tpl.Target, tpl.Tags,
	).Scan(&tpl.UpdatedAt)

	if err == pgx.ErrNoRows {
		return fmt.Errorf("模板不存在: %d", id)
	}
	if err != nil {
		return fmt.Errorf("更新模板失败: %w", err)
	}

	if oldName != tpl.Name {
		if _, err := tx.Exec(ctx, `
			UPDATE config_policies
			SET template_name = $2
			WHERE template_name = $1
		`, oldName, tpl.Name); err != nil {
			return fmt.Errorf("同步更新策略模板引用失败: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("提交事务失败: %w", err)
	}

	return nil
}

// Delete 删除模板
func (r *TemplateRepo) Delete(ctx context.Context, id int) error {
	query := `DELETE FROM templates WHERE id = $1`

	result, err := r.db.Pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("删除模板失败: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("模板不存在: %d", id)
	}

	return nil
}

// Exists 检查模板是否存在
func (r *TemplateRepo) Exists(ctx context.Context, name string) (bool, error) {
	query := `SELECT 1 FROM templates WHERE name = $1`

	var exists int
	err := r.db.Pool.QueryRow(ctx, query, name).Scan(&exists)
	if err == pgx.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("检查模板存在性失败: %w", err)
	}

	return true, nil
}

// GetDB 获取数据库实例
func (r *TemplateRepo) GetDB() *DB {
	return r.db
}
