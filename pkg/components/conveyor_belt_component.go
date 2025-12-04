package components

// ConveyorBeltComponent 传送带组件
// Story 19.5: 管理传送带状态和卡片队列
//
// 此组件用于追踪传送带的当前状态，支持：
// - 卡片队列管理（最多 10 张卡片）
// - 传动动画状态
// - 卡片生成计时
type ConveyorBeltComponent struct {
	// Cards 卡片队列（从左到右排列）
	// 最左边是最旧的卡片（SlotIndex = 0），最右边是最新的卡片
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

	// GenerationTimer 卡片生成计时器
	// 从 GenerationInterval 倒计时到 0
	GenerationTimer float64

	// GenerationInterval 卡片生成间隔（秒）
	// 当传送带未满时，每隔此时间生成一张新卡片
	GenerationInterval float64

	// SelectedCardIndex 当前选中的卡片索引
	// -1 表示没有选中任何卡片
	SelectedCardIndex int

	// FinalWaveTriggered 最终波是否已触发
	// 用于避免重复插入爆炸坚果
	FinalWaveTriggered bool
}

// ConveyorCard 传送带卡片
// 表示传送带上的单张卡片
type ConveyorCard struct {
	// CardType 卡片类型
	// 可选值: "wallnut_bowling", "explode_o_nut"
	CardType string

	// PositionX 卡片当前 X 位置（相对于传送带左边界的像素偏移）
	// 卡片从右侧进入后持续向左移动，直到停止
	PositionX float64

	// IsStopped 卡片是否已停止移动
	// 停止条件：到达左边界 或 碰到前面已停止的卡片
	IsStopped bool
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

// DefaultCardGenerationInterval 默认卡片生成间隔（秒）
const DefaultCardGenerationInterval = 3.0

// NewConveyorBeltComponent 创建传送带组件
// 使用默认配置初始化
func NewConveyorBeltComponent() *ConveyorBeltComponent {
	return &ConveyorBeltComponent{
		Cards:              make([]ConveyorCard, 0),
		Capacity:           DefaultConveyorCapacity,
		ScrollOffset:       0,
		IsActive:           false,
		GenerationTimer:    0,
		GenerationInterval: DefaultCardGenerationInterval,
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
