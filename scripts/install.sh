#!/usr/bin/env bash
# bot-ctl 一键安装器（Linux 专用）
# 下载静态二进制、校验 SHA256，并用简易 TUI 写入初始 stack.yaml。
set -euo pipefail

REPO_GITHUB="Qiscard/bot_install"
REPO_GITEE="qiscard/bot_install"
VERSION="${BOTCTL_VERSION:-v0.1.5}"
SOURCE="${BOTCTL_SOURCE:-}"
INSTALL_BIN="${BOTCTL_BIN_DIR:-/usr/local/bin}"
BIN_NAME="bot-ctl"

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

fetch_release() {
  local tmp asset base
  tmp="$(mktemp -d)"
  trap 'rm -rf "${tmp:-}"' RETURN
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

write_state() {
  local state_dir="${BOTCTL_STATE_DIR:-/etc/bot-ctl}"
  local state_file="$state_dir/install.json"
  local content
  content="$(cat <<EOF
{
  "source": "${SOURCE}",
  "version": "${VERSION}",
  "binary": "${INSTALL_BIN}/${BIN_NAME}",
  "updated_at": "$(date -u +%Y-%m-%dT%H:%M:%SZ)"
}
EOF
)"
  if [ -w "$state_dir" ] || mkdir -p "$state_dir" 2>/dev/null; then
    printf '%s\n' "$content" > "$state_file"
  else
    sudo mkdir -p "$state_dir"
    printf '%s\n' "$content" | sudo tee "$state_file" >/dev/null
  fi
  info "升级状态已写入 $state_file"
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

main() {
  choose_source
  fetch_release
  write_state
  cat <<EOF

安装完成。

下一步：
  bot-ctl doctor
  bot-ctl

EOF
}

main "$@"
