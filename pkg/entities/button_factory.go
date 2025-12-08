package entities

import (
	"github.com/decker502/pvz/pkg/components"
	"github.com/decker502/pvz/pkg/ecs"
	"github.com/decker502/pvz/pkg/game"
)

// NewMenuButton 创建菜单按钮实体（三段式可拉伸按钮）
//
// 参数：
//   - em: 实体管理器
//   - rm: 资源管理器（加载按钮图片）
//   - x, y: 按钮位置（屏幕坐标）
//   - text: 按钮文字
//   - fontSize: 文字大小
//   - textColor: 文字颜色 [R, G, B, A]
//   - middleWidth: 中间部分宽度（可拉伸）
//   - onClick: 点击回调函数
//
// 返回：
//   - 按钮实体ID
//   - 错误信息
func NewMenuButton(
	em *ecs.EntityManager,
	rm *game.ResourceManager,
	x, y float64,
	text string,
	fontSize float64,
	textColor [4]uint8,
	middleWidth float64,
	onClick func(),
) (ecs.EntityID, error) {
	// 加载三段式按钮图片
	leftImage, err := rm.LoadImageByID("IMAGE_BUTTON_LEFT")
	if err != nil {
		return 0, err
	}

	middleImage, err := rm.LoadImageByID("IMAGE_BUTTON_MIDDLE")
	if err != nil {
		return 0, err
	}

	rightImage, err := rm.LoadImageByID("IMAGE_BUTTON_RIGHT")
	if err != nil {
		return 0, err
	}

	// 加载字体
	font, err := rm.LoadFont("assets/fonts/SimHei.ttf", fontSize)
	if err != nil {
		return 0, err
	}

	// 计算按钮总尺寸（三段式：左边缘 + 中间拉伸 + 右边缘）
	leftWidth := float64(leftImage.Bounds().Dx())
	rightWidth := float64(rightImage.Bounds().Dx())
	totalWidth := leftWidth + middleWidth + rightWidth
	totalHeight := float64(leftImage.Bounds().Dy())

	// 创建按钮实体
	entity := em.CreateEntity()

	// 添加位置组件
	ecs.AddComponent(em, entity, &components.PositionComponent{
		X: x,
		Y: y,
	})

	// 添加按钮组件
	ecs.AddComponent(em, entity, &components.ButtonComponent{
		Type:         components.ButtonTypeNineSlice,
		LeftImage:    leftImage,
		MiddleImage:  middleImage,
		RightImage:   rightImage,
		MiddleWidth:  middleWidth,
		Text:         text,
		Font:         font,
		TextColor:    textColor,
		Width:          totalWidth,  // ✅ 初始化按钮尺寸
		Height:         totalHeight, // ✅ 初始化按钮尺寸
		State:          components.UINormal,
		Enabled:        true,
		OnClick:        onClick,
		ClickSoundID:   "SOUND_BUTTONCLICK",  // 释放时播放的音效
		PressedSoundID: "SOUND_GRAVEBUTTON",  // 按下时播放的音效（墓碑样式）
	})

	// 添加 UI 组件标记（方便过滤）
	ecs.AddComponent(em, entity, &components.UIComponent{
		State: components.UINormal,
	})

	// 添加菜单按钮标记（Story 8.8 - Task 6）
	// 用于在游戏冻结期间隐藏菜单按钮
	ecs.AddComponent(em, entity, &components.MenuButtonComponent{})

	return entity, nil
}
