#!/bin/sh

set -eu

GITHUB_REPO="ablate-ai/RuleFlow"
GITHUB_BRANCH="${RULEFLOW_BRANCH:-main}"
RAW_BASE="https://raw.githubusercontent.com/${GITHUB_REPO}/${GITHUB_BRANCH}"
INSTALL_DIR="${RULEFLOW_DIR:-$HOME/ruleflow}"

ENV_FILE="$INSTALL_DIR/.env"
COMPOSE_FILE="$INSTALL_DIR/deploy/docker-compose.yaml"
BIN_PATH="$INSTALL_DIR/ruleflow"

log() {
  printf '%s\n' "$1"
}

require_cmd() {
  if ! command -v "$1" >/dev/null 2>&1; then
    log "缺少依赖: $1"
    exit 1
  fi
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

detect_arch() {
  arch=$(uname -m)
  case "$arch" in
    x86_64)        printf 'amd64' ;;
    aarch64|arm64) printf 'arm64' ;;
    *)
      log "不支持的架构: $arch"
      exit 1
      ;;
  esac
}

download_files() {
  require_cmd curl

  log "创建安装目录: $INSTALL_DIR"
  mkdir -p "$INSTALL_DIR/deploy" "$INSTALL_DIR/migrations"

  log "下载 docker-compose.yaml..."
  curl -fsSL "$RAW_BASE/deploy/docker-compose.yaml" -o "$COMPOSE_FILE"

  log "下载 migrations/init.sql..."
  curl -fsSL "$RAW_BASE/migrations/init.sql" -o "$INSTALL_DIR/migrations/init.sql"

  log "下载 uninstall.sh..."
  curl -fsSL "$RAW_BASE/uninstall.sh" -o "$INSTALL_DIR/uninstall.sh"
  chmod +x "$INSTALL_DIR/uninstall.sh"
}

download_binary() {
  require_cmd curl

  ARCH=$(detect_arch)
  BINARY_NAME="ruleflow-linux-${ARCH}"
  DOWNLOAD_URL="https://github.com/${GITHUB_REPO}/releases/latest/download/${BINARY_NAME}"

  log "下载二进制: $DOWNLOAD_URL"
  curl -fsSL "$DOWNLOAD_URL" -o "$BIN_PATH"
  chmod +x "$BIN_PATH"
  log "下载完成: $BIN_PATH"
}

install_systemd_service() {
  SERVICE_FILE="/etc/systemd/system/ruleflow.service"

  log "创建 systemd 服务: $SERVICE_FILE"
  cat >"$SERVICE_FILE" <<EOF
[Unit]
Description=RuleFlow
After=network.target

[Service]
Type=simple
EnvironmentFile=$ENV_FILE
ExecStart=$BIN_PATH
Restart=on-failure
RestartSec=5s
StandardOutput=journal
StandardError=journal

[Install]
WantedBy=multi-user.target
EOF

  systemctl daemon-reload
  systemctl enable ruleflow
  systemctl restart ruleflow
}

if [ "$(id -u)" -ne 0 ]; then
  log "请以 root 身份运行，例如: curl ... | sudo sh"
  exit 1
fi

require_cmd docker

if ! docker info >/dev/null 2>&1; then
  log "Docker 未运行，请先启动 Docker Desktop 或 Docker Engine。"
  exit 1
fi

if ! docker compose version >/dev/null 2>&1; then
  log "当前 Docker 不支持 'docker compose'。请升级 Docker / Compose 插件。"
  exit 1
fi

download_files

# 初始化 .env
if [ ! -f "$ENV_FILE" ]; then
  touch "$ENV_FILE"
fi

ensure_kv POSTGRES_DB ruleflow
ensure_kv POSTGRES_USER ruleflow
ensure_kv POSTGRES_PASSWORD password
ensure_kv DATABASE_URL 'postgresql://ruleflow:password@localhost:5432/ruleflow?sslmode=disable'
ensure_kv REDIS_ADDR 'localhost:6379'

download_binary

log "启动基础设施（postgres + redis）..."
docker compose --env-file "$ENV_FILE" -f "$COMPOSE_FILE" up -d

log "启动 RuleFlow..."
install_systemd_service

PORT_VALUE=$(awk -F= '/^PORT=/{print $2}' "$ENV_FILE" | tail -n 1)
if [ -z "${PORT_VALUE:-}" ]; then
  PORT_VALUE=8080
fi

log "RuleFlow 已启动"
log "访问地址: http://localhost:$PORT_VALUE"
log "查看日志: journalctl -u ruleflow -f"
log "停止命令: systemctl stop ruleflow && docker compose --env-file \"$ENV_FILE\" -f \"$COMPOSE_FILE\" down"
