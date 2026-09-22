package version

import (
	"fmt"
	"runtime"
)

// 构建时通过 -ldflags 注入
var (
	Version   = "dev"
	GitCommit = "none"
	BuildDate = "unknown"
)

// Info 版本信息
type Info struct {
	Version   string `json:"version"`
	GitCommit string `json:"git_commit"`
	BuildDate string `json:"build_date"`
	GoVersion string `json:"go_version"`
	Platform  string `json:"platform"`
}

// Get 返回当前版本信息
func Get() Info {
	return Info{
		Version:   Version,
		GitCommit: GitCommit,
		BuildDate: BuildDate,
		GoVersion: runtime.Version(),
		Platform:  fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH),
	}
}

// String 返回可读的版本字符串
func (i Info) String() string {
	return fmt.Sprintf("bot-ctl %s (commit %s, built %s, %s, %s)",
		i.Version, i.GitCommit, i.BuildDate, i.GoVersion, i.Platform)
}
