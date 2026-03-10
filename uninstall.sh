#!/bin/sh

set -eu

DEFAULT_BOOTSTRAP_DIR=${RULEFLOW_DIR:-$HOME/RuleFlow}

bootstrap_uninstall() {
  INSTALL_DIR=$DEFAULT_BOOTSTRAP_DIR

  if [ ! -d "$INSTALL_DIR" ]; then
    log "未找到安装目录: $INSTALL_DIR"
    log "如安装在其他位置，请先设置 RULEFLOW_DIR 后重试。"
    exit 1
  fi

  log "进入安装目录执行卸载: $INSTALL_DIR"
  exec sh "$INSTALL_DIR/uninstall.sh"
}

ROOT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
ENV_FILE="$ROOT_DIR/.env.docker"
COMPOSE_FILE="$ROOT_DIR/deploy/docker-compose.yaml"

log() {
  printf '%s\n' "$1"
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
  bootstrap_uninstall
fi

log "停止并卸载 RuleFlow 容器、网络和数据卷..."
if [ -f "$ENV_FILE" ]; then
  docker compose --env-file "$ENV_FILE" -f "$COMPOSE_FILE" down -v --remove-orphans
else
  docker compose -f "$COMPOSE_FILE" down -v --remove-orphans
fi

if [ -f "$ENV_FILE" ]; then
  rm -f "$ENV_FILE"
  log "已删除环境文件: $ENV_FILE"
fi

log "RuleFlow 已卸载"
