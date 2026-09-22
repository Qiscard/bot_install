package cli

import (
	"github.com/bot-ctl/bot-ctl/internal/config"
	"github.com/bot-ctl/bot-ctl/internal/tui"

	"github.com/spf13/cobra"
)

var installDir string

// NewRootCmd 构造根命令；无子命令时启动 TUI 向导
func NewRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "bot-ctl",
		Short: "QQ 机器人（SnowLuma/NapCat/AstrBot）Docker 部署控制台",
		Long: "bot-ctl 是一个 Linux 专用的 Docker TUI 控制台，" +
			"用于部署与配置 QQ 机器人框架，并可将 QQ 下载的资源安全共享给 AstrBot。",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return tui.Run(resolveDir())
		},
	}
	root.PersistentFlags().StringVarP(&installDir, "dir", "d", "", "安装目录（默认 "+config.DefaultInstallDir+"）")

	root.AddCommand(
		newDoctorCmd(),
		newUpCmd(),
		newDownCmd(),
		newLogsCmd(),
		newStatusCmd(),
		newBridgeCmd(),
		newVersionCmd(),
	)
	return root
}

func resolveDir() string {
	if installDir != "" {
		return installDir
	}
	return config.DefaultInstallDir
}
