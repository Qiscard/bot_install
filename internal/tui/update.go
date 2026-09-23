package tui

import (
	"github.com/bot-ctl/bot-ctl/internal/version"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

// Update 处理消息与按键
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		if m.width > 8 {
			fieldWidth := m.width - 8
			if fieldWidth > 48 {
				fieldWidth = 48
			}
			m.dirInput.Width = fieldWidth
			for i := range m.verInputs {
				m.verInputs[i].Width = fieldWidth
			}
		}
		return m, nil

	case doctorMsg:
		m.diagDone = true
		m.diagOK = msg.ok
		m.diagText = msg.text
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)
	}

	// 透传给当前活跃输入框
	return m.updateActiveInput(msg)
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "q":
		if m.step != stepInstallDir && m.step != stepVersions {
			m.quitting = true
			return m, tea.Quit
		}
	case "esc":
		m.quitting = true
		return m, tea.Quit
	}

	switch m.step {
	case stepDoctor:
		return m.keyDoctor(msg)
	case stepSelectFrameworks:
		return m.keySelect(msg)
	case stepInstallDir:
		return m.keyInstallDir(msg)
	case stepVersions:
		return m.keyVersions(msg)
	case stepResource:
		return m.keyResource(msg)
	case stepPorts:
		return m.keyPorts(msg)
	case stepReview:
		return m.keyReview(msg)
	case stepDone:
		if msg.String() == "enter" {
			m.quitting = true
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m Model) keyDoctor(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "r":
		m.diagDone = false
		return m, runDoctorCmd(m.registry)
	case "enter":
		if m.diagDone {
			m.step = stepSelectFrameworks
		}
	}
	return m, nil
}

func (m Model) keySelect(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
	case "down", "j":
		if m.cursor < len(m.items)-1 {
			m.cursor++
		}
	case " ", "x":
		m.items[m.cursor].Checked = !m.items[m.cursor].Checked
	case "enter":
		if m.anyChecked() {
			m.errMsg = ""
			m.dirInput.Focus()
			m.step = stepInstallDir
		} else {
			m.errMsg = "请至少选择一个框架（空格勾选）"
		}
	}
	return m, nil
}

func (m Model) keyInstallDir(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		m.dirInput.Blur()
		m.initVersionInputs()
		m.step = stepVersions
		return m, nil
	}
	var cmd tea.Cmd
	m.dirInput, cmd = m.dirInput.Update(msg)
	return m, cmd
}

// initVersionInputs 为每个选中框架构造版本输入框
func (m *Model) initVersionInputs() {
	if m.versionInit {
		return
	}
	m.verInputs = nil
	for _, it := range m.items {
		if !it.Checked {
			continue
		}
		ti := textinput.New()
		ti.Placeholder = "latest（回车默认最新）"
		ti.CharLimit = 512
		ti.Width = 20
		m.verInputs = append(m.verInputs, ti)
	}
	m.verIndex = 0
	if len(m.verInputs) > 0 {
		m.verInputs[0].Focus()
	}
	m.versionInit = true
}

func (m Model) keyVersions(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		// 保存当前值到对应 item
		m.saveCurrentVersion()
		if m.verIndex < len(m.verInputs)-1 {
			m.verInputs[m.verIndex].Blur()
			m.verIndex++
			m.verInputs[m.verIndex].Focus()
			return m, nil
		}
		// 全部完成
		if m.canShareResources() {
			m.step = stepResource
		} else {
			m.preparePorts()
			m.step = stepPorts
		}
		return m, nil
	case "tab", "down":
		m.saveCurrentVersion()
		if m.verIndex < len(m.verInputs)-1 {
			m.verInputs[m.verIndex].Blur()
			m.verIndex++
			m.verInputs[m.verIndex].Focus()
		}
		return m, nil
	case "shift+tab", "up":
		m.saveCurrentVersion()
		if m.verIndex > 0 {
			m.verInputs[m.verIndex].Blur()
			m.verIndex--
			m.verInputs[m.verIndex].Focus()
		}
		return m, nil
	}
	var cmd tea.Cmd
	m.verInputs[m.verIndex], cmd = m.verInputs[m.verIndex].Update(msg)
	return m, cmd
}

// saveCurrentVersion 将当前版本输入写回对应的选中项
func (m *Model) saveCurrentVersion() {
	val := m.verInputs[m.verIndex].Value()
	idx := -1
	for i := range m.items {
		if m.items[i].Checked {
			idx++
			if idx == m.verIndex {
				m.items[i].Version = val
				return
			}
		}
	}
}

func (m Model) keyResource(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	cands := m.sourceCandidates()
	switch msg.String() {
	case " ", "x":
		m.resourceEnabled = !m.resourceEnabled
	case "up", "k":
		if m.resourceEnabled && len(cands) > 0 && m.resourceSource > 0 {
			m.resourceSource--
		}
	case "down", "j":
		if m.resourceEnabled && len(cands) > 0 && m.resourceSource < len(cands)-1 {
			m.resourceSource++
		}
	case "enter":
		if m.resourceEnabled && !m.hasConsumer() {
			m.errMsg = "已启用资源共享，但未选择消费方 AstrBot；仍可继续，桥会写入共享卷。"
		}
		m.preparePorts()
		m.step = stepPorts
	}
	return m, nil
}

func (m Model) keyPorts(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.customActive {
		switch msg.String() {
		case "enter":
			if err := m.addCustomPort(m.customInput.Value()); err != nil {
				m.errMsg = err.Error()
				return m, nil
			}
			m.customInput.SetValue("")
			m.customInput.Blur()
			m.customActive = false
			m.errMsg = ""
			return m, nil
		case "esc":
			m.customInput.Blur()
			m.customActive = false
			return m, nil
		}
		var cmd tea.Cmd
		m.customInput, cmd = m.customInput.Update(msg)
		return m, cmd
	}

	switch msg.String() {
	case "up", "k":
		if m.portCursor > 0 {
			m.portCursor--
		}
	case "down", "j":
		if m.portCursor < len(m.portItems)-1 {
			m.portCursor++
		}
	case " ":
		if len(m.portItems) > 0 {
			m.portItems[m.portCursor].Exposed = !m.portItems[m.portCursor].Exposed
		}
	case "a":
		m.customActive = true
		m.customInput.Focus()
	case "enter":
		m.prepareReview()
		m.step = stepReview
	}
	return m, nil
}

func (m Model) keyReview(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter", "y":
		if err := m.store.SaveStack(m.stack); err != nil {
			m.errMsg = "保存配置失败: " + err.Error()
			return m, nil
		}
		// 退出 altscreen 后再流式执行 docker pull/up，避免污染 TUI 画面。
		m.DeployAfterExit = true
		m.quitting = true
		return m, tea.Quit
	case "b":
		m.step = stepSelectFrameworks
	}
	return m, nil
}

func (m *Model) prepareReview() {
	m.stack = m.buildStack()
}

func (m Model) updateActiveInput(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch m.step {
	case stepInstallDir:
		var cmd tea.Cmd
		m.dirInput, cmd = m.dirInput.Update(msg)
		return m, cmd
	case stepVersions:
		if len(m.verInputs) > 0 {
			var cmd tea.Cmd
			m.verInputs[m.verIndex], cmd = m.verInputs[m.verIndex].Update(msg)
			return m, cmd
		}
	}
	return m, nil
}

// ---- 辅助 ----

func (m *Model) anyChecked() bool {
	for _, it := range m.items {
		if it.Checked {
			return true
		}
	}
	return false
}

// canShareResources 至少有一个来源框架被选中时才进入资源步骤
func (m *Model) canShareResources() bool {
	return len(m.sourceCandidates()) > 0
}

// versionKindLabel 返回版本规格的可读类型
func versionKindLabel(raw string) string {
	spec := version.Resolve(raw)
	switch spec.Kind {
	case version.SourceLatest:
		return "最新版"
	case version.SourceVersion:
		return "版本 " + spec.Version
	case version.SourceOCIImage:
		return "镜像 " + spec.Image
	case version.SourceGitHubRel:
		return "GitHub Release"
	case version.SourceGiteeRel:
		return "Gitee Release"
	case version.SourceCompose:
		return "Compose 直链"
	case version.SourceTarball:
		return "压缩包直链"
	default:
		return "未识别"
	}
}
