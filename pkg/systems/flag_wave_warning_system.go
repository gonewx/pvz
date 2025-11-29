package systems

import (
	"log"

	"github.com/decker502/pvz/internal/reanim"
	"github.com/decker502/pvz/pkg/components"
	"github.com/decker502/pvz/pkg/config"
	"github.com/decker502/pvz/pkg/ecs"
	"github.com/decker502/pvz/pkg/game"
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
//   - Phase 5 时创建 FinalWave.reanim 动画实体
//   - 播放警告音效
//   - Phase 完成后销毁实体
//
// 架构说明：
//   - 遵循 ECS 架构：系统只处理逻辑，不存储状态
//   - 复用 Story 11.3 的 FinalWaveWarningComponent 和动画资源
//   - 零耦合：通过读取 WaveTimerComponent 获取状态
type FlagWaveWarningSystem struct {
	entityManager    *ecs.EntityManager
	waveTimingSystem *WaveTimingSystem
	resourceManager  *game.ResourceManager

	// warningEntityID 当前红字警告实体ID（0 表示无）
	warningEntityID ecs.EntityID
}

// NewFlagWaveWarningSystem 创建红字警告动画系统
//
// 参数：
//   - em: 实体管理器
//   - wts: 波次计时系统（用于读取阶段状态）
//   - rm: 资源管理器（用于加载动画和音效）
//
// 返回：
//   - *FlagWaveWarningSystem: 新创建的系统实例
func NewFlagWaveWarningSystem(em *ecs.EntityManager, wts *WaveTimingSystem, rm *game.ResourceManager) *FlagWaveWarningSystem {
	return &FlagWaveWarningSystem{
		entityManager:    em,
		waveTimingSystem: wts,
		resourceManager:  rm,
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
//
// 使用 FinalWave.reanim 动画（复用 Story 11.3 的资源）
// 不添加 FinalWaveWarningComponent，避免被 FinalWaveWarningSystem 干扰
// 如果资源管理器为 nil 或动画加载失败，使用纯文本回退方案
func (s *FlagWaveWarningSystem) createWarningEntity() {
	// 如果没有资源管理器，直接使用纯文本回退方案
	if s.resourceManager == nil {
		s.createTextWarningEntity()
		return
	}

	// 尝试加载 FinalWave.reanim 资源
	reanimXML := s.resourceManager.GetReanimXML("FinalWave")
	if reanimXML == nil {
		log.Printf("[FlagWaveWarningSystem] WARNING: FinalWave.reanim not found, using text fallback")
		s.createTextWarningEntity()
		return
	}

	partImages := s.resourceManager.GetReanimPartImages("FinalWave")
	if len(partImages) == 0 {
		log.Printf("[FlagWaveWarningSystem] WARNING: FinalWave images not found, using text fallback")
		s.createTextWarningEntity()
		return
	}

	// 计算屏幕中心位置
	centerX := float64(config.ScreenWidth) / 2
	centerY := float64(config.ScreenHeight) / 2

	// 创建实体
	entityID := s.entityManager.CreateEntity()
	s.warningEntityID = entityID

	// 构建 MergedTracks
	mergedTracks := reanim.BuildMergedTracks(reanimXML)

	// 提取视觉轨道列表
	visualTracks := make([]string, 0, len(reanimXML.Tracks))
	for _, track := range reanimXML.Tracks {
		visualTracks = append(visualTracks, track.Name)
	}

	// 创建 ReanimComponent
	reanimComp := &components.ReanimComponent{
		ReanimName:        "FinalWave",
		ReanimXML:         reanimXML,
		PartImages:        partImages,
		MergedTracks:      mergedTracks,
		VisualTracks:      visualTracks,
		LogicalTracks:     []string{},
		CurrentFrame:      0,
		FrameAccumulator:  0.0,
		AnimationFPS:      float64(reanimXML.FPS),
		CurrentAnimations: []string{},
		AnimVisiblesMap:   make(map[string][]int),
		IsLooping:         false,
		ParentTracks:      nil,
		HiddenTracks:      nil,
	}

	// 添加位置组件
	posComp := &components.PositionComponent{
		X: centerX,
		Y: centerY,
	}

	// 添加 UI 组件（标记为 UI 元素，不受摄像机影响）
	uiComp := &components.UIComponent{
		State: components.UINormal,
	}

	// 添加所有组件（不添加 FinalWaveWarningComponent！）
	ecs.AddComponent(s.entityManager, entityID, reanimComp)
	ecs.AddComponent(s.entityManager, entityID, posComp)
	ecs.AddComponent(s.entityManager, entityID, uiComp)

	// 添加动画命令组件，播放 warning combo
	ecs.AddComponent(s.entityManager, entityID, &components.AnimationCommandComponent{
		UnitID:    "finalwave",
		ComboName: "warning",
		Processed: false,
	})

	// 播放音效：SOUND_AWOOGA（僵尸来袭音效）
	if audioPlayer := s.resourceManager.GetAudioPlayer("SOUND_AWOOGA"); audioPlayer != nil {
		audioPlayer.Rewind()
		audioPlayer.Play()
		log.Printf("[FlagWaveWarningSystem] Playing SOUND_AWOOGA")
	} else {
		log.Printf("[FlagWaveWarningSystem] WARNING: SOUND_AWOOGA not loaded")
	}

	log.Printf("[FlagWaveWarningSystem] Created FinalWave reanim entity (ID: %d) at (%.0f, %.0f)", entityID, centerX, centerY)
}

// createTextWarningEntity 创建纯文本警告实体（回退方案）
//
// 当 FinalWave.reanim 加载失败时使用
func (s *FlagWaveWarningSystem) createTextWarningEntity() {
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

	log.Printf("[FlagWaveWarningSystem] Created text warning entity (ID: %d) at (%.0f, %.0f) [fallback]", entityID, centerX, centerY)
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


