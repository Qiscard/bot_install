#!/usr/bin/env bash
# bot-ctl 一键安装器（Linux 专用）
# 下载静态二进制、校验 SHA256，并用简易 TUI 写入初始 stack.yaml。
set -euo pipefail

REPO_GITHUB="Qiscard/bot_install"
REPO_GITEE="qiscard/bot_install"
VERSION="${BOTCTL_VERSION:-v0.1.0}"
SOURCE="${BOTCTL_SOURCE:-}"
INSTALL_BIN="${BOTCTL_BIN_DIR:-/usr/local/bin}"
INSTALL_DIR="${BOTCTL_DIR:-/opt/bot-ctl}"
BIN_NAME="bot-ctl"

USE_SNOWLUMA=0
USE_NAPCAT=0
USE_ASTRBOT=0
USE_RESOURCE=0
RESOURCE_SOURCE="snowluma"

err() { printf '[错误] %s\n' "$*" >&2; exit 1; }
info() { printf '[信息] %s\n' "$*"; }

need_cmd() { command -v "$1" >/dev/null 2>&1 || err "缺少命令: $1"; }

[ "$(uname -s)" = "Linux" ] || err "bot-ctl 仅支持 Linux。"
need_cmd tar
need_cmd sha256sum
if command -v curl >/dev/null 2>&1; then
  DOWNLOADER="curl"
elif command -v wget >/dev/null 2>&1; then
  DOWNLOADER="wget"
else
  err "需要 curl 或 wget。"
fi

case "$(uname -m)" in
  x86_64|amd64) ARCH="amd64" ;;
  aarch64|arm64) ARCH="arm64" ;;
  *) err "不支持的架构: $(uname -m)" ;;
esac

download() {
  local url="$1" dest="$2"
  if [ "$DOWNLOADER" = "curl" ]; then
    if [ -n "${AUTH_HEADER:-}" ]; then
      curl -fsSL --retry 3 -H "$AUTH_HEADER" -o "$dest" "$url"
    else
      curl -fsSL --retry 3 -o "$dest" "$url"
    fi
  else
    if [ -n "${AUTH_HEADER:-}" ]; then
      wget -q -O "$dest" --header="$AUTH_HEADER" "$url"
    else
      wget -q -O "$dest" "$url"
    fi
  fi
}

pause() { printf '\n按回车继续...'; read -r _; }

choose_source() {
  [ -n "$SOURCE" ] && return 0
  while true; do
    clear
    cat <<'EOF'
bot-ctl 安装器

选择下载源：
  1) GitHub（公开，推荐）
  2) Gitee（当前仓库为私有，需要 GITEE_TOKEN）
  3) 本地离线包
EOF
    printf '请选择 [1-3，默认 1]: '
    read -r choice
    case "${choice:-1}" in
      1) SOURCE="github"; return 0 ;;
      2) SOURCE="gitee"; return 0 ;;
      3) SOURCE="offline"; return 0 ;;
      *) printf '无效选择\n'; sleep 1 ;;
    esac
  done
}

choose_project() {
  clear
  cat <<EOF
项目配置（直接回车使用默认值）

当前安装目录: $INSTALL_DIR
EOF
  printf '安装目录 [%s]: ' "$INSTALL_DIR"
  read -r value
  [ -n "$value" ] && INSTALL_DIR="$value"

  printf '版本 [%s]: ' "$VERSION"
  read -r value
  [ -n "$value" ] && VERSION="$value"

  printf '安装 SnowLuma? [y/N]: '
  read -r value; [[ "$value" =~ ^[Yy]$ ]] && USE_SNOWLUMA=1
  printf '安装 NapCat? [y/N]: '
  read -r value; [[ "$value" =~ ^[Yy]$ ]] && USE_NAPCAT=1
  printf '安装 AstrBot? [y/N]: '
  read -r value; [[ "$value" =~ ^[Yy]$ ]] && USE_ASTRBOT=1

  if [ "$USE_SNOWLUMA" -eq 1 ] || [ "$USE_NAPCAT" -eq 1 ]; then
    printf '启用 QQ 资源共享桥? [y/N]: '
    read -r value
    if [[ "$value" =~ ^[Yy]$ ]]; then
      USE_RESOURCE=1
      if [ "$USE_SNOWLUMA" -eq 1 ] && [ "$USE_NAPCAT" -eq 1 ]; then
        printf '资源来源 [snowluma/napcat，默认 snowluma]: '
        read -r value
        case "${value:-snowluma}" in
          snowluma|napcat) RESOURCE_SOURCE="$value" ;;
          *) err "资源来源只能是 snowluma 或 napcat" ;;
        esac
      elif [ "$USE_NAPCAT" -eq 1 ]; then
        RESOURCE_SOURCE="napcat"
      fi
    fi
  fi
}

confirm_config() {
  clear
  cat <<EOF
请确认安装配置：

  下载源:     $SOURCE
  版本:       $VERSION
  二进制目录: $INSTALL_BIN
  项目目录:   $INSTALL_DIR
  SnowLuma:   $USE_SNOWLUMA
  NapCat:     $USE_NAPCAT
  AstrBot:    $USE_ASTRBOT
  资源共享:   $USE_RESOURCE
  资源来源:   $RESOURCE_SOURCE

EOF
  printf '确认开始安装? [y/N]: '
  read -r value
  [[ "$value" =~ ^[Yy]$ ]] || err "已取消。"
}

fetch_release() {
  local tmp asset base
  tmp="$(mktemp -d)"
  trap 'rm -rf "$tmp"' RETURN
  asset="${BIN_NAME}_linux_${ARCH}.tar.gz"

  case "$SOURCE" in
    github)
      base="https://github.com/${REPO_GITHUB}/releases/download/${VERSION}"
      ;;
    gitee)
      : "${GITEE_TOKEN:?使用 Gitee 私有仓库时必须设置 GITEE_TOKEN}"
      AUTH_HEADER="Authorization: token ${GITEE_TOKEN}"
      base="https://gitee.com/api/v5/repos/${REPO_GITEE}/releases/tags/${VERSION}"
      ;;
    offline)
      [ -n "${BOTCTL_OFFLINE:-}" ] || err "离线模式需设置 BOTCTL_OFFLINE=/path/to/bot-ctl"
      install_binary "$BOTCTL_OFFLINE"
      return 0
      ;;
    *) err "未知下载源: $SOURCE" ;;
  esac

  info "下载 $asset"
  download "$base/$asset" "$tmp/$asset"
  info "下载校验文件"
  download "$base/$asset.sha256" "$tmp/$asset.sha256"
  (cd "$tmp" && sha256sum -c "$asset.sha256") || err "SHA256 校验失败。"
  tar -xzf "$tmp/$asset" -C "$tmp"
  [ -f "$tmp/$BIN_NAME" ] || err "安装包中没有 bot-ctl。"
  install_binary "$tmp/$BIN_NAME"
}

install_binary() {
  local src="$1"
  if [ -w "$INSTALL_BIN" ]; then
    install -m 0755 "$src" "$INSTALL_BIN/$BIN_NAME"
  else
    sudo install -m 0755 "$src" "$INSTALL_BIN/$BIN_NAME"
  fi
  info "二进制已安装到 $INSTALL_BIN/$BIN_NAME"
}

bool_text() { [ "$1" -eq 1 ] && printf 'true' || printf 'false'; }

write_stack() {
  mkdir -p "$INSTALL_DIR" 2>/dev/null || sudo mkdir -p "$INSTALL_DIR"
  local target="$INSTALL_DIR/stack.yaml"
  local content
  content="$(cat <<EOF
version: "1"
project_name: bot-ctl
install_dir: ${INSTALL_DIR}
services:
  snowluma:
    name: snowluma
    enabled: $(bool_text "$USE_SNOWLUMA")
    image: ghcr.io/snowluma/snowluma
    tag: latest
    network: bot-ctl-net
  napcat:
    name: napcat
    enabled: $(bool_text "$USE_NAPCAT")
    image: mlikiowa/napcat-docker
    tag: latest
    network: bot-ctl-net
  astrbot:
    name: astrbot
    enabled: $(bool_text "$USE_ASTRBOT")
    image: soulter/astrbot
    tag: latest
    network: bot-ctl-net
resource:
  enabled: $(bool_text "$USE_RESOURCE")
  source: ${RESOURCE_SOURCE}
  bridge_image: botctl/qq-resource-bridge
  bridge_tag: latest
  max_file_size: 104857600
  allowed_kinds: [image, video, audio, file]
  retention_days: 7
EOF
)"
  if [ -w "$INSTALL_DIR" ]; then
    printf '%s\n' "$content" > "$target"
  else
    printf '%s\n' "$content" | sudo tee "$target" >/dev/null
  fi
  info "项目配置已写入 $target"
}

main() {
  choose_source
  choose_project
  confirm_config
  fetch_release
  write_stack
  cat <<EOF

安装完成。

下一步：
  bot-ctl doctor
  bot-ctl --dir "$INSTALL_DIR"
  bot-ctl up --dir "$INSTALL_DIR"

EOF
}

main "$@"
