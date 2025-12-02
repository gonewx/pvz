package systems

import (
	"image/color"
	"log"

	"github.com/decker502/pvz/internal/reanim"
	"github.com/decker502/pvz/pkg/components"
	"github.com/decker502/pvz/pkg/config"
	"github.com/decker502/pvz/pkg/ecs"
	"github.com/decker502/pvz/pkg/entities"
	"github.com/decker502/pvz/pkg/game"
	"github.com/decker502/pvz/pkg/utils"
	"github.com/hajimehoshi/ebiten/v2"
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
//   - Phase 5 时使用 HouseofTerror28 位图字体生成红色警告文字
//   - 播放警告音效
//   - Phase 完成后销毁实体
//   - Bug Fix: 支持警告队列，顺序显示「一大波僵尸」和「最后一波」
//
// 架构说明：
//   - 遵循 ECS 架构：系统只处理逻辑，不存储状态
//   - Story 17.7 补充任务：使用 HouseofTerror28 位图字体动态渲染红色文字
//   - 零耦合：通过读取 WaveTimerComponent 获取状态
type FlagWaveWarningSystem struct {
	entityManager    *ecs.EntityManager
	waveTimingSystem *WaveTimingSystem
	resourceManager  *game.ResourceManager

	// warningEntityID 当前红字警告实体ID（0 表示无）
	warningEntityID ecs.EntityID

	// finalWaveEntityID 当前最终波警告实体ID（0 表示无）
	// Bug Fix: 用于支持旗帜波+最终波的顺序显示
	finalWaveEntityID ecs.EntityID

	// bitmapFont HouseofTerror28 位图字体（缓存）
	// Story 17.7 补充任务：用于渲染「一大波僵尸正在接近!」红色文字
	bitmapFont *utils.BitmapFont

	// bitmapFontLoaded 标记是否已尝试加载位图字体
	bitmapFontLoaded bool
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
		entityManager:     em,
		waveTimingSystem:  wts,
		resourceManager:   rm,
		warningEntityID:   0,
		finalWaveEntityID: 0,
		bitmapFont:        nil,
		bitmapFontLoaded:  false,
	}
}

// Update 更新红字警告系统
//
// 执行流程：
//  1. 检查 WaveTimingSystem 的警告阶段
//  2. Phase 5 时创建警告实体（如果不存在）
//  3. 更新动画状态（缩放、闪烁）
//  4. Phase 结束后销毁实体
//  5. Bug Fix: 检查警告队列，如有「最后一波」则继续显示
//
// 参数：
//   - deltaTime: 自上一帧以来经过的时间（秒）
func (s *FlagWaveWarningSystem) Update(deltaTime float64) {
	if s.waveTimingSystem == nil {
		return
	}

	// 先处理最终波警告实体的更新
	if s.finalWaveEntityID != 0 {
		s.updateFinalWaveWarning(deltaTime)
	}

	phase := s.waveTimingSystem.GetFlagWaveWarningPhase()

	// 检查是否需要创建红字警告实体
	if phase > 0 && s.warningEntityID == 0 {
		s.createWarningEntity()
	}

	// 检查是否需要销毁红字警告实体
	if phase == 0 && s.warningEntityID != 0 {
		s.destroyWarningEntity()

		// Bug Fix: 红字警告完成后，检查是否还有「最后一波」待显示
		s.checkAndTriggerFinalWaveWarning()
		return
	}

	// 更新现有红字警告实体的动画
	if s.warningEntityID != 0 {
		s.updateWarningAnimation(deltaTime)
	}
}

// checkAndTriggerFinalWaveWarning 检查并触发最终波白字警告
//
// Bug Fix: 当红字警告完成后，检查警告队列中是否有「最后一波」
// 如果有，推进队列并创建最终波警告实体
func (s *FlagWaveWarningSystem) checkAndTriggerFinalWaveWarning() {
	// 推进警告队列（刚完成的是 huge_wave）
	s.waveTimingSystem.AdvanceWarningQueue()

	// 检查下一个警告是否为 final_wave
	currentWarning := s.waveTimingSystem.GetCurrentWarning()
	if currentWarning != "final_wave" {
		return
	}

	// 创建最终波警告实体
	s.createFinalWaveWarningEntity()
}

// createFinalWaveWarningEntity 创建最终波白字警告实体
//
// Bug Fix: 使用 NewFinalWaveWarningEntity 创建「最后一波」提示
func (s *FlagWaveWarningSystem) createFinalWaveWarningEntity() {
	if s.resourceManager == nil {
		log.Printf("[FlagWaveWarningSystem] Cannot create final wave warning: no resource manager")
		return
	}

	centerX := float64(config.ScreenWidth) / 2
	centerY := float64(config.ScreenHeight) / 2

	entityID, err := entities.NewFinalWaveWarningEntity(
		s.entityManager,
		s.resourceManager,
		centerX,
		centerY,
	)

	if err != nil {
		log.Printf("[FlagWaveWarningSystem] ERROR: Failed to create FinalWave warning entity: %v", err)
		return
	}

	s.finalWaveEntityID = entityID

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
		log.Printf("[FlagWaveWarningSystem] Playing SOUND_AWOOGA for final wave")
	}

	log.Printf("[FlagWaveWarningSystem] Created final wave warning entity (ID: %d) - queued after huge wave",
		entityID)
}

// updateFinalWaveWarning 更新最终波警告动画
//
// Bug Fix: 检查 FinalWaveWarningComponent 的显示时间，超时则销毁
func (s *FlagWaveWarningSystem) updateFinalWaveWarning(deltaTime float64) {
	if s.finalWaveEntityID == 0 {
		return
	}

	warningComp, ok := ecs.GetComponent[*components.FinalWaveWarningComponent](s.entityManager, s.finalWaveEntityID)
	if !ok {
		// 组件不存在，可能已被其他系统销毁
		s.finalWaveEntityID = 0
		return
	}

	// 更新已显示时间
	warningComp.ElapsedTime += deltaTime

	// 检查是否超过显示时长
	if warningComp.ElapsedTime >= warningComp.DisplayTime {
		s.entityManager.DestroyEntity(s.finalWaveEntityID)
		log.Printf("[FlagWaveWarningSystem] Final wave warning completed, entity %d destroyed", s.finalWaveEntityID)
		s.finalWaveEntityID = 0

		// 推进警告队列
		s.waveTimingSystem.AdvanceWarningQueue()
	}
}

// createWarningEntity 创建红字警告实体
//
// Story 17.7 补充任务：复用 FinalWave.reanim 动画定义，替换图片为位图字体渲染的红色文字
// 1. 加载 FinalWave.reanim 动画参数（缩放、位移、透明度变化）
// 2. 用 HouseofTerror28 位图字体渲染「一大波僵尸正在接近!」红色文字
// 3. 替换 IMAGE_REANIM_FINALWAVE 为动态生成的红色文字图片
// 4. 创建带 ReanimComponent 的实体，让 ReanimSystem 播放动画
func (s *FlagWaveWarningSystem) createWarningEntity() {
	// 如果没有资源管理器，使用纯文本回退方案
	if s.resourceManager == nil {
		s.createTextWarningEntity()
		return
	}

	// 加载 FinalWave.reanim 动画定义
	reanimXML := s.resourceManager.GetReanimXML("FinalWave")
	if reanimXML == nil {
		log.Printf("[FlagWaveWarningSystem] WARNING: FinalWave.reanim not found, using text fallback")
		s.createTextWarningEntity()
		return
	}

	// 渲染红色警告文字图片
	textImage := s.renderWarningTextImage()
	if textImage == nil {
		log.Printf("[FlagWaveWarningSystem] WARNING: Failed to render text image, using text fallback")
		s.createTextWarningEntity()
		return
	}

	// 创建 partImages，用红色文字图片替换 IMAGE_REANIM_FINALWAVE
	partImages := map[string]*ebiten.Image{
		"IMAGE_REANIM_FINALWAVE": textImage,
	}

	// 计算实体位置
	// FinalWave.reanim 动画最终帧的相对位置是 (220.1, 260.1)
	// 为了让动画最终帧居中显示在屏幕中心，需要补偿这个偏移
	// 同时考虑图片尺寸（图片锚点在左上角，需要偏移半个图片尺寸）
	imgWidth := float64(textImage.Bounds().Dx())
	imgHeight := float64(textImage.Bounds().Dy())
	centerX := float64(config.ScreenWidth)/2 - imgWidth/2
	centerY := float64(config.ScreenHeight)/2 - imgHeight/2

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

	// 添加所有组件
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

	log.Printf("[FlagWaveWarningSystem] Created huge wave warning entity (ID: %d) at (%.0f, %.0f) with custom text image",
		entityID, centerX, centerY)
}

// createTextWarningEntity 创建纯文本警告实体（回退方案）
//
// Story 17.7 补充任务：使用 HouseofTerror28 位图字体渲染红色文字
// 当 FinalWave.reanim 加载失败时使用
func (s *FlagWaveWarningSystem) createTextWarningEntity() {
	entityID := s.entityManager.CreateEntity()
	s.warningEntityID = entityID

	// 计算屏幕中心位置（水平和垂直居中）
	centerX := float64(config.ScreenWidth) / 2
	centerY := float64(config.ScreenHeight) / 2

	// 尝试使用位图字体渲染红色文字图片
	var textImage *ebiten.Image
	textImage = s.renderWarningTextImage()

	// 添加警告组件
	warningComp := &components.FlagWaveWarningComponent{
		Text:            components.FlagWaveWarningText,
		TextImage:       textImage,
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

	if textImage != nil {
		log.Printf("[FlagWaveWarningSystem] Created text warning entity (ID: %d) with bitmap font at (%.0f, %.0f)", entityID, centerX, centerY)
	} else {
		log.Printf("[FlagWaveWarningSystem] Created text warning entity (ID: %d) at (%.0f, %.0f) [text fallback]", entityID, centerX, centerY)
	}
}

// renderWarningTextImage 获取预渲染的红色警告文字图片
//
// Story 17.7 补充任务：使用 HouseofTerror28 字体图集中预渲染的
// 「一大波僵尸正在接近!」中文文字，并应用红色着色
//
// 返回：
//   - *ebiten.Image: 渲染的文字图片，失败时返回 nil
func (s *FlagWaveWarningSystem) renderWarningTextImage() *ebiten.Image {
	// 懒加载位图字体（只尝试一次）
	if !s.bitmapFontLoaded {
		s.loadBitmapFont()
	}

	if s.bitmapFont == nil {
		return nil
	}

	// 使用红色提取预渲染的中文警告文字
	redColor := color.RGBA{R: 255, G: 0, B: 0, A: 255}
	textImage, err := s.bitmapFont.GetHugeWaveWarningImage(redColor)
	if err != nil {
		log.Printf("[FlagWaveWarningSystem] Failed to get huge wave warning image: %v", err)
		return nil
	}

	log.Printf("[FlagWaveWarningSystem] Got pre-rendered warning text image: %dx%d",
		textImage.Bounds().Dx(), textImage.Bounds().Dy())
	return textImage
}

// loadBitmapFont 加载 HouseofTerror28 位图字体
//
// Story 17.7 补充任务：懒加载位图字体，只尝试一次
func (s *FlagWaveWarningSystem) loadBitmapFont() {
	s.bitmapFontLoaded = true

	font, err := utils.LoadBitmapFont(
		"assets/data/HouseofTerror28.png",
		"assets/data/HouseofTerror28.txt",
	)
	if err != nil {
		log.Printf("[FlagWaveWarningSystem] Failed to load HouseofTerror28 font: %v", err)
		return
	}

	s.bitmapFont = font
	log.Printf("[FlagWaveWarningSystem] Loaded HouseofTerror28 bitmap font (%d characters)", len(font.CharMap))
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

// GetFinalWaveEntityID 获取当前最终波警告实体ID（用于测试）
func (s *FlagWaveWarningSystem) GetFinalWaveEntityID() ecs.EntityID {
	return s.finalWaveEntityID
}

// IsWarningActive 检查警告是否激活（包括红字和白字警告）
func (s *FlagWaveWarningSystem) IsWarningActive() bool {
	return s.warningEntityID != 0 || s.finalWaveEntityID != 0
}

// IsHugeWaveWarningEntityActive 检查红字警告实体是否激活
func (s *FlagWaveWarningSystem) IsHugeWaveWarningEntityActive() bool {
	return s.warningEntityID != 0
}

// IsFinalWaveWarningEntityActive 检查最终波白字警告实体是否激活
func (s *FlagWaveWarningSystem) IsFinalWaveWarningEntityActive() bool {
	return s.finalWaveEntityID != 0
}

// TriggerWarning 手动触发警告动画
//
// 用于调试和验证程序，直接创建警告实体而不依赖 WaveTimingSystem 状态
// 警告将显示约 4 秒后自动消失
//
// 返回：
//   - bool: true 表示成功触发，false 表示警告已存在
func (s *FlagWaveWarningSystem) TriggerWarning() bool {
	if s.warningEntityID != 0 {
		log.Printf("[FlagWaveWarningSystem] Warning already active, skipping trigger")
		return false
	}

	log.Printf("[FlagWaveWarningSystem] Manual warning trigger")
	s.createWarningEntity()
	return true
}

// DismissWarning 手动关闭警告动画
//
// 用于调试和验证程序，立即销毁警告实体
func (s *FlagWaveWarningSystem) DismissWarning() {
	if s.warningEntityID == 0 {
		return
	}

	s.destroyWarningEntity()
}
