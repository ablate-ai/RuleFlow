package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/c.chen/ruleflow/api"
	"github.com/c.chen/ruleflow/cache"
	"github.com/c.chen/ruleflow/config"
	"github.com/c.chen/ruleflow/database"
	"github.com/c.chen/ruleflow/internal/app"
	"github.com/c.chen/ruleflow/services"
)

func main() {
	// 加载配置
	cfg := config.Load()

	// 验证模板文件
	templateFile := app.ResolveProjectPath(cfg.RuleTemplateFile)
	if _, err := os.Stat(templateFile); err != nil {
		log.Printf("⚠️ 模板文件不可用: %s, err=%v\n", templateFile, err)
	} else {
		log.Printf("✅ 已加载模板文件: %s\n", templateFile)
	}

	// 初始化数据库和 Redis（可选）
	var db *database.DB
	var redisClient *cache.Client
	var subscriptionService *services.SubscriptionService

	// 尝试连接数据库
	if strings.TrimSpace(cfg.DatabaseURL) != "" {
		var err error
		db, err = database.New(cfg.DatabaseURL)
		if err != nil {
			log.Printf("⚠️ 数据库连接失败（将使用无数据库模式）: %v\n", err)
		} else {
			log.Printf("✅ 数据库连接成功\n")
			defer db.Close()
		}
	}

	// 尝试连接 Redis
	if os.Getenv("REDIS_ADDR") != "" {
		var err error
		redisClient, err = cache.New(cfg.RedisAddr, cfg.RedisPassword, cfg.RedisDB)
		if err != nil {
			log.Printf("⚠️ Redis 连接失败（将使用无缓存模式）: %v\n", err)
		} else {
			log.Printf("✅ Redis 连接成功\n")
			defer redisClient.Close()
		}
	}

	// 创建订阅服务（如果数据库和 Redis 都可用）
	var templateService *services.TemplateService
	var configPolicyService *services.ConfigPolicyService
	var nodeService *services.NodeService
	var subscriptionSyncService *services.SubscriptionSyncService
	if db != nil && redisClient != nil {
		subscriptionRepo := database.NewSubscriptionRepo(db)
		subscriptionCache := cache.NewSubscriptionCache(redisClient, time.Duration(cfg.CacheTTLSeconds)*time.Second)
		subscriptionService = services.NewSubscriptionService(subscriptionRepo, subscriptionCache)

		// 创建模板服务
		templateRepo := database.NewTemplateRepo(db)
		templateService = services.NewTemplateService(templateRepo)

		// 创建配置策略服务
		configPolicyRepo := database.NewConfigPolicyRepo(db)
		nodeRepo := database.NewNodeRepo(db)
		configPolicyService = services.NewConfigPolicyService(configPolicyRepo, subscriptionRepo, nodeRepo)

		// 创建节点服务
		nodeService = services.NewNodeService(nodeRepo)

		// 创建订阅同步服务
		subscriptionSyncService = services.NewSubscriptionSyncService(subscriptionRepo, nodeRepo)
	}

	// 创建 API 处理器
	var apiHandlers *api.Handlers
	if subscriptionService != nil && templateService != nil && configPolicyService != nil && nodeService != nil && subscriptionSyncService != nil {
		apiHandlers = api.NewHandlers(subscriptionService, templateService, configPolicyService, nodeService, subscriptionSyncService)
	}

	// 设置路由
	setupRoutes(cfg, apiHandlers, subscriptionService)

	// 启动服务器
	port := cfg.Port
	log.Printf("🚀 服务器启动: http://localhost:%s\n", port)
	log.Printf("💡 Clash 订阅: http://localhost:%s/sub?url=<订阅地址>[&template=<模板路径>]\n", port)
	log.Printf("💡 Stash 订阅: http://localhost:%s/sub?url=<订阅地址>&target=stash[&template=<模板路径>]\n", port)

	if subscriptionService != nil && templateService != nil {
		log.Printf("💡 数据库订阅: http://localhost:%s/sub/<订阅名称>?target=clash[&template=<模板路径>]\n", port)
		log.Printf("💡 管理界面: http://localhost:%s/web/admin.html\n", port)
		log.Printf("💡 管理接口: http://localhost:%s/api/subscriptions\n", port)
		log.Printf("💡 模板接口: http://localhost:%s/api/templates\n", port)
		log.Printf("💡 健康检查: http://localhost:%s/health\n", port)
	}

	// 优雅关闭
	server := &http.Server{
		Addr:    ":" + port,
		Handler: nil, // 使用默认的 http.DefaultServeMux
	}

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("❌ 服务器错误: %v\n", err)
		}
	}()

	// 等待中断信号
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("🛑 正在关闭服务器...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Printf("❌ 服务器关闭失败: %v\n", err)
	}

	log.Println("✅ 服务器已关闭")
}

func setupRoutes(cfg *config.Config, apiHandlers *api.Handlers, subscriptionService *services.SubscriptionService) {
	// 静态文件服务
	fs := http.FileServer(http.Dir(app.ResolveProjectPath("web")))
	http.Handle("/web/", http.StripPrefix("/web/", fs))

	// 主页和原有订阅接口（向后兼容）
	http.HandleFunc("/", app.IndexHandler)

	// 订阅接口（原有模式）
	http.HandleFunc("/sub", func(w http.ResponseWriter, r *http.Request) {
		subURL := r.URL.Query().Get("url")
		if subURL != "" {
			// 原有模式：直接从 URL 获取
			app.SubHandler(w, r)
		} else {
			// 新模式：需要订阅服务
			if subscriptionService == nil {
				http.Error(w, "数据库模式未启用", http.StatusServiceUnavailable)
				return
			}
			app.SubHandlerWithName(w, r, subscriptionService)
		}
	})

	// API 路由（如果启用）
	if apiHandlers != nil {
		// 应用中间件
		_ = api.LoggingMiddleware(api.CORSMiddleware(api.RecoveryMiddleware(http.DefaultServeMux)))

		// 订阅管理 API
		http.HandleFunc("/api/subscriptions", func(w http.ResponseWriter, r *http.Request) {
			switch r.Method {
			case http.MethodPost:
				apiHandlers.CreateSubscription(w, r)
			case http.MethodGet:
				apiHandlers.ListSubscriptions(w, r)
			default:
				api.SendError(w, http.StatusMethodNotAllowed, "方法不允许")
			}
		})

		http.HandleFunc("/api/subscriptions/", func(w http.ResponseWriter, r *http.Request) {
			// 检查是否是同步操作
			if strings.HasSuffix(r.URL.Path, "/sync") {
				if r.Method == http.MethodPost {
					apiHandlers.SyncSubscription(w, r)
				} else {
					api.SendError(w, http.StatusMethodNotAllowed, "方法不允许")
				}
				return
			}
			// 检查是否是获取同步状态
			if strings.HasSuffix(r.URL.Path, "/sync-status") {
				if r.Method == http.MethodGet {
					apiHandlers.GetSubscriptionSyncStatus(w, r)
				} else {
					api.SendError(w, http.StatusMethodNotAllowed, "方法不允许")
				}
				return
			}

			// 原有的订阅 CRUD 操作
			if strings.HasSuffix(r.URL.Path, "/refresh") {
				// /api/subscriptions/{name}/refresh
				apiHandlers.RefreshSubscription(w, r)
				return
			}

			switch r.Method {
			case http.MethodGet:
				apiHandlers.GetSubscription(w, r)
			case http.MethodPut, http.MethodPatch:
				apiHandlers.UpdateSubscription(w, r)
			case http.MethodDelete:
				apiHandlers.DeleteSubscription(w, r)
			default:
				api.SendError(w, http.StatusMethodNotAllowed, "方法不允许")
			}
		})

		// 模板管理 API
		http.HandleFunc("/api/templates", func(w http.ResponseWriter, r *http.Request) {
			switch r.Method {
			case http.MethodPost:
				apiHandlers.CreateTemplate(w, r)
			case http.MethodGet:
				apiHandlers.ListTemplates(w, r)
			default:
				api.SendError(w, http.StatusMethodNotAllowed, "方法不允许")
			}
		})

		http.HandleFunc("/api/templates/", func(w http.ResponseWriter, r *http.Request) {
			// 处理 /api/templates/{name} 路径
			switch r.Method {
			case http.MethodGet:
				apiHandlers.GetTemplate(w, r)
			case http.MethodPut, http.MethodPatch:
				apiHandlers.UpdateTemplate(w, r)
			case http.MethodDelete:
				apiHandlers.DeleteTemplate(w, r)
			default:
				api.SendError(w, http.StatusMethodNotAllowed, "方法不允许")
			}
		})

		// 配置策略管理 API
		http.HandleFunc("/api/config-policies", func(w http.ResponseWriter, r *http.Request) {
			switch r.Method {
			case http.MethodPost:
				apiHandlers.CreateConfigPolicy(w, r)
			case http.MethodGet:
				apiHandlers.ListConfigPolicies(w, r)
			default:
				api.SendError(w, http.StatusMethodNotAllowed, "方法不允许")
			}
		})

		http.HandleFunc("/api/config-policies/", func(w http.ResponseWriter, r *http.Request) {
			// 处理 /api/config-policies/{name} 路径
			switch r.Method {
			case http.MethodGet:
				apiHandlers.GetConfigPolicy(w, r)
			case http.MethodPut, http.MethodPatch:
				apiHandlers.UpdateConfigPolicy(w, r)
			case http.MethodDelete:
				apiHandlers.DeleteConfigPolicy(w, r)
			default:
				api.SendError(w, http.StatusMethodNotAllowed, "方法不允许")
			}
		})

		// 缓存管理 API
		http.HandleFunc("/api/cache", func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodDelete {
				apiHandlers.ClearCache(w, r)
			} else {
				api.SendError(w, http.StatusMethodNotAllowed, "方法不允许")
			}
		})

		http.HandleFunc("/api/cache/", func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodDelete {
				apiHandlers.ClearCache(w, r)
			} else {
				api.SendError(w, http.StatusMethodNotAllowed, "方法不允许")
			}
		})

		// 节点管理 API
		http.HandleFunc("/api/nodes", func(w http.ResponseWriter, r *http.Request) {
			switch r.Method {
			case http.MethodPost:
				apiHandlers.CreateNode(w, r)
			case http.MethodGet:
				apiHandlers.ListNodes(w, r)
			default:
				api.SendError(w, http.StatusMethodNotAllowed, "方法不允许")
			}
		})

		http.HandleFunc("/api/nodes/", func(w http.ResponseWriter, r *http.Request) {
			// 处理 /api/nodes/{id} 或 /api/nodes/batch 路径
			path := r.URL.Path
			if strings.Contains(path, "/batch") {
				// 批量操作
				if r.Method == http.MethodPatch || r.Method == http.MethodPost {
					apiHandlers.BatchNodeOperation(w, r)
				} else {
					api.SendError(w, http.StatusMethodNotAllowed, "方法不允许")
				}
				return
			}

			// 单个节点操作
			switch r.Method {
			case http.MethodGet:
				apiHandlers.GetNode(w, r)
			case http.MethodPut, http.MethodPatch:
				apiHandlers.UpdateNode(w, r)
			case http.MethodDelete:
				apiHandlers.DeleteNode(w, r)
			default:
				api.SendError(w, http.StatusMethodNotAllowed, "方法不允许")
			}
		})

		// 节点统计 API
		http.HandleFunc("/api/nodes/stats", func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodGet {
				apiHandlers.GetNodeStats(w, r)
			} else {
				api.SendError(w, http.StatusMethodNotAllowed, "方法不允许")
			}
		})

		// 健康检查
		http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
			apiHandlers.Health(w, r)
		})
	}
}
