package components

import "github.com/decker502/pvz/pkg/ecs"

// GuidedTutorialComponent 强引导教学组件
// Story 19.3: 强引导教学系统
// Story 19.x QA: 添加教学文本支持
//
// 此组件用于管理 Level 1-5 的强制铲子教学阶段的状态。
// 与 TutorialComponent 不同，这是"强制式"引导，限制玩家只能进行特定操作。
// 作为单例组件挂载到一个专用实体上，由 GuidedTutorialSystem 读取和更新。
type GuidedTutorialComponent struct {
	// IsActive 强引导模式是否激活
	// true: 限制玩家操作，只允许白名单中的操作
	// false: 正常模式，不限制操作
	IsActive bool

	// AllowedActions 允许的操作白名单
	// 强引导模式下只允许这些操作，其他操作静默忽略
	// 操作标识：
	//   - "click_shovel": 点击铲子槽位（进入/退出铲子模式）
	//   - "click_plant": 点击植物（铲子模式下移除植物）
	//   - "click_screen": 点击屏幕（推进 Dave 对话）
	AllowedActions []string

	// IdleTimer 空闲计时器（秒）
	// 记录玩家上次有效操作后经过的时间
	// 超过 IdleThreshold 后显示浮动箭头提示
	IdleTimer float64

	// IdleThreshold 空闲阈值（秒）
	// 玩家无操作超过此时间后显示浮动箭头
	// 默认值：5.0 秒
	IdleThreshold float64

	// ShowArrow 是否显示浮动箭头
	// true: 正在显示箭头指示符
	// false: 箭头已隐藏
	ShowArrow bool

	// ArrowTarget 箭头指向目标
	// "shovel": 箭头指向铲子槽位
	// "plant": 箭头指向草坪上的植物
	ArrowTarget string

	// ArrowEntityID 箭头粒子实体 ID
	// 用于管理箭头粒子效果的生命周期
	// 0 表示当前没有显示箭头
	ArrowEntityID ecs.EntityID

	// LastPlantCount 上一帧的植物数量
	// 用于检测植物数量变化（玩家移除了植物）
	LastPlantCount int

	// TransitionReady 转场条件是否满足
	// 当场上植物数量为 0 时设置为 true
	// 外部系统可以读取此标志来决定是否进入下一阶段
	TransitionReady bool

	// OnTransitionCallback 转场回调函数
	// 当转场条件满足时调用此回调
	// 由外部系统（如 GameScene）设置，遵循零耦合原则
	OnTransitionCallback func()

	// TextEntityID 教学文本实体 ID
	// Story 19.x QA: 用于显示铲子教学提示文本
	// 0 表示当前没有显示教学文本
	TextEntityID ecs.EntityID

	// TutorialTextKey 教学文本键（从 LawnStrings.txt 加载）
	// 默认为 "SHOVEL_INSTRUCTION"
	TutorialTextKey string
}
