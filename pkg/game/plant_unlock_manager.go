package game

import (
	"sort"

	"github.com/decker502/pvz/pkg/config"
)

// PlantUnlockManager 管理玩家的植物解锁进度
// 负责追踪哪些植物已经解锁，以及提供解锁查询接口
type PlantUnlockManager struct {
	unlockedPlants map[string]bool
	lastUnlocked   string // 最后一次解锁的植物ID（Story 8.3 新增）
}

// NewPlantUnlockManager 创建一个新的植物解锁管理器
// 初始化时包含第一章的默认解锁植物（开发阶段）
//
// 返回:
//   - *PlantUnlockManager: 新创建的植物解锁管理器实例
//
// 注意: 默认解锁列表仅用于开发阶段，Story 8.6 实现进度保存后将从存档加载
func NewPlantUnlockManager() *PlantUnlockManager {
	return &PlantUnlockManager{
		unlockedPlants: map[string]bool{
			// 第一章植物（1-1 到 1-10 逐步解锁）
			"peashooter": true, // 豌豆射手（1-1解锁）
			"sunflower":  true, // 向日葵（1-2解锁）
			"cherrybomb": true, // 樱桃炸弹（1-3解锁）
			"wallnut":    true, // 坚果墙（1-4解锁）
			"potatomine": true, // 土豆地雷（1-5解锁）
			"snowpea":    true, // 寒冰射手（1-6解锁）
			"chomper":    true, // 大嘴花（1-7解锁）
			"repeater":   true, // 双发射手（1-8解锁）
			// 更多植物将在后续章节解锁
		},
	}
}

// IsUnlocked 检查指定植物是否已解锁
// 参数:
//   - plantID: 植物ID（如 "peashooter", "sunflower"）
//
// 返回:
//   - bool: true 表示已解锁，false 表示未解锁
func (m *PlantUnlockManager) IsUnlocked(plantID string) bool {
	return m.unlockedPlants[plantID]
}

// UnlockPlant 解锁指定植物
// 参数:
//   - plantID: 要解锁的植物ID
//
// 注意: 如果植物已经解锁，此方法不会产生任何效果
func (m *PlantUnlockManager) UnlockPlant(plantID string) {
	// 只有在植物未解锁时才记录为"最后解锁"
	if !m.unlockedPlants[plantID] {
		m.lastUnlocked = plantID
	}
	m.unlockedPlants[plantID] = true
}

// GetUnlockedPlants 获取所有已解锁植物的ID列表
// 返回:
//   - []string: 已解锁植物的ID列表（按字母顺序排序）
func (m *PlantUnlockManager) GetUnlockedPlants() []string {
	plants := make([]string, 0, len(m.unlockedPlants))
	for plantID := range m.unlockedPlants {
		plants = append(plants, plantID)
	}

	// 按字母顺序排序，保证输出稳定
	sort.Strings(plants)

	return plants
}

// LoadFromSave 从存档文件加载植物解锁进度
// 此方法预留给 Story 8.6 实现，当前为空实现
//
// 返回:
//   - error: 加载失败时返回错误信息
//
// TODO(Story 8.6): 实现从存档文件加载解锁进度
func (m *PlantUnlockManager) LoadFromSave() error {
	// 预留给 Story 8.6 实现
	return nil
}

// SaveToFile 保存植物解锁进度到存档文件
// 此方法预留给 Story 8.6 实现，当前为空实现
//
// 返回:
//   - error: 保存失败时返回错误信息
//
// TODO(Story 8.6): 实现保存解锁进度到存档文件
func (m *PlantUnlockManager) SaveToFile() error {
	// 预留给 Story 8.6 实现
	return nil
}

// GetLastUnlocked 获取最后一次解锁的植物ID
// 用于关卡完成后触发奖励动画（Story 8.3）
//
// 返回:
//   - string: 最后一次解锁的植物ID，如果没有则返回空字符串
func (m *PlantUnlockManager) GetLastUnlocked() string {
	return m.lastUnlocked
}

// ClearLastUnlocked 清除最后解锁的植物ID
// 用于奖励动画显示后重置状态
func (m *PlantUnlockManager) ClearLastUnlocked() {
	m.lastUnlocked = ""
}

// PlantInfo 植物信息结构（名称和描述的文本键）
type PlantInfo struct {
	NameKey        string // LawnStrings.txt 中的名称键
	DescriptionKey string // LawnStrings.txt 中的描述键
}

// PlantInfoMap 植物信息映射表
// 存储所有植物的名称和描述文本键（从 LawnStrings.txt 读取）
var PlantInfoMap = map[string]PlantInfo{
	"peashooter": {
		NameKey:        "PEASHOOTER",
		DescriptionKey: "PEASHOOTER_TOOLTIP",
	},
	"sunflower": {
		NameKey:        "SUNFLOWER",
		DescriptionKey: "SUNFLOWER_TOOLTIP",
	},
	"cherrybomb": {
		NameKey:        "CHERRYBOMB",
		DescriptionKey: "CHERRYBOMB_TOOLTIP",
	},
	"wallnut": {
		NameKey:        "WALLNUT",
		DescriptionKey: "WALLNUT_TOOLTIP",
	},
	"potatomine": {
		NameKey:        "POTATOMINE_NAME",
		DescriptionKey: "POTATOMINE_DESC",
	},
	"snowpea": {
		NameKey:        "SNOWPEA_NAME",
		DescriptionKey: "SNOWPEA_DESC",
	},
	"chomper": {
		NameKey:        "CHOMPER_NAME",
		DescriptionKey: "CHOMPER_DESC",
	},
	"repeater": {
		NameKey:        "REPEATER_NAME",
		DescriptionKey: "REPEATER_DESC",
	},
}

// GetPlantInfo 获取植物的信息结构（名称和描述文本键）
// 参数:
//   - plantID: 植物ID
//
// 返回:
//   - PlantInfo: 植物信息结构
func (m *PlantUnlockManager) GetPlantInfo(plantID string) PlantInfo {
	info, ok := PlantInfoMap[plantID]
	if !ok {
		// 植物信息不存在，返回默认值
		return PlantInfo{
			NameKey:        "UNKNOWN_PLANT",
			DescriptionKey: "UNKNOWN_PLANT_DESC",
		}
	}
	return info
}

// GetPlantInfoWithStrings 获取植物的名称和描述（从 LawnStrings 加载）
// 参数:
//   - plantID: 植物ID
//   - lawnStrings: LawnStrings 实例
//
// 返回:
//   - name: 植物名称
//   - desc: 植物描述
func GetPlantInfoWithStrings(plantID string, lawnStrings *LawnStrings) (name, desc string) {
	info, ok := PlantInfoMap[plantID]
	if !ok {
		// 植物信息不存在，返回占位符
		return "[Unknown Plant]", "[No description available]"
	}

	name = lawnStrings.GetString(info.NameKey)
	desc = lawnStrings.GetString(info.DescriptionKey)
	return name, desc
}

// GetPlantSunCost 获取植物的阳光消耗值（从 config 包获取）
// 参数:
//   - plantID: 植物ID
//
// 返回:
//   - int: 阳光消耗值，如果植物ID未知则返回0
func GetPlantSunCost(plantID string) int {
	switch plantID {
	case "sunflower":
		return config.SunflowerSunCost
	case "peashooter":
		return config.PeashooterSunCost
	case "wallnut":
		return config.WallnutCost
	case "cherrybomb":
		return config.CherryBombSunCost
	case "potatomine":
		return 25 // TODO: 添加到 config 包
	case "snowpea":
		return 175 // TODO: 添加到 config 包
	case "chomper":
		return 150 // TODO: 添加到 config 包
	case "repeater":
		return 200 // TODO: 添加到 config 包
	default:
		return 0
	}
}
