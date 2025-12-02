package components

import (
	"github.com/decker502/pvz/pkg/types"
	"github.com/hajimehoshi/ebiten/v2"
)

// PlantType 是 types.PlantType 的类型别名，保持向后兼容
type PlantType = types.PlantType

// 植物类型常量（从 types 包重新导出，保持向后兼容）
const (
	PlantUnknown    = types.PlantUnknown
	PlantSunflower  = types.PlantSunflower
	PlantPeashooter = types.PlantPeashooter
	PlantWallnut    = types.PlantWallnut
	PlantCherryBomb = types.PlantCherryBomb
	PlantPotatoMine = types.PlantPotatoMine // Story 19.10
)

// PlantCardComponent 表示植物选择卡片的数据
// 包含植物类型、消耗、冷却等信息
// 此组件用于 ECS 架构中标识植物卡片实体，并存储其状态数据
//
// Story 6.3: 卡片采用多层渲染设计：
// - 背景层：卡片框 (SeedPacket_Larger.png)
// - 植物层：Reanim 渲染的植物预览图（离屏渲染到纹理）
// - 文字层：阳光数字（动态绘制）
// - 效果层：冷却遮罩/禁用效果
type PlantCardComponent struct {
	// PlantType 植物类型（向日葵、豌豆射手、坚果墙等）
	PlantType PlantType
	// SunCost 种植消耗的阳光数量
	SunCost int
	// CooldownTime 冷却总时间（秒）
	CooldownTime float64
	// CurrentCooldown 当前剩余冷却时间（秒）
	CurrentCooldown float64
	// IsAvailable 是否可用（考虑阳光和冷却）
	IsAvailable bool

	// Story 6.3: 多层渲染资源
	// BackgroundImage 卡片背景框图片（所有卡片共享）
	BackgroundImage *ebiten.Image
	// PlantIconTexture 植物预览图标（Reanim 离屏渲染生成的纹理）
	PlantIconTexture *ebiten.Image

	// Story 8.4: 卡片缩放
	// CardScale 卡片整体缩放因子（用于控制卡片显示大小，如 0.54 为标准大小，1.0 为原始大小）
	CardScale float64

	// Story 8.4: 卡片透明度（用于淡入淡出动画）
	// Alpha 卡片透明度（0.0 = 完全透明，1.0 = 完全不透明，默认 1.0）
	Alpha float64
}
