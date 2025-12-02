package components

import (
	"image"

	"github.com/decker502/pvz/pkg/ecs"
	"github.com/hajimehoshi/ebiten/v2"
)

// ShovelInteractionComponent 铲子交互组件
// Story 19.2: 铲子交互系统增强
//
// 此组件用于管理铲子选中状态、光标图标和植物高亮效果。
// 作为单例组件挂载到一个专用实体上，由 ShovelInteractionSystem 读取和更新。
type ShovelInteractionComponent struct {
	// IsSelected 铲子是否被选中
	// true: 进入铲子模式，鼠标变为铲子图标，可以移除植物
	// false: 正常模式
	IsSelected bool

	// CursorImage 铲子光标图标
	// 铲子模式下跟随鼠标移动的铲子图片
	CursorImage *ebiten.Image

	// HighlightedPlantEntity 当前高亮的植物实体ID
	// 鼠标悬停在植物上时设置为该植物的实体ID
	// 无悬停时为 0
	HighlightedPlantEntity ecs.EntityID

	// ShovelSlotBounds 铲子槽位的碰撞边界（屏幕坐标）
	// 用于检测鼠标点击是否在铲子槽位上
	ShovelSlotBounds image.Rectangle

	// CursorAnchorX 铲子光标锚点X偏移
	// 铲子尖端相对于图片左上角的X偏移
	CursorAnchorX float64

	// CursorAnchorY 铲子光标锚点Y偏移
	// 铲子尖端相对于图片左上角的Y偏移
	CursorAnchorY float64
}
