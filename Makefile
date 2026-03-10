ENV_FILE ?= .env
MIGRATION_FILE := migrations/init.sql
GOFLAGS ?= -buildvcs=false
GOCACHE_DIR ?= $(CURDIR)/.cache/go-build

.PHONY: help env-check migrate build run test

help:
	@echo "可用命令:"
	@echo "  make migrate        # 读取 .env 并初始化数据库"
	@echo "  make build          # 编译程序"
	@echo "  make run            # 读取 .env 后启动服务"
	@echo "  make test           # 运行测试"

env-check:
	@if [ ! -f "$(ENV_FILE)" ]; then \
		echo "缺少 $(ENV_FILE)"; \
		exit 1; \
	fi

migrate: env-check
	@bash -lc 'set -a; source "$(ENV_FILE)"; set +a; test -n "$$DATABASE_URL" || { echo "DATABASE_URL 未设置"; exit 1; }; psql "$$DATABASE_URL" -f "$(MIGRATION_FILE)"'

build:
	GOCACHE=$(GOCACHE_DIR) GOFLAGS="$(GOFLAGS)" go build -o ruleflow .

run: env-check
	@bash -lc 'set -a; source "$(ENV_FILE)"; set +a; GOCACHE="$(GOCACHE_DIR)" GOFLAGS="$(GOFLAGS)" go run .'

test:
	GOCACHE=$(GOCACHE_DIR) GOFLAGS="$(GOFLAGS)" go test ./...
