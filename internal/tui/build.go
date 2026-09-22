package tui

import (
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

	// 资源共享
	st.Resource.Enabled = m.resourceEnabled
	if m.resourceEnabled {
		st.Resource.Source = m.selectedSource()
	}
	return st
}

// applySpecToService 将版本规格落到服务配置
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
