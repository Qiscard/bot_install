package tui

import (
	"fmt"
	"strings"
)

// View 渲染当前步骤
func (m Model) View() string {
	if m.quitting {
		return ""
	}
	var b strings.Builder
	b.WriteString(titleStyle.Render(" bot-ctl · QQ 机器人部署控制台 "))
	b.WriteString("\n\n")

	switch m.step {
	case stepDoctor:
		b.WriteString(m.viewDoctor())
	case stepSelectFrameworks:
		b.WriteString(m.viewSelect())
	case stepInstallDir:
		b.WriteString(m.viewInstallDir())
	case stepVersions:
		b.WriteString(m.viewVersions())
	case stepResource:
		b.WriteString(m.viewResource())
	case stepReview:
		b.WriteString(m.viewReview())
	case stepApply:
		b.WriteString(m.viewApply())
	case stepDone:
		b.WriteString(m.viewDone())
	}

	if m.errMsg != "" {
		b.WriteString("\n" + warnStyle.Render("⚠ "+m.errMsg) + "\n")
	}
	return b.String()
}

func (m Model) viewDoctor() string {
	var b strings.Builder
	b.WriteString(fieldLabelStyle.Render("环境检查") + "\n\n")
	if !m.diagDone {
		b.WriteString("  正在检测 Docker 环境...\n")
		return b.String()
	}
	b.WriteString(m.diagText)
	b.WriteString("\n")
	if m.diagOK {
		b.WriteString(okStyle.Render("  环境就绪。") + "\n")
		b.WriteString(helpStyle.Render("回车继续 · r 重新检测 · q 退出"))
	} else {
		b.WriteString(errStyle.Render("  环境未就绪，请先安装/启动 Docker 与 Compose。") + "\n")
		b.WriteString(helpStyle.Render("r 重新检测 · q 退出"))
	}
	return b.String()
}

func (m Model) viewSelect() string {
	var b strings.Builder
	b.WriteString(fieldLabelStyle.Render("选择要安装的框架（多选）") + "\n\n")
	for i, it := range m.items {
		cursor := "  "
		if i == m.cursor {
			cursor = "▸ "
		}
		box := "[ ]"
		if it.Checked {
			box = checkedStyle.Render("[✓]")
		}
		line := fmt.Sprintf("%s%s %s", cursor, box, it.Label)
		if i == m.cursor {
			line = selectedStyle.Render(line)
		}
		b.WriteString(line + "\n")
	}
	b.WriteString(helpStyle.Render("↑/↓ 移动 · 空格 勾选 · 回车 下一步 · q 退出"))
	return b.String()
}

func (m Model) viewInstallDir() string {
	var b strings.Builder
	b.WriteString(fieldLabelStyle.Render("安装目录") + "\n\n")
	b.WriteString(subtitleStyle.Render("  compose 与配置将写入该目录；框架启动程序会被识别并记录。") + "\n\n")
	b.WriteString("  " + m.dirInput.View() + "\n\n")
	b.WriteString(helpStyle.Render("回车确认（留空使用默认） · esc 退出"))
	return b.String()
}

func (m Model) viewVersions() string {
	var b strings.Builder
	b.WriteString(fieldLabelStyle.Render("版本选择") + "\n\n")
	b.WriteString(subtitleStyle.Render("  默认最新版；可输入版本号（v1.10.0）或直链（GitHub/Gitee/压缩包/compose/镜像），自动识别。") + "\n\n")

	idx := 0
	for _, it := range m.items {
		if !it.Checked {
			continue
		}
		marker := "  "
		if idx == m.verIndex {
			marker = "▸ "
		}
		b.WriteString(fmt.Sprintf("%s%s\n", marker, fieldLabelStyle.Render(it.Label)))
		if idx < len(m.verInputs) {
			b.WriteString("    " + m.verInputs[idx].View() + "\n")
			preview := m.verInputs[idx].Value()
			if preview == "" {
				preview = "latest"
			}
			b.WriteString("    " + subtitleStyle.Render("→ "+versionKindLabel(preview)) + "\n")
		}
		b.WriteString("\n")
		idx++
	}
	b.WriteString(helpStyle.Render("回车 下一项/完成 · tab 切换 · esc 退出"))
	return b.String()
}

func (m Model) viewResource() string {
	var b strings.Builder
	b.WriteString(fieldLabelStyle.Render("QQ 资源共享") + "\n\n")
	b.WriteString(subtitleStyle.Render("  仅共享 QQ 下载的资源文件（图片/视频/语音/文件）到 AstrBot，") + "\n")
	b.WriteString(subtitleStyle.Render("  不共享登录态/数据库/配置等隐私数据。") + "\n\n")

	box := "[ ]"
	if m.resourceEnabled {
		box = checkedStyle.Render("[✓]")
	}
	b.WriteString(fmt.Sprintf("  %s 启用资源共享桥 (qq-resource-bridge)\n\n", box))

	if m.resourceEnabled {
		cands := m.sourceCandidates()
		b.WriteString("  选择资源来源：\n")
		for i, c := range cands {
			cursor := "   "
			mark := "○"
			if i == m.resourceSource {
				cursor = " ▸ "
				mark = okStyle.Render("●")
			}
			b.WriteString(fmt.Sprintf("%s%s %s\n", cursor, mark, c))
		}
		if !m.hasConsumer() {
			b.WriteString("\n" + warnStyle.Render("  注意：未选择 AstrBot，桥仍会写入共享卷 qq-resources。") + "\n")
		}
	}
	b.WriteString("\n" + helpStyle.Render("空格 开关 · ↑/↓ 选来源 · 回车 下一步"))
	return b.String()
}

func (m Model) viewReview() string {
	var b strings.Builder
	b.WriteString(fieldLabelStyle.Render("确认部署方案") + "\n\n")
	if m.stack == nil {
		b.WriteString("  （无内容）\n")
		return b.String()
	}
	body := &strings.Builder{}
	fmt.Fprintf(body, "项目名   : %s\n", m.stack.ProjectName)
	fmt.Fprintf(body, "安装目录 : %s\n", m.stack.InstallDir)
	fmt.Fprintf(body, "网络     : %s\n", "bot-ctl-net")
	body.WriteString("\n启用服务:\n")
	for _, it := range m.items {
		if !it.Checked {
			continue
		}
		svc := m.stack.Services[string(it.Type)]
		ver := it.Version
		if ver == "" {
			ver = "latest"
		}
		fmt.Fprintf(body, "  • %-20s %s:%s\n", it.Label, svc.Image, svc.Tag)
		fmt.Fprintf(body, "    版本规格: %s\n", versionKindLabel(ver))
	}
	if m.stack.Resource != nil && m.stack.Resource.Enabled {
		body.WriteString("\nQQ 资源共享: 启用\n")
		fmt.Fprintf(body, "  来源: %s → 共享卷 qq-resources → AstrBot (只读)\n", m.stack.Resource.Source)
	} else {
		body.WriteString("\nQQ 资源共享: 关闭\n")
	}
	b.WriteString(boxStyle.Render(body.String()))
	b.WriteString("\n" + helpStyle.Render("回车/y 生成并启动 · b 返回重选 · q 退出"))
	return b.String()
}

func (m Model) viewApply() string {
	var b strings.Builder
	b.WriteString(fieldLabelStyle.Render("正在部署") + "\n\n")
	b.WriteString("  生成 compose 并执行 docker compose pull / up...\n")
	b.WriteString(subtitleStyle.Render("  拉取镜像可能较慢，请耐心等待。") + "\n")
	return b.String()
}

func (m Model) viewDone() string {
	var b strings.Builder
	if m.errMsg == "" {
		b.WriteString(okStyle.Render("✓ 部署完成") + "\n\n")
		if m.composePath != "" {
			fmt.Fprintf(&b, "  compose 文件: %s\n", m.composePath)
		}
		b.WriteString("  使用 `bot-ctl logs` 查看日志，`bot-ctl status` 查看状态。\n")
	} else {
		b.WriteString(errStyle.Render("✗ 部署失败") + "\n\n")
		fmt.Fprintf(&b, "  %s\n", m.errMsg)
		if m.composePath != "" {
			fmt.Fprintf(&b, "  已生成 compose: %s（可手动排查）\n", m.composePath)
		}
	}
	b.WriteString("\n" + helpStyle.Render("回车 退出"))
	return b.String()
}
