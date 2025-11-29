package systems

import (
	"log"

	"github.com/decker502/pvz/pkg/components"
	"github.com/decker502/pvz/pkg/config"
	"github.com/decker502/pvz/pkg/ecs"
)

// 最终波白字动画常量
const (
	// FinalWaveTextInitialScale 白字初始缩放
	FinalWaveTextInitialScale = 3.0

	// FinalWaveTextFinalScale 白字最终缩放
	FinalWaveTextFinalScale = 1.5

	// FinalWaveTextScaleDurationCs 缩放动画时长（厘秒）
	FinalWaveTextScaleDurationCs = 50

	// FinalWaveTextFadeInDurationCs 淡入动画时长（厘秒）
	FinalWaveTextFadeInDurationCs = 20
)

// FinalWaveTextSystem 最终波白字管理系统
//
// Story 17.7: 最终波 "FINAL WAVE" 白字显示系统
//
// 职责：
//   - 监控 WaveTimingSystem 的最终波状态
//   - 当最终波倒计时为 0 时创建白字实体
//   - 管理动画生命周期（缩放、淡入效果）
//   - 500cs 后标记完成
//
// 架构说明：
//   - 遵循 ECS 架构：系统只处理逻辑，不存储状态
//   - 通过 FinalWaveTextComponent 管理动画生命周期
//   - 零耦合：通过读取 WaveTimingSystem 获取状态
type FinalWaveTextSystem struct {
	entityManager    *ecs.EntityManager
	waveTimingSystem *WaveTimingSystem

	// textEntityID 当前白字实体ID（0 表示无）
	textEntityID ecs.EntityID
}

// NewFinalWaveTextSystem 创建最终波白字系统
//
// 参数：
//   - em: 实体管理器
//   - wts: 波次计时系统（用于读取状态）
//
// 返回：
//   - *FinalWaveTextSystem: 新创建的系统实例
func NewFinalWaveTextSystem(em *ecs.EntityManager, wts *WaveTimingSystem) *FinalWaveTextSystem {
	return &FinalWaveTextSystem{
		entityManager:    em,
		waveTimingSystem: wts,
		textEntityID:     0,
	}
}

// Update 更新最终波白字系统
//
// 执行流程：
//  1. 检查 WaveTimingSystem 的最终波白字状态
//  2. 激活时创建白字实体（如果不存在）
//  3. 更新动画状态（缩放、淡入）
//  4. 500cs 后标记完成
//
// 参数：
//   - deltaTime: 自上一帧以来经过的时间（秒）
func (s *FinalWaveTextSystem) Update(deltaTime float64) {
	if s.waveTimingSystem == nil {
		return
	}

	isActive := s.waveTimingSystem.IsFinalWaveTextActive()

	// 检查是否需要创建白字实体
	if isActive && s.textEntityID == 0 {
		s.createTextEntity()
	}

	// 更新现有白字实体的动画
	if s.textEntityID != 0 {
		s.updateTextAnimation(deltaTime)
	}
}

// createTextEntity 创建最终波白字实体
func (s *FinalWaveTextSystem) createTextEntity() {
	entityID := s.entityManager.CreateEntity()
	s.textEntityID = entityID

	// 计算屏幕中心位置
	centerX := float64(config.ScreenWidth) / 2
	centerY := float64(config.ScreenHeight) / 2

	// 添加白字组件
	textComp := &components.FinalWaveTextComponent{
		Text:            components.FinalWaveText,
		ElapsedTimeCs:   0,
		TotalDurationCs: FinalWaveTextDurationCs,
		Scale:           FinalWaveTextInitialScale,
		Alpha:           0.0, // 从透明开始淡入
		IsActive:        true,
		IsComplete:      false,
		X:               centerX,
		Y:               centerY,
	}

	ecs.AddComponent(s.entityManager, entityID, textComp)

	log.Printf("[FinalWaveTextSystem] Created text entity (ID: %d) at (%.0f, %.0f)", entityID, centerX, centerY)
}

// updateTextAnimation 更新白字动画
func (s *FinalWaveTextSystem) updateTextAnimation(deltaTime float64) {
	textComp, ok := ecs.GetComponent[*components.FinalWaveTextComponent](s.entityManager, s.textEntityID)
	if !ok {
		return
	}

	// 更新已显示时间
	deltaCsInt := int(deltaTime * 100)
	textComp.ElapsedTimeCs += deltaCsInt

	// 更新淡入动画（从 0 增加到 1.0）
	if textComp.ElapsedTimeCs < FinalWaveTextFadeInDurationCs {
		progress := float64(textComp.ElapsedTimeCs) / float64(FinalWaveTextFadeInDurationCs)
		textComp.Alpha = progress
	} else {
		textComp.Alpha = 1.0
	}

	// 更新缩放动画（从 3.0 缩小到 1.5）
	if textComp.ElapsedTimeCs < FinalWaveTextScaleDurationCs {
		progress := float64(textComp.ElapsedTimeCs) / float64(FinalWaveTextScaleDurationCs)
		textComp.Scale = FinalWaveTextInitialScale - (FinalWaveTextInitialScale-FinalWaveTextFinalScale)*progress
	} else {
		textComp.Scale = FinalWaveTextFinalScale
	}

	// 检查是否完成
	if textComp.ElapsedTimeCs >= textComp.TotalDurationCs && !textComp.IsComplete {
		textComp.IsComplete = true
		log.Printf("[FinalWaveTextSystem] Text display complete (500cs)")
	}
}

// DestroyTextEntity 销毁白字实体
//
// 在游戏胜利后由 LevelSystem 调用
func (s *FinalWaveTextSystem) DestroyTextEntity() {
	if s.textEntityID == 0 {
		return
	}

	s.entityManager.DestroyEntity(s.textEntityID)
	log.Printf("[FinalWaveTextSystem] Destroyed text entity (ID: %d)", s.textEntityID)
	s.textEntityID = 0
}

// GetTextEntityID 获取当前白字实体ID（用于测试）
func (s *FinalWaveTextSystem) GetTextEntityID() ecs.EntityID {
	return s.textEntityID
}

// IsTextActive 检查白字是否激活
func (s *FinalWaveTextSystem) IsTextActive() bool {
	return s.textEntityID != 0
}

// IsTextComplete 检查白字显示是否完成
func (s *FinalWaveTextSystem) IsTextComplete() bool {
	if s.textEntityID == 0 {
		return false
	}

	textComp, ok := ecs.GetComponent[*components.FinalWaveTextComponent](s.entityManager, s.textEntityID)
	if !ok {
		return false
	}

	return textComp.IsComplete
}


