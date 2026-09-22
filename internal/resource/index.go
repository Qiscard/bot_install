package resource

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/bot-ctl/bot-ctl/pkg/model"
)

// 共享目录约定（对应 QQ_RESOURCE_SHARING.md 第 3 节）
const (
	SubImages = "images"
	SubVideos = "videos"
	SubAudio  = "audio"
	SubFiles  = "files"
	IndexName = "index.json"
)

// SubdirFor 返回资源类型对应的子目录
func SubdirFor(kind model.ResourceKind) string {
	switch kind {
	case model.ResourceKindImage:
		return SubImages
	case model.ResourceKindVideo:
		return SubVideos
	case model.ResourceKindAudio:
		return SubAudio
	default:
		return SubFiles
	}
}

// Index 是共享目录的资源索引，供 AstrBot 插件读取
type Index struct {
	Version   string              `json:"version"`
	UpdatedAt time.Time           `json:"updated_at"`
	Resources []model.ResourceRef `json:"resources"`
}

// Store 管理共享目录下的资源与索引
type Store struct {
	root string
	mu   sync.Mutex
}

// NewStore 创建资源存储，root 为共享卷挂载点（桥容器内为 /output）
func NewStore(root string) *Store {
	return &Store{root: root}
}

// EnsureLayout 创建标准子目录结构
func (s *Store) EnsureLayout() error {
	for _, sub := range []string{SubImages, SubVideos, SubAudio, SubFiles} {
		if err := os.MkdirAll(filepath.Join(s.root, sub), 0o755); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) indexPath() string { return filepath.Join(s.root, IndexName) }

// LoadIndex 读取索引；不存在返回空索引
func (s *Store) LoadIndex() (*Index, error) {
	data, err := os.ReadFile(s.indexPath())
	if errors.Is(err, os.ErrNotExist) {
		return &Index{Version: "1", Resources: []model.ResourceRef{}}, nil
	}
	if err != nil {
		return nil, err
	}
	var idx Index
	if err := json.Unmarshal(data, &idx); err != nil {
		return nil, err
	}
	if idx.Resources == nil {
		idx.Resources = []model.ResourceRef{}
	}
	return &idx, nil
}

// saveIndex 原子写入索引
func (s *Store) saveIndex(idx *Index) error {
	idx.Version = "1"
	idx.UpdatedAt = time.Now()
	data, err := json.MarshalIndent(idx, "", "  ")
	if err != nil {
		return err
	}
	tmp := s.indexPath() + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, s.indexPath())
}

// Upsert 写入或更新一条资源引用到索引
func (s *Store) Upsert(ref model.ResourceRef) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	idx, err := s.LoadIndex()
	if err != nil {
		return err
	}
	replaced := false
	for i := range idx.Resources {
		if idx.Resources[i].ResourceID == ref.ResourceID {
			idx.Resources[i] = ref
			replaced = true
			break
		}
	}
	if !replaced {
		idx.Resources = append(idx.Resources, ref)
	}
	return s.saveIndex(idx)
}

// Prune 删除超过保留期或缺失文件的资源条目
func (s *Store) Prune(retentionDays int) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	idx, err := s.LoadIndex()
	if err != nil {
		return 0, err
	}
	cutoff := time.Now().AddDate(0, 0, -retentionDays)
	kept := make([]model.ResourceRef, 0, len(idx.Resources))
	removed := 0
	for _, r := range idx.Resources {
		abs := filepath.Join(s.root, r.Path)
		fi, statErr := os.Stat(abs)
		expired := r.ExpiresAt != nil && r.ExpiresAt.Before(time.Now())
		tooOld := retentionDays > 0 && fi != nil && fi.ModTime().Before(cutoff)
		if statErr != nil || expired || tooOld {
			if statErr == nil {
				_ = os.Remove(abs)
			}
			removed++
			continue
		}
		kept = append(kept, r)
	}
	idx.Resources = kept
	if err := s.saveIndex(idx); err != nil {
		return removed, err
	}
	return removed, nil
}

// Root 返回共享目录根
func (s *Store) Root() string { return s.root }
