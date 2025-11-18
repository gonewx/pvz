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

	// ✅ 首先更新所有对话框的悬停状态（每帧都更新）
	s.updateDialogHoverStates(dialogEntities)

	// 检测 ESC 键按下
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		// 关闭���有对话框
		for _, entityID := range dialogEntities {
			s.destroyDialogAndChildren(entityID)
		}
		return
	}

	// ✅ 修改为释放时执行：检测鼠标左键释放（而不是按下）
	if inpututil.IsMouseButtonJustReleased(ebiten.MouseButtonLeft) {
		mouseX, mouseY := ebiten.CursorPosition()
		log.Printf("[DialogInputSystem] 检测到鼠标释放: (%d, %d)", mouseX, mouseY)

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

			// ✅ Story 12.4: 检查是否点击了用户列表项
			clickedUserIndex := s.getClickedUserListItem(mouseX, mouseY, entityID, dialogComp, posComp.X, posComp.Y)
			if clickedUserIndex >= 0 {
				log.Printf("[DialogInputSystem] ✅ 点击了用户���表项: %d", clickedUserIndex)

				// 获取 UserListComponent
				userList, ok := ecs.GetComponent[*components.UserListComponent](s.entityManager, entityID)
				if ok {
					// 检查是否点击了"建立一位新用户"项
					if clickedUserIndex == len(userList.Users) {
						// 点击了"建立一位新用户"，触发新建用户操作
						log.Printf("[DialogInputSystem] 点击了'建立一位新用户'，触发按钮回调")
						// 查找"好"按钮并触发回调（UserActionSwitch 会处理新建用户）
						for i := range dialogComp.Buttons {
							btn := &dialogComp.Buttons[i]
							if btn.Label == "好" {
								// 先更新选中索引
								userList.SelectedIndex = clickedUserIndex
								// 然后触发按钮回调
								if btn.OnClick != nil {
									btn.OnClick()
								}
								return
							}
						}
					} else {
						// 普通用户项，只更新选中索引
						userList.SelectedIndex = clickedUserIndex
						log.Printf("[DialogInputSystem] 更新选中索引为: %d", clickedUserIndex)
					}
				}
				return
			}

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
				log.Printf("[DialogInputSystem] 点击了对话框外部")

				// 如果对话框设置了 AutoClose=false，查找"取消"按钮并触发回调
				if !dialogComp.AutoClose {
					for i := range dialogComp.Buttons {
						btn := &dialogComp.Buttons[i]
						if btn.Label == "取消" {
							log.Printf("[DialogInputSystem] 触发'取消'按钮回调")
							if btn.OnClick != nil {
								btn.OnClick()
							}
							return
						}
					}
				}

				// 如果没有"取消"按钮或 AutoClose=true，直接销毁对话框
				log.Printf("[DialogInputSystem] 直接关闭对话框 %d", entityID)
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

// getClickedUserListItem 获取被点击的用户列表项索引（Story 12.4）
// 返回值：
//   - -1: 没有点击列表项
//   - 0 ~ len(users)-1: 点击了普通用户项
//   - len(users): 点击了"建立一位新用户"项
func (s *DialogInputSystem) getClickedUserListItem(mouseX, mouseY int, entityID ecs.EntityID, dialog *components.DialogComponent, dialogX, dialogY float64) int {
	// 检查是否有 UserListComponent
	userList, ok := ecs.GetComponent[*components.UserListComponent](s.entityManager, entityID)
	if !ok {
		return -1
	}

	mx := float64(mouseX)
	my := float64(mouseY)

	// 列表区域配置（使用统一常量）
	listX := dialogX + components.UserListPadding
	listWidth := dialog.Width - components.UserListPadding*2
	itemHeight := userList.ItemHeight

	// 检查每个用户列表项
	for i := range userList.Users {
		itemY := dialogY + components.UserListStartY + float64(i)*itemHeight

		if mx >= listX &&
			mx <= listX+listWidth &&
			my >= itemY &&
			my <= itemY+itemHeight {
			log.Printf("[DialogInputSystem] ✅ 点击了用户列表项 %d", i)
			return i
		}
	}

	// 检查"建立一位新用户"项
	newUserIndex := len(userList.Users)
	newUserItemY := dialogY + components.UserListStartY + float64(newUserIndex)*itemHeight

	if mx >= listX &&
		mx <= listX+listWidth &&
		my >= newUserItemY &&
		my <= newUserItemY+itemHeight {
		log.Printf("[DialogInputSystem] ✅ 点击了'建立一位新用户'")
		return newUserIndex
	}

	return -1
}

// updateDialogHoverStates 更新所有对话框的悬停状态和按下状态
// 每帧调用,负责更新用户列表的 HoveredIndex 和对话框按钮的 HoveredButtonIdx 和 PressedButtonIdx
func (s *DialogInputSystem) updateDialogHoverStates(dialogEntities []ecs.EntityID) {
	mouseX, mouseY := ebiten.CursorPosition()
	mx := float64(mouseX)
	my := float64(mouseY)

	// ✅ 检测鼠标左键是否按下
	isMousePressed := ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft)

	// 遍历所有对话框
	for _, entityID := range dialogEntities {
		dialogComp, ok1 := ecs.GetComponent[*components.DialogComponent](s.entityManager, entityID)
		posComp, ok2 := ecs.GetComponent[*components.PositionComponent](s.entityManager, entityID)

		if !ok1 || !ok2 || !dialogComp.IsVisible {
			continue
		}

		dialogX := posComp.X
		dialogY := posComp.Y

		// ✅ 修复：先检查按钮悬停（支持按钮超出对话框边界的情况）
		hoveredBtnIdx := -1
		for i := range dialogComp.Buttons {
			btn := &dialogComp.Buttons[i]
			btnX := dialogX + btn.X
			btnY := dialogY + btn.Y

			if mx >= btnX && mx <= btnX+btn.Width &&
				my >= btnY && my <= btnY+btn.Height {
				hoveredBtnIdx = i
				break
			}
		}
		dialogComp.HoveredButtonIdx = hoveredBtnIdx

		// ✅ 更新按钮按下状态（只有悬停+鼠标按下时才设置 PressedButtonIdx）
		if isMousePressed && hoveredBtnIdx >= 0 {
			dialogComp.PressedButtonIdx = hoveredBtnIdx
		} else {
			dialogComp.PressedButtonIdx = -1
		}

		// 检查鼠标是否在对话框内（用于用户列表检测）
		isInDialog := mx >= dialogX && mx <= dialogX+dialogComp.Width &&
			my >= dialogY && my <= dialogY+dialogComp.Height

		if isInDialog {
			// ✅ 更新用户列表悬停状态（如果有）
			userList, ok := ecs.GetComponent[*components.UserListComponent](s.entityManager, entityID)
			if ok {
				// 鼠标在对话框内,检测悬停项
				listX := dialogX + components.UserListPadding
				listWidth := dialogComp.Width - components.UserListPadding*2
				itemHeight := userList.ItemHeight

				hoveredIndex := -1
				totalItems := len(userList.Users) + 1 // +1 for "建立一位新用户"

				for i := 0; i < totalItems; i++ {
					itemY := dialogY + components.UserListStartY + float64(i)*itemHeight

					if mx >= listX && mx <= listX+listWidth &&
						my >= itemY && my <= itemY+itemHeight {
						hoveredIndex = i
						break
					}
				}

				userList.HoveredIndex = hoveredIndex
			}
		} else {
			// 鼠标不在对话框内,重置用户列表悬停状态
			userList, ok := ecs.GetComponent[*components.UserListComponent](s.entityManager, entityID)
			if ok {
				userList.HoveredIndex = -1
			}
		}
	}
}
