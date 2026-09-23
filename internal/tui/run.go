package tui

import (
	"context"
	"fmt"
	"os"

	"github.com/bot-ctl/bot-ctl/internal/adapter"
	"github.com/bot-ctl/bot-ctl/internal/config"
	"github.com/bot-ctl/bot-ctl/internal/credentials"
	"github.com/bot-ctl/bot-ctl/internal/docker"
	"github.com/bot-ctl/bot-ctl/pkg/model"

	tea "github.com/charmbracelet/bubbletea"
)

// Run 启动交互式 TUI 向导；用户确认后在普通终端流式执行部署。
func Run(installDir string) error {
	if installDir == "" {
		installDir = config.DefaultInstallDir
	}
	store := config.NewStore(installDir)
	reg := adapter.NewRegistry()
	m := New(store, reg)

	p := tea.NewProgram(m, tea.WithAltScreen())
	final, err := p.Run()
	if err != nil {
		return err
	}

	fm, ok := final.(Model)
	if !ok || !fm.DeployAfterExit || fm.stack == nil {
		return nil
	}
	return deploy(fm.stack)
}

// deploy 生成 compose 并执行 docker compose pull/up（输出直达终端）。
func deploy(st *model.StackConfig) error {
	data, err := docker.GenerateCompose(st)
	if err != nil {
		return err
	}
	prov := docker.NewProvider(st.InstallDir, st.ProjectName)
	ctx := context.Background()
	if _, err := prov.Detect(ctx); err != nil {
		return err
	}
	path, err := prov.WriteCompose(data)
	if err != nil {
		return err
	}
	fmt.Println("已生成 compose:", path)
	fmt.Println("拉取镜像中（国内网络可能较慢）...")
	if err := prov.Pull(ctx); err != nil {
		return fmt.Errorf("拉取镜像失败: %w", err)
	}
	fmt.Println("启动服务中...")
	if err := prov.Up(ctx); err != nil {
		return fmt.Errorf("启动服务失败: %w", err)
	}
	fmt.Fprintln(os.Stdout, "✓ 部署完成。使用 `bot-ctl logs` 查看日志，`bot-ctl status` 查看状态。")
	return credentials.CaptureInitial(st.InstallDir)
}
