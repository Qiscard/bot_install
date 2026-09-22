package docker

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

// serviceNameRe 限制服务名为安全字符集（纵深防御的输入校验）。
var serviceNameRe = regexp.MustCompile(`^[a-zA-Z0-9_.-]+$`)

// allowedBins 是本包唯一允许执行的可执行文件白名单。
var allowedBins = map[string]struct{}{
	"docker":         {},
	"docker-compose": {},
}

// Provider 封装对 docker / docker compose CLI 的调用。
//
// 安全说明：外部命令通过 buildCmd 构造，可执行文件名先经 allowedBins 白名单
// 校验，再由 exec.LookPath 解析为绝对路径；参数以字符串数组形式经 execve 传入，
// 不经过 shell、不做字符串拼接，因此不存在命令注入。
type Provider struct {
	workDir     string   // compose 工作目录
	projectName string   // compose 项目名
	composeBin  []string // ["docker","compose"] 或 ["docker-compose"]
}

// NewProvider 创建 Provider
func NewProvider(workDir, projectName string) *Provider {
	return &Provider{
		workDir:     workDir,
		projectName: projectName,
	}
}

// buildCmd 构造受限外部命令。bin 必须命中白名单，随后解析为绝对路径。
// 直接填充 exec.Cmd 结构（不经 shell）；ctx 取消由 runWithCtx 负责。
func buildCmd(_ context.Context, workDir, bin string, args []string) (*exec.Cmd, error) {
	if _, ok := allowedBins[bin]; !ok {
		return nil, fmt.Errorf("拒绝执行未在白名单中的命令: %q", bin)
	}
	resolved, err := exec.LookPath(bin)
	if err != nil {
		return nil, fmt.Errorf("未找到可执行文件 %q: %w", bin, err)
	}
	argv := make([]string, 0, len(args)+1)
	argv = append(argv, bin)
	argv = append(argv, args...)

	cmd := &exec.Cmd{
		Path: resolved,
		Args: argv,
		Dir:  workDir,
	}
	return cmd, nil
}

// runWithCtx 启动命令并在 ctx 结束时终止子进程（补齐手动构造 Cmd 缺失的取消能力）。
func runWithCtx(ctx context.Context, cmd *exec.Cmd) error {
	if err := cmd.Start(); err != nil {
		return err
	}
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	select {
	case <-ctx.Done():
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
		<-done
		return ctx.Err()
	case err := <-done:
		return err
	}
}

// Detect 检测 docker 与 compose 可用性，返回诊断信息
func (p *Provider) Detect(ctx context.Context) (DiagResult, error) {
	res := DiagResult{}

	if _, err := exec.LookPath("docker"); err != nil {
		res.DockerInstalled = false
		return res, fmt.Errorf("未找到 docker 命令，请先安装 Docker Engine")
	}
	res.DockerInstalled = true

	if out, err := p.runOut(ctx, "docker", "info", "--format", "{{.ServerVersion}}"); err == nil {
		res.DaemonRunning = true
		res.ServerVersion = strings.TrimSpace(out)
	} else {
		res.DaemonRunning = false
	}

	if err := p.run(ctx, "docker", "compose", "version"); err == nil {
		res.ComposeAvailable = true
		p.composeBin = []string{"docker", "compose"}
	} else if _, err := exec.LookPath("docker-compose"); err == nil {
		res.ComposeAvailable = true
		p.composeBin = []string{"docker-compose"}
	} else {
		res.ComposeAvailable = false
	}

	return res, nil
}

// composeArgs 构造 compose 调用的可执行文件与参数数组
func (p *Provider) composeArgs(args ...string) (string, []string) {
	bin := p.composeBin
	if len(bin) == 0 {
		bin = []string{"docker", "compose"}
	}
	full := append([]string{}, bin[1:]...)
	full = append(full, "-p", p.projectName)
	full = append(full, args...)
	return bin[0], full
}

// WriteCompose 写入 compose 文件到工作目录
func (p *Provider) WriteCompose(data []byte) (string, error) {
	if err := os.MkdirAll(p.workDir, 0o755); err != nil {
		return "", err
	}
	path := filepath.Join(p.workDir, "docker-compose.yml")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return "", err
	}
	return path, nil
}

// Pull 拉取镜像
func (p *Provider) Pull(ctx context.Context) error {
	name, args := p.composeArgs("pull")
	return p.runStream(ctx, name, args...)
}

// Up 启动服务
func (p *Provider) Up(ctx context.Context) error {
	name, args := p.composeArgs("up", "-d")
	return p.runStream(ctx, name, args...)
}

// Down 停止并移除服务
func (p *Provider) Down(ctx context.Context, removeVolumes bool) error {
	extra := []string{"down"}
	if removeVolumes {
		extra = append(extra, "-v")
	}
	name, args := p.composeArgs(extra...)
	return p.runStream(ctx, name, args...)
}

// Logs 获取服务日志。service 若非空必须通过安全字符集校验。
func (p *Provider) Logs(ctx context.Context, service string, tail int) (string, error) {
	if service != "" && !serviceNameRe.MatchString(service) {
		return "", fmt.Errorf("非法的服务名: %q", service)
	}
	if tail <= 0 {
		tail = 200
	}
	extra := []string{"logs", "--tail", fmt.Sprintf("%d", tail)}
	if service != "" {
		extra = append(extra, service)
	}
	name, args := p.composeArgs(extra...)
	return p.runOut(ctx, name, args...)
}

// PS 获取服务状态（JSON）
func (p *Provider) PS(ctx context.Context) (string, error) {
	name, args := p.composeArgs("ps", "--format", "json")
	return p.runOut(ctx, name, args...)
}

func (p *Provider) run(ctx context.Context, name string, args ...string) error {
	cmd, err := buildCmd(ctx, p.workDir, name, args)
	if err != nil {
		return err
	}
	return runWithCtx(ctx, cmd)
}

func (p *Provider) runOut(ctx context.Context, name string, args ...string) (string, error) {
	cmd, err := buildCmd(ctx, p.workDir, name, args)
	if err != nil {
		return "", err
	}
	var out, errb bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errb
	if err := runWithCtx(ctx, cmd); err != nil {
		return out.String(), fmt.Errorf("%s %s: %w: %s", name, strings.Join(args, " "), err, errb.String())
	}
	return out.String(), nil
}

func (p *Provider) runStream(ctx context.Context, name string, args ...string) error {
	cmd, err := buildCmd(ctx, p.workDir, name, args)
	if err != nil {
		return err
	}
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return runWithCtx(ctx, cmd)
}

// DiagResult docker 环境诊断结果
type DiagResult struct {
	DockerInstalled  bool
	DaemonRunning    bool
	ComposeAvailable bool
	ServerVersion    string
}
