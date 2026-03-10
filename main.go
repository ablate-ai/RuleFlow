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
	var subscriptionCache *cache.SubscriptionCache
	if db != nil && redisClient != nil {
		subscriptionRepo := database.NewSubscriptionRepo(db)
		subscriptionCache = cache.NewSubscriptionCache(redisClient, time.Duration(cfg.CacheTTLSeconds)*time.Second)
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
		subscriptionSyncService = services.NewSubscriptionSyncService(subscriptionRepo, nodeRepo, subscriptionCache)
	}

	// 启动自动刷新调度器
	schedulerCtx, cancelScheduler := context.WithCancel(context.Background())
	defer cancelScheduler()
	if subscriptionSyncService != nil {
		scheduler := services.NewSubscriptionScheduler(database.NewSubscriptionRepo(db), subscriptionSyncService)
		scheduler.Start(schedulerCtx)
	}

	// 创建 API 处理器
	var apiHandlers *api.Handlers
	if subscriptionService != nil && templateService != nil && configPolicyService != nil && nodeService != nil && subscriptionSyncService != nil {
		apiHandlers = api.NewHandlers(subscriptionService, templateService, configPolicyService, nodeService, subscriptionSyncService, subscriptionCache)
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
		Handler: api.LoggingMiddleware(api.CORSMiddleware(api.RecoveryMiddleware(http.DefaultServeMux))),
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

	// 收到第二次信号时强制退出
	signal.Reset(syscall.SIGINT, syscall.SIGTERM)

	log.Println("🛑 正在关闭服务器...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Printf("⚠️ 优雅关闭超时，强制关闭: %v\n", err)
		_ = server.Close()
	}

	log.Println("✅ 服务器已关闭")
	os.Exit(0)
}

func setupRoutes(cfg *config.Config, apiHandlers *api.Handlers, subscriptionService *services.SubscriptionService) {
	if cfg.AdminPassword != "" {
		log.Printf("🔒 Web 控制台鉴权已启用\n")
	} else {
		log.Printf("⚠️ ADMIN_PASSWORD 未设置，Web 控制台无需鉴权\n")
	}

	// 登录页（不需要鉴权，直接提供静态文件）
	http.HandleFunc("/login", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			// 处理登录表单
			pass := r.FormValue("password")
			if cfg.AdminPassword == "" || pass == cfg.AdminPassword {
				api.SetSessionCookie(w, cfg.AdminPassword)
				next := r.FormValue("next")
				if next == "" {
					next = "/web/index.html"
				}
				http.Redirect(w, r, next, http.StatusFound)
			} else {
				http.Redirect(w, r, "/login?error=1&next="+r.FormValue("next"), http.StatusFound)
			}
			return
		}
		http.ServeFile(w, r, app.ResolveProjectPath("web/login.html"))
	})

	// 退出登录
	http.HandleFunc("/logout", func(w http.ResponseWriter, r *http.Request) {
		api.ClearSessionCookie(w)
		http.Redirect(w, r, "/login", http.StatusFound)
	})

	// 静态文件服务（需要鉴权）
	webAuth := api.WebAuthMiddleware(cfg.AdminPassword)
	fs := http.FileServer(http.Dir(app.ResolveProjectPath("web")))
	http.Handle("/web/", webAuth(http.StripPrefix("/web/", fs)))

	// 主页和原有订阅接口（向后兼容）
	http.Handle("/", webAuth(http.HandlerFunc(app.IndexHandler)))

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

	// API 路由（统一用子 mux，整体加鉴权）
	apiMux := http.NewServeMux()

	if apiHandlers == nil {
		// 数据库未启用时，所有 /api/ 请求返回 JSON 错误而非 HTML
		apiMux.HandleFunc("/api/", func(w http.ResponseWriter, r *http.Request) {
			api.SendError(w, http.StatusServiceUnavailable, "数据库模式未启用，API 功能不可用")
		})
	} else {
		// 订阅管理 API
		apiMux.HandleFunc("/api/subscriptions", func(w http.ResponseWriter, r *http.Request) {
			switch r.Method {
			case http.MethodPost:
				apiHandlers.CreateSubscription(w, r)
			case http.MethodGet:
				apiHandlers.ListSubscriptions(w, r)
			default:
				api.SendError(w, http.StatusMethodNotAllowed, "方法不允许")
			}
		})

		apiMux.HandleFunc("/api/subscriptions/", func(w http.ResponseWriter, r *http.Request) {
			if strings.HasSuffix(r.URL.Path, "/sync") {
				if r.Method == http.MethodPost {
					apiHandlers.SyncSubscription(w, r)
				} else {
					api.SendError(w, http.StatusMethodNotAllowed, "方法不允许")
				}
				return
			}
			if strings.HasSuffix(r.URL.Path, "/sync-status") {
				if r.Method == http.MethodGet {
					apiHandlers.GetSubscriptionSyncStatus(w, r)
				} else {
					api.SendError(w, http.StatusMethodNotAllowed, "方法不允许")
				}
				return
			}
			if strings.HasSuffix(r.URL.Path, "/refresh") {
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
		apiMux.HandleFunc("/api/templates", func(w http.ResponseWriter, r *http.Request) {
			switch r.Method {
			case http.MethodPost:
				apiHandlers.CreateTemplate(w, r)
			case http.MethodGet:
				apiHandlers.ListTemplates(w, r)
			default:
				api.SendError(w, http.StatusMethodNotAllowed, "方法不允许")
			}
		})

		apiMux.HandleFunc("/api/templates/", func(w http.ResponseWriter, r *http.Request) {
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
		apiMux.HandleFunc("/api/config-policies", func(w http.ResponseWriter, r *http.Request) {
			switch r.Method {
			case http.MethodPost:
				apiHandlers.CreateConfigPolicy(w, r)
			case http.MethodGet:
				apiHandlers.ListConfigPolicies(w, r)
			default:
				api.SendError(w, http.StatusMethodNotAllowed, "方法不允许")
			}
		})

		apiMux.HandleFunc("/api/config-policies/", func(w http.ResponseWriter, r *http.Request) {
			// /api/config-policies/{id}/cache
			if strings.HasSuffix(r.URL.Path, "/cache") {
				if r.Method == http.MethodDelete {
					apiHandlers.ClearPolicyConfigCache(w, r)
				} else {
					api.SendError(w, http.StatusMethodNotAllowed, "方法不允许")
				}
				return
			}
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
		apiMux.HandleFunc("/api/cache", func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodDelete {
				apiHandlers.ClearCache(w, r)
			} else {
				api.SendError(w, http.StatusMethodNotAllowed, "方法不允许")
			}
		})

		apiMux.HandleFunc("/api/cache/", func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodDelete {
				apiHandlers.ClearCache(w, r)
			} else {
				api.SendError(w, http.StatusMethodNotAllowed, "方法不允许")
			}
		})

		// 节点管理 API
		apiMux.HandleFunc("/api/nodes", func(w http.ResponseWriter, r *http.Request) {
			switch r.Method {
			case http.MethodPost:
				apiHandlers.CreateNode(w, r)
			case http.MethodGet:
				apiHandlers.ListNodes(w, r)
			default:
				api.SendError(w, http.StatusMethodNotAllowed, "方法不允许")
			}
		})

		apiMux.HandleFunc("/api/nodes/stats", func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodGet {
				apiHandlers.GetNodeStats(w, r)
			} else {
				api.SendError(w, http.StatusMethodNotAllowed, "方法不允许")
			}
		})

		apiMux.HandleFunc("/api/nodes/import", func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodPost {
				apiHandlers.ImportNodes(w, r)
			} else {
				api.SendError(w, http.StatusMethodNotAllowed, "方法不允许")
			}
		})

		apiMux.HandleFunc("/api/nodes/", func(w http.ResponseWriter, r *http.Request) {
			path := r.URL.Path
			if strings.Contains(path, "/batch") {
				if r.Method == http.MethodPatch || r.Method == http.MethodPost {
					apiHandlers.BatchNodeOperation(w, r)
				} else {
					api.SendError(w, http.StatusMethodNotAllowed, "方法不允许")
				}
				return
			}
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

		// 配置生成（供客户端直接订阅）GET /config?token=xxx，不需要鉴权
		http.HandleFunc("/config", func(w http.ResponseWriter, r *http.Request) {
			apiHandlers.GenerateConfig(w, r)
		})

		// 健康检查，不需要鉴权
		http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
			apiHandlers.Health(w, r)
		})
	}

	// 将 API 子 mux 挂载到默认 mux，整体加鉴权
	http.Handle("/api/", api.APIAuthMiddleware(cfg.AdminPassword)(apiMux))
}
