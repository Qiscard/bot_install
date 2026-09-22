package resource

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"mime"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/bot-ctl/bot-ctl/pkg/api"
	"github.com/bot-ctl/bot-ctl/pkg/model"
	"github.com/google/uuid"
)

// Bridge 从 OneBot 来源拉取 QQ 资源，落地到共享目录并登记索引。
// 只搬运“下载的资源文件”（图片/视频/语音/文件），绝不搬运登录态、
// 数据库、配置等隐私数据。
type Bridge struct {
	client   *api.OneBotClient
	store    *Store
	maxSize  int64
	allowed  map[model.ResourceKind]struct{}
	retention int
}

// Options 桥配置
type Options struct {
	OneBotURL     string
	Token         string
	OutputRoot    string
	MaxSize       int64
	AllowedKinds  []model.ResourceKind
	RetentionDays int
}

// NewBridge 创建资源桥
func NewBridge(opts Options) *Bridge {
	allowed := map[model.ResourceKind]struct{}{}
	if len(opts.AllowedKinds) == 0 {
		opts.AllowedKinds = []model.ResourceKind{
			model.ResourceKindImage, model.ResourceKindVideo,
			model.ResourceKindAudio, model.ResourceKindFile,
		}
	}
	for _, k := range opts.AllowedKinds {
		allowed[k] = struct{}{}
	}
	if opts.MaxSize <= 0 {
		opts.MaxSize = 100 << 20
	}
	return &Bridge{
		client:    api.NewOneBotClient(opts.OneBotURL, opts.Token),
		store:     NewStore(opts.OutputRoot),
		maxSize:   opts.MaxSize,
		allowed:   allowed,
		retention: opts.RetentionDays,
	}
}

// Init 初始化共享目录结构
func (b *Bridge) Init() error {
	return b.store.EnsureLayout()
}

// FetchRequest 一次资源搬运请求
type FetchRequest struct {
	Kind   model.ResourceKind
	FileID string // OneBot file_id / file
}

// Fetch 拉取单个资源到共享目录，返回清洗后的 ResourceRef。
func (b *Bridge) Fetch(ctx context.Context, req FetchRequest) (*model.ResourceRef, error) {
	if _, ok := b.allowed[req.Kind]; !ok {
		return nil, fmt.Errorf("资源类型 %q 不在允许清单内", req.Kind)
	}

	var fi *api.FileInfo
	var err error
	switch req.Kind {
	case model.ResourceKindImage:
		fi, err = b.client.GetImage(ctx, req.FileID)
	case model.ResourceKindAudio:
		fi, err = b.client.GetRecord(ctx, req.FileID, "mp3")
	default:
		fi, err = b.client.GetFile(ctx, req.FileID)
	}
	if err != nil {
		return nil, err
	}
	if fi.File == "" {
		return nil, fmt.Errorf("来源未返回可用的文件路径")
	}

	// 打开来源文件（来源容器内路径，通过共享挂载或本地路径可达）
	src, err := os.Open(fi.File)
	if err != nil {
		return nil, fmt.Errorf("打开来源文件失败: %w", err)
	}
	defer src.Close()

	if declared := parseSize(fi.FileSize); declared > 0 && declared > b.maxSize {
		return nil, fmt.Errorf("文件大小 %d 超过上限 %d", declared, b.maxSize)
	}

	// 目标路径
	sub := SubdirFor(req.Kind)
	fileName := sanitizeFileName(pickName(fi))
	resourceID := uuid.NewString()
	destName := resourceID + extOf(fileName)
	destRel := sub + "/" + destName
	destAbs := filepath.Join(b.store.Root(), sub, destName)

	// 拷贝 + 计算 sha256 + 限流
	written, sum, err := copyWithHash(destAbs, src, b.maxSize)
	if err != nil {
		_ = os.Remove(destAbs)
		return nil, err
	}

	ref := model.ResourceRef{
		ResourceID: resourceID,
		Kind:       req.Kind,
		Path:       destRel,
		FileName:   fileName,
		Mime:       detectMime(fileName),
		Size:       written,
		SHA256:     sum,
		Ready:      true,
	}
	clean := SanitizeRef(ref)
	if err := b.store.Upsert(clean); err != nil {
		return nil, err
	}
	return &clean, nil
}

// Prune 清理过期资源
func (b *Bridge) Prune() (int, error) {
	return b.store.Prune(b.retention)
}

func copyWithHash(dest string, src io.Reader, limit int64) (int64, string, error) {
	out, err := os.OpenFile(dest, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return 0, "", err
	}
	defer out.Close()

	h := sha256.New()
	// 多写 1 字节用于探测是否超限
	limited := io.LimitReader(src, limit+1)
	mw := io.MultiWriter(out, h)
	n, err := io.Copy(mw, limited)
	if err != nil {
		return n, "", err
	}
	if n > limit {
		return n, "", fmt.Errorf("文件超过大小上限 %d 字节", limit)
	}
	return n, hex.EncodeToString(h.Sum(nil)), nil
}

func pickName(fi *api.FileInfo) string {
	if fi.FileName != "" {
		return fi.FileName
	}
	return filepath.Base(fi.File)
}

func extOf(name string) string {
	return filepath.Ext(name)
}

func detectMime(name string) string {
	if m := mime.TypeByExtension(extOf(name)); m != "" {
		return m
	}
	return "application/octet-stream"
}

func parseSize(s string) int64 {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0
	}
	n, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0
	}
	return n
}
