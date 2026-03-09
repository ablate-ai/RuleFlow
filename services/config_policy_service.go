package services

import (
	"context"
	"fmt"
	"strings"

	"github.com/c.chen/ruleflow/database"
)

// ConfigPolicyService 配置策略服务
type ConfigPolicyService struct {
	policyRepo *database.ConfigPolicyRepo
	subRepo    *database.SubscriptionRepo
	nodeRepo   *database.NodeRepo
}

// NewConfigPolicyService 创建配置策略服务
func NewConfigPolicyService(
	policyRepo *database.ConfigPolicyRepo,
	subRepo *database.SubscriptionRepo,
	nodeRepo *database.NodeRepo,
) *ConfigPolicyService {
	return &ConfigPolicyService{
		policyRepo: policyRepo,
		subRepo:    subRepo,
		nodeRepo:   nodeRepo,
	}
}

// Create 创建配置策略
func (s *ConfigPolicyService) Create(ctx context.Context, policy *database.ConfigPolicy) error {
	// 验证订阅源是否存在
	for _, subName := range policy.SubscriptionNames {
		_, err := s.subRepo.GetByName(ctx, subName)
		if err != nil {
			return fmt.Errorf("订阅源不存在: %s", subName)
		}
	}

	// 验证目标类型
	if policy.Target != "clash" && policy.Target != "stash" {
		return fmt.Errorf("不支持的目标类型: %s", policy.Target)
	}

	return s.policyRepo.Create(ctx, policy)
}

// GetByName 根据名称获取配置策略
func (s *ConfigPolicyService) GetByName(ctx context.Context, name string) (*database.ConfigPolicy, error) {
	return s.policyRepo.GetByName(ctx, name)
}

// GetByToken 根据 token 获取配置策略
func (s *ConfigPolicyService) GetByToken(ctx context.Context, token string) (*database.ConfigPolicy, error) {
	return s.policyRepo.GetByToken(ctx, token)
}

// List 获取所有配置策略
func (s *ConfigPolicyService) List(ctx context.Context) ([]*database.ConfigPolicy, error) {
	return s.policyRepo.List(ctx)
}

// Update 更新配置策略
func (s *ConfigPolicyService) Update(ctx context.Context, policy *database.ConfigPolicy) error {
	// 验证订阅源是否存在
	for _, subName := range policy.SubscriptionNames {
		_, err := s.subRepo.GetByName(ctx, subName)
		if err != nil {
			return fmt.Errorf("订阅源不存在: %s", subName)
		}
	}

	// 验证目标类型
	if policy.Target != "clash" && policy.Target != "stash" {
		return fmt.Errorf("不支持的目标类型: %s", policy.Target)
	}

	return s.policyRepo.Update(ctx, policy)
}

// Delete 删除配置策略
func (s *ConfigPolicyService) Delete(ctx context.Context, name string) error {
	return s.policyRepo.Delete(ctx, name)
}

// GetEnabled 获取所有启用的配置策略
func (s *ConfigPolicyService) GetEnabled(ctx context.Context) ([]*database.ConfigPolicy, error) {
	return s.policyRepo.GetEnabled(ctx)
}

// ValidateConfig 验证配置策略
func (s *ConfigPolicyService) ValidateConfig(policy *database.ConfigPolicy) error {
	// 验证名称
	if policy.Name == "" {
		return fmt.Errorf("配置策略名称不能为空")
	}

	// 验证订阅源
	if len(policy.SubscriptionNames) == 0 {
		return fmt.Errorf("至少需要选择一个订阅源")
	}

	// 验证目标类型
	if policy.Target != "clash" && policy.Target != "stash" {
		return fmt.Errorf("不支持的目标类型: %s (支持: clash, stash)", policy.Target)
	}

	return nil
}

// GetNodesForPolicy 获取策略对应的节点
// 根据 policy.subscription_names 从节点表筛选节点
func (s *ConfigPolicyService) GetNodesForPolicy(ctx context.Context, policy *database.ConfigPolicy) ([]database.Node, error) {
	// 构建来源筛选条件
	// 策略的 subscription_names 对应节点的 source 字段（格式：subscription:{name}）
	sources := make([]string, 0, len(policy.SubscriptionNames))
	for _, subName := range policy.SubscriptionNames {
		sources = append(sources, fmt.Sprintf("subscription:%s", subName))
	}

	// 收集所有节点
	allNodes := make([]database.Node, 0)
	for _, source := range sources {
		nodes, err := s.nodeRepo.List(ctx, database.NodeFilter{
			Source: source,
		})
		if err != nil {
			return nil, fmt.Errorf("获取节点失败 (来源: %s): %w", source, err)
		}
		allNodes = append(allNodes, nodes...)
	}

	// 应用节点过滤条件
	if policy.NodeFilters != nil && len(policy.NodeFilters) > 0 {
		allNodes = s.applyNodeFilters(allNodes, policy.NodeFilters)
	}

	return allNodes, nil
}

// applyNodeFilters 应用节点过滤条件
func (s *ConfigPolicyService) applyNodeFilters(nodes []database.Node, filters map[string]interface{}) []database.Node {
	filtered := make([]database.Node, 0, len(nodes))

	for _, node := range nodes {
		// 默认包含节点
		include := true

		// 按协议筛选
		if protocols, ok := filters["protocols"].([]interface{}); ok {
			protoSet := make(map[string]bool)
			for _, p := range protocols {
				if protoStr, ok := p.(string); ok {
					protoSet[protoStr] = true
				}
			}
			if len(protoSet) > 0 && !protoSet[node.Protocol] {
				include = false
			}
		}

		// 按关键词筛选
		if keywords, ok := filters["keywords"].([]interface{}); ok && include {
			matches := false
			nodeNameLower := strings.ToLower(node.Name)
			for _, kw := range keywords {
				if keywordStr, ok := kw.(string); ok {
					if strings.Contains(nodeNameLower, strings.ToLower(keywordStr)) {
						matches = true
						break
					}
				}
			}
			if len(keywords) > 0 && !matches {
				include = false
			}
		}

		// 按标签筛选
		if tags, ok := filters["tags"].([]interface{}); ok && include {
			tagSet := make(map[string]bool)
			for _, t := range tags {
				if tagStr, ok := t.(string); ok {
					tagSet[tagStr] = true
				}
			}
			if len(tagSet) > 0 {
				hasTag := false
				for _, nodeTag := range node.Tags {
					if tagSet[nodeTag] {
						hasTag = true
						break
					}
				}
				if !hasTag {
					include = false
				}
			}
		}

		// 只包含启用的节点
		if include && !node.Enabled {
			include = false
		}

		if include {
			filtered = append(filtered, node)
		}
	}

	return filtered
}

// ApplyNodeFilters 应用节点过滤条件（保留向后兼容）
func (s *ConfigPolicyService) ApplyNodeFilters(nodes []*database.ConfigPolicy, filters map[string]interface{}) []*database.ConfigPolicy {
	// 这里保留原有的方法签名以向后兼容
	// 实际的过滤逻辑现在使用 applyNodeFilters 方法处理 Node 类型
	return nodes
}

// GetPolicyWithNodes 获取策略及其节点
func (s *ConfigPolicyService) GetPolicyWithNodes(ctx context.Context, policyName string) (*database.ConfigPolicy, []database.Node, error) {
	policy, err := s.GetByName(ctx, policyName)
	if err != nil {
		return nil, nil, err
	}

	nodes, err := s.GetNodesForPolicy(ctx, policy)
	if err != nil {
		return nil, nil, err
	}

	return policy, nodes, nil
}
