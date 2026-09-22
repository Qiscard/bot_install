#!/usr/bin/env bash
# bot-ctl 安装脚本（Linux 专用）
#
# 用法（推荐先下载并校验，再本地执行，不要把下载直接管道给解释器）：
#     curl -fsSLO https://<host>/install.sh
#     curl -fsSLO https://<host>/install.sh.sha256
#     sha256sum -c install.sh.sha256 && bash install.sh
#
# 环境变量：
#   BOTCTL_VERSION   指定版本，默认 latest
#   BOTCTL_SOURCE    下载源：github（默认）/ gitee / offline
#   BOTCTL_MIRROR    自定义镜像基址（offline 或私有分发时使用）
#   BOTCTL_INSTALL   安装路径，默认 /usr/local/bin
#   BOTCTL_OFFLINE   offline 源时的本地二进制路径
set -euo pipefail

REPO_GITHUB="https://github.com/Qiscard/bot_install"
REPO_GITEE="https://gitee.com/Qiscard/bot_install"

VERSION="${BOTCTL_VERSION:-latest}"
SOURCE="${BOTCTL_SOURCE:-github}"
INSTALL_DIR="${BOTCTL_INSTALL:-/usr/local/bin}"
BIN_NAME="bot-ctl"

err()  { echo "[错误] $*" >&2; exit 1; }
info() { echo "[信息] $*"; }

# 仅支持 Linux
[ "$(uname -s)" = "Linux" ] || err "bot-ctl 仅支持 Linux。"

# 架构识别
case "$(uname -m)" in
  x86_64|amd64) ARCH="amd64" ;;
  aarch64|arm64) ARCH="arm64" ;;
  *) err "不支持的架构: $(uname -m)" ;;
esac

TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT

resolve_base() {
  case "$SOURCE" in
    github) echo "$REPO_GITHUB" ;;
    gitee)  echo "$REPO_GITEE" ;;
    offline) echo "${BOTCTL_MIRROR:-}" ;;
    *) err "未知下载源: $SOURCE（github/gitee/offline）" ;;
  esac
}

download() {
  local url="$1" dest="$2"
  info "下载: $url"
  if command -v curl >/dev/null 2>&1; then
    curl -fsSL --retry 3 -o "$dest" "$url"
  elif command -v wget >/dev/null 2>&1; then
    wget -q -O "$dest" "$url"
  else
    err "需要 curl 或 wget。"
  fi
}

# offline 源：直接使用本地二进制
if [ "$SOURCE" = "offline" ] && [ -n "${BOTCTL_OFFLINE:-}" ]; then
  info "使用本地二进制: $BOTCTL_OFFLINE"
  install -m 0755 "$BOTCTL_OFFLINE" "$INSTALL_DIR/$BIN_NAME"
  info "已安装到 $INSTALL_DIR/$BIN_NAME"
  "$INSTALL_DIR/$BIN_NAME" version || true
  exit 0
fi

BASE="$(resolve_base)"
[ -n "$BASE" ] || err "无法解析下载源基址。"

# 构造下载地址（release 资产命名：bot-ctl_linux_${ARCH}.tar.gz）
if [ "$VERSION" = "latest" ]; then
  REL="releases/latest/download"
else
  REL="releases/download/${VERSION}"
fi
ASSET="${BIN_NAME}_linux_${ARCH}.tar.gz"
TARBALL_URL="${BASE}/${REL}/${ASSET}"
CHECKSUM_URL="${TARBALL_URL}.sha256"

download "$TARBALL_URL" "$TMP/$ASSET"

# 校验和验证（存在则强制校验）
if download "$CHECKSUM_URL" "$TMP/$ASSET.sha256" 2>/dev/null; then
  info "校验 SHA256..."
  ( cd "$TMP" && sha256sum -c "$ASSET.sha256" ) || err "校验和不匹配，已中止。"
else
  info "警告：未获取到校验和文件，跳过校验（建议手动核对）。"
fi

tar -xzf "$TMP/$ASSET" -C "$TMP"
[ -f "$TMP/$BIN_NAME" ] || err "解压后未找到 $BIN_NAME。"

install -m 0755 "$TMP/$BIN_NAME" "$INSTALL_DIR/$BIN_NAME"
info "已安装到 $INSTALL_DIR/$BIN_NAME"
"$INSTALL_DIR/$BIN_NAME" version || true

echo
info "运行 'bot-ctl' 启动交互式部署向导。"
