#!/bin/sh

set -eu

GITHUB_REPO="ablate-ai/RuleFlow"
REPO_URL=${RULEFLOW_REPO_URL:-https://github.com/${GITHUB_REPO}.git}
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
BIN_PATH="$ROOT_DIR/ruleflow"

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

detect_arch() {
  arch=$(uname -m)
  case "$arch" in
    x86_64)  printf 'amd64' ;;
    aarch64|arm64) printf 'arm64' ;;
    *)
      log "不支持的架构: $arch"
      exit 1
      ;;
  esac
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
ensure_kv DATABASE_URL 'postgresql://ruleflow:password@localhost:5432/ruleflow?sslmode=disable'
ensure_kv REDIS_ADDR 'localhost:6379'

# 下载二进制（如果不存在或用户要求更新）
if [ ! -f "$BIN_PATH" ]; then
  download_binary
fi

log "启动基础设施（postgres + redis）..."
docker compose --env-file "$ENV_FILE" -f "$COMPOSE_FILE" up -d

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

log "启动 RuleFlow..."
install_systemd_service
log "RuleFlow 已注册为 systemd 服务"
log "查看日志: journalctl -u ruleflow -f"
log "停止命令: systemctl stop ruleflow && docker compose --env-file \"$ENV_FILE\" -f \"$COMPOSE_FILE\" down"

PORT_VALUE=$(awk -F= '/^PORT=/{print $2}' "$ENV_FILE" | tail -n 1)
if [ -z "${PORT_VALUE:-}" ]; then
  PORT_VALUE=8080
fi

log "访问地址: http://localhost:$PORT_VALUE"
