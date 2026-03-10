#!/bin/sh

set -eu

REPO_URL=${RULEFLOW_REPO_URL:-https://github.com/ablate-ai/RuleFlow.git}
DEFAULT_BOOTSTRAP_DIR=${RULEFLOW_DIR:-$HOME/RuleFlow}

bootstrap_install() {
  require_cmd git

  INSTALL_DIR=$DEFAULT_BOOTSTRAP_DIR

  if [ -d "$INSTALL_DIR/.git" ]; then
    log "检测到已有仓库，开始更新: $INSTALL_DIR"
    git -C "$INSTALL_DIR" pull --ff-only
  else
    log "开始克隆仓库到: $INSTALL_DIR"
    git clone "$REPO_URL" "$INSTALL_DIR"
  fi

  log "进入仓库执行安装..."
  exec sh "$INSTALL_DIR/install.sh"
}

ROOT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
ENV_FILE="$ROOT_DIR/.env.docker"
ENV_EXAMPLE="$ROOT_DIR/.env.example"
COMPOSE_FILE="$ROOT_DIR/deploy/docker-compose.yaml"

log() {
  printf '%s\n' "$1"
}

ensure_kv() {
  key=$1
  value=$2

  if grep -q "^$key=" "$ENV_FILE"; then
    tmp_file=$(mktemp)
    awk -F= -v key="$key" -v value="$value" '
      BEGIN { updated = 0 }
      $1 == key {
        print key "=" value
        updated = 1
        next
      }
      { print }
      END {
        if (updated == 0) {
          print key "=" value
        }
      }
    ' "$ENV_FILE" >"$tmp_file"
    mv "$tmp_file" "$ENV_FILE"
  else
    printf '%s=%s\n' "$key" "$value" >>"$ENV_FILE"
  fi
}

require_cmd() {
  if ! command -v "$1" >/dev/null 2>&1; then
    log "缺少依赖: $1"
    exit 1
  fi
}

require_cmd docker

if ! docker info >/dev/null 2>&1; then
  log "Docker 未运行，请先启动 Docker Desktop 或 Docker Engine。"
  exit 1
fi

if ! docker compose version >/dev/null 2>&1; then
  log "当前 Docker 不支持 'docker compose'。请升级 Docker / Compose 插件。"
  exit 1
fi

if [ ! -f "$COMPOSE_FILE" ]; then
  bootstrap_install
fi

if [ ! -f "$ENV_FILE" ]; then
  if [ -f "$ENV_EXAMPLE" ]; then
    cp "$ENV_EXAMPLE" "$ENV_FILE"
    log "已根据 .env.example 生成 .env.docker"
  else
    log "缺少 .env.example，无法自动生成 .env.docker"
    exit 1
  fi
fi

ensure_kv POSTGRES_DB ruleflow
ensure_kv POSTGRES_USER ruleflow
ensure_kv POSTGRES_PASSWORD password
ensure_kv DATABASE_URL 'postgresql://ruleflow:password@postgres:5432/ruleflow?sslmode=disable'
ensure_kv REDIS_ADDR 'redis:6379'

log "开始构建并启动 RuleFlow..."
docker compose --env-file "$ENV_FILE" -f "$COMPOSE_FILE" up -d --build

PORT_VALUE=$(awk -F= '/^PORT=/{print $2}' "$ENV_FILE" | tail -n 1)
if [ -z "${PORT_VALUE:-}" ]; then
  PORT_VALUE=8080
fi

log "RuleFlow 已启动"
log "访问地址: http://localhost:$PORT_VALUE"
log "停止命令: docker compose --env-file \"$ENV_FILE\" -f \"$COMPOSE_FILE\" down"
