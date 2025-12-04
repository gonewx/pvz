package game

import (
	"time"
)

// BattleSaveVersion 战斗存档版本号
// 用于版本兼容性检查，当数据结构发生不兼容变更时递增
// v2: 添加保龄球模式支持（BowlingNuts, ConveyorBelt, LevelPhase, DaveDialogue, GuidedTutorial）
const BattleSaveVersion = 2

// BattleSaveData 战斗存档数据结构
//
// Story 18.1: 战斗状态序列化系统
//
// 包含完整的战斗状态，用于保存和恢复游戏战斗进度。
// 使用 gob 二进制格式序列化，具有以下优势：
//   - 紧凑的二进制格式
//   - Go 原生支持，无需第三方库
//   - 类型安全
//   - 不易被用户手动修改（防作弊）
type BattleSaveData struct {
	// 版本和元数据
	Version  int       // 存档版本号，用于兼容性检查
	SaveTime time.Time // 保存时间

	// 关卡状态
	LevelID             string  // 关卡ID，如 "1-2"
	LevelTime           float64 // 关卡已进行时间（秒）
	CurrentWaveIndex    int     // 当前波次索引（0表示第一波）
	SpawnedWaves        []bool  // 已生成波次标记
	TotalZombiesSpawned int     // 已生成僵尸总数
	ZombiesKilled       int     // 已消灭僵尸数
	Sun                 int     // 当前阳光数量

	// 教学状态
	Tutorial *TutorialSaveData // 教学进度数据（可选，非教学关卡为 nil）

	// 实体数据
	Plants      []PlantData      // 植物数据
	Zombies     []ZombieData     // 僵尸数据
	Projectiles []ProjectileData // 子弹数据
	Suns        []SunData        // 阳光数据
	Lawnmowers  []LawnmowerData  // 除草车数据

	// 保龄球模式数据（Level 1-5）
	BowlingNuts    []BowlingNutData    // 保龄球坚果数据
	ConveyorBelt   *ConveyorBeltData   // 传送带数据（可选）
	LevelPhase     *LevelPhaseData     // 关卡阶段数据（可选）
	DaveDialogue   *DaveDialogueData   // Dave 对话数据（可选）
	GuidedTutorial *GuidedTutorialData // 强引导教学数据（可选）
}

// TutorialSaveData 教学进度序列化数据
//
// 包含教学系统的状态，用于正确恢复教学流程
type TutorialSaveData struct {
	CurrentStepIndex int             // 当前教学步骤索引
	CompletedSteps   map[string]bool // 已完成的步骤（trigger -> completed）
	IsActive         bool            // 教学是否激活
	PlantCount       int             // 已种植的植物数量
	SunflowerCount   int             // 已种植的向日葵数量
}

// PlantData 植物序列化数据
//
// 包含植物实体的核心状态，用于恢复植物实体。
// 字段与 PlantComponent、HealthComponent 等组件对应。
type PlantData struct {
	PlantType        string  // 植物类型ID，如 "peashooter", "sunflower"
	GridRow          int     // 所在草坪行 (0-4, 从上到下)
	GridCol          int     // 所在草坪列 (0-8, 从左到右)
	Health           int     // 当前生命值
	MaxHealth        int     // 最大生命值
	AttackCooldown   float64 // 攻击冷却剩余时间（秒）
	TimerTargetTime  float64 // 计时器目标时间（秒），用于恢复向日葵等变周期植物
	BlinkTimer       float64 // 眨眼计时器（秒）
	AttackAnimState  int     // 攻击动画状态 (0=空闲, 1=攻击中)
}

// ZombieData 僵尸序列化数据
//
// 包含僵尸实体的核心状态，用于恢复僵尸实体。
// 字段与 BehaviorComponent、HealthComponent、ArmorComponent 等组件对应。
type ZombieData struct {
	ZombieType   string  // 僵尸类型ID，如 "basic", "conehead", "buckethead"
	X            float64 // X坐标（世界坐标）
	Y            float64 // Y坐标（世界坐标）
	VelocityX    float64 // X轴速度（像素/秒）
	Health       int     // 当前生命值
	MaxHealth    int     // 最大生命值
	ArmorHealth  int     // 当前护甲值
	ArmorMax     int     // 最大护甲值
	Lane         int     // 所在行号（1-5）
	BehaviorType string  // 行为类型，如 "basic", "eating", "dying"
	IsEating     bool    // 是否正在啃食
}

// ProjectileData 子弹序列化数据
//
// 包含子弹实体的核心状态，用于恢复子弹实体。
type ProjectileData struct {
	Type      string  // 子弹类型ID，如 "pea", "snow_pea"
	X         float64 // X坐标（世界坐标）
	Y         float64 // Y坐标（世界坐标）
	VelocityX float64 // X轴速度（像素/秒）
	Damage    int     // 伤害值
	Lane      int     // 所在行号（1-5）
}

// SunData 阳光序列化数据
//
// 包含阳光实体的核心状态，用于恢复阳光实体。
type SunData struct {
	X            float64 // X坐标（世界坐标）
	Y            float64 // Y坐标（世界坐标）
	VelocityY    float64 // Y轴速度（像素/秒，用于下落/上升）
	Lifetime     float64 // 剩余生命周期（秒）
	Value        int     // 阳光值（通常为25）
	IsCollecting bool    // 是否正在被收集
	TargetX      float64 // 收集目标X坐标
	TargetY      float64 // 收集目标Y坐标
}

// LawnmowerData 除草车序列化数据
//
// 包含除草车实体的核心状态，用于恢复除草车实体。
type LawnmowerData struct {
	Lane      int     // 所在行号（1-5）
	X         float64 // X坐标（世界坐标）
	Triggered bool    // 是否已触发（僵尸到达左侧）
	Active    bool    // 是否激活（正在移动）
}

// BattleSaveInfo 战斗存档信息预览
//
// 用于在不加载完整存档的情况下显示存档信息。
// 包含关卡ID、保存时间、阳光数量和波次进度。
type BattleSaveInfo struct {
	LevelID   string    // 关卡ID，如 "1-2"
	SaveTime  time.Time // 保存时间
	Sun       int       // 阳光数量
	WaveIndex int       // 当前波次索引
}

// NewBattleSaveData 创建一个新的战斗存档数据结构
//
// 返回一个初始化的 BattleSaveData 实例，版本号和时间已设置。
func NewBattleSaveData() *BattleSaveData {
	return &BattleSaveData{
		Version:      BattleSaveVersion,
		SaveTime:     time.Now(),
		SpawnedWaves: []bool{},
		Plants:       []PlantData{},
		Zombies:      []ZombieData{},
		Projectiles:  []ProjectileData{},
		Suns:         []SunData{},
		Lawnmowers:   []LawnmowerData{},
		BowlingNuts:  []BowlingNutData{},
	}
}

// ToBattleSaveInfo 从完整存档数据提取预览信息
//
// 返回 BattleSaveInfo 结构，用于快速显示存档概要。
func (b *BattleSaveData) ToBattleSaveInfo() *BattleSaveInfo {
	return &BattleSaveInfo{
		LevelID:   b.LevelID,
		SaveTime:  b.SaveTime,
		Sun:       b.Sun,
		WaveIndex: b.CurrentWaveIndex,
	}
}

// =============================================================================
// 保龄球模式数据结构（Level 1-5）
// =============================================================================

// BowlingNutData 保龄球坚果序列化数据
//
// 包含保龄球坚果实体的核心状态，用于恢复滚动中的坚果。
type BowlingNutData struct {
	X                 float64 // X坐标（世界坐标）
	Y                 float64 // Y坐标（世界坐标）
	VelocityX         float64 // 水平移动速度（像素/秒）
	VelocityY         float64 // 垂直移动速度（像素/秒）
	Row               int     // 所在行号（0-4）
	IsRolling         bool    // 是否正在滚动
	IsBouncing        bool    // 是否正在弹射中
	TargetRow         int     // 弹射目标行（0-4）
	IsExplosive       bool    // 是否为爆炸坚果
	BounceCount       int     // 弹射次数
	CollisionCooldown float64 // 碰撞冷却时间（秒）
	BounceDirection   int     // 弹射方向（-1=向上, 1=向下, 0=未弹射）
}

// ConveyorBeltData 传送带序列化数据
//
// 包含传送带的完整状态，用于恢复卡片队列和生成进度。
type ConveyorBeltData struct {
	Cards              []ConveyorCardData // 卡片队列
	Capacity           int                // 最大容量
	ScrollOffset       float64            // 传动动画偏移量
	IsActive           bool               // 是否激活
	GenerationTimer    float64            // 卡片生成计时器
	GenerationInterval float64            // 卡片生成间隔（秒）
	SelectedCardIndex  int                // 当前选中的卡片索引
	FinalWaveTriggered bool               // 最终波是否已触发
}

// ConveyorCardData 传送带卡片序列化数据
type ConveyorCardData struct {
	CardType      string  // 卡片类型
	SlideProgress float64 // 滑入动画进度
	SlotIndex     int     // 槽位索引
}

// LevelPhaseData 关卡阶段序列化数据
//
// 包含多阶段关卡的当前状态，用于恢复阶段流程。
type LevelPhaseData struct {
	CurrentPhase        int     // 当前阶段编号（1=铲子教学, 2=保龄球）
	PhaseState          string  // 阶段状态（active/transitioning/completed）
	TransitionProgress  float64 // 转场动画进度
	TransitionStep      int     // 转场序列当前步骤
	ConveyorBeltY       float64 // 传送带当前 Y 位置
	ConveyorBeltVisible bool    // 传送带是否可见
	ShowRedLine         bool    // 是否显示红线
}

// DaveDialogueData Dave 对话序列化数据
//
// 包含 Dave 对话的当前状态，用于恢复对话流程。
type DaveDialogueData struct {
	DialogueKeys     []string // 对话文本 key 列表
	CurrentLineIndex int      // 当前对话行索引
	CurrentText      string   // 当前显示的文本内容
	IsVisible        bool     // 对话气泡是否可见
	State            int      // Dave 当前状态（0=Hidden, 1=Entering, 2=Talking, 3=Leaving）
	Expression       string   // 当前表情状态
	DaveX            float64  // Dave X 坐标
	DaveY            float64  // Dave Y 坐标
}

// GuidedTutorialData 强引导教学序列化数据
//
// 包含强引导教学的状态，用于恢复铲子教学阶段。
type GuidedTutorialData struct {
	IsActive        bool     // 强引导模式是否激活
	AllowedActions  []string // 允许的操作白名单
	IdleTimer       float64  // 空闲计时器（秒）
	IdleThreshold   float64  // 空闲阈值（秒）
	ShowArrow       bool     // 是否显示浮动箭头
	ArrowTarget     string   // 箭头指向目标
	LastPlantCount  int      // 上一帧的植物数量
	TransitionReady bool     // 转场条件是否满足
	TutorialTextKey string   // 教学文本键
}
