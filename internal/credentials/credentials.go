package credentials

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/bot-ctl/bot-ctl/internal/docker"
)

// Credential 是首次启动时捕获的一条初始凭据。
type Credential struct {
	Service   string `json:"service"`
	Username  string `json:"username"`
	Password  string `json:"password"`
	Source    string `json:"source"`
	CreatedAt string `json:"created_at"`
}

var (
	astrbotUserPattern = regexp.MustCompile(`(?i)initial username:\s*(\S+)`)
	astrbotPassPattern = regexp.MustCompile(`(?i)initial password:\s*(\S+)`)
	snowUserPattern    = regexp.MustCompile(`(?i)user=([^\s]+)`)
	snowPassPattern    = regexp.MustCompile(`(?i)password=([^\s]+)`)
)

// Path 返回凭据文件路径。
func Path(installDir string) string {
	return filepath.Join(installDir, "secrets", "initial-credentials.json")
}

// Load 读取已保存的凭据。
func Load(installDir string) ([]Credential, error) {
	data, err := os.ReadFile(Path(installDir))
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var items []Credential
	if err := json.Unmarshal(data, &items); err != nil {
		return nil, err
	}
	return items, nil
}

// Save 以仅所有者可读写的权限保存凭据。
func Save(installDir string, items []Credential) error {
	dir := filepath.Join(installDir, "secrets")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(items, "", "  ")
	if err != nil {
		return err
	}
	path := Path(installDir)
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return err
	}
	return os.Chmod(path, 0o600)
}

// CaptureFromLogs 从服务日志中提取初始凭据。
func CaptureFromLogs(existing []Credential, service, logs string) []Credential {
	switch service {
	case "astrbot":
		user := firstMatch(astrbotUserPattern, logs)
		pass := firstMatch(astrbotPassPattern, logs)
		if user != "" && pass != "" {
			return upsert(existing, Credential{Service: service, Username: user, Password: pass, Source: "container-log"})
		}
	case "snowluma":
		user := firstMatch(snowUserPattern, logs)
		pass := firstMatch(snowPassPattern, logs)
		if user != "" && pass != "" {
			return upsert(existing, Credential{Service: service, Username: user, Password: pass, Source: "container-log"})
		}
	}
	return existing
}

// CaptureInitial 读取容器首次启动日志并保存新发现的凭据。
func CaptureInitial(installDir string) error {
	items, err := Load(installDir)
	if err != nil {
		return err
	}
	provider := docker.NewProvider(installDir, "bot-ctl")
	changed := false
	for _, service := range []string{"astrbot", "snowluma"} {
		logs, err := provider.Logs(context.Background(), service, 300)
		if err != nil {
			continue
		}
		next := CaptureFromLogs(items, service, logs)
		if len(next) != len(items) {
			changed = true
		}
		items = next
	}
	if !changed {
		return nil
	}
	if err := Save(installDir, items); err != nil {
		return err
	}
	fmt.Printf("初始凭据已保存到 %s\n", Path(installDir))
	fmt.Println("请立即登录并修改密码。之后可运行 bot-ctl password 查看。")
	return nil
}

func upsert(items []Credential, item Credential) []Credential {
	item.CreatedAt = time.Now().UTC().Format(time.RFC3339)
	for i := range items {
		if items[i].Service == item.Service {
			items[i] = item
			return items
		}
	}
	return append(items, item)
}

func firstMatch(pattern *regexp.Regexp, text string) string {
	scanner := bufio.NewScanner(strings.NewReader(text))
	for scanner.Scan() {
		if match := pattern.FindStringSubmatch(scanner.Text()); len(match) == 2 {
			return match[1]
		}
	}
	return ""
}
