package cli

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/bot-ctl/bot-ctl/internal/resource"
	"github.com/bot-ctl/bot-ctl/pkg/model"

	"github.com/spf13/cobra"
)

func newBridgeCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "bridge",
		Short: "QQ 资源共享桥（读取 OneBot 资源写入共享卷）",
	}
	c.AddCommand(newBridgeServeCmd(), newBridgeFetchCmd(), newBridgePruneCmd())
	return c
}

// bridgeOptionsFromEnv 从环境变量读取桥配置（容器内运行）
func bridgeOptionsFromEnv() resource.Options {
	max := int64(100 << 20)
	if v := os.Getenv("BRIDGE_MAX_SIZE"); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil && n > 0 {
			max = n
		}
	}
	retention := 7
	if v := os.Getenv("BRIDGE_RETENTION_DAYS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			retention = n
		}
	}
	out := os.Getenv("BRIDGE_OUTPUT")
	if out == "" {
		out = "/output"
	}
	onebot := os.Getenv("BRIDGE_ONEBOT")
	if onebot == "" {
		onebot = "http://snowluma:3000"
	}
	return resource.Options{
		OneBotURL:     onebot,
		Token:         os.Getenv("BRIDGE_TOKEN"),
		OutputRoot:    out,
		MaxSize:       max,
		RetentionDays: retention,
	}
}

func newBridgeServeCmd() *cobra.Command {
	var interval time.Duration
	var listen string
	c := &cobra.Command{
		Use:   "serve",
		Short: "以守护模式运行：提供 HTTP 接口并定期清理过期资源",
		RunE: func(cmd *cobra.Command, args []string) error {
			br := resource.NewBridge(bridgeOptionsFromEnv())
			if err := br.Init(); err != nil {
				return err
			}
			if v := os.Getenv("BRIDGE_LISTEN"); v != "" {
				listen = v
			}
			fmt.Printf("qq-resource-bridge 已启动，共享目录已初始化，监听 %s。\n", listen)

			ctx, stop := signal.NotifyContext(cmd.Context(), syscall.SIGINT, syscall.SIGTERM)
			defer stop()

			// HTTP 接口
			srv := resource.NewServer(br)
			go func() {
				if err := srv.Serve(ctx, listen); err != nil {
					fmt.Fprintln(os.Stderr, "HTTP 服务错误:", err)
				}
			}()

			// 定期清理
			ticker := time.NewTicker(interval)
			defer ticker.Stop()
			for {
				select {
				case <-ctx.Done():
					fmt.Println("收到退出信号，停止。")
					return nil
				case <-ticker.C:
					if n, err := br.Prune(); err != nil {
						fmt.Fprintln(os.Stderr, "清理失败:", err)
					} else if n > 0 {
						fmt.Printf("已清理 %d 个过期资源。\n", n)
					}
				}
			}
		},
	}
	c.Flags().DurationVar(&interval, "interval", time.Hour, "清理巡检间隔")
	c.Flags().StringVar(&listen, "listen", ":8787", "HTTP 监听地址")
	return c
}

func newBridgeFetchCmd() *cobra.Command {
	var kind, fileID string
	c := &cobra.Command{
		Use:   "fetch",
		Short: "手动拉取一个 QQ 资源到共享目录",
		RunE: func(cmd *cobra.Command, args []string) error {
			if fileID == "" {
				return fmt.Errorf("必须提供 --file-id")
			}
			br := resource.NewBridge(bridgeOptionsFromEnv())
			if err := br.Init(); err != nil {
				return err
			}
			ctx, cancel := context.WithTimeout(cmd.Context(), 2*time.Minute)
			defer cancel()
			ref, err := br.Fetch(ctx, resource.FetchRequest{
				Kind:   model.ResourceKind(kind),
				FileID: fileID,
			})
			if err != nil {
				return err
			}
			fmt.Printf("已共享: %s (%s, %d 字节)\n  路径: %s\n  sha256: %s\n",
				ref.FileName, ref.Kind, ref.Size, ref.Path, ref.SHA256)
			return nil
		},
	}
	c.Flags().StringVar(&kind, "kind", "file", "资源类型 image/video/audio/file")
	c.Flags().StringVar(&fileID, "file-id", "", "OneBot file_id / file")
	return c
}

func newBridgePruneCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "prune",
		Short: "立即清理过期/缺失资源",
		RunE: func(cmd *cobra.Command, args []string) error {
			br := resource.NewBridge(bridgeOptionsFromEnv())
			n, err := br.Prune()
			if err != nil {
				return err
			}
			fmt.Printf("已清理 %d 个资源。\n", n)
			return nil
		},
	}
}
