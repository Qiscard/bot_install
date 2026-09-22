package resource

import (
	"testing"

	"github.com/bot-ctl/bot-ctl/pkg/model"
)

func TestSanitizeRefStripsPrivacy(t *testing.T) {
	in := model.ResourceRef{
		ResourceID: "abc",
		Kind:       model.ResourceKindImage,
		Path:       "/root/QQ/nt/data/image/xyz.jpg",
		FileName:   "../../etc/passwd",
		Mime:       "image/jpeg",
		Size:       1024,
		SHA256:     "deadbeef",
		Metadata: map[string]string{
			"uin":        "10001",
			"group_id":   "22222",
			"cookie":     "skey=xxx",
			"download_url": "https://gchat.qpic.cn/x?rkey=secret",
			"caption":    "safe caption",
			"width":      "800",
		},
	}
	out := SanitizeRef(in)

	if out.Path != "images/xyz.jpg" {
		t.Errorf("path not normalized to shared relative: %q", out.Path)
	}
	if out.FileName != "passwd" {
		t.Errorf("filename not base-sanitized: %q", out.FileName)
	}
	for _, bad := range []string{"uin", "group_id", "cookie", "download_url"} {
		if _, ok := out.Metadata[bad]; ok {
			t.Errorf("sensitive/heuristic key %q leaked into metadata", bad)
		}
	}
	if v := out.Metadata["caption"]; v != "safe caption" {
		t.Errorf("safe metadata dropped: %q", v)
	}
	if v := out.Metadata["width"]; v != "800" {
		t.Errorf("safe content metadata 'width' dropped: %q", v)
	}
}

func TestSanitizeRefRelativePathKept(t *testing.T) {
	in := model.ResourceRef{
		Kind: model.ResourceKindVideo,
		Path: "videos/clip.mp4",
	}
	out := SanitizeRef(in)
	if out.Path != "videos/clip.mp4" {
		t.Errorf("relative path altered: %q", out.Path)
	}
}
