package services

import (
	"context"
	"fmt"

	"github.com/c.chen/ruleflow/database"
)

// TemplateService 模板服务
type TemplateService struct {
	templateRepo *database.TemplateRepo
}

// NewTemplateService 创建模板服务
func NewTemplateService(templateRepo *database.TemplateRepo) *TemplateService {
	return &TemplateService{
		templateRepo: templateRepo,
	}
}

// CreateTemplate 创建模板
func (s *TemplateService) CreateTemplate(ctx context.Context, tpl *database.Template) error {
	// 检查名称是否已存在
	exists, err := s.templateRepo.Exists(ctx, tpl.Name)
	if err != nil {
		return err
	}
	if exists {
		return fmt.Errorf("模板名称已存在: %s", tpl.Name)
	}

	// 如果是第一个模板，自动设置为默认
	if tpl.IsDefault {
		return s.templateRepo.Create(ctx, tpl)
	}

	// 检查是否已有默认模板
	_, err = s.templateRepo.GetDefault(ctx)
	if err != nil && err.Error() == "未找到默认模板" {
		// 没有默认模板，将此模板设为默认
		tpl.IsDefault = true
	}

	return s.templateRepo.Create(ctx, tpl)
}

// GetTemplateByName 根据名称获取模板
func (s *TemplateService) GetTemplateByName(ctx context.Context, name string) (*database.Template, error) {
	return s.templateRepo.GetByName(ctx, name)
}

// GetDefaultTemplate 获取默认模板
func (s *TemplateService) GetDefaultTemplate(ctx context.Context) (*database.Template, error) {
	return s.templateRepo.GetDefault(ctx)
}

// ListTemplates 列出所有模板
func (s *TemplateService) ListTemplates(ctx context.Context) ([]database.Template, error) {
	return s.templateRepo.List(ctx)
}

// UpdateTemplate 更新模板
func (s *TemplateService) UpdateTemplate(ctx context.Context, tpl *database.Template) error {
	return s.templateRepo.Update(ctx, tpl)
}

// DeleteTemplate 删除模板
func (s *TemplateService) DeleteTemplate(ctx context.Context, name string) error {
	return s.templateRepo.Delete(ctx, name)
}

// Health 检查服务健康状态
func (s *TemplateService) Health(ctx context.Context) map[string]string {
	status := map[string]string{
		"templates": "available",
	}

	// 简单测试数据库连接
	_, err := s.templateRepo.List(ctx)
	if err != nil {
		status["status"] = "unhealthy"
		status["error"] = err.Error()
	} else {
		status["status"] = "healthy"
	}

	return status
}
