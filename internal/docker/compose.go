package docker

import (
	"fmt"
	"sort"

	"github.com/bot-ctl/bot-ctl/internal/config"
	"github.com/bot-ctl/bot-ctl/pkg/model"
	"gopkg.in/yaml.v3"
)

// composeFile 对应 docker-compose.yml 顶层结构
type composeFile struct {
	Services map[string]composeService `yaml:"services"`
	Networks map[string]composeNetwork `yaml:"networks,omitempty"`
	Volumes  map[string]composeVolume  `yaml:"volumes,omitempty"`
}

type composeService struct {
	Image       string            `yaml:"image"`
	ContainerName string          `yaml:"container_name,omitempty"`
	Entrypoint  []string          `yaml:"entrypoint,omitempty"`
	Command     []string          `yaml:"command,omitempty"`
	Restart     string            `yaml:"restart,omitempty"`
	Ports       []string          `yaml:"ports,omitempty"`
	Volumes     []string          `yaml:"volumes,omitempty"`
	Environment map[string]string `yaml:"environment,omitempty"`
	DependsOn   []string          `yaml:"depends_on,omitempty"`
	Networks    []string          `yaml:"networks,omitempty"`
	CapAdd      []string          `yaml:"cap_add,omitempty"`
	SecurityOpt []string          `yaml:"security_opt,omitempty"`
	ShmSize     string            `yaml:"shm_size,omitempty"`
}

type composeNetwork struct {
	Driver string `yaml:"driver,omitempty"`
}

type composeVolume struct {
	Driver string `yaml:"driver,omitempty"`
}

// GenerateCompose 从栈配置生成 docker-compose.yml 内容
func GenerateCompose(st *model.StackConfig) ([]byte, error) {
	cf := composeFile{
		Services: map[string]composeService{},
		Networks: map[string]composeNetwork{config.DefaultNetwork: {Driver: "bridge"}},
		Volumes:  map[string]composeVolume{},
	}

	// 资源共享卷
	if st.Resource != nil && st.Resource.Enabled {
		cf.Volumes[config.QQResourceVol] = composeVolume{Driver: "local"}
	}

	for name, svc := range st.Services {
		if svc == nil || !svc.Enabled {
			continue
		}
		cs := buildService(name, svc, st)
		cf.Services[name] = cs
	}

	// 资源桥服务
	if st.Resource != nil && st.Resource.Enabled {
		cf.Services["qq-resource-bridge"] = buildBridgeService(st)
	}

	if len(cf.Services) == 0 {
		return nil, fmt.Errorf("没有启用任何服务，无法生成 compose")
	}

	return yaml.Marshal(cf)
}

func buildService(name string, svc *model.ServiceConfig, st *model.StackConfig) composeService {
	cs := composeService{
		Image:         imageRef(svc),
		ContainerName: fmt.Sprintf("%s-%s", st.ProjectName, name),
		Restart:       "unless-stopped",
		Environment:   svc.Environment,
		Networks:      []string{config.DefaultNetwork},
	}

	// 端口：仅暴露 Exposed=true 的映射到宿主机
	for _, p := range svc.Ports {
		if !p.Exposed {
			continue
		}
		cs.Ports = append(cs.Ports, formatPort(p))
	}
	sort.Strings(cs.Ports)

	// 卷
	for _, v := range svc.Volumes {
		cs.Volumes = append(cs.Volumes, formatVolume(v))
	}

	// SnowLuma 特殊要求：ptrace + seccomp + shm
	if name == "snowluma" {
		cs.CapAdd = []string{"SYS_PTRACE"}
		cs.SecurityOpt = []string{"seccomp=unconfined"}
		cs.ShmSize = "1gb"
	}

	// AstrBot 挂载只读资源目录（消费方）
	if name == "astrbot" && st.Resource != nil && st.Resource.Enabled {
		cs.Volumes = append(cs.Volumes,
			fmt.Sprintf("%s:%s:ro", config.QQResourceVol, config.QQResourceMount))
	}

	return cs
}

// buildBridgeService 资源桥：读取 OneBot 资源，写入共享卷
func buildBridgeService(st *model.StackConfig) composeService {
	source := st.Resource.Source
	if source == "" {
		source = "snowluma"
	}
	return composeService{
		Image:         "m.daocloud.io/docker.io/library/alpine:3.20",
		ContainerName: fmt.Sprintf("%s-qq-resource-bridge", st.ProjectName),
		Entrypoint:    []string{"/usr/local/bin/bot-ctl"},
		Command:       []string{"bridge", "serve"},
		Restart:       "unless-stopped",
		Networks:      []string{config.DefaultNetwork},
		DependsOn:     []string{source},
		Environment: map[string]string{
			"BRIDGE_SOURCE":   source,
			"BRIDGE_ONEBOT":   fmt.Sprintf("http://%s:%d", source, config.SnowLumaHTTPPort),
			"BRIDGE_OUTPUT":   "/output",
			"BRIDGE_MAX_SIZE": fmt.Sprintf("%d", st.Resource.MaxFileSize),
			"BRIDGE_RETENTION_DAYS": fmt.Sprintf("%d", st.Resource.RetentionDays),
		},
		Volumes: []string{
			"/usr/local/bin/bot-ctl:/usr/local/bin/bot-ctl:ro",
			fmt.Sprintf("%s:/output", config.QQResourceVol),
		},
	}
}

func imageRef(svc *model.ServiceConfig) string {
	tag := svc.Tag
	if tag == "" {
		tag = "latest"
	}
	return fmt.Sprintf("%s:%s", svc.Image, tag)
}

func formatPort(p model.PortMapping) string {
	host := p.Host
	if host == "" {
		host = config.LoopbackBind
	}
	proto := ""
	if p.Protocol == "udp" {
		proto = "/udp"
	}
	return fmt.Sprintf("%s:%d:%d%s", host, p.HostPort, p.Container, proto)
}

func formatVolume(v model.VolumeMapping) string {
	s := fmt.Sprintf("%s:%s", v.Source, v.Target)
	if v.ReadOnly {
		s += ":ro"
	}
	return s
}
