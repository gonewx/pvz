// Package utils 提供通用工具函数
package utils

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

// InputState 存储当前帧的输入状态
// 用于统一处理鼠标和触摸输入
type InputState struct {
	// 是否有点击/触摸事件刚刚发生
	JustPressed bool
	// 点击/触摸位置
	X, Y int
	// 是否有活动的触摸
	IsTouching bool
}

// GetInputState 获取当前帧的输入状态
// 同时支持鼠标点击和触摸输入，优先检测触摸
func GetInputState() InputState {
	state := InputState{}

	// 首先检查触摸输入（移动设备）
	touchIDs := inpututil.AppendJustPressedTouchIDs(nil)
	if len(touchIDs) > 0 {
		// 有新的触摸事件
		state.JustPressed = true
		state.X, state.Y = ebiten.TouchPosition(touchIDs[0])
		state.IsTouching = true
		return state
	}

	// 检查是否有活动的触摸（用于悬停检测）
	allTouchIDs := ebiten.AppendTouchIDs(nil)
	if len(allTouchIDs) > 0 {
		state.X, state.Y = ebiten.TouchPosition(allTouchIDs[0])
		state.IsTouching = true
		return state
	}

	// 其次检查鼠标输入（桌面设备）
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		state.JustPressed = true
		state.X, state.Y = ebiten.CursorPosition()
		return state
	}

	// 获取鼠标位置用于悬停检测
	state.X, state.Y = ebiten.CursorPosition()
	return state
}

// IsJustTouchedOrClicked 检查是否刚刚发生点击或触摸
// 返回是否点击以及点击位置
func IsJustTouchedOrClicked() (bool, int, int) {
	// 检查触摸
	touchIDs := inpututil.AppendJustPressedTouchIDs(nil)
	if len(touchIDs) > 0 {
		x, y := ebiten.TouchPosition(touchIDs[0])
		return true, x, y
	}

	// 检查鼠标
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		x, y := ebiten.CursorPosition()
		return true, x, y
	}

	return false, 0, 0
}

// GetPointerPosition 获取当前指针位置（触摸或鼠标）
// 优先返回触摸位置，如果没有触摸则返回鼠标位置
func GetPointerPosition() (int, int) {
	// 检查触摸
	touchIDs := ebiten.AppendTouchIDs(nil)
	if len(touchIDs) > 0 {
		return ebiten.TouchPosition(touchIDs[0])
	}

	// 返回鼠标位置
	return ebiten.CursorPosition()
}

// IsPointerPressed 检查是否有指针按下（鼠标左键或触摸）
func IsPointerPressed() bool {
	// 检查触摸
	touchIDs := ebiten.AppendTouchIDs(nil)
	if len(touchIDs) > 0 {
		return true
	}

	// 检查鼠标
	return ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft)
}

// GetPointerState 获取指针的完整状态
// 返回：是否按下、X坐标、Y坐标
func GetPointerState() (pressed bool, x, y int) {
	// 检查触摸
	touchIDs := ebiten.AppendTouchIDs(nil)
	if len(touchIDs) > 0 {
		x, y = ebiten.TouchPosition(touchIDs[0])
		return true, x, y
	}

	// 检查鼠标
	x, y = ebiten.CursorPosition()
	pressed = ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft)
	return pressed, x, y
}

// IsTouchDevice 检测当前是否为触摸设备
// 通过检查是否有活动的触摸来判断
func IsTouchDevice() bool {
	touchIDs := ebiten.AppendTouchIDs(nil)
	return len(touchIDs) > 0
}

// 保存最后一次触摸位置（用于触摸释放时获取位置）
var lastTouchX, lastTouchY int

// UpdateLastTouchPosition 更新最后一次触摸位置
// 应该在每帧更新时调用
func UpdateLastTouchPosition() {
	touchIDs := ebiten.AppendTouchIDs(nil)
	if len(touchIDs) > 0 {
		lastTouchX, lastTouchY = ebiten.TouchPosition(touchIDs[0])
	}
}

// IsPointerJustReleased 检查是否刚刚释放指针（触摸或鼠标）
// 返回是否释放以及释放位置
func IsPointerJustReleased() (bool, int, int) {
	// 检查触摸释放
	releasedTouchIDs := inpututil.AppendJustReleasedTouchIDs(nil)
	if len(releasedTouchIDs) > 0 {
		// 触摸释放时使用保存的最后触摸位置
		return true, lastTouchX, lastTouchY
	}

	// 检查鼠标释放
	if inpututil.IsMouseButtonJustReleased(ebiten.MouseButtonLeft) {
		x, y := ebiten.CursorPosition()
		return true, x, y
	}

	return false, 0, 0
}

// IsPointerJustPressed 检查是否刚刚按下指针（触摸或鼠标）
// 返回是否按下以及按下位置
func IsPointerJustPressed() (bool, int, int) {
	// 检查触摸按下
	touchIDs := inpututil.AppendJustPressedTouchIDs(nil)
	if len(touchIDs) > 0 {
		x, y := ebiten.TouchPosition(touchIDs[0])
		// 同时更新最后触摸位置
		lastTouchX, lastTouchY = x, y
		return true, x, y
	}

	// 检查鼠标按下
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		x, y := ebiten.CursorPosition()
		return true, x, y
	}

	return false, 0, 0
}

// ============================================================================
// 拖拽状态管理器 - 用于移动端植物放置的拖拽交互
// ============================================================================

// DragState 拖拽状态
type DragState int

const (
	// DragStateNone 无拖拽
	DragStateNone DragState = iota
	// DragStateStarted 拖拽开始（刚按下）
	DragStateStarted
	// DragStateDragging 拖拽中（按住移动）
	DragStateDragging
	// DragStateEnded 拖拽结束（释放）
	DragStateEnded
)

// DragInfo 拖拽信息
type DragInfo struct {
	// State 当前拖拽状态
	State DragState
	// StartX, StartY 拖拽起始位置（屏幕坐标）
	StartX, StartY int
	// CurrentX, CurrentY 当前位置（屏幕坐标）
	CurrentX, CurrentY int
	// TouchID 当前跟踪的触摸ID（-1��示鼠标）
	TouchID ebiten.TouchID
	// IsTouchInput 是否为触摸输入（区分触摸和鼠标）
	IsTouchInput bool
}

// DragManager 拖拽管理器
// 跟踪触摸/鼠标的拖拽状态
type DragManager struct {
	info         DragInfo
	lastTouchIDs []ebiten.TouchID
}

// 全局拖拽管理器实例
var globalDragManager = &DragManager{
	info: DragInfo{
		State:   DragStateNone,
		TouchID: -1,
	},
}

// GetDragManager 获取全局拖拽管理器
func GetDragManager() *DragManager {
	return globalDragManager
}

// Update 更新拖拽状态（每帧调用一次）
func (dm *DragManager) Update() {
	// 获取当前触摸ID列表
	currentTouchIDs := ebiten.AppendTouchIDs(nil)

	switch dm.info.State {
	case DragStateNone:
		// 检测新的拖拽开始
		dm.checkDragStart(currentTouchIDs)

	case DragStateStarted:
		// 从开始状态转换到拖拽中
		dm.info.State = DragStateDragging
		dm.updateCurrentPosition(currentTouchIDs)

	case DragStateDragging:
		// 检测拖拽结束或更新位置
		if dm.checkDragEnd(currentTouchIDs) {
			dm.info.State = DragStateEnded
		} else {
			dm.updateCurrentPosition(currentTouchIDs)
		}

	case DragStateEnded:
		// 结束状态只持续一帧，下一帧重置
		dm.Reset()
	}

	// 保存当前触摸ID列表用于下一帧比较
	dm.lastTouchIDs = make([]ebiten.TouchID, len(currentTouchIDs))
	copy(dm.lastTouchIDs, currentTouchIDs)
}

// checkDragStart 检测拖拽开始
func (dm *DragManager) checkDragStart(currentTouchIDs []ebiten.TouchID) {
	// 优先检测触摸输入
	justPressedTouchIDs := inpututil.AppendJustPressedTouchIDs(nil)
	if len(justPressedTouchIDs) > 0 {
		touchID := justPressedTouchIDs[0]
		x, y := ebiten.TouchPosition(touchID)
		dm.info = DragInfo{
			State:        DragStateStarted,
			StartX:       x,
			StartY:       y,
			CurrentX:     x,
			CurrentY:     y,
			TouchID:      touchID,
			IsTouchInput: true,
		}
		return
	}

	// 检测鼠标输入
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		x, y := ebiten.CursorPosition()
		dm.info = DragInfo{
			State:        DragStateStarted,
			StartX:       x,
			StartY:       y,
			CurrentX:     x,
			CurrentY:     y,
			TouchID:      -1,
			IsTouchInput: false,
		}
	}
}

// checkDragEnd 检测拖���结束
func (dm *DragManager) checkDragEnd(currentTouchIDs []ebiten.TouchID) bool {
	if dm.info.IsTouchInput {
		// 检测触摸释放
		for _, id := range currentTouchIDs {
			if id == dm.info.TouchID {
				return false // 触摸仍然活跃
			}
		}
		return true // 触摸已释放
	}

	// 检测鼠标释放
	return !ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft)
}

// updateCurrentPosition 更新当前位置
func (dm *DragManager) updateCurrentPosition(currentTouchIDs []ebiten.TouchID) {
	if dm.info.IsTouchInput {
		// 更新触摸位置
		for _, id := range currentTouchIDs {
			if id == dm.info.TouchID {
				dm.info.CurrentX, dm.info.CurrentY = ebiten.TouchPosition(id)
				return
			}
		}
	} else {
		// 更新鼠标位置
		dm.info.CurrentX, dm.info.CurrentY = ebiten.CursorPosition()
	}
}

// Reset 重置拖拽状态
func (dm *DragManager) Reset() {
	dm.info = DragInfo{
		State:   DragStateNone,
		TouchID: -1,
	}
}

// GetState 获取当前拖拽状态
func (dm *DragManager) GetState() DragState {
	return dm.info.State
}

// GetInfo 获取完整拖拽信息
func (dm *DragManager) GetInfo() DragInfo {
	return dm.info
}

// IsDragging 是否正在拖拽
func (dm *DragManager) IsDragging() bool {
	return dm.info.State == DragStateDragging
}

// JustStarted 是否刚开始拖拽（本帧）
func (dm *DragManager) JustStarted() bool {
	return dm.info.State == DragStateStarted
}

// JustEnded 是否刚结束拖拽（本帧）
func (dm *DragManager) JustEnded() bool {
	return dm.info.State == DragStateEnded
}

// GetDragDistance 获取拖拽距离（从起点到当前位置）
func (dm *DragManager) GetDragDistance() (dx, dy int) {
	return dm.info.CurrentX - dm.info.StartX, dm.info.CurrentY - dm.info.StartY
}

// IsTouchDrag 是否为触摸拖拽
func (dm *DragManager) IsTouchDrag() bool {
	return dm.info.IsTouchInput
}
