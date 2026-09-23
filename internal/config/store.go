package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/bot-ctl/bot-ctl/pkg/model"
	"gopkg.in/yaml.v3"
)

const (
	stackFileName = "stack.yaml"
	stateFileName = "state.json"
)

// Store 负责栈配置与状态的持久化
type Store struct {
	baseDir string
}

// NewStore 创建配置存储，baseDir 通常为安装目录
func NewStore(baseDir string) *Store {
	return &Store{baseDir: baseDir}
}

func (s *Store) stackPath() string { return filepath.Join(s.baseDir, stackFileName) }
func (s *Store) statePath() string { return filepath.Join(s.baseDir, stateFileName) }

// EnsureDir 确保基础目录存在
func (s *Store) EnsureDir() error {
	return os.MkdirAll(s.baseDir, 0o755)
}

// LoadStack 加载栈配置；不存在时返回默认栈
func (s *Store) LoadStack() (*model.StackConfig, error) {
	data, err := os.ReadFile(s.stackPath())
	if errors.Is(err, os.ErrNotExist) {
		st := DefaultStack()
		st.InstallDir = s.baseDir
		return st, nil
	}
	if err != nil {
		return nil, fmt.Errorf("读取 stack 配置失败: %w", err)
	}
	var st model.StackConfig
	if err := yaml.Unmarshal(data, &st); err != nil {
		return nil, fmt.Errorf("解析 stack 配置失败: %w", err)
	}
	normalizeStack(&st)
	return &st, nil
}

// normalizeStack 将旧版本保存的镜像迁移到当前默认可拉取镜像。
func normalizeStack(st *model.StackConfig) {
	defaults := map[string]string{
		"snowluma": SnowLumaImage,
		"napcat":   NapCatImage,
		"astrbot":  AstrBotImage,
	}
	legacy := map[string]map[string]struct{}{
		"snowluma": {"ghcr.io/snowluma/snowluma": {}},
		"napcat":   {"mlikiowa/napcat-docker": {}},
		"astrbot":  {"soulter/astrbot": {}},
	}
	for name, svc := range st.Services {
		if svc == nil {
			continue
		}
		if _, ok := legacy[name][svc.Image]; ok {
			svc.Image = defaults[name]
		}
	}
}

// SaveStack 保存栈配置
func (s *Store) SaveStack(st *model.StackConfig) error {
	if err := s.EnsureDir(); err != nil {
		return err
	}
	now := time.Now()
	if st.CreatedAt.IsZero() {
		st.CreatedAt = now
	}
	st.UpdatedAt = now
	data, err := yaml.Marshal(st)
	if err != nil {
		return fmt.Errorf("序列化 stack 配置失败: %w", err)
	}
	if err := writeFileAtomic(s.stackPath(), data, 0o644); err != nil {
		return fmt.Errorf("写入 stack 配置失败: %w", err)
	}
	return nil
}

// LoadState 加载实例状态
func (s *Store) LoadState() (*model.InstanceState, error) {
	data, err := os.ReadFile(s.statePath())
	if errors.Is(err, os.ErrNotExist) {
		return &model.InstanceState{ProjectName: DefaultProjectName, InstallDir: s.baseDir}, nil
	}
	if err != nil {
		return nil, err
	}
	var st model.InstanceState
	if err := json.Unmarshal(data, &st); err != nil {
		return nil, err
	}
	return &st, nil
}

// SaveState 保存实例状态（JSON，与 state.json 文件名及模型标签一致）
func (s *Store) SaveState(st *model.InstanceState) error {
	if err := s.EnsureDir(); err != nil {
		return err
	}
	st.UpdatedAt = time.Now()
	data, err := json.MarshalIndent(st, "", "  ")
	if err != nil {
		return err
	}
	return writeFileAtomic(s.statePath(), data, 0o644)
}

// writeFileAtomic 原子写入，避免中断损坏
func writeFileAtomic(path string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".tmp-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Chmod(tmpName, perm); err != nil {
		return err
	}
	return os.Rename(tmpName, path)
}
