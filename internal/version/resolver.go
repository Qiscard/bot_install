package version

import (
	"regexp"
	"strings"
)

// SourceKind 版本来源类型
type SourceKind string

const (
	SourceLatest      SourceKind = "latest"        // 默认最新版
	SourceVersion     SourceKind = "version"       // 指定版本号 v1.10.0
	SourceGitHubRel   SourceKind = "github_release" // GitHub release 链接
	SourceGiteeRel    SourceKind = "gitee_release"  // Gitee release 链接
	SourceTarball     SourceKind = "tarball"       // 压缩包直链
	SourceCompose     SourceKind = "compose"       // compose 文件直链
	SourceOCIImage    SourceKind = "oci_image"     // OCI 镜像引用
	SourceUnknown     SourceKind = "unknown"
)

// Spec 版本规格
type Spec struct {
	Kind    SourceKind `json:"kind"`
	Raw     string     `json:"raw"`
	Version string     `json:"version"` // 解析出的版本号（若有）
	URL     string     `json:"url"`     // 解析出的 URL（若有）
	Image   string     `json:"image"`   // 解析出的镜像引用（若有）
}

var (
	versionRe   = regexp.MustCompile(`^v?\d+\.\d+(\.\d+)?([-.][0-9A-Za-z.]+)?$`)
	githubRelRe = regexp.MustCompile(`github\.com/[^/]+/[^/]+/releases`)
	giteeRelRe  = regexp.MustCompile(`gitee\.com/[^/]+/[^/]+/releases`)
	tarballRe   = regexp.MustCompile(`\.(tar\.gz|tgz|tar\.xz|zip)($|\?)`)
	composeRe   = regexp.MustCompile(`(docker-)?compose[^/]*\.ya?ml($|\?)`)
	ociImageRe  = regexp.MustCompile(`^([a-z0-9.\-]+(:[0-9]+)?/)?[a-z0-9._\-/]+(:[\w.\-]+)?(@sha256:[a-f0-9]{64})?$`)
)

// Resolve 自动识别用户输入的版本规格
func Resolve(input string) Spec {
	raw := strings.TrimSpace(input)
	if raw == "" || strings.EqualFold(raw, "latest") {
		return Spec{Kind: SourceLatest, Raw: raw, Version: "latest"}
	}

	// 纯版本号
	if versionRe.MatchString(raw) {
		return Spec{Kind: SourceVersion, Raw: raw, Version: normalizeVersion(raw)}
	}

	// URL 类型
	if strings.HasPrefix(raw, "http://") || strings.HasPrefix(raw, "https://") {
		switch {
		case githubRelRe.MatchString(raw):
			return Spec{Kind: SourceGitHubRel, Raw: raw, URL: raw, Version: extractVersionFromURL(raw)}
		case giteeRelRe.MatchString(raw):
			return Spec{Kind: SourceGiteeRel, Raw: raw, URL: raw, Version: extractVersionFromURL(raw)}
		case composeRe.MatchString(raw):
			return Spec{Kind: SourceCompose, Raw: raw, URL: raw}
		case tarballRe.MatchString(raw):
			return Spec{Kind: SourceTarball, Raw: raw, URL: raw}
		default:
			return Spec{Kind: SourceUnknown, Raw: raw, URL: raw}
		}
	}

	// OCI 镜像引用（包含 : 或 / 或 @sha256）
	if strings.Contains(raw, "/") || strings.Contains(raw, ":") || strings.Contains(raw, "@") {
		if ociImageRe.MatchString(raw) {
			return Spec{Kind: SourceOCIImage, Raw: raw, Image: raw}
		}
	}

	return Spec{Kind: SourceUnknown, Raw: raw}
}

func normalizeVersion(v string) string {
	if !strings.HasPrefix(v, "v") {
		return "v" + v
	}
	return v
}

func extractVersionFromURL(url string) string {
	// .../releases/tag/v1.10.0 or .../releases/download/v1.10.0/...
	re := regexp.MustCompile(`/(?:tag|download)/(v?\d+\.\d+(?:\.\d+)?[-.\w]*)`)
	if m := re.FindStringSubmatch(url); len(m) > 1 {
		return m[1]
	}
	return ""
}
