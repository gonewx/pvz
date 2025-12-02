// Package types 定义共享的基础类型
// 这个包不依赖任何其他业务包，用于解决循环引用问题
package types

// PlantType 定义植物的类型
type PlantType int

const (
	// PlantUnknown 未知植物类型
	PlantUnknown PlantType = iota
	// PlantSunflower 向日葵
	PlantSunflower
	// PlantPeashooter 豌豆射手
	PlantPeashooter
	// PlantWallnut 坚果墙
	PlantWallnut
	// PlantCherryBomb 樱桃炸弹
	PlantCherryBomb
	// PlantPotatoMine 土豆地雷 (Story 19.10)
	PlantPotatoMine
)

// String 返回植物类型的字符串表示
func (p PlantType) String() string {
	switch p {
	case PlantSunflower:
		return "Sunflower"
	case PlantPeashooter:
		return "Peashooter"
	case PlantWallnut:
		return "Wallnut"
	case PlantCherryBomb:
		return "CherryBomb"
	case PlantPotatoMine:
		return "PotatoMine"
	default:
		return "Unknown"
	}
}
