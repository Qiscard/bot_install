package adapter

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/bot-ctl/bot-ctl/internal/config"
	"github.com/bot-ctl/bot-ctl/pkg/model"
)

// AstrBot 适配器（资源消费方）
type AstrBot struct{}

// NewAstrBot 创建 AstrBot 适配器
func NewAstrBot() *AstrBot { return &AstrBot{} }

func (a *AstrBot) Type() model.FrameworkType { return model.FrameworkAstrBot }
func (a *AstrBot) DisplayName() string       { return "AstrBot" }

func (a *AstrBot) DefaultService() *model.ServiceConfig {
	return config.DefaultAstrBot()
}

func (a *AstrBot) Capabilities() Capabilities {
	return Capabilities{
		OneBotV11:     false,
		ProvidesFiles: false,
		ConsumesFiles: true,
		NeedsPtrace:   false,
	}
}

// DetectStartup 识别 AstrBot 的 compose 入口与 data 目录
func (a *AstrBot) DetectStartup(installDir string) (StartupInfo, error) {
	info, err := detectCompose(installDir, []string{"astrbot", "AstrBot"})
	if err != nil {
		return info, err
	}
	// AstrBot 官方数据目录 /AstrBot/data，不做调控，仅记录
	info.DataDirs = []string{config.AstrBotDataDir}
	return info, nil
}

// HealthCheck 通过 Web 管理端口探活
func (a *AstrBot) HealthCheck(ctx context.Context, endpoint string) error {
	client := &http.Client{Timeout: 5 * time.Second}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return err
	}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("AstrBot 健康检查失败: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 500 {
		return fmt.Errorf("AstrBot 返回 HTTP %d", resp.StatusCode)
	}
	return nil
}
