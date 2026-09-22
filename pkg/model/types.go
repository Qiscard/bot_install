package model

import (
	"time"
)

// FrameworkType 支持的框架类型
type FrameworkType string

const (
	FrameworkSnowLuma FrameworkType = "snowluma"
	FrameworkNapCat   FrameworkType = "napcat"
	FrameworkAstrBot  FrameworkType = "astrbot"
)

// ResourceKind 资源类型
type ResourceKind string

const (
	ResourceKindImage ResourceKind = "image"
	ResourceKindVideo ResourceKind = "video"
	ResourceKindAudio ResourceKind = "audio"
	ResourceKindFile  ResourceKind = "file"
)

// ResourceRef 统一资源引用
type ResourceRef struct {
	ResourceID string            `json:"resource_id"`
	Kind       ResourceKind      `json:"kind"`
	Path       string            `json:"path"`
	FileName   string            `json:"file_name"`
	Mime       string            `json:"mime"`
	Size       int64             `json:"size"`
	SHA256     string            `json:"sha256"`
	Ready      bool              `json:"ready"`
	ExpiresAt  *time.Time        `json:"expires_at,omitempty"`
	Metadata   map[string]string `json:"metadata,omitempty"`
}

// ServiceConfig 服务配置
type ServiceConfig struct {
	Name        string            `yaml:"name" json:"name"`
	Enabled     bool              `yaml:"enabled" json:"enabled"`
	Image       string            `yaml:"image" json:"image"`
	Tag         string            `yaml:"tag" json:"tag"`
	Ports       []PortMapping     `yaml:"ports" json:"ports"`
	Volumes     []VolumeMapping   `yaml:"volumes" json:"volumes"`
	Environment map[string]string `yaml:"environment" json:"environment"`
	DependsOn   []string          `yaml:"depends_on" json:"depends_on"`
	Network     string            `yaml:"network" json:"network"`
}

// PortMapping 端口映射
type PortMapping struct {
	Host      string `yaml:"host" json:"host"`           // 宿主机绑定地址，默认 127.0.0.1
	HostPort  int    `yaml:"host_port" json:"host_port"` // 宿主机端口
	Container int    `yaml:"container" json:"container"` // 容器端口
	Protocol  string `yaml:"protocol" json:"protocol"`   // tcp/udp
	Exposed   bool   `yaml:"exposed" json:"exposed"`     // 是否暴露到宿主机
}

// VolumeMapping 卷映射
type VolumeMapping struct {
	Source    string `yaml:"source" json:"source"`       // 宿主机路径或卷名
	Target    string `yaml:"target" json:"target"`       // 容器内路径
	ReadOnly  bool   `yaml:"read_only" json:"read_only"` // 是否只读
	Type      string `yaml:"type" json:"type"`           // bind / volume
}

// StackConfig 部署栈配置
type StackConfig struct {
	Version     string                    `yaml:"version" json:"version"`
	ProjectName string                    `yaml:"project_name" json:"project_name"`
	InstallDir  string                    `yaml:"install_dir" json:"install_dir"`
	Services    map[string]*ServiceConfig `yaml:"services" json:"services"`
	Resource    *ResourceConfig           `yaml:"resource" json:"resource"`
	CreatedAt   time.Time                 `yaml:"created_at" json:"created_at"`
	UpdatedAt   time.Time                 `yaml:"updated_at" json:"updated_at"`
}

// ResourceConfig 资源共享配置
type ResourceConfig struct {
	Enabled      bool     `yaml:"enabled" json:"enabled"`
	Source       string   `yaml:"source" json:"source"` // snowluma / napcat
	BridgeImage  string   `yaml:"bridge_image" json:"bridge_image"`
	BridgeTag    string   `yaml:"bridge_tag" json:"bridge_tag"`
	MaxFileSize  int64    `yaml:"max_file_size" json:"max_file_size"`
	AllowedKinds []string `yaml:"allowed_kinds" json:"allowed_kinds"`
	RetentionDays int     `yaml:"retention_days" json:"retention_days"`
}

// InstanceState 实例状态
type InstanceState struct {
	ProjectName string            `json:"project_name"`
	InstallDir  string            `json:"install_dir"`
	Services    []ServiceState    `json:"services"`
	Resource    *ResourceState    `json:"resource"`
	UpdatedAt   time.Time         `json:"updated_at"`
}

// ServiceState 服务状态
type ServiceState struct {
	Name      string    `json:"name"`
	Running   bool      `json:"running"`
	Healthy   bool      `json:"healthy"`
	Version   string    `json:"version"`
	Ports     []int     `json:"ports"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ResourceState 资源状态
type ResourceState struct {
	Enabled     bool      `json:"enabled"`
	BridgeRunning bool    `json:"bridge_running"`
	ResourceCount int     `json:"resource_count"`
	UpdatedAt   time.Time `json:"updated_at"`
}
