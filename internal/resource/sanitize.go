package resource

import (
	"path/filepath"
	"strings"

	"github.com/bot-ctl/bot-ctl/pkg/model"
)

// sensitiveMetaKeys 是禁止写入 ResourceRef.Metadata 的隐私字段。
// 对应 QQ_RESOURCE_SHARING.md 第 4.2 节“绝不包含”的清单。
var sensitiveMetaKeys = map[string]struct{}{
	"uin": {}, "qq": {}, "account": {}, "user_id": {}, "sender": {},
	"group_id": {}, "group": {}, "discuss_id": {},
	"cookie": {}, "cookies": {}, "token": {}, "skey": {}, "p_skey": {},
	"message": {}, "raw_message": {}, "content": {},
	"host_path": {}, "abs_path": {}, "internal_path": {},
	"url": {}, "download_url": {}, "signature": {}, "sign": {},
}

// SanitizeRef 清洗资源引用，剥离任何隐私/敏感信息，只保留可安全共享的字段。
// 返回的 ResourceRef 仅含相对路径与内容元数据，绝不含 QQ 账号/群/路径/凭证。
func SanitizeRef(ref model.ResourceRef) model.ResourceRef {
	clean := model.ResourceRef{
		ResourceID: ref.ResourceID,
		Kind:       ref.Kind,
		Path:       toRelativeShared(ref.Path, ref.Kind),
		FileName:   sanitizeFileName(ref.FileName),
		Mime:       ref.Mime,
		Size:       ref.Size,
		SHA256:     ref.SHA256,
		Ready:      ref.Ready,
		ExpiresAt:  ref.ExpiresAt,
	}

	// 元数据仅保留白名单外的、明确安全的内容型键
	if len(ref.Metadata) > 0 {
		safe := map[string]string{}
		for k, v := range ref.Metadata {
			if _, bad := sensitiveMetaKeys[strings.ToLower(k)]; bad {
				continue
			}
			if looksSensitive(k) {
				continue
			}
			safe[k] = v
		}
		if len(safe) > 0 {
			clean.Metadata = safe
		}
	}
	return clean
}

// toRelativeShared 将路径规约为共享目录内的相对路径（sub/filename）。
// 若已是相对路径则原样返回；否则仅取文件名并拼上类型子目录。
func toRelativeShared(p string, kind model.ResourceKind) string {
	p = filepath.ToSlash(p)
	if p != "" && !strings.HasPrefix(p, "/") && !strings.Contains(p, ":") {
		// 已是相对路径
		return p
	}
	base := filepath.Base(p)
	return SubdirFor(kind) + "/" + base
}

// sanitizeFileName 去除路径分量，只保留基本文件名
func sanitizeFileName(name string) string {
	name = filepath.Base(filepath.ToSlash(name))
	name = strings.TrimSpace(name)
	if name == "" || name == "." || name == "/" {
		return "file"
	}
	return name
}

// looksSensitive 对键名做启发式判断，拦截含敏感词的自定义键
func looksSensitive(k string) bool {
	lk := strings.ToLower(k)
	for _, needle := range []string{"qq", "uin", "cookie", "token", "secret", "key", "sign", "path", "url", "session", "auth", "sender", "group"} {
		if strings.Contains(lk, needle) {
			return true
		}
	}
	return false
}
