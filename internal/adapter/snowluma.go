package adapter

import (
	"context"
	"os"
	"path/filepath"

	"github.com/bot-ctl/bot-ctl/internal/config"
	"github.com/bot-ctl/bot-ctl/pkg/api"
	"github.com/bot-ctl/bot-ctl/pkg/model"
)

// SnowLuma 适配器
type SnowLuma struct{}

// NewSnowLuma 创建 SnowLuma 适配器
func NewSnowLuma() *SnowLuma { return &SnowLuma{} }

func (s *SnowLuma) Type() model.FrameworkType { return model.FrameworkSnowLuma }
func (s *SnowLuma) DisplayName() string       { return "SnowLuma (OneBot v11)" }

func (s *SnowLuma) DefaultService() *model.ServiceConfig {
	return config.DefaultSnowLuma()
}

func (s *SnowLuma) Capabilities() Capabilities {
	return Capabilities{
		OneBotV11:     true,
		ProvidesFiles: true,
		ConsumesFiles: false,
		NeedsPtrace:   true,
	}
}

// DetectStartup 在安装目录下识别 SnowLuma 的 compose 入口
func (s *SnowLuma) DetectStartup(installDir string) (StartupInfo, error) {
	return detectCompose(installDir, []string{
		"snowluma", "SnowLuma",
	})
}

// HealthCheck 通过 OneBot get_status 探活
func (s *SnowLuma) HealthCheck(ctx context.Context, endpoint string) error {
	return api.NewOneBotClient(endpoint, "").Ping(ctx)
}

// detectCompose 尝试在 installDir 及常见子目录寻找 compose 文件
func detectCompose(installDir string, hints []string) (StartupInfo, error) {
	candidates := []string{
		filepath.Join(installDir, "docker-compose.yml"),
		filepath.Join(installDir, "docker-compose.yaml"),
		filepath.Join(installDir, "compose.yml"),
		filepath.Join(installDir, "compose.yaml"),
	}
	for _, h := range hints {
		candidates = append(candidates,
			filepath.Join(installDir, h, "docker-compose.yml"),
			filepath.Join(installDir, h, "docker-compose.yaml"),
		)
	}
	for _, c := range candidates {
		if fi, err := os.Stat(c); err == nil && !fi.IsDir() {
			return StartupInfo{
				Entry:       c,
				ComposeFile: c,
				Found:       true,
			}, nil
		}
	}
	return StartupInfo{Found: false}, nil
}
