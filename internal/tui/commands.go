package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/bot-ctl/bot-ctl/internal/adapter"
	"github.com/bot-ctl/bot-ctl/internal/docker"

	tea "github.com/charmbracelet/bubbletea"
)

// doctorMsg 环境诊断结果
type doctorMsg struct {
	ok   bool
	text string
}

// runDoctorCmd 执行 docker 环境诊断
func runDoctorCmd(reg *adapter.Registry) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		prov := docker.NewProvider(".", "bot-ctl-probe")
		diag, err := prov.Detect(ctx)

		var b strings.Builder
		line := func(ok bool, label, detail string) {
			mark := "✗"
			if ok {
				mark = "✓"
			}
			b.WriteString(fmt.Sprintf("  %s %s %s\n", mark, label, detail))
		}
		line(diag.DockerInstalled, "Docker CLI", "")
		line(diag.DaemonRunning, "Docker 守护进程", diag.ServerVersion)
		line(diag.ComposeAvailable, "Docker Compose", "")

		ok := diag.DockerInstalled && diag.DaemonRunning && diag.ComposeAvailable
		if err != nil && !ok {
			b.WriteString("\n  " + err.Error() + "\n")
		}
		return doctorMsg{ok: ok, text: b.String()}
	}
}
