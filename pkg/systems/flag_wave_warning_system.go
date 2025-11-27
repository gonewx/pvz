package systems

import (
	"log"

	"github.com/decker502/pvz/pkg/components"
	"github.com/decker502/pvz/pkg/config"
	"github.com/decker502/pvz/pkg/ecs"
)

// 红字警告动画常量
const (
	// FlagWaveWarningInitialScale 红字初始缩放
	FlagWaveWarningInitialScale = 2.0

	// FlagWaveWarningFinalScale 红字最终缩放
	FlagWaveWarningFinalScale = 1.0

	// FlagWaveWarningScaleDurationCs 缩放动画时长（厘秒）
	FlagWaveWarningScaleDurationCs = 30

	// FlagWaveWarningFlashCycleCs 闪烁周期（厘秒）
	FlagWaveWarningFlashCycleCs = 15
)

// FlagWaveWarningSystem 红字警告动画管理系统
//
// Story 17.7: 旗帜波红字警告动画系统
//
// 职责：
//   - 监控 WaveTimerComponent.FlagWaveCountdownPhase
//   - Phase 5 时创建红字警告实体
//   - 管理动画生命周期（缩放、闪烁效果）
//   - Phase 完成后销毁实体
//
// 架构说明：
//   - 遵循 ECS 架构：系统只处理逻辑，不存储状态
//   - 通过 FlagWaveWarningComponent 管理动画生命周期
//   - 零耦合：通过读取 WaveTimerComponent 获取状态
type FlagWaveWarningSystem struct {
	entityManager    *ecs.EntityManager
	waveTimingSystem *WaveTimingSystem

	// warningEntityID 当前红字警告实体ID（0 表示无）
	warningEntityID ecs.EntityID
}

// NewFlagWaveWarningSystem 创建红字警告动画系统
//
// 参数：
//   - em: 实体管理器
//   - wts: 波次计时系统（用于读取阶段状态）
//
// 返回：
//   - *FlagWaveWarningSystem: 新创建的系统实例
func NewFlagWaveWarningSystem(em *ecs.EntityManager, wts *WaveTimingSystem) *FlagWaveWarningSystem {
	return &FlagWaveWarningSystem{
		entityManager:    em,
		waveTimingSystem: wts,
		warningEntityID:  0,
	}
}

// Update 更新红字警告系统
//
// 执行流程：
//  1. 检查 WaveTimingSystem 的警告阶段
//  2. Phase 5 时创建警告实体（如果不存在）
//  3. 更新动画状态（缩放、闪烁）
//  4. Phase 结束后销毁实体
//
// 参数：
//   - deltaTime: 自上一帧以来经过的时间（秒）
func (s *FlagWaveWarningSystem) Update(deltaTime float64) {
	if s.waveTimingSystem == nil {
		return
	}

	phase := s.waveTimingSystem.GetFlagWaveWarningPhase()

	// 检查是否需要创建警告实体
	if phase > 0 && s.warningEntityID == 0 {
		s.createWarningEntity()
	}

	// 检查是否需要销毁警告实体
	if phase == 0 && s.warningEntityID != 0 {
		s.destroyWarningEntity()
		return
	}

	// 更新现有警告实体的动画
	if s.warningEntityID != 0 {
		s.updateWarningAnimation(deltaTime)
	}
}

// createWarningEntity 创建红字警告实体
func (s *FlagWaveWarningSystem) createWarningEntity() {
	entityID := s.entityManager.CreateEntity()
	s.warningEntityID = entityID

	// 计算屏幕中心位置
	centerX := float64(config.ScreenWidth) / 2
	centerY := float64(config.ScreenHeight) / 3 // 上方 1/3 处

	// 添加警告组件
	warningComp := &components.FlagWaveWarningComponent{
		Text:            components.FlagWaveWarningText,
		Phase:           5,
		ElapsedTimeCs:   0,
		TotalDurationCs: FlagWarningTotalDurationCs,
		Scale:           FlagWaveWarningInitialScale,
		Alpha:           1.0,
		FlashTimer:      0,
		FlashVisible:    true,
		IsActive:        true,
		X:               centerX,
		Y:               centerY,
	}

	ecs.AddComponent(s.entityManager, entityID, warningComp)

	log.Printf("[FlagWaveWarningSystem] Created warning entity (ID: %d) at (%.0f, %.0f)", entityID, centerX, centerY)
}

// destroyWarningEntity 销毁红字警告实体
func (s *FlagWaveWarningSystem) destroyWarningEntity() {
	if s.warningEntityID == 0 {
		return
	}

	s.entityManager.DestroyEntity(s.warningEntityID)
	log.Printf("[FlagWaveWarningSystem] Destroyed warning entity (ID: %d)", s.warningEntityID)
	s.warningEntityID = 0
}

// updateWarningAnimation 更新警告动画
func (s *FlagWaveWarningSystem) updateWarningAnimation(deltaTime float64) {
	warningComp, ok := ecs.GetComponent[*components.FlagWaveWarningComponent](s.entityManager, s.warningEntityID)
	if !ok {
		return
	}

	// 更新已显示时间
	deltaCsInt := int(deltaTime * 100)
	warningComp.ElapsedTimeCs += deltaCsInt

	// 更新缩放动画（从 2.0 缩小到 1.0）
	if warningComp.ElapsedTimeCs < FlagWaveWarningScaleDurationCs {
		progress := float64(warningComp.ElapsedTimeCs) / float64(FlagWaveWarningScaleDurationCs)
		warningComp.Scale = FlagWaveWarningInitialScale - (FlagWaveWarningInitialScale-FlagWaveWarningFinalScale)*progress
	} else {
		warningComp.Scale = FlagWaveWarningFinalScale
	}

	// 更新闪烁效果
	warningComp.FlashTimer += deltaTime * 100
	if warningComp.FlashTimer >= FlagWaveWarningFlashCycleCs {
		warningComp.FlashTimer -= FlagWaveWarningFlashCycleCs
		warningComp.FlashVisible = !warningComp.FlashVisible
	}

	// 更新阶段
	phase := s.waveTimingSystem.GetFlagWaveWarningPhase()
	warningComp.Phase = phase
}

// GetWarningEntityID 获取当前警告实体ID（用于测试）
func (s *FlagWaveWarningSystem) GetWarningEntityID() ecs.EntityID {
	return s.warningEntityID
}

// IsWarningActive 检查警告是否激活
func (s *FlagWaveWarningSystem) IsWarningActive() bool {
	return s.warningEntityID != 0
}

