// Package config 提供游戏配置管理
package config

import (
	"github.com/gonewx/pvz/pkg/types"
)

// ZombieInfo 包含僵尸的所有元数据信息
// 新增僵尸时只需在 zombieRegistry 中添加一条记录即可
type ZombieInfo struct {
	Type       types.ZombieType // 僵尸类型枚举
	ID         string           // 配置文件ID (如 "basic", "conehead")
	Aliases    []string         // ID别名 (如 "zombie" 也指向 basic)
	UnitID     string           // 动画单元ID (如 "zombie", "zombie_conehead")
	ReanimName string           // Reanim 资源名称
	Health     int              // 基础生命值
}

// zombieRegistry 僵尸注册表 - 集中管理所有僵尸的元数据
// 新增僵尸时只需在此处添加一条记录
var zombieRegistry = []ZombieInfo{
	{
		Type:       types.ZombieBasic,
		ID:         "basic",
		Aliases:    []string{"zombie"},
		UnitID:     types.UnitIDZombie,
		ReanimName: "Zombie",
		Health:     ZombieDefaultHealth,
	},
	{
		Type:       types.ZombieConehead,
		ID:         "conehead",
		Aliases:    nil,
		UnitID:     types.UnitIDZombieConehead,
		ReanimName: "Zombie_conehead",
		Health:     ZombieDefaultHealth + ConeheadZombieArmorHealth,
	},
	{
		Type:       types.ZombieBuckethead,
		ID:         "buckethead",
		Aliases:    nil,
		UnitID:     types.UnitIDZombieBuckethead,
		ReanimName: "Zombie_buckethead",
		Health:     ZombieDefaultHealth + BucketheadZombieArmorHealth,
	},
	{
		Type:       types.ZombieFlag,
		ID:         "flag",
		Aliases:    nil,
		UnitID:     types.UnitIDZombieFlag,
		ReanimName: "Zombie",
		Health:     ZombieDefaultHealth,
	},
	{
		Type:       types.ZombiePolevaulter,
		ID:         "polevaulter",
		Aliases:    nil,
		UnitID:     types.UnitIDZombiePolevaulter,
		ReanimName: "Zombie_polevaulter",
		Health:     PolevaulterZombieHealth,
	},
}

// 预构建的索引映射，避免每次查询都遍历
var (
	zombieByID   map[string]*ZombieInfo
	zombieByType map[types.ZombieType]*ZombieInfo
)

func init() {
	zombieByID = make(map[string]*ZombieInfo, len(zombieRegistry)*2)
	zombieByType = make(map[types.ZombieType]*ZombieInfo, len(zombieRegistry))

	for i := range zombieRegistry {
		info := &zombieRegistry[i]
		// 主ID
		zombieByID[info.ID] = info
		// 别名
		for _, alias := range info.Aliases {
			zombieByID[alias] = info
		}
		// 类型映射
		zombieByType[info.Type] = info
	}
}

// GetZombieByID 根据配置ID获取僵尸信息
func GetZombieByID(id string) *ZombieInfo {
	return zombieByID[id]
}

// GetZombieByType 根据ZombieType获取僵尸信息
func GetZombieByType(zombieType types.ZombieType) *ZombieInfo {
	return zombieByType[zombieType]
}

// ZombieIDToType 将配置ID转换为ZombieType
func ZombieIDToType(id string) types.ZombieType {
	if info := zombieByID[id]; info != nil {
		return info.Type
	}
	return types.ZombieUnknown
}

// ZombieTypeToID 将ZombieType转换为配置ID
func ZombieTypeToID(zombieType types.ZombieType) string {
	if info := zombieByType[zombieType]; info != nil {
		return info.ID
	}
	return ""
}

// GetZombieUnitID 获取僵尸的动画单元ID
func GetZombieUnitID(zombieType types.ZombieType) string {
	if info := zombieByType[zombieType]; info != nil {
		return info.UnitID
	}
	return ""
}

// GetZombieUnitIDByID 根据配置ID获取动画单元ID
func GetZombieUnitIDByID(id string) string {
	if info := zombieByID[id]; info != nil {
		return info.UnitID
	}
	return ""
}

// GetZombieReanimName 获取僵尸的Reanim资源名称
func GetZombieReanimName(zombieType types.ZombieType) string {
	if info := zombieByType[zombieType]; info != nil {
		return info.ReanimName
	}
	return ""
}

// GetZombieHealth 获取僵尸的基础生命值
func GetZombieHealth(zombieType types.ZombieType) int {
	if info := zombieByType[zombieType]; info != nil {
		return info.Health
	}
	return 0
}

// GetZombieHealthByID 根据配置ID获取基础生命值
func GetZombieHealthByID(id string) int {
	if info := zombieByID[id]; info != nil {
		return info.Health
	}
	return 0
}

// IsValidZombieID 检查ID是否是有效的僵尸ID
func IsValidZombieID(id string) bool {
	return zombieByID[id] != nil
}

// GetAllZombies 返回所有注册的僵尸信息
func GetAllZombies() []ZombieInfo {
	result := make([]ZombieInfo, len(zombieRegistry))
	copy(result, zombieRegistry)
	return result
}
