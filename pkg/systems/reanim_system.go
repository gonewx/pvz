package systems

import (
	"fmt"
	"log"

	"github.com/decker502/pvz/internal/reanim"
	"github.com/decker502/pvz/pkg/components"
	"github.com/decker502/pvz/pkg/config"
	"github.com/decker502/pvz/pkg/ecs"
	"github.com/hajimehoshi/ebiten/v2"
)

// ReanimSystem 是 Reanim 动画系统
// 基于 animation_showcase/AnimationCell 重写，简化并修复 Epic 13 遗留问题
//
// Story 13.8 重构目标：
// - API 数量从 50+ 减少到 2 个核心 API
// - 代码行数从 2808 减少到 ~1000 行
// - 与 AnimationCell 保持一致的逻辑
type ReanimSystem struct {
	entityManager *ecs.EntityManager
	configManager *config.ReanimConfigManager

	// 游戏 TPS（用于帧推进计算）
	targetTPS float64
}

// NewReanimSystem 创建新的 Reanim 动画系统
func NewReanimSystem(em *ecs.EntityManager) *ReanimSystem {
	return &ReanimSystem{
		entityManager: em,
		targetTPS:     60.0, // 默认 60 TPS
	}
}

// SetConfigManager 设置配置管理器
func (s *ReanimSystem) SetConfigManager(cm *config.ReanimConfigManager) {
	s.configManager = cm
}

// SetTargetTPS 设置目标 TPS（用于帧推进计算）
func (s *ReanimSystem) SetTargetTPS(tps float64) {
	s.targetTPS = tps
}

// ==================================================================
// 核心 API (Core APIs)
// ==================================================================

// PlayAnimation 播放单个动画（基础 API，不读配置）
// 用于简单场景，不需要配置文件的支持
//
// 参数：
//   - entityID: 实体 ID
//   - animName: 动画名称（如 "anim_idle"）
//
// 返回：
//   - error: 如果实体不存在或没有 ReanimComponent，返回错误
func (s *ReanimSystem) PlayAnimation(entityID ecs.EntityID, animName string) error {
	comp, ok := ecs.GetComponent[*components.ReanimComponent](s.entityManager, entityID)
	if !ok {
		return fmt.Errorf("entity %d does not have ReanimComponent", entityID)
	}

	if comp.ReanimXML == nil {
		return fmt.Errorf("entity %d has no ReanimXML data", entityID)
	}

	// ✅ Story 13.8 Bug Fix #9: 自动初始化基础字段（如果尚未初始化）
	// 原因：zombie_factory 等调用者只设置 ReanimXML 和 PartImages
	// rebuildAnimationData 需要 MergedTracks 存在
	if comp.MergedTracks == nil {
		comp.MergedTracks = reanim.BuildMergedTracks(comp.ReanimXML)
		comp.VisualTracks, comp.LogicalTracks = s.analyzeTrackTypes(comp.ReanimXML)
		comp.AnimationFPS = float64(comp.ReanimXML.FPS)
		comp.IsLooping = true
		comp.LastRenderFrame = -1
	}

	// ✅ 单个动画模式：清空配置相关字段
	// 单个动画模式下，不使用 HiddenTracks, ParentTracks, TrackAnimationBinding
	// 这些都依赖 Reanim 文件本身的定义
	comp.HiddenTracks = nil
	comp.ParentTracks = nil
	comp.TrackAnimationBinding = nil

	// 设置当前动画列表
	comp.CurrentAnimations = []string{animName}
	comp.CurrentFrame = 0
	comp.FrameAccumulator = 0
	comp.IsFinished = false
	comp.IsLooping = true // ✅ 显式设置为循环播放

	// 重建动画数据
	s.rebuildAnimationData(comp)

	// 计算并缓存 CenterOffset（基于第一帧）
	s.calculateCenterOffset(comp)

	// 标记缓存失效
	comp.LastRenderFrame = -1

	return nil
}

// AddAnimation 添加一个动画到当前播放列表（累加模式）
// 用于同时播放多个独立动画（如背景 + 云朵 + 草）
//
// 参数：
//   - entityID: 实体 ID
//   - animName: 动画名称（如 "anim_cloud1"）
//
// 返回：
//   - error: 如果实体不存在或没有 ReanimComponent，返回错误
func (s *ReanimSystem) AddAnimation(entityID ecs.EntityID, animName string) error {
	comp, ok := ecs.GetComponent[*components.ReanimComponent](s.entityManager, entityID)
	if !ok {
		return fmt.Errorf("entity %d does not have ReanimComponent", entityID)
	}

	if comp.ReanimXML == nil {
		return fmt.Errorf("entity %d has no ReanimXML data", entityID)
	}

	// ✅ 自动初始化基础字段（如果尚未初始化）
	if comp.MergedTracks == nil {
		comp.MergedTracks = reanim.BuildMergedTracks(comp.ReanimXML)
		comp.VisualTracks, comp.LogicalTracks = s.analyzeTrackTypes(comp.ReanimXML)
		comp.AnimationFPS = float64(comp.ReanimXML.FPS)
		comp.IsLooping = true
		comp.LastRenderFrame = -1
	}

	// ✅ 添加动画到列表（而不是替换）
	comp.CurrentAnimations = append(comp.CurrentAnimations, animName)

	// 重建动画数据（为新动画构建 AnimVisiblesMap）
	s.rebuildAnimationData(comp)

	// 标记缓存失效
	comp.LastRenderFrame = -1

	log.Printf("[ReanimSystem] AddAnimation: entity %d, added animation '%s', total animations: %d",
		entityID, animName, len(comp.CurrentAnimations))

	return nil
}

// FinalizeAnimations 完成动画设置（生成轨道绑定）
// 在使用 PlayAnimation + AddAnimation 添加完所有动画后调用
// 自动分析轨道绑定，确定哪个轨道由哪个动画控制
//
// 参数：
//   - entityID: 实体 ID
//
// 返回：
//   - error: 如果实体不存在或没有 ReanimComponent，返回错误
func (s *ReanimSystem) FinalizeAnimations(entityID ecs.EntityID) error {
	comp, ok := ecs.GetComponent[*components.ReanimComponent](s.entityManager, entityID)
	if !ok {
		return fmt.Errorf("entity %d does not have ReanimComponent", entityID)
	}

	// 自动生成轨道绑定
	comp.TrackAnimationBinding = s.analyzeTrackBinding(comp)

	// 标记缓存失效
	comp.LastRenderFrame = -1

	log.Printf("[ReanimSystem] FinalizeAnimations: entity %d, generated %d track bindings",
		entityID, len(comp.TrackAnimationBinding))

	return nil
}

// PlayCombo 播放配置组合（推荐 API，应用所有配置）
// 从配置管理器读取 combo 配置，应用所有设置（hidden_tracks, parent_tracks, binding）
//
// 参数：
//   - entityID: 实体 ID
//   - unitID: 单位 ID（如 "peashooter", "sunflower"）
//   - comboName: 组合名称（如 "attack", "idle"）。如果为空，使用第一个 combo
//
// 返回：
//   - error: 如果实体不存在、配置缺失，返回错误
func (s *ReanimSystem) PlayCombo(entityID ecs.EntityID, unitID, comboName string) error {
	comp, ok := ecs.GetComponent[*components.ReanimComponent](s.entityManager, entityID)
	if !ok {
		return fmt.Errorf("entity %d does not have ReanimComponent", entityID)
	}

	if comp.ReanimXML == nil {
		return fmt.Errorf("entity %d has no ReanimXML data", entityID)
	}

	// ✅ Story 13.8 Bug Fix: 自动初始化基础字段（如果尚未初始化）
	// 原因：plant_card_factory 等调用者只设置 ReanimXML 和 PartImages
	// 需要 PlayCombo 自动初始化 MergedTracks, VisualTracks 等字段
	if comp.MergedTracks == nil {
		comp.ReanimName = unitID
		comp.MergedTracks = reanim.BuildMergedTracks(comp.ReanimXML)
		comp.VisualTracks, comp.LogicalTracks = s.analyzeTrackTypes(comp.ReanimXML)
		comp.AnimationFPS = float64(comp.ReanimXML.FPS)
		// IsLooping 默认为 true，会在后面根据配置覆盖
		comp.IsLooping = true
		comp.LastRenderFrame = -1
		log.Printf("[ReanimSystem] PlayCombo: 初始化实体 %d, ReanimName='%s', VisualTracks=%d, LogicalTracks=%d, FPS=%.1f",
			entityID, comp.ReanimName, len(comp.VisualTracks), len(comp.LogicalTracks), comp.AnimationFPS)
	}

	if s.configManager == nil {
		return fmt.Errorf("config manager not set, cannot play combo")
	}

	// 获取单位配置
	unitConfig, err := s.configManager.GetUnit(unitID)
	if err != nil {
		return fmt.Errorf("failed to get config for unit %s: %w", unitID, err)
	}

	// 查找 combo 配置
	var combo *config.AnimationComboConfig
	if comboName == "" {
		// 使用第一个 combo
		if len(unitConfig.AnimationCombos) > 0 {
			combo = &unitConfig.AnimationCombos[0]
		}
	} else {
		// 查找指定 combo
		for i := range unitConfig.AnimationCombos {
			if unitConfig.AnimationCombos[i].Name == comboName {
				combo = &unitConfig.AnimationCombos[i]
				break
			}
		}
	}

	if combo == nil {
		return fmt.Errorf("no combo found for unit %s, combo %s", unitID, comboName)
	}

	// 1. 设置动画列表
	comp.CurrentAnimations = combo.Animations
	comp.CurrentFrame = 0
	comp.FrameAccumulator = 0
	comp.IsFinished = false

	// ✅ 加载单个动画的 FPS 和速度倍率配置
	// 从 unitConfig.AvailableAnimations 中读取每个动画的 FPS 和 Speed
	// 并设置到 AnimationFPSOverrides 和 AnimationSpeedOverrides 中
	if comp.AnimationFPSOverrides == nil {
		comp.AnimationFPSOverrides = make(map[string]float64)
	}
	if comp.AnimationSpeedOverrides == nil {
		comp.AnimationSpeedOverrides = make(map[string]float64)
	}
	for _, animInfo := range unitConfig.AvailableAnimations {
		// 如果配置中指定了 FPS，应用到 AnimationFPSOverrides
		if animInfo.FPS > 0 {
			comp.AnimationFPSOverrides[animInfo.Name] = animInfo.FPS
			log.Printf("[ReanimSystem] PlayCombo: 动画 %s 使用独立 FPS = %.1f", animInfo.Name, animInfo.FPS)
		}
		// 如果配置中指定了 Speed，应用到 AnimationSpeedOverrides
		if animInfo.Speed > 0 {
			comp.AnimationSpeedOverrides[animInfo.Name] = animInfo.Speed
			log.Printf("[ReanimSystem] PlayCombo: 动画 %s 使用速度倍率 = %.2f", animInfo.Name, animInfo.Speed)
		}
	}

	// 应用循环设置（如果配置中指定了）
	if combo.Loop != nil {
		comp.IsLooping = *combo.Loop
		log.Printf("[ReanimSystem] PlayCombo: entity %d, unit %s, combo %s → loop: %v", entityID, unitID, comboName, *combo.Loop)
	} else {
		// 默认循环
		comp.IsLooping = true
	}

	log.Printf("[ReanimSystem] PlayCombo: entity %d, unit %s, combo %s → animations: %v, loop: %v",
		entityID, unitID, comboName, combo.Animations, comp.IsLooping)

	// 2. 应用父子关系
	if len(combo.ParentTracks) > 0 {
		comp.ParentTracks = combo.ParentTracks
		log.Printf("[ReanimSystem] PlayCombo: applied %d parent tracks", len(combo.ParentTracks))
	} else {
		comp.ParentTracks = nil
	}

	// 3. 应用隐藏轨道
	if len(combo.HiddenTracks) > 0 {
		comp.HiddenTracks = make(map[string]bool)
		for _, track := range combo.HiddenTracks {
			comp.HiddenTracks[track] = true
		}
		log.Printf("[ReanimSystem] PlayCombo: hiding %d tracks", len(combo.HiddenTracks))
	} else {
		comp.HiddenTracks = nil
	}

	// 4. 重建动画数据
	s.rebuildAnimationData(comp)

	// 5. 分析轨道绑定
	if combo.BindingStrategy == "auto" {
		comp.TrackAnimationBinding = s.analyzeTrackBinding(comp)
		log.Printf("[ReanimSystem] PlayCombo: auto-generated %d track bindings", len(comp.TrackAnimationBinding))
	} else if combo.BindingStrategy == "manual" && len(combo.ManualBindings) > 0 {
		comp.TrackAnimationBinding = combo.ManualBindings
		log.Printf("[ReanimSystem] PlayCombo: applied %d manual bindings", len(combo.ManualBindings))
	} else {
		comp.TrackAnimationBinding = nil
	}

	// 标记缓存失效
	// 计算并缓存 CenterOffset（基于第一帧）
	s.calculateCenterOffset(comp)

	comp.LastRenderFrame = -1

	return nil
}

// ==================================================================
// 系统更新 (System Update)
// ==================================================================

// Update 更新所有 Reanim 组件的动画帧
// 基于 AnimationCell.Update() 的逻辑
// ✅ Story 13.8 Bug Fix #10: 完全匹配参考实现
//   - currentFrame 无限增长，不在 Update 中做循环检查
//   - 循环逻辑完全由 findControllingAnimation 的取模处理
//   - 支持多动画组合（不同轨道可以有不同的帧数）
func (s *ReanimSystem) Update(deltaTime float64) {
	entities := ecs.GetEntitiesWith1[*components.ReanimComponent](s.entityManager)

	// Debug: 检查是否有 sodroll 实体
	for _, id := range entities {
		comp, exists := ecs.GetComponent[*components.ReanimComponent](s.entityManager, id)
		if exists && comp.ReanimName == "sodroll" && comp.CurrentFrame < 3 {
			log.Printf("[ReanimSystem] 🟫 Update: sodroll entity %d, frame=%d, FPS=%.1f",
				id, comp.CurrentFrame, comp.AnimationFPS)
		}
	}

	for _, id := range entities {
		comp, exists := ecs.GetComponent[*components.ReanimComponent](s.entityManager, id)
		if !exists {
			continue
		}

		// 跳过没有数据的组件
		if comp.ReanimXML == nil {
			continue
		}

		// 跳过暂停的动画
		if comp.IsPaused {
			continue
		}

		// ✅ 为每个动画独立推进帧（支持不同动画不同 FPS）
		// 初始化 AnimationFrameIndices（如果尚未初始化）
		if comp.AnimationFrameIndices == nil {
			comp.AnimationFrameIndices = make(map[string]float64)
			for _, animName := range comp.CurrentAnimations {
				comp.AnimationFrameIndices[animName] = 0.0
			}
		}

		// 为每个动画独立推进帧
		for _, animName := range comp.CurrentAnimations {
			// 检查是否暂停
			if comp.AnimationPausedStates != nil {
				if isPaused, exists := comp.AnimationPausedStates[animName]; exists && isPaused {
					continue // 跳过暂停的动画
				}
			}

			// 获取该动画的 FPS
			animFPS := comp.AnimationFPS // 默认使用全局 FPS
			if comp.AnimationFPSOverrides != nil {
				if fps, hasOverride := comp.AnimationFPSOverrides[animName]; hasOverride {
					animFPS = fps
				}
			}

			// ✅ 获取该动画的速度倍率
			animSpeed := 1.0 // 默认正常速度
			if comp.AnimationSpeedOverrides != nil {
				if speed, hasOverride := comp.AnimationSpeedOverrides[animName]; hasOverride && speed > 0 {
					animSpeed = speed
				}
			}

			// 推进该动画的帧索引（应用速度倍率）
			// frameIncrement = (FPS / targetTPS) * speedMultiplier
			// 例如：FPS=12, TPS=60, speed=0.2 → increment = (12/60) * 0.2 = 0.04 帧/tick
			frameIncrement := (animFPS / s.targetTPS) * animSpeed
			comp.AnimationFrameIndices[animName] += frameIncrement
		}

		// 同步更新 CurrentFrame（用于后备和非循环动画检测）
		// 使用第一个非暂停动画的帧索引
		for _, animName := range comp.CurrentAnimations {
			if comp.AnimationPausedStates != nil {
				if isPaused, exists := comp.AnimationPausedStates[animName]; exists && isPaused {
					continue
				}
			}
			comp.CurrentFrame = int(comp.AnimationFrameIndices[animName])
			break
		}

		// ✅ 检查非循环动画是否已完成
		// 支持混合模式：即使全局 IsLooping=true，也要检测单个非循环动画的完成状态
		if !comp.IsFinished {
			// 检查是否所有非循环动画都已完成
			allNonLoopingAnimsFinished := false

			// 如果全局非循环（旧逻辑）
			if !comp.IsLooping {
				// 计算动画的最大帧数（所有当前播放动画中的最大可见帧数）
				maxVisibleFrames := 0
				for _, animName := range comp.CurrentAnimations {
					if animVisibles, exists := comp.AnimVisiblesMap[animName]; exists {
						visibleCount := countVisibleFrames(animVisibles)
						if visibleCount > maxVisibleFrames {
							maxVisibleFrames = visibleCount
						}
					}
				}

				// 如果当前帧已经到达或超过最大帧数，标记动画完成
				if maxVisibleFrames > 0 && comp.CurrentFrame >= maxVisibleFrames {
					allNonLoopingAnimsFinished = true
				}
			} else if comp.AnimationLoopStates != nil {
				// ✅ 新逻辑：检测单个非循环动画是否完成（即使全局 IsLooping=true）
				// 只检查非循环动画的完成状态
				hasNonLoopingAnims := false
				allNonLoopingComplete := true

				for _, animName := range comp.CurrentAnimations {
					// 获取该动画的循环状态
					isLooping := comp.IsLooping // 默认使用全局状态
					if loopState, hasState := comp.AnimationLoopStates[animName]; hasState {
						isLooping = loopState
					}

					// 如果该动画是非循环的
					if !isLooping {
						hasNonLoopingAnims = true
						if animVisibles, exists := comp.AnimVisiblesMap[animName]; exists {
							visibleCount := countVisibleFrames(animVisibles)
							// 检查该动画是否完成
							if visibleCount > 0 && comp.CurrentFrame < visibleCount {
								allNonLoopingComplete = false
								break
							}
						}
					}
				}

				// 如果有非循环动画且全部完成，设置 IsFinished
				if hasNonLoopingAnims && allNonLoopingComplete {
					allNonLoopingAnimsFinished = true
				}
			}

			// 设置 IsFinished 标志
			if allNonLoopingAnimsFinished {
				comp.IsFinished = true
				log.Printf("[ReanimSystem] 非循环动画完成: entity=%d, ReanimName=%s, CurrentFrame=%d", id, comp.ReanimName, comp.CurrentFrame)
			}
		}
	}
}

// ==================================================================
// 渲染缓存 (Render Cache)
// ==================================================================

// prepareRenderCache 准备渲染缓存
// 基于 AnimationCell.updateRenderCache() 的逻辑
// 关键修复：检查 HiddenTracks（Story 13.8 核心 Bug 修复）
func (s *ReanimSystem) prepareRenderCache(comp *components.ReanimComponent) {
	// Debug: 无条件打印向日葵和 SodRoll 的缓存准备
	if comp.ReanimName == "sunflower" && comp.CurrentFrame < 3 {
		log.Printf("[ReanimSystem] 🌻 prepareRenderCache 被调用: frame=%d", comp.CurrentFrame)
	}
	if comp.ReanimName == "sodroll" && comp.CurrentFrame < 3 {
		log.Printf("[ReanimSystem] 🟫 SodRoll prepareRenderCache 被调用: frame=%d, VisualTracks=%d",
			comp.CurrentFrame, len(comp.VisualTracks))
	}

	// 重用切片避免分配
	comp.CachedRenderData = comp.CachedRenderData[:0]

	visibleCount := 0
	skippedHidden := 0
	skippedNoAnim := 0
	skippedNoFrames := 0
	skippedNoImage := 0

	for _, trackName := range comp.VisualTracks {
		// Debug: 打印向日葵的所有轨道名称
		if comp.ReanimName == "sunflower" && comp.CurrentFrame == 0 {
			log.Printf("[ReanimSystem] 🔍 sunflower 轨道: %s", trackName)
		}

		// ✅ 关键修复：检查隐藏轨道（黑名单）
		if comp.HiddenTracks != nil && comp.HiddenTracks[trackName] {
			skippedHidden++
			continue
		}

		// 查找控制该轨道的动画
		controllingAnim, logicalFrame := s.findControllingAnimation(comp, trackName)
		if controllingAnim == "" {
			skippedNoAnim++
			// Debug: 记录没有控制动画的轨道
			if comp.ReanimName == "sunflower" && comp.CurrentFrame == 0 {
				log.Printf("[ReanimSystem] ⚠️ sunflower 轨道 %s: 没有找到控制动画", trackName)
			}
			continue
		}

		// Debug: 记录 anim_idle 相关轨道的控制信息
		if comp.ReanimName == "sunflower" && comp.CurrentFrame < 3 && (trackName == "anim_idle" || controllingAnim == "anim_idle") {
			log.Printf("[ReanimSystem] 📍 sunflower frame %d: 轨道 %s 由动画 %s 控制, logicalFrame=%.2f",
				comp.CurrentFrame, trackName, controllingAnim, logicalFrame)
		}

		// 获取轨道的帧数组和动画可见性数组
		mergedFrames, ok := comp.MergedTracks[trackName]
		if !ok || len(mergedFrames) == 0 {
			skippedNoFrames++
			continue
		}

		animVisibles, ok := comp.AnimVisiblesMap[controllingAnim]
		if !ok {
			skippedNoFrames++
			continue
		}

		// ✅ 使用帧插值获取平滑的帧数据
		frame := s.getInterpolatedFrame(controllingAnim, logicalFrame, animVisibles, mergedFrames)

		// ✅ 图片继承逻辑：如果插值后的帧没有图片，向前搜索最近的有图片的帧
		// 原版 PvZ 的 Reanim 系统会继承上一帧的图片（类似 Flash 的关键帧）
		if frame.ImagePath == "" {
			// 获取整数帧索引用于图片继承搜索
			physicalFrame := mapLogicalToPhysical(int(logicalFrame), animVisibles)
			if physicalFrame < 0 {
				skippedNoImage++
				continue
			}

			// 向前搜索有图片的帧
			foundImage := false
			for i := physicalFrame - 1; i >= 0; i-- {
				if i < len(mergedFrames) && mergedFrames[i].ImagePath != "" {
					// 继承前一帧的图片路径，但保留当前帧的变换属性
					frame.ImagePath = mergedFrames[i].ImagePath
					foundImage = true
					// Debug: 向日葵 anim_idle 轨道的图片继承
					if comp.ReanimName == "sunflower" && trackName == "anim_idle" && comp.CurrentFrame < 5 {
						log.Printf("[ReanimSystem] 🔧 SunFlower anim_idle frame %d 继承图片: %s (从帧 %d)",
							physicalFrame, frame.ImagePath, i)
					}
					break
				}
			}
			// 如果整个轨道都没有图片，才跳过
			if !foundImage {
				skippedNoImage++
				if comp.ReanimName == "sunflower" && trackName == "anim_idle" {
					log.Printf("[ReanimSystem] ❌ SunFlower anim_idle frame %d: 整个轨道都没有图片!", physicalFrame)
				}
				continue
			}
		} else if comp.ReanimName == "sunflower" && trackName == "anim_idle" && comp.CurrentFrame < 5 {
			// Debug: 原生图片
			log.Printf("[ReanimSystem] ✅ SunFlower anim_idle frame %.2f 原生图片: %s", logicalFrame, frame.ImagePath)
		}

		// 计算父轨道偏移
		offsetX, offsetY := 0.0, 0.0
		if parentTrackName, hasParent := comp.ParentTracks[trackName]; hasParent {
			childAnimName, _ := s.findControllingAnimation(comp, trackName)
			parentAnimName, _ := s.findControllingAnimation(comp, parentTrackName)

			// 只有当子轨道和父轨道使用不同动画时，才应用偏移
			if childAnimName != parentAnimName && childAnimName != "" && parentAnimName != "" {
				offsetX, offsetY = s.getParentOffset(comp, parentTrackName)
			}
		}

		// 获取图片
		img, ok := comp.PartImages[frame.ImagePath]
		if !ok || img == nil {
			// Debug: 记录找不到图片的情况
			if comp.ReanimName == "sunflower" && trackName == "anim_idle" && comp.CurrentFrame < 5 {
				log.Printf("[ReanimSystem] ⚠️ SunFlower anim_idle frame %.2f: 图片 %s 不存在于 PartImages", logicalFrame, frame.ImagePath)
			}
			continue
		}

		// Debug: 成功获取图片
		if comp.ReanimName == "sunflower" && trackName == "anim_idle" && comp.CurrentFrame < 5 {
			log.Printf("[ReanimSystem] ✅ SunFlower anim_idle frame %.2f: 成功获取图片 %s (尺寸: %dx%d)",
				logicalFrame, frame.ImagePath, img.Bounds().Dx(), img.Bounds().Dy())
		}

		// 添加到缓存
		comp.CachedRenderData = append(comp.CachedRenderData, components.RenderPartData{
			Img:     img,
			Frame:   frame,
			OffsetX: offsetX,
			OffsetY: offsetY,
		})
		visibleCount++
	}

	// Debug: 只在有变化时输出日志（避免刷屏）
	// 特殊调试：向日葵每帧都打印（前 10 帧）
	if comp.ReanimName == "sunflower" && comp.CurrentFrame < 10 {
		log.Printf("[ReanimSystem] 🔍 SunFlower frame %d → %d visible parts (skipped: hidden=%d, noAnim=%d, noFrames=%d, noImage=%d)",
			comp.CurrentFrame, visibleCount, skippedHidden, skippedNoAnim, skippedNoFrames, skippedNoImage)
	} else if len(comp.CachedRenderData) > 0 && comp.CurrentFrame%30 == 0 {
		log.Printf("[ReanimSystem] prepareRenderCache: %s frame %d → %d visible parts (skipped: hidden=%d, noAnim=%d, noFrames=%d, noImage=%d)",
			comp.ReanimName, comp.CurrentFrame, visibleCount, skippedHidden, skippedNoAnim, skippedNoFrames, skippedNoImage)
	}
}

// GetRenderData 获取渲染数据（供 RenderSystem 使用）
// 如果缓存失效，会自动重建缓存
func (s *ReanimSystem) GetRenderData(entityID ecs.EntityID) []components.RenderPartData {
	comp, ok := ecs.GetComponent[*components.ReanimComponent](s.entityManager, entityID)
	if !ok {
		return nil
	}

	// 检查缓存是否失效
	if comp.LastRenderFrame != comp.CurrentFrame {
		s.prepareRenderCache(comp)
		comp.LastRenderFrame = comp.CurrentFrame
	}

	return comp.CachedRenderData
}

// ==================================================================
// 辅助方法 (Helper Methods)
// ==================================================================

// rebuildAnimationData 重建动画数据（AnimVisiblesMap）
// 基于 AnimationCell.rebuildAnimationData()
func (s *ReanimSystem) rebuildAnimationData(comp *components.ReanimComponent) {
	comp.AnimVisiblesMap = make(map[string][]int)

	for _, animName := range comp.CurrentAnimations {
		animVisibles := buildVisiblesArray(comp.ReanimXML, comp.MergedTracks, animName)
		comp.AnimVisiblesMap[animName] = animVisibles
	}
}

// analyzeTrackBinding 自动分析轨道绑定
// 基于 AnimationCell.analyzeTrackBinding()
func (s *ReanimSystem) analyzeTrackBinding(comp *components.ReanimComponent) map[string]string {
	binding := make(map[string]string)

	// 1. 分析视觉轨道
	for _, trackName := range comp.VisualTracks {
		frames, ok := comp.MergedTracks[trackName]
		if !ok {
			continue
		}

		var bestAnim string
		var bestScore float64

		for _, animName := range comp.CurrentAnimations {
			animVisibles := comp.AnimVisiblesMap[animName]
			firstVisible, lastVisible := findVisibleWindow(animVisibles)

			if firstVisible < 0 || lastVisible >= len(frames) {
				continue
			}

			// 检查是否有图片
			hasImage := false
			for i := firstVisible; i <= lastVisible && i < len(frames); i++ {
				if frames[i].ImagePath != "" {
					hasImage = true
					break
				}
			}

			if !hasImage {
				continue
			}

			// 计算评分
			variance := calculatePositionVariance(frames, firstVisible, lastVisible)
			score := 1.0 + variance

			if score > bestScore {
				bestScore = score
				bestAnim = animName
			}
		}

		if bestAnim != "" {
			binding[trackName] = bestAnim
		}
	}

	// 2. 分析逻辑轨道
	for _, trackName := range comp.LogicalTracks {
		frames, ok := comp.MergedTracks[trackName]
		if !ok || len(frames) == 0 {
			continue
		}

		var bestAnim string
		var maxVariance float64

		for _, animName := range comp.CurrentAnimations {
			animVisibles := comp.AnimVisiblesMap[animName]
			firstVisible, lastVisible := findVisibleWindow(animVisibles)

			if firstVisible < 0 || lastVisible >= len(frames) {
				continue
			}

			variance := calculatePositionVariance(frames, firstVisible, lastVisible)

			if variance > maxVariance {
				maxVariance = variance
				bestAnim = animName
			}
		}

		if bestAnim != "" && maxVariance > 0.1 {
			binding[trackName] = bestAnim
		}
	}

	return binding
}

// findControllingAnimation 查找控制指定轨道的动画
// 基于 AnimationCell.findControllingAnimation()
// 返回：动画名称、浮点逻辑帧索引（用于插值）
func (s *ReanimSystem) findControllingAnimation(comp *components.ReanimComponent, trackName string) (string, float64) {
	// 优先使用绑定
	if comp.TrackAnimationBinding != nil {
		if animName, exists := comp.TrackAnimationBinding[trackName]; exists {
			animVisibles := comp.AnimVisiblesMap[animName]
			visibleCount := countVisibleFrames(animVisibles)
			if visibleCount > 0 {
				// ✅ 检查该动画是否被暂停
				// 如果暂停，返回第 0 帧（初始帧），使动画保持静止但可见
				if comp.AnimationPausedStates != nil {
					if isPaused, hasPausedState := comp.AnimationPausedStates[animName]; hasPausedState && isPaused {
						return animName, 0.0
					}
				}

				// ✅ 从 AnimationFrameIndices 获取该动画的独立帧索引（浮点数）
				var currentFrame float64
				if comp.AnimationFrameIndices != nil {
					if frameIndex, exists := comp.AnimationFrameIndices[animName]; exists {
						currentFrame = frameIndex
					} else {
						currentFrame = float64(comp.CurrentFrame) // 后备：使用共享帧
					}
				} else {
					currentFrame = float64(comp.CurrentFrame) // 后备：使用共享帧
				}

				// ✅ 检查该动画的单独循环状态
				isLooping := comp.IsLooping // 默认使用全局循环状态
				if comp.AnimationLoopStates != nil {
					if loopState, hasState := comp.AnimationLoopStates[animName]; hasState {
						isLooping = loopState // 使用该动画的独立循环状态
					}
				}

				var animLogicalFrame float64
				if isLooping {
					// 循环模式：使用浮点取模
					animLogicalFrame = currentFrame
					for animLogicalFrame >= float64(visibleCount) {
						animLogicalFrame -= float64(visibleCount)
					}
				} else {
					animLogicalFrame = currentFrame
					// 如果超出范围，钳制到最后一帧
					if animLogicalFrame >= float64(visibleCount) {
						animLogicalFrame = float64(visibleCount - 1)
					}
				}
				return animName, animLogicalFrame
			}
		}
	}

	// 默认使用第一个动画
	if len(comp.CurrentAnimations) > 0 {
		animName := comp.CurrentAnimations[0]
		animVisibles := comp.AnimVisiblesMap[animName]
		visibleCount := countVisibleFrames(animVisibles)
		if visibleCount > 0 {
			// ✅ 检查该动画是否被暂停
			// 如果暂停，返回第 0 帧（初始帧），使动画保持静止但可见
			if comp.AnimationPausedStates != nil {
				if isPaused, hasPausedState := comp.AnimationPausedStates[animName]; hasPausedState && isPaused {
					return animName, 0.0
				}
			}

			// ✅ 从 AnimationFrameIndices 获取该动画的独立帧索引（浮点数）
			var currentFrame float64
			if comp.AnimationFrameIndices != nil {
				if frameIndex, exists := comp.AnimationFrameIndices[animName]; exists {
					currentFrame = frameIndex
				} else {
					currentFrame = float64(comp.CurrentFrame) // 后备：使用共享帧
				}
			} else {
				currentFrame = float64(comp.CurrentFrame) // 后备：使用共享帧
			}

			// ✅ 检查该动画的单独循环状态
			isLooping := comp.IsLooping // 默认使用全局循环状态
			if comp.AnimationLoopStates != nil {
				if loopState, hasState := comp.AnimationLoopStates[animName]; hasState {
					isLooping = loopState // 使用该动画的独立循环状态
				}
			}

			var animLogicalFrame float64
			if isLooping {
				// 循环模式：使用浮点取模
				animLogicalFrame = currentFrame
				for animLogicalFrame >= float64(visibleCount) {
					animLogicalFrame -= float64(visibleCount)
				}
			} else {
				animLogicalFrame = currentFrame
				// 如果超出范围，钳制到最后一帧
				if animLogicalFrame >= float64(visibleCount) {
					animLogicalFrame = float64(visibleCount - 1)
				}
			}
			return animName, animLogicalFrame
		}
	}

	return "", -1.0
}

// getInterpolatedFrame 获取插值后的帧数据
// 参数：
//   - animName: 动画名称
//   - logicalFrame: 浮点逻辑帧索引（如 2.7 表示第 2 帧和第 3 帧之间，插值因子 0.7）
//   - animVisibles: 动画可见性数组
//   - mergedFrames: 轨道的累加帧数组
// 返回：插值后的帧数据
func (s *ReanimSystem) getInterpolatedFrame(
	animName string,
	logicalFrame float64,
	animVisibles []int,
	mergedFrames []reanim.Frame,
) reanim.Frame {
	// 1. 获取整数部分和小数部分
	frame1Index := int(logicalFrame)                    // 当前帧索引
	frame2Index := frame1Index + 1                      // 下一帧索引
	t := logicalFrame - float64(frame1Index)            // 插值因子 (0.0 - 1.0)

	// 2. 映射逻辑帧到物理帧
	physicalFrame1 := mapLogicalToPhysical(frame1Index, animVisibles)
	physicalFrame2 := mapLogicalToPhysical(frame2Index, animVisibles)

	// 3. 边界检查
	if physicalFrame1 < 0 || physicalFrame1 >= len(mergedFrames) {
		return reanim.Frame{} // 返回空帧
	}
	if physicalFrame2 < 0 || physicalFrame2 >= len(mergedFrames) {
		// 如果下一帧越界，直接返回当前帧（不插值）
		return mergedFrames[physicalFrame1]
	}

	// 4. 获取两个帧
	f1 := mergedFrames[physicalFrame1]
	f2 := mergedFrames[physicalFrame2]

	// 5. 线性插值
	result := reanim.Frame{
		ImagePath: f1.ImagePath, // 图片引用不插值，使用第一帧的
	}

	// 插值位置 (X, Y)
	if f1.X != nil && f2.X != nil {
		interpolatedX := *f1.X + (*f2.X-*f1.X)*t
		result.X = &interpolatedX
	} else if f1.X != nil {
		result.X = f1.X
	}

	if f1.Y != nil && f2.Y != nil {
		interpolatedY := *f1.Y + (*f2.Y-*f1.Y)*t
		result.Y = &interpolatedY
	} else if f1.Y != nil {
		result.Y = f1.Y
	}

	// 插值缩放 (ScaleX, ScaleY)
	if f1.ScaleX != nil && f2.ScaleX != nil {
		interpolatedScaleX := *f1.ScaleX + (*f2.ScaleX-*f1.ScaleX)*t
		result.ScaleX = &interpolatedScaleX
	} else if f1.ScaleX != nil {
		result.ScaleX = f1.ScaleX
	}

	if f1.ScaleY != nil && f2.ScaleY != nil {
		interpolatedScaleY := *f1.ScaleY + (*f2.ScaleY-*f1.ScaleY)*t
		result.ScaleY = &interpolatedScaleY
	} else if f1.ScaleY != nil {
		result.ScaleY = f1.ScaleY
	}

	// 插值倾斜角度 (SkewX, SkewY)
	if f1.SkewX != nil && f2.SkewX != nil {
		interpolatedSkewX := *f1.SkewX + (*f2.SkewX-*f1.SkewX)*t
		result.SkewX = &interpolatedSkewX
	} else if f1.SkewX != nil {
		result.SkewX = f1.SkewX
	}

	if f1.SkewY != nil && f2.SkewY != nil {
		interpolatedSkewY := *f1.SkewY + (*f2.SkewY-*f1.SkewY)*t
		result.SkewY = &interpolatedSkewY
	} else if f1.SkewY != nil {
		result.SkewY = f1.SkewY
	}

	// FrameNum 不插值（可见性标志），使用第一帧的
	result.FrameNum = f1.FrameNum

	return result
}

// getParentOffset 获取父轨道的偏移量
// 基于 AnimationCell.getParentOffset() (animation_cell.go:454-499)
//
// ✅ Story 13.8 Bug Fix #8: 修复父子偏移计算逻辑
//   - animation_showcase 逐步初始化坐标（先设为 0，有值则覆盖）
//   - 旧实现同时检查两个指针，导致 nil 值处理不正确
// ✅ 支持浮点帧索引和帧插值
func (s *ReanimSystem) getParentOffset(comp *components.ReanimComponent, parentTrackName string) (float64, float64) {
	parentFrames, ok := comp.MergedTracks[parentTrackName]
	if !ok || len(parentFrames) == 0 {
		return 0, 0
	}

	parentAnimName, parentLogicalFrame := s.findControllingAnimation(comp, parentTrackName)
	if parentAnimName == "" || parentLogicalFrame < 0 {
		return 0, 0
	}

	parentAnimVisibles := comp.AnimVisiblesMap[parentAnimName]
	if len(parentAnimVisibles) == 0 {
		return 0, 0
	}

	// 获取第一个可见帧的索引
	firstVisibleFrameIndex := -1
	for i, v := range parentAnimVisibles {
		if v == 0 {
			firstVisibleFrameIndex = i
			break
		}
	}

	// 映射到物理帧
	firstPhysicalFrame := mapLogicalToPhysical(firstVisibleFrameIndex, parentAnimVisibles)
	if firstPhysicalFrame < 0 || firstPhysicalFrame >= len(parentFrames) {
		return 0, 0
	}

	// ✅ 与 animation_showcase 完全一致的逻辑（animation_cell.go:479-498）
	// 先初始化为 0，然后逐步设置有效值
	initX, initY := 0.0, 0.0
	if parentFrames[firstPhysicalFrame].X != nil {
		initX = *parentFrames[firstPhysicalFrame].X
	}
	if parentFrames[firstPhysicalFrame].Y != nil {
		initY = *parentFrames[firstPhysicalFrame].Y
	}

	// ✅ 使用帧插值获取父轨道当前帧的平滑位置
	currentFrame := s.getInterpolatedFrame(parentAnimName, parentLogicalFrame, parentAnimVisibles, parentFrames)

	currentX, currentY := initX, initY
	if currentFrame.X != nil {
		currentX = *currentFrame.X
	}
	if currentFrame.Y != nil {
		currentY = *currentFrame.Y
	}

	return currentX - initX, currentY - initY
}

// ==================================================================
// 全局辅助函数 (Global Helper Functions)
// 基于 animation_showcase 的实现
// ==================================================================

// buildVisiblesArray 构建动画的可见性数组
func buildVisiblesArray(reanimXML *reanim.ReanimXML, mergedTracks map[string][]reanim.Frame, animName string) []int {
	var animTrack *reanim.Track
	for i := range reanimXML.Tracks {
		if reanimXML.Tracks[i].Name == animName {
			animTrack = &reanimXML.Tracks[i]
			break
		}
	}

	if animTrack == nil {
		return []int{}
	}

	standardFrameCount := 0
	for _, track := range reanimXML.Tracks {
		if len(track.Frames) > standardFrameCount {
			standardFrameCount = len(track.Frames)
		}
	}

	if standardFrameCount == 0 {
		return []int{}
	}

	visibles := make([]int, standardFrameCount)
	currentValue := 0

	for i := 0; i < standardFrameCount; i++ {
		if i < len(animTrack.Frames) {
			frame := animTrack.Frames[i]
			if frame.FrameNum != nil {
				currentValue = *frame.FrameNum
			}
		}
		visibles[i] = currentValue
	}

	return visibles
}

// countVisibleFrames 计算可见帧数（非隐藏帧的数量）
// animVisibles 中：-1 表示隐藏，>= 0 表示可见
func countVisibleFrames(animVisibles []int) int {
	count := 0
	for _, visible := range animVisibles {
		if visible >= 0 {
			count++
		}
	}
	return count
}

// mapLogicalToPhysical 将逻辑帧号映射到物理帧号
func mapLogicalToPhysical(logicalFrameNum int, animVisibles []int) int {
	if len(animVisibles) == 0 {
		return logicalFrameNum
	}

	logicalIndex := 0
	for i := 0; i < len(animVisibles); i++ {
		if animVisibles[i] == 0 {
			if logicalIndex == logicalFrameNum {
				return i
			}
			logicalIndex++
		}
	}

	return -1
}

// findVisibleWindow 查找动画的可见时间窗口
func findVisibleWindow(animVisibles []int) (int, int) {
	firstVisible, lastVisible := -1, -1
	for i, v := range animVisibles {
		if v == 0 {
			if firstVisible == -1 {
				firstVisible = i
			}
			lastVisible = i
		}
	}
	return firstVisible, lastVisible
}

// calculatePositionVariance 计算位置方差
func calculatePositionVariance(frames []reanim.Frame, startIdx, endIdx int) float64 {
	if startIdx < 0 || endIdx >= len(frames) || startIdx > endIdx {
		return 0.0
	}

	sumX, sumY := 0.0, 0.0
	count := 0
	for i := startIdx; i <= endIdx; i++ {
		if frames[i].X != nil && frames[i].Y != nil {
			sumX += *frames[i].X
			sumY += *frames[i].Y
			count++
		}
	}

	if count == 0 {
		return 0.0
	}

	meanX := sumX / float64(count)
	meanY := sumY / float64(count)

	variance := 0.0
	for i := startIdx; i <= endIdx; i++ {
		if frames[i].X != nil && frames[i].Y != nil {
			dx := *frames[i].X - meanX
			dy := *frames[i].Y - meanY
			variance += dx*dx + dy*dy
		}
	}

	return variance / float64(count)
}

// ==================================================================
// 兼容性方法（临时保留，用于过渡）
// ==================================================================

// InitReanimComponent 初始化 Reanim 组件的基础数据
// 用于实体工厂创建实体时的初始化
func (s *ReanimSystem) InitReanimComponent(
	entityID ecs.EntityID,
	reanimName string,
	reanimXML *reanim.ReanimXML,
	partImages map[string]*ebiten.Image,
	mergedTracks map[string][]reanim.Frame,
	visualTracks []string,
	logicalTracks []string,
) error {
	comp, ok := ecs.GetComponent[*components.ReanimComponent](s.entityManager, entityID)
	if !ok {
		return fmt.Errorf("entity %d does not have ReanimComponent", entityID)
	}

	comp.ReanimName = reanimName
	comp.ReanimXML = reanimXML
	comp.PartImages = partImages
	comp.MergedTracks = mergedTracks
	comp.VisualTracks = visualTracks
	comp.LogicalTracks = logicalTracks
	comp.AnimationFPS = float64(reanimXML.FPS)
	comp.IsLooping = true
	comp.LastRenderFrame = -1

	return nil
}

// PrepareStaticPreview prepares a Reanim entity for static preview (e.g., plant card icons).
// Story 13.8: 简化版本，使用配置驱动的方式
//
// 策略：
// 1. 播放默认动画组合
// 2. 将当前帧设置为中间帧（最佳预览帧）
// 3. 暂停动画播放（IsPaused = true）
//
// Parameters:
//   - entityID: the ID of the entity to prepare for static preview
//   - reanimName: the Reanim resource name (e.g., "sunflower", "peashooter")
//
// Returns:
//   - An error if preparation fails
func (s *ReanimSystem) PrepareStaticPreview(entityID ecs.EntityID, reanimName string) error {
	// 使用 PlayCombo 播放默认动画
	if err := s.PlayCombo(entityID, reanimName, ""); err != nil {
		return fmt.Errorf("failed to play default animation: %w", err)
	}

	// 获取组件
	comp, ok := ecs.GetComponent[*components.ReanimComponent](s.entityManager, entityID)
	if !ok {
		return fmt.Errorf("entity %d does not have ReanimComponent", entityID)
	}

	// 查找最佳预览帧（使用第一个动画的中间帧）
	if len(comp.CurrentAnimations) > 0 {
		animName := comp.CurrentAnimations[0]
		if visibles, ok := comp.AnimVisiblesMap[animName]; ok && len(visibles) > 0 {
			// 使用中间帧作为预览帧
			bestFrame := len(visibles) / 2
			comp.CurrentFrame = bestFrame
			log.Printf("[ReanimSystem] PrepareStaticPreview: %s set to frame %d/%d",
				reanimName, bestFrame, len(visibles))
		}
	}

	// 暂停动画播放（静态预览）
	comp.IsPaused = true
	comp.IsLooping = false

	// 强制更新渲染缓存
	s.prepareRenderCache(comp)

	return nil
}

// RenderToTexture 将指定实体的 Reanim 渲染到目标纹理（离屏渲染）
// 用于生成植物卡片图标等静态纹理
//
// 参数：
//   - entityID: 实体 ID
//   - target: 目标纹理（调用者创建）
//
// 返回：
//   - error: 如果实体不存在或没有必要组件，返回错误
func (s *ReanimSystem) RenderToTexture(entityID ecs.EntityID, target *ebiten.Image) error {
	// 验证实体拥有必要的组件
	pos, hasPos := ecs.GetComponent[*components.PositionComponent](s.entityManager, entityID)
	_, hasReanim := ecs.GetComponent[*components.ReanimComponent](s.entityManager, entityID)

	if !hasPos || !hasReanim {
		return fmt.Errorf("entity %d missing required components for rendering", entityID)
	}

	// 获取渲染数据（自动更新缓存）
	renderData := s.GetRenderData(entityID)
	if len(renderData) == 0 {
		return fmt.Errorf("entity %d has no render data", entityID)
	}

	// Step 1: 计算所有可见部件的 bounding box（用于居中）
	// 这是 Story 13.8 Bug Fix：替代旧的 CenterOffset 机制
	minX, maxX := 9999.0, -9999.0
	minY, maxY := 9999.0, -9999.0
	hasVisibleParts := false

	for _, partData := range renderData {
		if partData.Img == nil {
			continue
		}

		frame := partData.Frame
		if frame.FrameNum != nil && *frame.FrameNum == -1 {
			continue
		}

		// 计算部件位置
		partX := getFloat(frame.X) + partData.OffsetX
		partY := getFloat(frame.Y) + partData.OffsetY

		// 获取图片尺寸
		bounds := partData.Img.Bounds()
		w := float64(bounds.Dx())
		h := float64(bounds.Dy())

		// 考虑缩放
		scaleX := getFloat(frame.ScaleX)
		scaleY := getFloat(frame.ScaleY)
		if scaleX == 0 {
			scaleX = 1.0
		}
		if scaleY == 0 {
			scaleY = 1.0
		}

		// 计算部件的 bounding box（考虑图片尺寸）
		partMinX := partX
		partMaxX := partX + w*scaleX
		partMinY := partY
		partMaxY := partY + h*scaleY

		if partMinX < minX {
			minX = partMinX
		}
		if partMaxX > maxX {
			maxX = partMaxX
		}
		if partMinY < minY {
			minY = partMinY
		}
		if partMaxY > maxY {
			maxY = partMaxY
		}

		hasVisibleParts = true
	}

	// Step 2: 计算居中偏移
	// 目标：将 bounding box 的中心对齐到实体的 Position
	centerOffsetX := 0.0
	centerOffsetY := 0.0
	if hasVisibleParts {
		boundingWidth := maxX - minX
		boundingHeight := maxY - minY
		centerOffsetX = -(minX + boundingWidth/2)
		centerOffsetY = -(minY + boundingHeight/2)
	}

	// Step 3: 渲染所有部件（应用居中偏移）
	for _, partData := range renderData {
		if partData.Img == nil {
			continue
		}

		frame := partData.Frame

		// 跳过隐藏帧（FrameNum == -1）
		if frame.FrameNum != nil && *frame.FrameNum == -1 {
			continue
		}

		// 计算部件位置（相对于实体原点）
		partX := getFloat(frame.X) + partData.OffsetX
		partY := getFloat(frame.Y) + partData.OffsetY

		// 应用变换
		opts := &ebiten.DrawImageOptions{}

		// 1. 缩放（先应用缩放，再应用旋转和平移）
		scaleX := getFloat(frame.ScaleX)
		scaleY := getFloat(frame.ScaleY)
		if scaleX == 0 {
			scaleX = 1.0
		}
		if scaleY == 0 {
			scaleY = 1.0
		}
		opts.GeoM.Scale(scaleX, scaleY)

		// 2. 旋转（如果需要）
		// 注意：Reanim 使用弧度制
		// 这里暂不处理旋转，因为大部分植物图标不需要

		// 3. 平移到最终位置（应用居中偏移）
		// 使用 Position 作为基准点（离屏渲染，不减去摄像机偏移）
		finalX := pos.X + partX + centerOffsetX
		finalY := pos.Y + partY + centerOffsetY
		opts.GeoM.Translate(finalX, finalY)

		// 绘制部件
		target.DrawImage(partData.Img, opts)
	}

	return nil
}

// analyzeTrackTypes 分析轨道类型（视觉轨道 vs 逻辑轨道）
// 基于 animation_showcase/animation_cell.go:670-700
//
// ✅ Story 13.8 Bug Fix #7: 修复僵尸动画错误
//   - animation_showcase 只跳过植物的 4 个动画定义轨道
//   - 僵尸的 anim_walk/anim_eat 等应该被分类为 logicalTracks（无图片）
//   - 与 animation_showcase 保持完全一致
func (s *ReanimSystem) analyzeTrackTypes(reanimXML *reanim.ReanimXML) (visualTracks []string, logicalTracks []string) {
	// ✅ Bug Fix: 先检查轨道是否有图片，再决定是否跳过
	// 原因：向日葵的 anim_idle 轨道包含头部图像，不应该被跳过
	// animation_showcase 的逻辑可能不适用于所有植物
	animationDefinitionTracks := map[string]bool{
		"anim_idle":      true,
		"anim_shooting":  true,
		"anim_head_idle": true,
		"anim_full_idle": true,
	}

	for _, track := range reanimXML.Tracks {
		// 先检查轨道是否包含图片
		hasImage := false
		for _, frame := range track.Frames {
			if frame.ImagePath != "" {
				hasImage = true
				break
			}
		}

		// ✅ 关键修复：如果轨道包含图片，即使名称在 animationDefinitionTracks 中，
		// 也应该作为视觉轨道处理（例如向日葵的 anim_idle 轨道）
		if hasImage {
			visualTracks = append(visualTracks, track.Name)
		} else if animationDefinitionTracks[track.Name] {
			// 只有在没有图片的情况下，才跳过动画定义轨道
			logicalTracks = append(logicalTracks, track.Name)
		} else {
			// 其他无图片轨道也作为逻辑轨道
			logicalTracks = append(logicalTracks, track.Name)
		}
	}

	return visualTracks, logicalTracks
}

// calculateCenterOffset 计算并缓存 CenterOffset
// 在第一帧计算所有可见部件的 bounding box 中心,避免每帧重新计算导致位置抖动
func (s *ReanimSystem) calculateCenterOffset(comp *components.ReanimComponent) {
	// 确保已初始化
	if comp.MergedTracks == nil || len(comp.VisualTracks) == 0 {
		comp.CenterOffsetX = 0
		comp.CenterOffsetY = 0
		return
	}

	// 强制帧索引为 0,计算第一帧的 bounding box
	comp.CurrentFrame = 0

	// 准备第一帧的渲染数据
	s.prepareRenderCache(comp)

	if len(comp.CachedRenderData) == 0 {
		comp.CenterOffsetX = 0
		comp.CenterOffsetY = 0
		return
	}

	// 计算 bounding box
	minX, maxX := 9999.0, -9999.0
	minY, maxY := 9999.0, -9999.0

	for _, partData := range comp.CachedRenderData {
		if partData.Img == nil {
			continue
		}

		frame := partData.Frame
		if frame.FrameNum != nil && *frame.FrameNum == -1 {
			continue
		}

		// 计算部件位置
		partX := getFloat(frame.X) + partData.OffsetX
		partY := getFloat(frame.Y) + partData.OffsetY

		// 获取图片尺寸
		bounds := partData.Img.Bounds()
		w := float64(bounds.Dx())
		h := float64(bounds.Dy())

		// 考虑缩放
		scaleX := getFloat(frame.ScaleX)
		scaleY := getFloat(frame.ScaleY)
		if scaleX == 0 {
			scaleX = 1.0
		}
		if scaleY == 0 {
			scaleY = 1.0
		}

		// 计算部件的 bounding box（考虑图片尺寸）
		partMinX := partX
		partMaxX := partX + w*scaleX
		partMinY := partY
		partMaxY := partY + h*scaleY

		if partMinX < minX {
			minX = partMinX
		}
		if partMaxX > maxX {
			maxX = partMaxX
		}
		if partMinY < minY {
			minY = partMinY
		}
		if partMaxY > maxY {
			maxY = partMaxY
		}
	}

	// 计算中心点坐标
	comp.CenterOffsetX = (minX + maxX) / 2
	comp.CenterOffsetY = (minY + maxY) / 2
}

