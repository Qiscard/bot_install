package tui

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/bot-ctl/bot-ctl/internal/config"
	"github.com/bot-ctl/bot-ctl/internal/version"
	"github.com/bot-ctl/bot-ctl/pkg/model"
)

// buildStack 依据向导选择构建 StackConfig（未选中的框架保持 disabled）
func (m *Model) buildStack() *model.StackConfig {
	st := config.DefaultStack()
	st.InstallDir = m.dirInput.Value()
	if st.InstallDir == "" {
		st.InstallDir = config.DefaultInstallDir
	}

	for _, it := range m.items {
		key := string(it.Type)
		svc, ok := st.Services[key]
		if !ok {
			continue
		}
		svc.Enabled = it.Checked
		if !it.Checked {
			continue
		}
		// 解析版本规格 -> tag / image
		spec := version.Resolve(it.Version)
		applySpecToService(svc, spec)
	}

	applyPortChoices(st, m.portItems)

	// 资源共享
	st.Resource.Enabled = m.resourceEnabled
	if m.resourceEnabled {
		st.Resource.Source = m.selectedSource()
	}
	return st
}

// applySpecToService 将版本规格落到服务配置
func (m *Model) preparePorts() {
	if len(m.portItems) > 0 {
		return
	}
	defaults := map[model.FrameworkType][]portItem{
		model.FrameworkAstrBot: {
			{Service: "astrbot", HostPort: config.AstrBotWebPort, Container: config.AstrBotWebPort, Exposed: true},
			{Service: "astrbot", HostPort: config.AstrBotWSPort, Container: config.AstrBotWSPort, Exposed: true},
		},
		model.FrameworkSnowLuma: {
			{Service: "snowluma", HostPort: config.SnowLumaWebPort, Container: config.SnowLumaWebPort, Exposed: true},
			{Service: "snowluma", HostPort: config.SnowLumaVNCPort, Container: config.SnowLumaVNCPort, Exposed: true},
		},
		model.FrameworkNapCat: {
			{Service: "napcat", HostPort: config.NapCatWebUI, Container: config.NapCatWebUI, Exposed: true},
		},
	}
	for _, item := range m.items {
		if item.Checked {
			m.portItems = append(m.portItems, defaults[item.Type]...)
		}
	}
}

func (m *Model) addCustomPort(raw string) error {
	parts := strings.Split(strings.TrimSpace(raw), ":")
	if len(parts) != 3 {
		return fmt.Errorf("格式应为 服务名:宿主机端口:容器端口")
	}
	hostPort, err := strconv.Atoi(parts[1])
	if err != nil || hostPort < 1 || hostPort > 65535 {
		return fmt.Errorf("宿主机端口无效")
	}
	containerPort, err := strconv.Atoi(parts[2])
	if err != nil || containerPort < 1 || containerPort > 65535 {
		return fmt.Errorf("容器端口无效")
	}
	m.portItems = append(m.portItems, portItem{
		Service: parts[0], HostPort: hostPort, Container: containerPort, Exposed: true, Custom: true,
	})
	return nil
}

func applyPortChoices(st *model.StackConfig, items []portItem) {
	for _, item := range items {
		svc := st.Services[item.Service]
		if svc == nil {
			continue
		}
		replaced := false
		for i := range svc.Ports {
			if svc.Ports[i].Container == item.Container && svc.Ports[i].HostPort == item.HostPort {
				svc.Ports[i].Exposed = item.Exposed
				if item.Exposed {
					svc.Ports[i].Host = "0.0.0.0"
				}
				replaced = true
			}
		}
		if !replaced {
			svc.Ports = append(svc.Ports, model.PortMapping{
				Host: "0.0.0.0", HostPort: item.HostPort, Container: item.Container,
				Protocol: "tcp", Exposed: item.Exposed,
			})
		}
	}
}

func applySpecToService(svc *model.ServiceConfig, spec version.Spec) {
	switch spec.Kind {
	case version.SourceLatest:
		svc.Tag = "latest"
	case version.SourceVersion:
		svc.Tag = spec.Version
	case version.SourceOCIImage:
		if img, tag := splitImageTag(spec.Image); img != "" {
			svc.Image = img
			svc.Tag = tag
		}
	case version.SourceGitHubRel, version.SourceGiteeRel:
		if spec.Version != "" {
			svc.Tag = spec.Version
		}
	case version.SourceCompose, version.SourceTarball:
		// 记录到环境变量，供 apply 阶段的高级流程使用
		if svc.Environment == nil {
			svc.Environment = map[string]string{}
		}
		svc.Environment["BOTCTL_SOURCE_URL"] = spec.URL
	}
}

func splitImageTag(ref string) (string, string) {
	// 处理 registry/name:tag，忽略 @sha256 摘要场景的简化
	lastColon := -1
	lastSlash := -1
	for i, c := range ref {
		if c == ':' {
			lastColon = i
		}
		if c == '/' {
			lastSlash = i
		}
	}
	if lastColon > lastSlash && lastColon != -1 {
		return ref[:lastColon], ref[lastColon+1:]
	}
	return ref, "latest"
}

// sourceCandidates 返回可作为 QQ 资源来源的已选框架
func (m *Model) sourceCandidates() []model.FrameworkType {
	var out []model.FrameworkType
	for _, it := range m.items {
		if !it.Checked {
			continue
		}
		if a, ok := m.registry.Get(it.Type); ok && a.Capabilities().ProvidesFiles {
			out = append(out, it.Type)
		}
	}
	return out
}

// selectedSource 返回当前选中的资源来源
func (m *Model) selectedSource() string {
	cands := m.sourceCandidates()
	if len(cands) == 0 {
		return ""
	}
	if m.resourceSource < 0 || m.resourceSource >= len(cands) {
		return string(cands[0])
	}
	return string(cands[m.resourceSource])
}

// hasConsumer 是否选中了资源消费方（AstrBot）
func (m *Model) hasConsumer() bool {
	for _, it := range m.items {
		if !it.Checked {
			continue
		}
		if a, ok := m.registry.Get(it.Type); ok && a.Capabilities().ConsumesFiles {
			return true
		}
	}
	return false
}
