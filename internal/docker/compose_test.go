package docker

import (
	"strings"
	"testing"

	"github.com/bot-ctl/bot-ctl/internal/config"
)

func TestGenerateComposeBasic(t *testing.T) {
	st := config.DefaultStack()
	st.Services["snowluma"].Enabled = true
	st.Services["astrbot"].Enabled = true
	st.Resource.Enabled = true
	st.Resource.Source = "snowluma"

	out, err := GenerateCompose(st)
	if err != nil {
		t.Fatalf("GenerateCompose error: %v", err)
	}
	s := string(out)

	// SnowLuma 特殊要求
	if !strings.Contains(s, "SYS_PTRACE") {
		t.Error("SnowLuma 缺少 SYS_PTRACE")
	}
	if !strings.Contains(s, "seccomp=unconfined") {
		t.Error("SnowLuma 缺少 seccomp=unconfined")
	}
	if !strings.Contains(s, "shm_size") {
		t.Error("SnowLuma 缺少 shm_size")
	}
	// 资源桥与共享卷
	if !strings.Contains(s, "qq-resource-bridge") {
		t.Error("缺少资源桥服务")
	}
	if !strings.Contains(s, "qq-resources") {
		t.Error("缺少共享卷")
	}
	// AstrBot 只读挂载
	if !strings.Contains(s, config.QQResourceMount+":ro") {
		t.Error("AstrBot 未以只读方式挂载共享目录")
	}
	if !strings.Contains(s, "alpine:3.20") || !strings.Contains(s, "/usr/local/bin/bot-ctl:/usr/local/bin/bot-ctl:ro") {
		t.Error("资源桥未复用已安装的 bot-ctl")
	}
	if strings.Contains(s, "botctl/qq-resource-bridge") {
		t.Error("不应拉取尚未发布的资源桥镜像")
	}
	// AstrBot Web 端口暴露
	if !strings.Contains(s, "6185") {
		t.Error("AstrBot Web 端口未暴露")
	}
}

func TestGenerateComposeNoServices(t *testing.T) {
	st := config.DefaultStack()
	if _, err := GenerateCompose(st); err == nil {
		t.Error("空栈应返回错误")
	}
}

func TestGenerateComposeOneBotPortsNotExposed(t *testing.T) {
	st := config.DefaultStack()
	st.Services["snowluma"].Enabled = true
	out, err := GenerateCompose(st)
	if err != nil {
		t.Fatal(err)
	}
	s := string(out)
	// OneBot 端口默认不暴露到宿主机（Exposed=false）
	if strings.Contains(s, "3000:3000") {
		t.Error("OneBot HTTP 端口不应默认发布到宿主机")
	}
}
