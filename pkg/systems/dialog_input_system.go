package systems

import (
	"log"

	"github.com/decker502/pvz/pkg/components"
	"github.com/decker502/pvz/pkg/ecs"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

// DialogInputSystem 对话框输入系统
// 负责处理对话框的用户交互
//
// 职责：
//   - 检测鼠标点击
//   - 检测 ESC 键按下
//   - 判断点击位置（按钮/对话框外部）
//   - 关闭对话框（销毁实体）
type DialogInputSystem struct {
	entityManager *ecs.EntityManager
}

// NewDialogInputSystem 创建对话框输入系统
func NewDialogInputSystem(em *ecs.EntityManager) *DialogInputSystem {
	return &DialogInputSystem{
		entityManager: em,
	}
}

// Update 更新对话框输入处理
func (s *DialogInputSystem) Update(deltaTime float64) {
	// 查询所有对话框实体
	dialogEntities := ecs.GetEntitiesWith2[*components.DialogComponent, *components.PositionComponent](s.entityManager)

	if len(dialogEntities) == 0 {
		return
	}

	// 检测 ESC 键按下
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		// 关闭所有对话框
		for _, entityID := range dialogEntities {
			s.entityManager.DestroyEntity(entityID)
		}
		return
	}

	// 检测鼠标左键点击
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		mouseX, mouseY := ebiten.CursorPosition()
		log.Printf("[DialogInputSystem] 检测到鼠标点击: (%d, %d)", mouseX, mouseY)

		for _, entityID := range dialogEntities {
			dialogComp, ok := ecs.GetComponent[*components.DialogComponent](s.entityManager, entityID)
			if !ok {
				continue
			}

			posComp, ok := ecs.GetComponent[*components.PositionComponent](s.entityManager, entityID)
			if !ok {
				continue
			}

			if !dialogComp.IsVisible {
				continue
			}

			log.Printf("[DialogInputSystem] 对话框位置: (%.0f, %.0f), 大小: (%.0f, %.0f)",
				posComp.X, posComp.Y, dialogComp.Width, dialogComp.Height)

			// 检查是否点击了按钮
			if s.isClickOnButton(mouseX, mouseY, dialogComp, posComp.X, posComp.Y) {
				// 点击了按钮，关闭对话框
				log.Printf("[DialogInputSystem] ✅ 点击了按钮，关闭对话框 %d", entityID)
				s.entityManager.DestroyEntity(entityID)
				return
			}

			// 检查是否点击了对话框外部
			if !s.isClickInDialog(mouseX, mouseY, dialogComp, posComp.X, posComp.Y) {
				// 点击了对话框外部，关闭对话框
				log.Printf("[DialogInputSystem] 点击了对话框外部，关闭对话框 %d", entityID)
				s.entityManager.DestroyEntity(entityID)
				return
			}

			log.Printf("[DialogInputSystem] 点击在对话框内部但不在按钮上")
		}
	}
}

// isClickInDialog 判断点击是否在对话框内部
func (s *DialogInputSystem) isClickInDialog(mouseX, mouseY int, dialog *components.DialogComponent, dialogX, dialogY float64) bool {
	mx := float64(mouseX)
	my := float64(mouseY)

	return mx >= dialogX &&
		mx <= dialogX+dialog.Width &&
		my >= dialogY &&
		my <= dialogY+dialog.Height
}

// isClickOnButton 判断点击是否在按钮上
func (s *DialogInputSystem) isClickOnButton(mouseX, mouseY int, dialog *components.DialogComponent, dialogX, dialogY float64) bool {
	mx := float64(mouseX)
	my := float64(mouseY)

	for _, btn := range dialog.Buttons {
		// 按钮绝对位置
		btnX := dialogX + btn.X
		btnY := dialogY + btn.Y

		log.Printf("[DialogInputSystem] 检查按钮 '%s': 位置=(%.0f, %.0f), 大小=(%.0f, %.0f), 鼠标=(%.0f, %.0f)",
			btn.Label, btnX, btnY, btn.Width, btn.Height, mx, my)

		if mx >= btnX &&
			mx <= btnX+btn.Width &&
			my >= btnY &&
			my <= btnY+btn.Height {
			log.Printf("[DialogInputSystem] ✅ 鼠标在按钮 '%s' 上", btn.Label)
			return true
		}
	}

	return false
}
