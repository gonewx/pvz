package entities

import (
	"log"

	"github.com/gonewx/pvz/pkg/components"
	"github.com/gonewx/pvz/pkg/ecs"
	"github.com/gonewx/pvz/pkg/game"
)

// 虚拟键盘布局常量
const (
	// 键盘尺寸配置（相对于 800x600 分辨率）
	VirtualKeyboardHeight    = 200.0 // 键盘总高度
	VirtualKeyboardKeyWidth  = 65.0  // 标准按键宽度
	VirtualKeyboardKeyHeight = 42.0  // 按键高度
	VirtualKeyboardSpacing   = 5.0   // 按键间距
	VirtualKeyboardPadding   = 10.0  // 键盘边距
)

// NewVirtualKeyboardEntity 创建虚拟键盘实体
// 仅在移动端创建，桌面端不需要虚拟键盘
//
// 参数：
//   - em: 实体管理器
//   - rm: 资源管理器（用于加载字体）
//   - screenWidth: 屏幕宽度
//   - screenHeight: 屏幕高度
//
// 返回：
//   - ecs.EntityID: 虚拟键盘实体ID
//   - error: 创建失败时返回错误
func NewVirtualKeyboardEntity(em *ecs.EntityManager, rm *game.ResourceManager, screenWidth, screenHeight float64) (ecs.EntityID, error) {
	entity := em.CreateEntity()

	// 计算键盘布局参数
	keyboardY := screenHeight - VirtualKeyboardHeight - VirtualKeyboardPadding
	keyboardX := VirtualKeyboardPadding

	// 创建虚拟键盘组件
	keyboardComp := &components.VirtualKeyboardComponent{
		IsVisible:         false, // 初始不可见
		ShiftActive:       false,
		NumericMode:       false,
		KeyStates:         make(map[string]bool),
		PressedKey:        "",
		PressedTimer:      0,
		TargetInputEntity: 0,
		KeyWidth:          VirtualKeyboardKeyWidth,
		KeyHeight:         VirtualKeyboardKeyHeight,
		KeySpacing:        VirtualKeyboardSpacing,
		KeyboardY:         keyboardY,
		KeyboardX:         keyboardX,
		ScreenWidth:       screenWidth,
		ScreenHeight:      screenHeight,
	}

	// 添加组件
	ecs.AddComponent(em, entity, keyboardComp)

	// 添加 UI 组件标记（不受相机影响）
	ecs.AddComponent(em, entity, &components.UIComponent{})

	log.Printf("[VirtualKeyboardFactory] Created virtual keyboard entity (ID=%d, keyboardY=%.1f)", entity, keyboardY)

	return entity, nil
}

// CalculateKeyboardLayout 计算当前模式下的键盘布局
// 返回所有按键的位置和尺寸信息
//
// 参数：
//   - kb: 虚拟键盘组件
//
// 返回：
//   - [][]KeyInfo: 二维数组，每行的按键信息
func CalculateKeyboardLayout(kb *components.VirtualKeyboardComponent) [][]components.KeyInfo {
	// 选择当前布局
	var layout [][]string
	if kb.NumericMode {
		layout = components.KeyboardLayoutNumeric
	} else if kb.ShiftActive {
		layout = components.KeyboardLayoutUpper
	} else {
		layout = components.KeyboardLayoutLower
	}

	result := make([][]components.KeyInfo, len(layout))

	// 计算每行的起始 Y 坐标
	rowY := kb.KeyboardY

	for rowIdx, row := range layout {
		rowKeys := make([]components.KeyInfo, len(row))

		// 计算这一行的总宽度（用于居中）
		totalWidth := 0.0
		for _, keyAction := range row {
			widthFactor := components.GetKeyWidthFactor(keyAction)
			totalWidth += kb.KeyWidth * widthFactor
		}
		totalWidth += float64(len(row)-1) * kb.KeySpacing

		// 计算起始 X（居中）
		startX := (kb.ScreenWidth - totalWidth) / 2
		keyX := startX

		for keyIdx, keyAction := range row {
			widthFactor := components.GetKeyWidthFactor(keyAction)
			keyWidth := kb.KeyWidth * widthFactor

			rowKeys[keyIdx] = components.KeyInfo{
				Label:       components.GetKeyLabel(keyAction),
				Action:      keyAction,
				X:           keyX,
				Y:           rowY,
				Width:       keyWidth,
				Height:      kb.KeyHeight,
				WidthFactor: widthFactor,
			}

			keyX += keyWidth + kb.KeySpacing
		}

		result[rowIdx] = rowKeys
		rowY += kb.KeyHeight + kb.KeySpacing
	}

	return result
}

// GetAllKeys 获取当前布局的所有按键（扁平化列表）
func GetAllKeys(kb *components.VirtualKeyboardComponent) []components.KeyInfo {
	layout := CalculateKeyboardLayout(kb)
	var allKeys []components.KeyInfo
	for _, row := range layout {
		allKeys = append(allKeys, row...)
	}
	return allKeys
}
