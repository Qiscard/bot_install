package version

import "testing"

func TestResolve(t *testing.T) {
	cases := []struct {
		in   string
		want SourceKind
	}{
		{"", SourceLatest},
		{"latest", SourceLatest},
		{"v1.10.0", SourceVersion},
		{"1.10.0", SourceVersion},
		{"v2.3", SourceVersion},
		{"https://github.com/foo/bar/releases/tag/v1.2.3", SourceGitHubRel},
		{"https://gitee.com/foo/bar/releases/download/v1.2.3/x.tar.gz", SourceGiteeRel},
		{"https://example.com/app.tar.gz", SourceTarball},
		{"https://example.com/docker-compose.yml", SourceCompose},
		{"ghcr.io/snowluma/snowluma:latest", SourceOCIImage},
		{"soulter/astrbot:v3", SourceOCIImage},
	}
	for _, c := range cases {
		got := Resolve(c.in)
		if got.Kind != c.want {
			t.Errorf("Resolve(%q).Kind = %v, want %v", c.in, got.Kind, c.want)
		}
	}
}

func TestResolveVersionNormalized(t *testing.T) {
	got := Resolve("1.10.0")
	if got.Version != "v1.10.0" {
		t.Errorf("normalize = %q, want v1.10.0", got.Version)
	}
}

func TestExtractVersionFromURL(t *testing.T) {
	got := Resolve("https://github.com/foo/bar/releases/tag/v1.2.3")
	if got.Version != "v1.2.3" {
		t.Errorf("extracted version = %q, want v1.2.3", got.Version)
	}
}
