package cli

import (
	"archive/tar"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

const (
	githubRepo = "Qiscard/bot_install"
	giteeRepo  = "qiscard/bot_install"
)

// InstallState 是首次安装写入、升级时读取的状态。
type InstallState struct {
	Source    string `json:"source"`
	Version   string `json:"version"`
	Binary    string `json:"binary"`
	UpdatedAt string `json:"updated_at"`
}

func statePath() string {
	if value := os.Getenv("BOTCTL_STATE"); value != "" {
		return value
	}
	return "/etc/bot-ctl/install.json"
}

func loadState() (InstallState, error) {
	var state InstallState
	data, err := os.ReadFile(statePath())
	if err != nil {
		return state, err
	}
	err = json.Unmarshal(data, &state)
	return state, err
}

func saveState(state InstallState) error {
	path := statePath()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	state.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	return writeAtomic(path, data, 0o644)
}

func latestVersion(source, token string) (string, error) {
	var url string
	switch source {
	case "github":
		url = "https://api.github.com/repos/" + githubRepo + "/releases/latest"
	case "gitee":
		if token == "" {
			return "", errors.New("Gitee 升级需要设置 GITEE_TOKEN")
		}
		url = "https://gitee.com/api/v5/repos/" + giteeRepo + "/releases/latest"
	default:
		return "", fmt.Errorf("不支持的升级源: %s", source)
	}
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "bot-ctl")
	if token != "" {
		req.Header.Set("Authorization", "token "+token)
	}
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("查询最新版本失败: HTTP %d", resp.StatusCode)
	}
	var body struct {
		TagName string `json:"tag_name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return "", err
	}
	if body.TagName == "" {
		return "", errors.New("最新版本响应缺少 tag_name")
	}
	return body.TagName, nil
}

func downloadAsset(source, version, name, token, dest string) error {
	var url string
	switch source {
	case "github":
		url = fmt.Sprintf("https://github.com/%s/releases/download/%s/%s", githubRepo, version, name)
	case "gitee":
		url = fmt.Sprintf("https://gitee.com/api/v5/repos/%s/releases/tags/%s/%s", giteeRepo, version, name)
	default:
		return fmt.Errorf("不支持的升级源: %s", source)
	}
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "bot-ctl")
	if token != "" {
		req.Header.Set("Authorization", "token "+token)
	}
	client := &http.Client{Timeout: 5 * time.Minute}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("下载 %s 失败: HTTP %d", name, resp.StatusCode)
	}
	file, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer file.Close()
	_, err = io.Copy(file, resp.Body)
	return err
}

func verifySHA256(filePath, checksumPath string) error {
	data, err := os.ReadFile(checksumPath)
	if err != nil {
		return err
	}
	fields := strings.Fields(string(data))
	if len(fields) == 0 {
		return errors.New("校验文件为空")
	}
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return err
	}
	actual := hex.EncodeToString(hash.Sum(nil))
	if !strings.EqualFold(actual, fields[0]) {
		return errors.New("SHA256 校验失败")
	}
	return nil
}

func extractBinary(archivePath, dest string) error {
	file, err := os.Open(archivePath)
	if err != nil {
		return err
	}
	defer file.Close()
	gzipReader, err := gzip.NewReader(file)
	if err != nil {
		return err
	}
	defer gzipReader.Close()
	reader := tar.NewReader(gzipReader)
	for {
		header, err := reader.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return err
		}
		if filepath.Base(header.Name) != "bot-ctl" || header.Typeflag != tar.TypeReg {
			continue
		}
		out, err := os.OpenFile(dest, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o755)
		if err != nil {
			return err
		}
		if _, err := io.Copy(out, reader); err != nil {
			out.Close()
			return err
		}
		return out.Close()
	}
	return errors.New("安装包中没有 bot-ctl")
}

func replaceBinary(target, updated string) error {
	backup := target + ".bak"
	_ = os.Remove(backup)
	if _, err := os.Stat(target); err == nil {
		if err := os.Rename(target, backup); err != nil {
			return err
		}
	}
	if err := os.Rename(updated, target); err != nil {
		_ = os.Rename(backup, target)
		return err
	}
	return os.Chmod(target, 0o755)
}

func newUpgradeCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "upgrade",
		Short: "检查并升级 bot-ctl",
		RunE: func(cmd *cobra.Command, args []string) error {
			state, err := loadState()
			if err != nil {
				return fmt.Errorf("未找到安装状态，请先运行 install.sh: %w", err)
			}
			if state.Source == "" || state.Binary == "" {
				return errors.New("安装状态不完整")
			}
			token := os.Getenv("GITEE_TOKEN")
			latest, err := latestVersion(state.Source, token)
			if err != nil {
				return err
			}
			if latest == state.Version {
				fmt.Printf("已是最新版本 %s。\n", latest)
				return nil
			}
			fmt.Printf("发现新版本 %s（当前 %s），开始升级。\n", latest, state.Version)
			arch := runtime.GOARCH
			asset := fmt.Sprintf("bot-ctl_linux_%s.tar.gz", arch)
			tmp, err := os.MkdirTemp("", "bot-ctl-upgrade-*")
			if err != nil {
				return err
			}
			defer os.RemoveAll(tmp)
			archive := filepath.Join(tmp, asset)
			checksum := archive + ".sha256"
			if err := downloadAsset(state.Source, latest, asset, token, archive); err != nil {
				return err
			}
			if err := downloadAsset(state.Source, latest, asset+".sha256", token, checksum); err != nil {
				return err
			}
			if err := verifySHA256(archive, checksum); err != nil {
				return err
			}
			updated := filepath.Join(tmp, "bot-ctl")
			if err := extractBinary(archive, updated); err != nil {
				return err
			}
			if err := replaceBinary(state.Binary, updated); err != nil {
				return err
			}
			state.Version = latest
			if err := saveState(state); err != nil {
				return err
			}
			fmt.Printf("已升级到 %s。\n", latest)
			return nil
		},
	}
}

func writeAtomic(path string, data []byte, mode os.FileMode) error {
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, mode); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}
