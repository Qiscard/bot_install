package config

import (
	"github.com/bot-ctl/bot-ctl/pkg/model"
)

// 官方默认配置常量
const (
	DefaultProjectName = "bot-ctl"
	DefaultInstallDir  = "/opt/bot-ctl"
	DefaultNetwork     = "bot-ctl-net"

	// SnowLuma 官方默认
	SnowLumaImage    = "motricseven7/snowluma"
	SnowLumaTag      = "latest"
	SnowLumaHTTPPort = 3000 // OneBot HTTP
	SnowLumaWSPort   = 3001 // OneBot 正向 WS

	// NapCat 官方默认
	NapCatImage    = "m.daocloud.io/docker.io/mlikiowa/napcat-docker"
	NapCatTag      = "latest"
	NapCatHTTPPort = 3000
	NapCatWSPort   = 3001
	NapCatWebUI    = 6099

	// AstrBot 官方默认
	AstrBotImage    = "m.daocloud.io/docker.io/soulter/astrbot"
	AstrBotTag      = "latest"
	AstrBotWebPort  = 6185
	AstrBotDataDir  = "/AstrBot/data" // 官方默认，不做调控

	// 资源桥
	BridgeImage    = "botctl/qq-resource-bridge"
	BridgeTag      = "latest"
	QQResourceVol  = "qq-resources"
	QQResourceMount = "/opt/astrbot/qq-resources"
)

// LoopbackBind 默认宿主机绑定地址（仅本机可访问）
const LoopbackBind = "127.0.0.1"

// DefaultSnowLuma 返回 SnowLuma 官方默认服务配置
func DefaultSnowLuma() *model.ServiceConfig {
	return &model.ServiceConfig{
		Name:    "snowluma",
		Enabled: false,
		Image:   SnowLumaImage,
		Tag:     SnowLumaTag,
		Network: DefaultNetwork,
		Ports: []model.PortMapping{
			{Host: LoopbackBind, HostPort: SnowLumaHTTPPort, Container: SnowLumaHTTPPort, Protocol: "tcp", Exposed: false},
			{Host: LoopbackBind, HostPort: SnowLumaWSPort, Container: SnowLumaWSPort, Protocol: "tcp", Exposed: false},
		},
		Environment: map[string]string{},
	}
}

// DefaultNapCat 返回 NapCat 官方默认服务配置
func DefaultNapCat() *model.ServiceConfig {
	return &model.ServiceConfig{
		Name:    "napcat",
		Enabled: false,
		Image:   NapCatImage,
		Tag:     NapCatTag,
		Network: DefaultNetwork,
		Ports: []model.PortMapping{
			{Host: LoopbackBind, HostPort: NapCatWebUI, Container: NapCatWebUI, Protocol: "tcp", Exposed: false},
			{Host: LoopbackBind, HostPort: NapCatHTTPPort, Container: NapCatHTTPPort, Protocol: "tcp", Exposed: false},
			{Host: LoopbackBind, HostPort: NapCatWSPort, Container: NapCatWSPort, Protocol: "tcp", Exposed: false},
		},
		Environment: map[string]string{},
	}
}

// DefaultAstrBot 返回 AstrBot 官方默认服务配置
func DefaultAstrBot() *model.ServiceConfig {
	return &model.ServiceConfig{
		Name:    "astrbot",
		Enabled: false,
		Image:   AstrBotImage,
		Tag:     AstrBotTag,
		Network: DefaultNetwork,
		Ports: []model.PortMapping{
			{Host: LoopbackBind, HostPort: AstrBotWebPort, Container: AstrBotWebPort, Protocol: "tcp", Exposed: true},
		},
		Environment: map[string]string{},
	}
}

// DefaultResourceConfig 返回资源共享默认配置
func DefaultResourceConfig() *model.ResourceConfig {
	return &model.ResourceConfig{
		Enabled:       false,
		Source:        "snowluma",
		BridgeImage:   BridgeImage,
		BridgeTag:     BridgeTag,
		MaxFileSize:   100 * 1024 * 1024, // 100MB
		AllowedKinds:  []string{"image", "video", "audio", "file"},
		RetentionDays: 7,
	}
}

// DefaultStack 返回一个包含官方默认的空栈
func DefaultStack() *model.StackConfig {
	return &model.StackConfig{
		Version:     "1",
		ProjectName: DefaultProjectName,
		InstallDir:  DefaultInstallDir,
		Services: map[string]*model.ServiceConfig{
			"snowluma": DefaultSnowLuma(),
			"napcat":   DefaultNapCat(),
			"astrbot":  DefaultAstrBot(),
		},
		Resource: DefaultResourceConfig(),
	}
}
