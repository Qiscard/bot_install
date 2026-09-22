package tui

import (
	"github.com/bot-ctl/bot-ctl/internal/adapter"
	"github.com/bot-ctl/bot-ctl/internal/config"
	"github.com/bot-ctl/bot-ctl/pkg/model"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

// step 向导步骤
type step int

const (
	stepDoctor step = iota
	stepSelectFrameworks
	stepInstallDir
	stepVersions
	stepResource
	stepReview
	stepApply
	stepDone
)

// frameworkItem 多选列表项
type frameworkItem struct {
	Type     model.FrameworkType
	Label    string
	Checked  bool
	Version  string // 用户输入的版本规格
}

// Model 是根 TUI 模型
type Model struct {
	step     step
	cursor   int
	width    int
	height   int
	quitting bool
	errMsg   string

	registry *adapter.Registry
	store    *config.Store

	// doctor
	diagDone bool
	diagText string
	diagOK   bool

	// 框架多选
	items []frameworkItem

	// 安装目录输入
	dirInput textinput.Model

	// 版本输入（按当前选中的框架）
	verInputs   []textinput.Model
	verIndex    int
	versionInit bool

	// 资源共享
	resourceEnabled bool
	resourceSource  int // 0 = 第一个可作为来源的框架
	resourceCursor  int

	// 生成结果
	stack           *model.StackConfig
	composePath     string
	DeployAfterExit bool // 用户在确认页选择部署；退出 TUI 后由 Run 执行流式部署
}

// New 创建 TUI 模型
func New(store *config.Store, reg *adapter.Registry) Model {
	items := make([]frameworkItem, 0)
	for _, a := range reg.All() {
		items = append(items, frameworkItem{
			Type:    a.Type(),
			Label:   a.DisplayName(),
			Checked: false,
			Version: "",
		})
	}
	// 追加“QQ 资源共享桥”作为一个可选项由 resource 步骤处理

	di := textinput.New()
	di.Placeholder = config.DefaultInstallDir
	di.SetValue(config.DefaultInstallDir)
	di.CharLimit = 256
	di.Width = 48

	return Model{
		step:     stepDoctor,
		registry: reg,
		store:    store,
		items:    items,
		dirInput: di,
	}
}

// Init 启动时触发环境诊断
func (m Model) Init() tea.Cmd {
	return runDoctorCmd(m.registry)
}
