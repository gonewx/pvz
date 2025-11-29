package game

import (
	"time"
)

// BattleSaveVersion 战斗存档版本号
// 用于版本兼容性检查，当数据结构发生不兼容变更时递增
const BattleSaveVersion = 1

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
	LevelID             string // 关卡ID，如 "1-2"
	LevelTime           float64 // 关卡已进行时间（秒）
	CurrentWaveIndex    int     // 当前波次索引（0表示第一波）
	SpawnedWaves        []bool  // 已生成波次标记
	TotalZombiesSpawned int     // 已生成僵尸总数
	ZombiesKilled       int     // 已消灭僵尸数
	Sun                 int     // 当前阳光数量

	// 实体数据
	Plants      []PlantData      // 植物数据
	Zombies     []ZombieData     // 僵尸数据
	Projectiles []ProjectileData // 子弹数据
	Suns        []SunData        // 阳光数据
	Lawnmowers  []LawnmowerData  // 除草车数据
}

// PlantData 植物序列化数据
//
// 包含植物实体的核心状态，用于恢复植物实体。
// 字段与 PlantComponent、HealthComponent 等组件对应。
type PlantData struct {
	PlantType      string  // 植物类型ID，如 "peashooter", "sunflower"
	GridRow        int     // 所在草坪行 (0-4, 从上到下)
	GridCol        int     // 所在草坪列 (0-8, 从左到右)
	Health         int     // 当前生命值
	MaxHealth      int     // 最大生命值
	AttackCooldown float64 // 攻击冷却剩余时间（秒）
	BlinkTimer     float64 // 眨眼计时器（秒）
	AttackAnimState int    // 攻击动画状态 (0=空闲, 1=攻击中)
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
		Version:     BattleSaveVersion,
		SaveTime:    time.Now(),
		SpawnedWaves: []bool{},
		Plants:      []PlantData{},
		Zombies:     []ZombieData{},
		Projectiles: []ProjectileData{},
		Suns:        []SunData{},
		Lawnmowers:  []LawnmowerData{},
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
