package cli

import (
	"context"
	"fmt"
	"time"

	"github.com/bot-ctl/bot-ctl/internal/config"
	"github.com/bot-ctl/bot-ctl/internal/docker"
	"github.com/bot-ctl/bot-ctl/internal/version"

	"github.com/spf13/cobra"
)

func newDoctorCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "检查 Docker 环境是否就绪",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := context.WithTimeout(cmd.Context(), 15*time.Second)
			defer cancel()
			prov := docker.NewProvider(".", "bot-ctl-probe")
			diag, err := prov.Detect(ctx)
			mark := func(ok bool) string {
				if ok {
					return "✓"
				}
				return "✗"
			}
			fmt.Printf("%s Docker CLI\n", mark(diag.DockerInstalled))
			fmt.Printf("%s Docker 守护进程 %s\n", mark(diag.DaemonRunning), diag.ServerVersion)
			fmt.Printf("%s Docker Compose\n", mark(diag.ComposeAvailable))
			if err != nil {
				fmt.Println(err.Error())
			}
			if !(diag.DockerInstalled && diag.DaemonRunning && diag.ComposeAvailable) {
				return fmt.Errorf("环境未就绪")
			}
			fmt.Println("环境就绪。")
			return nil
		},
	}
}

func newUpCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "up",
		Short: "根据已保存配置生成 compose 并启动服务",
		RunE: func(cmd *cobra.Command, args []string) error {
			store := config.NewStore(resolveDir())
			st, err := store.LoadStack()
			if err != nil {
				return err
			}
			data, err := docker.GenerateCompose(st)
			if err != nil {
				return err
			}
			prov := docker.NewProvider(st.InstallDir, st.ProjectName)
			if _, err := prov.Detect(cmd.Context()); err != nil {
				return err
			}
			path, err := prov.WriteCompose(data)
			if err != nil {
				return err
			}
			fmt.Println("已生成:", path)
			if err := prov.Pull(cmd.Context()); err != nil {
				return err
			}
			return prov.Up(cmd.Context())
		},
	}
}

func newDownCmd() *cobra.Command {
	var removeVolumes bool
	c := &cobra.Command{
		Use:   "down",
		Short: "停止并移除服务",
		RunE: func(cmd *cobra.Command, args []string) error {
			store := config.NewStore(resolveDir())
			st, err := store.LoadStack()
			if err != nil {
				return err
			}
			prov := docker.NewProvider(st.InstallDir, st.ProjectName)
			if _, err := prov.Detect(cmd.Context()); err != nil {
				return err
			}
			return prov.Down(cmd.Context(), removeVolumes)
		},
	}
	c.Flags().BoolVar(&removeVolumes, "volumes", false, "同时删除数据卷（危险）")
	return c
}

func newLogsCmd() *cobra.Command {
	var tail int
	c := &cobra.Command{
		Use:   "logs [service]",
		Short: "查看服务日志",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			store := config.NewStore(resolveDir())
			st, err := store.LoadStack()
			if err != nil {
				return err
			}
			svc := ""
			if len(args) == 1 {
				svc = args[0]
			}
			prov := docker.NewProvider(st.InstallDir, st.ProjectName)
			if _, err := prov.Detect(cmd.Context()); err != nil {
				return err
			}
			out, err := prov.Logs(cmd.Context(), svc, tail)
			if err != nil {
				return err
			}
			fmt.Print(out)
			return nil
		},
	}
	c.Flags().IntVar(&tail, "tail", 200, "显示末尾行数")
	return c
}

func newStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "查看服务运行状态",
		RunE: func(cmd *cobra.Command, args []string) error {
			store := config.NewStore(resolveDir())
			st, err := store.LoadStack()
			if err != nil {
				return err
			}
			prov := docker.NewProvider(st.InstallDir, st.ProjectName)
			if _, err := prov.Detect(cmd.Context()); err != nil {
				return err
			}
			out, err := prov.PS(cmd.Context())
			if err != nil {
				return err
			}
			fmt.Print(out)
			return nil
		},
	}
}

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "显示版本信息",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println(version.Get().String())
		},
	}
}
