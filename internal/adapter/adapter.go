package adapter

import (
	"context"

	"github.com/bot-ctl/bot-ctl/pkg/model"
)

// Adapter 描述一个可被 bot-ctl 管理的框架服务。
// 每个框架提供其默认服务配置、健康检查与能力声明。
type Adapter interface {
	// Type 返回框架类型
	Type() model.FrameworkType
	// DisplayName 人类可读名称
	DisplayName() string
	// DefaultService 返回官方默认服务配置
	DefaultService() *model.ServiceConfig
	// DetectStartup 根据安装目录识别启动程序/入口，返回描述
	DetectStartup(installDir string) (StartupInfo, error)
	// Capabilities 声明能力
	Capabilities() Capabilities
	// HealthCheck 对运行中的服务做健康检查
	HealthCheck(ctx context.Context, endpoint string) error
}

// StartupInfo 启动程序识别结果
type StartupInfo struct {
	Entry       string   // 入口文件/命令，如 docker-compose.yml
	ComposeFile string   // compose 文件路径（若有）
	DataDirs    []string // 数据目录
	Found       bool
}

// Capabilities 框架能力声明
type Capabilities struct {
	OneBotV11     bool // 是否兼容 OneBot v11
	ProvidesFiles bool // 是否可作为 QQ 资源来源
	ConsumesFiles bool // 是否作为资源消费方（AstrBot）
	NeedsPtrace   bool // 是否需要 SYS_PTRACE
}

// Registry 适配器注册表
type Registry struct {
	adapters map[model.FrameworkType]Adapter
}

// NewRegistry 创建并注册内置适配器
func NewRegistry() *Registry {
	r := &Registry{adapters: map[model.FrameworkType]Adapter{}}
	r.Register(NewSnowLuma())
	r.Register(NewNapCat())
	r.Register(NewAstrBot())
	return r
}

// Register 注册适配器
func (r *Registry) Register(a Adapter) {
	r.adapters[a.Type()] = a
}

// Get 按类型获取适配器
func (r *Registry) Get(t model.FrameworkType) (Adapter, bool) {
	a, ok := r.adapters[t]
	return a, ok
}

// All 返回全部适配器（稳定顺序）
func (r *Registry) All() []Adapter {
	order := []model.FrameworkType{
		model.FrameworkSnowLuma,
		model.FrameworkNapCat,
		model.FrameworkAstrBot,
	}
	out := make([]Adapter, 0, len(order))
	for _, t := range order {
		if a, ok := r.adapters[t]; ok {
			out = append(out, a)
		}
	}
	return out
}
