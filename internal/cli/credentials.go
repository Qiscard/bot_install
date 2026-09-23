package cli

import (
	"fmt"

	"github.com/bot-ctl/bot-ctl/internal/credentials"
	"github.com/spf13/cobra"
)

func newPasswordCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "password",
		Short: "查看已保存的初始凭据",
		RunE: func(cmd *cobra.Command, args []string) error {
			dir := resolveDir()
			items, err := credentials.Load(dir)
			if err != nil {
				return err
			}
			if len(items) == 0 {
				return fmt.Errorf("尚未保存初始凭据，可查看 %s", credentials.Path(dir))
			}
			fmt.Printf("凭据文件: %s\n", credentials.Path(dir))
			for _, item := range items {
				fmt.Printf("%s  用户名: %s  初始密码: %s\n", item.Service, item.Username, item.Password)
			}
			return nil
		},
	}
}
