package systems

import (
	"log"
	"sort"

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
			s.destroyDialogAndChildren(entityID)
		}
		return
	}

	// 检测鼠标左键点击
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		mouseX, mouseY := ebiten.CursorPosition()
		log.Printf("[DialogInputSystem] 检测到鼠标点击: (%d, %d)", mouseX, mouseY)

		// ✅ Story 12.4: 按 ID 倒序排序，优先处理最上层（ID 最大）的对话框
		sort.Slice(dialogEntities, func(i, j int) bool {
			return dialogEntities[i] > dialogEntities[j]
		})

		// 只处理第一个（最上层）对话框
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
			clickedButton := s.getClickedButton(mouseX, mouseY, dialogComp, posComp.X, posComp.Y)
			if clickedButton != nil {
				// 触发按钮回调
				log.Printf("[DialogInputSystem] ✅ 点击了按钮 '%s'", clickedButton.Label)
				if clickedButton.OnClick != nil {
					clickedButton.OnClick()
				}

				// 根据 AutoClose 标志决定是否关闭对话框
				// AutoClose 默认为 false，所以需要显式检查
				if dialogComp.AutoClose {
					log.Printf("[DialogInputSystem] AutoClose=true, 关闭对话框 %d", entityID)
					s.destroyDialogAndChildren(entityID)
				} else {
					log.Printf("[DialogInputSystem] AutoClose=false, 保持对话框打开")
				}
				return
			}

			// 检查是否点击了对话框外部
			if !s.isClickInDialog(mouseX, mouseY, dialogComp, posComp.X, posComp.Y) {
				// 点击了对话框外部，关闭对话框
				log.Printf("[DialogInputSystem] 点击了对话框外部，关闭对话框 %d", entityID)
				s.destroyDialogAndChildren(entityID)
				return
			}

			log.Printf("[DialogInputSystem] 点击在对话框内部但不在按钮上")
		}
	}
}

// destroyDialogAndChildren 销毁对话框及其所有子实体
func (s *DialogInputSystem) destroyDialogAndChildren(dialogID ecs.EntityID) {
	// 获取对话框组件
	dialogComp, ok := ecs.GetComponent[*components.DialogComponent](s.entityManager, dialogID)
	if ok && len(dialogComp.ChildEntities) > 0 {
		// 先销毁所有子实体
		for _, childID := range dialogComp.ChildEntities {
			log.Printf("[DialogInputSystem] 销毁子实体 %d", childID)
			s.entityManager.DestroyEntity(childID)
		}
	}

	// 最后销毁对话框本身
	s.entityManager.DestroyEntity(dialogID)
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

// getClickedButton 获取被点击的按钮（如果有）
func (s *DialogInputSystem) getClickedButton(mouseX, mouseY int, dialog *components.DialogComponent, dialogX, dialogY float64) *components.DialogButton {
	mx := float64(mouseX)
	my := float64(mouseY)

	for i := range dialog.Buttons {
		btn := &dialog.Buttons[i]
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
			return btn
		}
	}

	return nil
}

// isClickOnButton 判断点击是否在按钮上（已废弃，使用 getClickedButton 代替）
func (s *DialogInputSystem) isClickOnButton(mouseX, mouseY int, dialog *components.DialogComponent, dialogX, dialogY float64) bool {
	return s.getClickedButton(mouseX, mouseY, dialog, dialogX, dialogY) != nil
}
