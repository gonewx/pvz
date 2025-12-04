package components

// ConveyorBeltComponent 传送带组件
// Story 19.5 & 19.12: 管理传送带状态和卡片队列
//
// 此组件用于追踪传送带的当前状态，支持：
// - 卡片队列管理（最多 10 张卡片）
// - 传动动画状态
// - 卡片移动和生成
type ConveyorBeltComponent struct {
	// Cards 卡片队列（按生成顺序排列）
	// 最早生成的卡片在前（靠左），最新生成的卡片在后（靠右）
	Cards []ConveyorCard

	// Capacity 最大容量
	// 默认为 10 张卡片
	Capacity int

	// ScrollOffset 传动动画偏移量
	// 范围：0.0 到 纹理高度，用于实现传动带滚动效果
	ScrollOffset float64

	// IsActive 是否激活
	// 只有激活状态才会生成新卡片和响应交互
	IsActive bool

	// NextSpacing 下一个坚果的间隔距离（像素）
	// Story 19.12: 用于距离驱动的生成逻辑
	NextSpacing float64

	// SelectedCardIndex 当前选中的卡片索引
	// -1 表示没有选中任何卡片
	SelectedCardIndex int

	// FinalWaveTriggered 最终波是否已触发
	// 用于避免重复插入爆炸坚果
	FinalWaveTriggered bool
}

// ConveyorCard 传送带卡片
// Story 19.12: 表示传送带上的单张卡片
type ConveyorCard struct {
	// CardType 卡片类型
	// 可选值: "wallnut_bowling", "explode_o_nut"
	CardType string

	// PositionX 在传送带上的 X 位置（局部坐标，像素）
	// Story 19.12: 0 = 传送带最左侧，ConveyorBeltWidth = 最右侧
	// 卡片随传送带向左移动，此值逐渐减小
	PositionX float64

	// IsAtLeftEdge 是否已到达左侧边缘
	// Story 19.12: 到达 ConveyorNutStopX 后设为 true，停止移动
	IsAtLeftEdge bool
}

// CardType 常量
const (
	// CardTypeWallnutBowling 保龄球坚果
	CardTypeWallnutBowling = "wallnut_bowling"

	// CardTypeExplodeONut 爆炸坚果
	CardTypeExplodeONut = "explode_o_nut"
)

// DefaultConveyorCapacity 默认传送带容量
const DefaultConveyorCapacity = 10

// DefaultNutSpacing 默认坚果间隔（像素）
// Story 19.12: 基础间隔，由配置覆盖
const DefaultNutSpacing = 80.0

// NewConveyorBeltComponent 创建传送带组件
// 使用默认配置初始化
func NewConveyorBeltComponent() *ConveyorBeltComponent {
	return &ConveyorBeltComponent{
		Cards:              make([]ConveyorCard, 0),
		Capacity:           DefaultConveyorCapacity,
		ScrollOffset:       0,
		IsActive:           false,
		NextSpacing:        DefaultNutSpacing,
		SelectedCardIndex:  -1,
		FinalWaveTriggered: false,
	}
}

// IsFull 检查传送带是否已满
func (c *ConveyorBeltComponent) IsFull() bool {
	return len(c.Cards) >= c.Capacity
}

// IsEmpty 检查传送带是否为空
func (c *ConveyorBeltComponent) IsEmpty() bool {
	return len(c.Cards) == 0
}

// CardCount 获取当前卡片数量
func (c *ConveyorBeltComponent) CardCount() int {
	return len(c.Cards)
}
