package game

import "sort"

// PlantUnlockManager 管理玩家的植物解锁进度
// 负责追踪哪些植物已经解锁，以及提供解锁查询接口
type PlantUnlockManager struct {
	unlockedPlants map[string]bool
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

