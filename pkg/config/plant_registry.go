// Package config 提供游戏配置管理
package config

import (
	"github.com/gonewx/pvz/pkg/types"
)

// PlantInfo 包含植物的所有元数据信息
// 新增植物时只需在 plantRegistry 中添加一条记录即可
type PlantInfo struct {
	Type       types.PlantType // 植物类型枚举
	ID         string          // 配置文件ID (如 "sunflower")
	ReanimName string          // Reanim 资源名称 (如 "SunFlower")
	SunCost    int             // 阳光消耗
}

// plantRegistry 植物注册表 - 集中管理所有植物的元数据
// 新增植物时只需在此处添加一条记录
// 注意：SunCost 在 init() 中从常量初始化，这里的值仅作为默认值
var plantRegistry = []PlantInfo{
	{Type: types.PlantSunflower, ID: "sunflower", ReanimName: "SunFlower"},
	{Type: types.PlantPeashooter, ID: "peashooter", ReanimName: "PeaShooterSingle"},
	{Type: types.PlantCherryBomb, ID: "cherrybomb", ReanimName: "CherryBomb"},
	{Type: types.PlantWallnut, ID: "wallnut", ReanimName: "Wallnut"},
	{Type: types.PlantPotatoMine, ID: "potatomine", ReanimName: "PotatoMine"},
	{Type: types.PlantSnowPea, ID: "snowpea", ReanimName: "SnowPea"},
	{Type: types.PlantChomper, ID: "chomper", ReanimName: "Chomper"},
}

// 预构建的索引映射，避免每次查询都遍历
var (
	plantByID   map[string]*PlantInfo
	plantByType map[types.PlantType]*PlantInfo
)

// sunCostMap 阳光消耗映射表（从常量初始化）
var sunCostMap map[types.PlantType]int

func init() {
	// 初始化阳光消耗映射（引用现有常量）
	sunCostMap = map[types.PlantType]int{
		types.PlantSunflower:  SunflowerSunCost,
		types.PlantPeashooter: PeashooterSunCost,
		types.PlantCherryBomb: CherryBombSunCost,
		types.PlantWallnut:    WallnutCost,
		types.PlantPotatoMine: PotatoMineSunCost,
		types.PlantSnowPea:    SnowPeaSunCost,
		types.PlantChomper:    ChomperSunCost,
	}

	// 构建索引映射
	plantByID = make(map[string]*PlantInfo, len(plantRegistry))
	plantByType = make(map[types.PlantType]*PlantInfo, len(plantRegistry))

	for i := range plantRegistry {
		info := &plantRegistry[i]
		// 从常量映射中获取阳光消耗
		if cost, ok := sunCostMap[info.Type]; ok {
			info.SunCost = cost
		}
		plantByID[info.ID] = info
		plantByType[info.Type] = info
	}
}

// GetPlantByID 根据配置ID获取植物信息
func GetPlantByID(id string) *PlantInfo {
	return plantByID[id]
}

// GetPlantByType 根据PlantType获取植物信息
func GetPlantByType(plantType types.PlantType) *PlantInfo {
	return plantByType[plantType]
}

// PlantIDToType 将配置ID转换为PlantType
func PlantIDToType(id string) types.PlantType {
	if info := plantByID[id]; info != nil {
		return info.Type
	}
	return types.PlantUnknown
}

// PlantTypeToID 将PlantType转换为配置ID
func PlantTypeToID(plantType types.PlantType) string {
	if info := plantByType[plantType]; info != nil {
		return info.ID
	}
	return ""
}

// GetPlantReanimName 获取植物的Reanim资源名称
func GetPlantReanimName(plantType types.PlantType) string {
	if info := plantByType[plantType]; info != nil {
		return info.ReanimName
	}
	return ""
}

// GetPlantReanimNameByID 根据配置ID获取Reanim资源名称
func GetPlantReanimNameByID(id string) string {
	if info := plantByID[id]; info != nil {
		return info.ReanimName
	}
	return ""
}

// GetPlantSunCost 根据配置ID获取阳光消耗
func GetPlantSunCost(id string) int {
	if info := plantByID[id]; info != nil {
		return info.SunCost
	}
	return 0
}

// GetPlantSunCostByType 根据PlantType获取阳光消耗
func GetPlantSunCostByType(plantType types.PlantType) int {
	if info := plantByType[plantType]; info != nil {
		return info.SunCost
	}
	return 0
}

// GetAllPlants 返回所有注册的植物信息
func GetAllPlants() []PlantInfo {
	result := make([]PlantInfo, len(plantRegistry))
	copy(result, plantRegistry)
	return result
}
