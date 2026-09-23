# bot-ctl

Linux 专用的 Docker TUI 控制台，用一条命令完成 QQ 机器人框架（SnowLuma / NapCat / AstrBot）的部署与配置，并可将 QQ 下载的资源安全共享给 AstrBot 插件。

> 资源共享的完整协议与安全边界见 [QQ_RESOURCE_SHARING.md](./QQ_RESOURCE_SHARING.md)。

## 特性

- **一条命令、交互式 TUI**：环境自检 → 多选框架 → 选安装目录 → 选版本 → 资源共享 → 确认部署。
- **官方默认优先**：每一项都可直接回车采用官方默认，或输入自定义值。不调控 SnowLuma 的 QQ 数据与 AstrBot 的 `data` 目录。
- **版本自动识别**：默认最新版；支持版本号（`v1.10.0`）或直链（GitHub/Gitee Release、压缩包、compose 文件、OCI 镜像），自动判别类型。
- **多选安装**：SnowLuma、NapCat、AstrBot 自由勾选。
- **QQ 资源共享桥**：仅把下载的图片/视频/语音/文件搬运到独立共享卷 `qq-resources`，AstrBot 以只读方式挂载；登录态、Cookie、数据库、配置等隐私数据绝不外泄。
- **国内网络友好**：安装脚本支持 GitHub / Gitee / 离线三种源，先校验 SHA256 再执行。

## 一键安装

服务器只需 Docker，不需要安装 Go。以下命令会下载对应架构的静态二进制和 SHA256 校验文件，校验通过后安装到 `/usr/local/bin/bot-ctl`。

### GitHub

```bash
set -euo pipefail
cd /tmp
arch="$(uname -m)"
case "$arch" in
  x86_64|amd64) asset="bot-ctl_linux_amd64.tar.gz" ;;
  aarch64|arm64) asset="bot-ctl_linux_arm64.tar.gz" ;;
  *) echo "不支持的架构: $arch" >&2; exit 1 ;;
esac
base="https://github.com/Qiscard/bot_install/releases/download/v0.1.0"
curl -fsSLO "$base/$asset"
curl -fsSLO "$base/$asset.sha256"
sha256sum -c "$asset.sha256"
tar -xzf "$asset"
sudo install -m 0755 bot-ctl /usr/local/bin/bot-ctl
bot-ctl version
```

### Gitee

仓库当前为私有仓库，下载附件前需先设置具有仓库读取权限的 `GITEE_TOKEN`：

```bash
set -euo pipefail
cd /tmp
: "${GITEE_TOKEN:?请先 export GITEE_TOKEN=你的Gitee私人令牌}"
arch="$(uname -m)"
case "$arch" in
  x86_64|amd64) asset="bot-ctl_linux_amd64.tar.gz" ;;
  aarch64|arm64) asset="bot-ctl_linux_arm64.tar.gz" ;;
  *) echo "不支持的架构: $arch" >&2; exit 1 ;;
esac
base="https://gitee.com/api/v5/repos/qiscard/bot_install/releases/tags/v0.1.0"
curl -fsSL -H "Authorization: token ${GITEE_TOKEN}" \
  -o "$asset" "$base/$asset"
curl -fsSL -H "Authorization: token ${GITEE_TOKEN}" \
  -o "$asset.sha256" "$base/$asset.sha256"
sha256sum -c "$asset.sha256"
tar -xzf "$asset"
sudo install -m 0755 bot-ctl /usr/local/bin/bot-ctl
bot-ctl version
```

### 其他安装方式

```bash
BOTCTL_VERSION=v0.1.0 bash install.sh                      # 指定版本
BOTCTL_SOURCE=offline BOTCTL_OFFLINE=./bot-ctl bash install.sh  # 使用本地二进制
```

## 使用

```bash
bot-ctl                 # 启动交互式部署向导（默认）
bot-ctl doctor          # 检查 Docker 环境
bot-ctl up              # 按已保存配置生成 compose 并启动
bot-ctl down            # 停止并移除（--volumes 一并删卷，危险）
bot-ctl logs [service]  # 查看日志（--tail N）
bot-ctl status          # 查看运行状态
bot-ctl version         # 版本信息
```

资源桥（通常由 compose 自动拉起，也可手动）：

```bash
bot-ctl bridge serve                       # 守护模式：HTTP 接口 + 定期清理
bot-ctl bridge fetch --kind image --file-id <id>   # 手动共享一个资源
bot-ctl bridge prune                       # 立即清理过期/缺失资源
```

## 端口与访问

- OneBot 端口（3000/3001）默认**不发布**到宿主机，仅容器网络内 `snowluma:3000` 可达。
- AstrBot Web（6185）默认绑定 `127.0.0.1`；公网访问请自行加反向代理或 SSH 隧道。
- 不做端口映射时，公网 IP 无法直接访问对应服务。

## 从源码构建

```bash
go mod tidy
make build            # 产物在 bin/bot-ctl
make build-linux      # 交叉编译 linux amd64 + arm64 到 dist/
make dist             # 打包 tar.gz 并生成 .sha256
make bridge-image     # 构建 qq-resource-bridge 镜像
make test             # 运行测试
```

## 目录结构

```
cmd/bot-ctl        入口
internal/cli       Cobra 命令
internal/tui       Bubble Tea 向导
internal/docker    compose 生成 + docker CLI 封装
internal/adapter   SnowLuma / NapCat / AstrBot 适配器
internal/resource  QQ 资源共享桥（索引、清洗、HTTP 接口）
internal/config    栈配置与官方默认
internal/version   版本信息与版本规格解析
pkg/model          共享数据模型（含 ResourceRef）
pkg/api            OneBot v11 最小客户端
```

## 平台

仅支持 Linux（amd64 / arm64）。
