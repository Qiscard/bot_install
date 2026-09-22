package adapter

import (
	"context"

	"github.com/bot-ctl/bot-ctl/internal/config"
	"github.com/bot-ctl/bot-ctl/pkg/api"
	"github.com/bot-ctl/bot-ctl/pkg/model"
)

// NapCat 适配器
type NapCat struct{}

// NewNapCat 创建 NapCat 适配器
func NewNapCat() *NapCat { return &NapCat{} }

func (n *NapCat) Type() model.FrameworkType { return model.FrameworkNapCat }
func (n *NapCat) DisplayName() string       { return "NapCat (OneBot v11)" }

func (n *NapCat) DefaultService() *model.ServiceConfig {
	return config.DefaultNapCat()
}

func (n *NapCat) Capabilities() Capabilities {
	return Capabilities{
		OneBotV11:     true,
		ProvidesFiles: true,
		ConsumesFiles: false,
		NeedsPtrace:   false,
	}
}

// DetectStartup 识别 NapCat 的 compose 入口
func (n *NapCat) DetectStartup(installDir string) (StartupInfo, error) {
	return detectCompose(installDir, []string{"napcat", "NapCat"})
}

// HealthCheck 通过 OneBot get_status 探活
func (n *NapCat) HealthCheck(ctx context.Context, endpoint string) error {
	return api.NewOneBotClient(endpoint, "").Ping(ctx)
}
