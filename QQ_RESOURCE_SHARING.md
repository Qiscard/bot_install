# QQ 资源共享方案（QQ Resource Sharing）

> 本文档定义 SnowLuma / NapCat 与 AstrBot 之间的 QQ 资源共享协议、目录约定、数据流与安全边界。
> 目标：只共享 QQ 下载的图片、视频、语音和文件，不共享登录态、配置、数据库等隐私数据。

---

## 1. 背景与目标

### 1.1 问题

SnowLuma / NapCat 作为 QQ 客户端运行时，收到消息后会产生资源文件（图片、视频、语音、文件等）。这些资源存放在各自的 QQ 内部数据目录中，例如：

```text
qq-client-config/QQ/nt_qq_<账号哈希>/nt_data/File
qq-client-config/QQ/nt_qq_<账号哈希>/nt_data/Video
qq-client-config/QQ/nt_qq_<账号哈希>/nt_data/Ptt
```

AstrBot 插件无法直接访问这些路径，因为：

- 不同容器的文件系统彼此隔离；
- QQ 内部路径不稳定，可能随版本变化；
- 直接共享 QQ 数据卷会暴露登录态、Cookie、数据库等敏感信息；
- 宿主机路径在容器内不可直接使用。

### 1.2 目标

| 目标 | 说明 |
|------|------|
| 资源可访问 | AstrBot 插件能读取 QQ 下载的图片、视频、语音、文件 |
| 隐私隔离 | 不暴露 QQ 登录态、配置、数据库、缓存、日志 |
| 路径统一 | AstrBot 只依赖固定容器路径，不关心来源是 SnowLuma 还是 NapCat |
| 版本兼容 | 不依赖 QQ 客户端内部目录结构，适配器负责适配 |
| 官方兼容 | 不修改 SnowLuma / NapCat 的官方数据布局 |
| 可选开关 | 用户可选择是否启用资源共享，默认关闭 |

---

## 2. 架构总览

```text
┌─────────────────────────────────────────────────────────────────┐
│  QQ 客户端（SnowLuma / NapCat）                                  │
│    │                                                             │
│    │ 消息事件携带 file_id / url / 文件信息                        │
│    ▼                                                             │
│  OneBot API（/get_file, /get_image, /get_record, ...）            │
│    │                                                             │
│    │ 返回资源元数据（file、url、file_size、file_name 等）         │
│    ▼                                                             │
│  qq-resource-bridge                                              │
│    │  1. 校验资源类型（MIME、扩展名、大小）                        │
│    │  2. 通过流式下载或只读源卷获取内容                           │
│    │  3. 写入临时文件，原子重命名到目标位置                        │
│    │  4. 生成统一资源对象 ResourceRef                             │
│    ▼                                                             │
│  qq-resources volume                                             │
│    │                                                             │
│    │ 统一目录结构：images/ videos/ audio/ files/ index.json       │
│    ▼                                                             │
│  AstrBot 容器                                                    │
│    └── /opt/astrbot/qq-resources（只读挂载）                     │
│         └── 插件读取 / 发送文件                                   │
└─────────────────────────────────────────────────────────────────┘
```

### 2.1 组件职责

| 组件 | 职责 |
|------|------|
| SnowLuma / NapCat | 接收 QQ 消息、提供 OneBot API、维护官方数据卷 |
| qq-resource-bridge | 从 API 拉取资源、写入共享卷、生成索引 |
| qq-resources volume | 独立的资源共享卷，AstrBot 只读挂载 |
| AstrBot | 通过固定路径读取资源，插件使用统一接口 |

---

## 3. 目录与路径约定

### 3.1 共享卷名称

```text
qq-resources
```

### 3.2 容器内路径

| 容器 | 挂载路径 | 权限 |
|------|----------|------|
| qq-resource-bridge | `/output` | 读写 |
| AstrBot | `/opt/astrbot/qq-resources` | 只读 |
| SnowLuma / NapCat | 不挂载（可选） | — |

### 3.3 目录结构

```text
/opt/astrbot/qq-resources/
├── images/                 # 图片（jpg, png, gif, webp, bmp）
├── videos/                 # 视频（mp4, mov, webm, mkv, avi）
├── audio/                  # 语音（amr, mp3, wav, ogg, flac）
├── files/                  # 文件（pdf, zip, 7z, doc, docx, xls, xlsx, txt）
├── index.json              # 资源索引
└── .metadata/              # 桥接器元数据（可选，不挂载给 AstrBot）
```

### 3.4 文件命名规则

文件名由桥接器生成，不使用 QQ 原始文件名，避免冲突和注入：

```text
{resource_id}.{ext}
```

`resource_id` 由桥接器生成（UUID 或内容哈希），例如：

```text
/opt/astrbot/qq-resources/images/2026/09/3f8a2b1c4d5e6f7g.jpg
/opt/astrbot/qq-resources/videos/2026/09/9e1d2c3b4a5f6e7d.mp4
/opt/astrbot/qq-resources/files/2026/09/report-abc123.pdf
```

---

## 4. 资源对象协议（ResourceRef）

### 4.1 统一资源对象

```json
{
  "resource_id": "3f8a2b1c4d5e6f7g",
  "kind": "image",
  "path": "/opt/astrbot/qq-resources/images/2026/09/3f8a2b1c4d5e6f7g.jpg",
  "file_name": "photo.jpg",
  "mime": "image/jpeg",
  "size": 248102,
  "sha256": "a1b2c3d4e5f6...",
  "ready": true,
  "expires_at": null
}
```

### 4.2 字段说明

| 字段 | 类型 | 说明 |
|------|------|------|
| `resource_id` | string | 桥接器生成的资源唯一标识 |
| `kind` | string | 资源类型：`image`、`video`、`audio`、`file` |
| `path` | string | AstrBot 容器内可读路径 |
| `file_name` | string | 原始文件名（可选，用于展示） |
| `mime` | string | MIME 类型 |
| `size` | integer | 文件大小（字节） |
| `sha256` | string | 内容摘要（用于去重和校验） |
| `ready` | boolean | 文件是否已完整写入 |
| `expires_at` | string \| null | 临时资源过期时间（ISO 8601） |

### 4.3 不返回的字段

以下信息**不得**包含在 ResourceRef 中：

- QQ 账号、UIN、账号哈希；
- 群号、群名、发送人昵称；
- 消息正文、消息 ID；
- QQ 内部路径；
- 宿主机路径；
- Cookie、Token、认证信息；
- 下载 URL 中的敏感参数。

---

## 5. API 调用流程

### 5.1 消息事件触发

OneBot 消息事件携带资源标识，例如：

```json
{
  "type": "image",
  "data": {
    "file": "image_id_123",
    "url": "https://..."
  }
}
```

### 5.2 桥接器处理流程

```text
1. 解析消息事件，提取 resource_id
2. 根据 kind 调用对应 API：
   - image  → POST /get_image
   - record → POST /get_record
   - file   → POST /get_file 或 /get_group_file_url / /get_private_file_url
3. 获取资源元数据（file、url、file_size、file_name）
4. 调用 POST /download_file_stream 获取文件内容
5. 写入临时文件：/output/.tmp/<random>
6. 校验 MIME 类型、大小限制、SHA-256
7. 原子重命名到目标目录：/output/images/2026/09/<resource_id>.jpg
8. 更新 index.json
9. 返回 ResourceRef（path 为 AstrBot 容器内路径）
```

### 5.3 流式下载协议

SnowLuma / NapCat 的 `download_file_stream` 返回流式帧：

```text
file_info → file_chunk* → file_complete
```

桥接器必须：

- 在 `file_complete` 前不得声明资源可用；
- 使用临时文件写入，完成后原子重命名；
- 校验 `file_info` 中的 `file_size` 与实际接收大小一致；
- 校验 `sha256`（如果提供）。

---

## 6. Compose 配置示例

```yaml
version: "3.8"

services:
  snowluma:
    image: motricseven7/snowluma:latest
    # ... 官方配置 ...
    networks:
      - bot-network
    # 不挂载 qq-resources，官方数据卷保持独立

  napcat:
    image: mlikiowa/napcat-docker:latest
    # ... 官方配置 ...
    networks:
      - bot-network

  qq-resource-bridge:
    image: bot-ctl/qq-resource-bridge:latest
    networks:
      - bot-network
    volumes:
      - qq-resources:/output
    environment:
      - BRIDGE_SOURCE=napcat          # 或 snowluma
      - BRIDGE_API_URL=http://napcat:3000
      - BRIDGE_OUTPUT_DIR=/output
      - BRIDGE_MAX_FILE_SIZE=104857600   # 100MB
      - BRIDGE_ALLOWED_KINDS=image,video,audio,file
      - BRIDGE_RETENTION_DAYS=30
    depends_on:
      - napcat

  astrbot:
    image: soulter/astrbot:latest
    networks:
      - bot-network
    volumes:
      - ./data:/AstrBot/data
      - qq-resources:/opt/astrbot/qq-resources:ro
    environment:
      - QQ_RESOURCE_DIR=/opt/astrbot/qq-resources

volumes:
  qq-resources:

networks:
  bot-network:
```

---

## 7. 资源索引（index.json）

桥接器维护一个可选的索引文件，方便插件查询：

```json
{
  "schema_version": 1,
  "updated_at": "2026-09-22T10:00:00Z",
  "resources": [
    {
      "resource_id": "3f8a2b1c4d5e6f7g",
      "kind": "image",
      "path": "images/2026/09/3f8a2b1c4d5e6f7g.jpg",
      "mime": "image/jpeg",
      "size": 248102,
      "sha256": "a1b2c3d4e5f6...",
      "created_at": "2026-09-22T09:55:00Z"
    }
  ]
}
```

插件可读取：

```python
import json
from pathlib import Path

index_path = Path("/opt/astrbot/qq-resources/index.json")
if index_path.exists():
    index = json.loads(index_path.read_text())
    for resource in index.get("resources", []):
        print(resource["path"], resource["mime"])
```

---

## 8. AstrBot 插件使用方式

### 8.1 直接读取文件

```python
from pathlib import Path

resource_path = Path("/opt/astrbot/qq-resources/images/2026/09/3f8a2b1c4d5e6f7g.jpg")
if resource_path.exists():
    content = resource_path.read_bytes()
    # 使用 content...
```

### 8.2 使用统一辅助接口（推荐）

`bot-ctl` 可提供 AstrBot 插件辅助库：

```python
from botctl_resources import resolve_resource, list_recent

# 根据消息段解析资源
resource = await resolve_resource(segment)
if resource and resource.ready:
    with open(resource.path, "rb") as f:
        data = f.read()

# 列出最近资源
for r in list_recent(kind="image", limit=10):
    print(r.path, r.mime, r.size)
```

---

## 9. 安全边界

### 9.1 桥接器必须拒绝的内容

| 类型 | 处理方式 |
|------|----------|
| QQ 登录态、Cookie、Token | 不读取、不复制、不暴露 |
| QQ 数据库（nt_db） | 不读取 |
| QQ 配置（mmkv、msf、auth） | 不读取 |
| QQ 日志、缓存、Crashpad | 不读取 |
| 系统表情、头像、缩略图 | 默认排除 |
| 可执行文件、脚本、动态库 | 拒绝 |
| 超过大小限制的文件 | 拒绝并记录 |
| 软链接指向共享根目录外 | 拒绝 |

### 9.2 桥接器权限

```text
- 不挂载 Docker Socket
- 不使用 privileged
- 不访问公网（除非显式配置代理）
- 不修改 SnowLuma / NapCat 官方数据卷
- 只写 qq-resources volume
```

### 9.3 外部 URL 下载

如果桥接器需要直接下载外部 URL，必须：

- 只允许 `http` / `https`；
- 校验目标 host；
- 拒绝 localhost、环回地址、私有地址、保留地址；
- 限制重定向到允许域名；
- 设置超时、大小限制、MIME 校验。

推荐优先使用 NapCat / SnowLuma 的 `download_file_stream`，让来源框架处理外部 URL，桥接器只接收内部流。

---

## 10. 生命周期与清理

### 10.1 资源保留策略

默认保留 30 天，可通过环境变量配置：

```text
BRIDGE_RETENTION_DAYS=30
```

### 10.2 清理规则

- 超过保留时间的资源自动删除；
- 桥接器重启时清理 `.tmp` 目录中的未完成文件；
- `index.json` 同步更新，删除不存在的资源记录。

### 10.3 手动清理

```bash
# 查看资源数量
ls /opt/astrbot/qq-resources/images | wc -l

# 清理 30 天前的资源
find /opt/astrbot/qq-resources -type f -mtime +30 -delete
```

---

## 11. 兼容性与降级

### 11.1 如果 SnowLuma / NapCat 不支持流式下载

降级方案：使用只读源卷 + 目录扫描。

```yaml
qq-resource-bridge:
  volumes:
    - qq-client-config:/source/qq:ro
    - qq-resources:/output
```

桥接器扫描：

```text
/source/qq/QQ/nt_qq_*/nt_data/File
/source/qq/QQ/nt_qq_*/nt_data/Video
/source/qq/QQ/nt_qq_*/nt_data/Ptt
```

但必须：

- 排除 Thumb、ThumbTemp、Cache、Emoji、avatar、log、db 等；
- 根据文件头识别 MIME，不依赖扩展名；
- 跳过符号链接；
- 不读取隐藏文件和配置目录。

### 11.2 如果 AstrBot 无法挂载共享卷

桥接器可以提供 HTTP 服务（仅内部网络）：

```text
http://qq-resource-bridge:8080/resource/<resource_id>
```

AstrBot 通过内部 URL 获取资源。此模式需要桥接器提供认证和访问控制。

---

## 12. 验收标准

| 检查项 | 通过条件 |
|--------|----------|
| 资源隔离 | AstrBot 无法读取 QQ 登录态、配置、数据库 |
| 路径统一 | AstrBot 插件只使用 `/opt/astrbot/qq-resources` |
| 类型过滤 | 只允许图片、视频、语音、文件进入共享目录 |
| 原子写入 | 文件在 `file_complete` 前不可见 |
| 内容校验 | SHA-256 与元数据一致 |
| 索引同步 | `index.json` 与实际文件一致 |
| 保留策略 | 过期资源自动清理 |
| 权限最小化 | 桥接器不使用 root、不挂载 Docker Socket |
| 版本兼容 | SnowLuma 和 NapCat 使用相同资源协议 |

---

## 13. 参考文档

- SnowLuma API: https://snowluma.github.io/zh/docs/api
- NapCat API: https://napneko.github.io/api/4.18.28
- SnowLuma Docker 部署: https://snowluma.github.io/zh/docs/guide/deploy/docker
- AstrBot Docker 部署: https://docs.astrbot.app/deploy/astrbot/docker.html
