package systems

import (
	"fmt"
	"log"
	"strings"

	"github.com/decker502/pvz/internal/reanim"
	"github.com/decker502/pvz/pkg/components"
	"github.com/decker502/pvz/pkg/config"
	"github.com/decker502/pvz/pkg/ecs"
)

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

	// 原因：zombie_factory 等调用者只设置 ReanimXML 和 PartImages
	// rebuildAnimationData 需要 MergedTracks 存在
	if comp.MergedTracks == nil {
		comp.MergedTracks = reanim.BuildMergedTracks(comp.ReanimXML)
		comp.VisualTracks, comp.LogicalTracks = s.analyzeTrackTypes(comp.ReanimXML)
		comp.AnimationFPS = float64(comp.ReanimXML.FPS)
		comp.IsLooping = true
		comp.LastRenderFrame = -1
	}

	// 单个动画模式下，ParentTracks 不使用（依赖 Reanim 文件本身的定义）
	comp.ParentTracks = nil

	// Story 12.4: 保留现有的 HiddenTracks（如果已设置）
	// 首次启动时需要在整个流程中保持轨道隐藏状态
	// 只在 HiddenTracks 未初始化时才清空
	if comp.HiddenTracks == nil {
		// 未设置 HiddenTracks，保持为 nil（默认行为）
		log.Printf("[ReanimSystem] PlayAnimation: HiddenTracks is nil, keeping it nil")
	} else {
		// 保留现有的 HiddenTracks
		log.Printf("[ReanimSystem] PlayAnimation: Preserving HiddenTracks (count=%d)", len(comp.HiddenTracks))
		for trackName := range comp.HiddenTracks {
			log.Printf("[ReanimSystem]   - Hidden track: %s", trackName)
		}
	}
	// 否则保留现有的 HiddenTracks

	// 设置当前动画列表
	comp.CurrentAnimations = []string{animName}
	comp.CurrentFrame = 0
	comp.FrameAccumulator = 0
	comp.IsFinished = false
	comp.IsLooping = true // 显式设置为循环播放

	// 重建动画数据
	s.rebuildAnimationData(comp)

	// 计算并缓存 CenterOffset（基于第一帧）
	s.calculateCenterOffset(comp)

	// 标记缓存失效
	comp.LastRenderFrame = -1

	return nil
}

// PlayAnimationWithConfig 播放单个动画（带配置）
// 与 PlayAnimation 类似，但从配置文件中读取 loop 设置
//
// 参数：
//   - entityID: 实体 ID
//   - unitID: 单位 ID（用于查找配置，如 "loadbar_sprout"）
//   - animName: 动画名称（如 "anim_sprout"）
//
// 返回：
//   - error: 如果实体不存在、没有 ReanimComponent、或配置读取失败，返回错误
func (s *ReanimSystem) PlayAnimationWithConfig(entityID ecs.EntityID, unitID, animName string) error {
	comp, ok := ecs.GetComponent[*components.ReanimComponent](s.entityManager, entityID)
	if !ok {
		return fmt.Errorf("entity %d does not have ReanimComponent", entityID)
	}

	if comp.ReanimXML == nil {
		return fmt.Errorf("entity %d has no ReanimXML data", entityID)
	}

	// 初始化 MergedTracks（如果需要）
	if comp.MergedTracks == nil {
		comp.MergedTracks = reanim.BuildMergedTracks(comp.ReanimXML)
		comp.VisualTracks, comp.LogicalTracks = s.analyzeTrackTypes(comp.ReanimXML)
		comp.AnimationFPS = float64(comp.ReanimXML.FPS)
		comp.IsLooping = true // 默认值
		comp.LastRenderFrame = -1
	}

	// 单个动画模式下，ParentTracks 不使用
	comp.ParentTracks = nil

	// 保留现有的 HiddenTracks
	if comp.HiddenTracks == nil {
		log.Printf("[ReanimSystem] PlayAnimationWithConfig: HiddenTracks is nil, keeping it nil")
	} else {
		log.Printf("[ReanimSystem] PlayAnimationWithConfig: Preserving HiddenTracks (count=%d)", len(comp.HiddenTracks))
	}

	// 设置当前动画列表
	comp.CurrentAnimations = []string{animName}
	comp.CurrentFrame = 0
	comp.FrameAccumulator = 0
	comp.IsFinished = false

	// 从配置中读取 loop 设置
	shouldLoop := true // 默认循环
	if s.configManager != nil {
		unitConfig, err := s.configManager.GetUnit(unitID)
		if err == nil {
			// 查找动画配置
			for _, animInfo := range unitConfig.AvailableAnimations {
				if animInfo.Name == animName {
					// animInfo.Loop 是 *bool 类型
					// nil = 使用默认值 true（循环）
					// &false = 显式设置为不循环
					// &true = 显式设置为循环
					if animInfo.Loop != nil {
						shouldLoop = *animInfo.Loop
						if !shouldLoop {
							log.Printf("[ReanimSystem] PlayAnimationWithConfig: 动画 %s (unit=%s) 配置为不循环", animName, unitID)
						}
					}
					break
				}
			}
		} else {
			log.Printf("[ReanimSystem] PlayAnimationWithConfig: 无法获取单位配置 %s: %v，使用默认循环设置", unitID, err)
		}
	}

	comp.IsLooping = shouldLoop

	// 重建动画数据
	s.rebuildAnimationData(comp)

	// 计算并缓存 CenterOffset
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

	if comp.MergedTracks == nil {
		comp.MergedTracks = reanim.BuildMergedTracks(comp.ReanimXML)
		comp.VisualTracks, comp.LogicalTracks = s.analyzeTrackTypes(comp.ReanimXML)
		comp.AnimationFPS = float64(comp.ReanimXML.FPS)
		comp.IsLooping = true
		comp.LastRenderFrame = -1
	}

	comp.CurrentAnimations = append(comp.CurrentAnimations, animName)

	// 如果 AnimationFrameIndices 已经存在但没有该动画的条目，添加初始值
	if comp.AnimationFrameIndices == nil {
		comp.AnimationFrameIndices = make(map[string]float64)
	}
	if _, exists := comp.AnimationFrameIndices[animName]; !exists {
		comp.AnimationFrameIndices[animName] = 0.0
		log.Printf("[ReanimSystem] AddAnimation: initialized frame index for '%s' = 0.0", animName)
	}

	// 重建动画数据（为新动画构建 AnimVisiblesMap）
	s.rebuildAnimationData(comp)

	// 标记缓存失效
	comp.LastRenderFrame = -1

	log.Printf("[ReanimSystem] AddAnimation: entity %d, added animation '%s', total animations: %d",
		entityID, animName, len(comp.CurrentAnimations))

	return nil
}

// finalizeAnimations 完成动画设置（内部方法）
// 新的渲染逻辑直接从动画遍历到轨道，无需绑定关系
//
// 参数：
//   - entityID: 实体 ID
//
// 返回：
//   - error: 如果实体不存在或没有 ReanimComponent，返回错误
func (s *ReanimSystem) finalizeAnimations(entityID ecs.EntityID) error {
	comp, ok := ecs.GetComponent[*components.ReanimComponent](s.entityManager, entityID)
	if !ok {
		return fmt.Errorf("entity %d does not have ReanimComponent", entityID)
	}

	// 确保每个动画都有独立的帧索引
	//         这样非循环动画（如 anim_open）在完成后保持在最后一帧
	if comp.AnimationFrameIndices == nil {
		comp.AnimationFrameIndices = make(map[string]float64)
	}
	for _, animName := range comp.CurrentAnimations {
		if _, exists := comp.AnimationFrameIndices[animName]; !exists {
			comp.AnimationFrameIndices[animName] = 0.0
		}
	}

	// 标记缓存失效
	comp.LastRenderFrame = -1

	log.Printf("[ReanimSystem] finalizeAnimations: entity %d, animations: %v, initialized frame indices",
		entityID, comp.CurrentAnimations)

	return nil
}

// PlayCombo 播放配置组合（推荐 API，应用所有配置）
// 从配置管理器读取 combo 配置，应用所有设置（hidden_tracks, parent_tracks, binding）
//
// Story 5.4.1: 支持运行时单位切换（如僵尸切换到烧焦僵尸）
// 当 unitID 与当前 ReanimName 不同时，自动重新加载 Reanim 数据
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

	// Story 5.4.1: 检测单位切换
	// 当请求的 unitID 与当前 ReanimName 不同时，需要重新加载 Reanim 数据
	// 注意：使用忽略大小写的比较，因为配置文件中的 ID 通常是小写，而 ReanimName 可能是原始大小写
	unitSwitched := false
	if comp.ReanimName != "" && !strings.EqualFold(comp.ReanimName, unitID) {
		// 单位切换：需要从 ResourceLoader 加载新的 Reanim 数据
		if s.resourceLoader == nil {
			return fmt.Errorf("cannot switch unit from %s to %s: resourceLoader not set", comp.ReanimName, unitID)
		}

		// 获取单位配置以确定 Reanim 文件名
		if s.configManager == nil {
			return fmt.Errorf("cannot switch unit: configManager not set")
		}
		unitConfig, err := s.configManager.GetUnit(unitID)
		if err != nil {
			return fmt.Errorf("failed to get config for unit %s: %w", unitID, err)
		}

		// 从配置中提取 Reanim 文件名（去掉路径和扩展名）
		// 优先从 ReanimFile 路径提取，因为这是实际的资源文件名
		// 例如 "data/reanim/Zombie.reanim" -> "Zombie"
		reanimFileName := ""
		if unitConfig.ReanimFile != "" {
			reanimFileName = strings.TrimSuffix(unitConfig.ReanimFile, ".reanim")
			if idx := strings.LastIndex(reanimFileName, "/"); idx != -1 {
				reanimFileName = reanimFileName[idx+1:]
			}
		}

		// 如果 ReanimFile 解析结果为空，回退到 Name
		if reanimFileName == "" {
			reanimFileName = unitConfig.Name
		}

		// 加载新的 Reanim 数据
		newReanimXML := s.resourceLoader.GetReanimXML(reanimFileName)
		newPartImages := s.resourceLoader.GetReanimPartImages(reanimFileName)

		if newReanimXML == nil || newPartImages == nil {
			return fmt.Errorf("failed to load Reanim resources for unit %s (file: %s)", unitID, reanimFileName)
		}

		log.Printf("[ReanimSystem] PlayCombo: 单位切换 %s -> %s，重新加载 Reanim 数据 (file: %s)",
			comp.ReanimName, unitID, reanimFileName)

		// 替换 Reanim 数据
		comp.ReanimXML = newReanimXML
		comp.PartImages = newPartImages
		comp.MergedTracks = nil // 清空，下面会重新构建
		unitSwitched = true
	}

	// 原因：plant_card_factory 等调用者只设置 ReanimXML 和 PartImages
	// 需要 PlayCombo 自动初始化 MergedTracks, VisualTracks 等字段
	// Story 5.4.1: 单位切换后也需要重新初始化
	if comp.MergedTracks == nil || unitSwitched {
		comp.ReanimName = unitID
		comp.MergedTracks = reanim.BuildMergedTracks(comp.ReanimXML)
		comp.VisualTracks, comp.LogicalTracks = s.analyzeTrackTypes(comp.ReanimXML)
		comp.AnimationFPS = float64(comp.ReanimXML.FPS)
		// IsLooping 默认为 true，会在后面根据配置覆盖
		comp.IsLooping = true
		comp.LastRenderFrame = -1
		// 清空旧的动画状态
		comp.HiddenTracks = nil
		comp.AnimationFrameIndices = nil
		comp.AnimationFPSOverrides = nil
		comp.AnimationSpeedOverrides = nil
		comp.AnimationLoopStates = nil
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

	if unitConfig == nil {
		return fmt.Errorf("unit config is nil for %s", unitID)
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

	// 重置每个动画的帧索引（修复：防止非循环动画检查误判为已完成）
	if comp.AnimationFrameIndices == nil {
		comp.AnimationFrameIndices = make(map[string]float64)
	}
	for _, animName := range comp.CurrentAnimations {
		comp.AnimationFrameIndices[animName] = 0.0
	}

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

	// 应用独立的动画循环状态（如果配置中指定了）
	if len(combo.AnimationLoopStates) > 0 {
		if comp.AnimationLoopStates == nil {
			comp.AnimationLoopStates = make(map[string]bool)
		}
		for animName, loopState := range combo.AnimationLoopStates {
			comp.AnimationLoopStates[animName] = loopState
			log.Printf("[ReanimSystem] PlayCombo: 动画 %s 独立循环状态 = %v", animName, loopState)
		}
	} else {
		// 清除之前的独立循环状态
		comp.AnimationLoopStates = nil
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

	// 新的渲染逻辑直接从动画遍历到轨道，无需绑定关系

	// 计算并缓存 CenterOffset（基于第一帧）
	// 检查配置中是否手动指定了 CenterOffset
	if len(unitConfig.CenterOffset) == 2 {
		// 使用配置指定的 CenterOffset
		comp.CenterOffsetX = unitConfig.CenterOffset[0]
		comp.CenterOffsetY = unitConfig.CenterOffset[1]
		log.Printf("[ReanimSystem] PlayCombo: 使用配置的 CenterOffset: %s → (%.1f, %.1f)",
			unitID, comp.CenterOffsetX, comp.CenterOffsetY)
	} else {
		// 自动计算 CenterOffset
		s.calculateCenterOffset(comp)
	}

	comp.LastRenderFrame = -1

	return nil
}

// PlayComboWithOptions 播放配置组合（带选项）
// 扩展 PlayCombo，支持动画进度保留等高级选项
//
// 参数：
//   - entityID: 实体 ID
//   - unitID: 单位 ID（如 "peashooter", "sunflower"）
//   - comboName: 组合名称（如 "attack", "idle"）。如果为空，使用第一个 combo
//   - preserveProgress: 是否保留当前动画进度（平滑过渡）
//
// 返回：
//   - error: 如果实体不存在、配置缺失，返回错误
func (s *ReanimSystem) PlayComboWithOptions(entityID ecs.EntityID, unitID, comboName string, preserveProgress bool) error {
	// 如果不需要保留进度，直接调用 PlayCombo
	if !preserveProgress {
		return s.PlayCombo(entityID, unitID, comboName)
	}

	// 获取当前动画进度
	comp, ok := ecs.GetComponent[*components.ReanimComponent](s.entityManager, entityID)
	if !ok {
		return fmt.Errorf("entity %d does not have ReanimComponent", entityID)
	}

	// 计算当前动画的相对进度（0.0 - 1.0）
	var currentProgress float64 = 0.0
	if len(comp.CurrentAnimations) > 0 && comp.AnimationFrameIndices != nil {
		// 使用第一个动画的进度作为参考
		firstAnim := comp.CurrentAnimations[0]
		if animVisibles, exists := comp.AnimVisiblesMap[firstAnim]; exists {
			visibleCount := countVisibleFrames(animVisibles)
			if visibleCount > 0 {
				currentFrameIndex := comp.AnimationFrameIndices[firstAnim]
				currentProgress = currentFrameIndex / float64(visibleCount)
				// 确保进度在 0-1 范围内
				if currentProgress > 1.0 {
					currentProgress = currentProgress - float64(int(currentProgress))
				}
				log.Printf("[ReanimSystem] PlayComboWithOptions: 保留进度 %.2f (frame %.1f / %d)",
					currentProgress, currentFrameIndex, visibleCount)
			}
		}
	}

	// 调用 PlayCombo 设置新动画
	err := s.PlayCombo(entityID, unitID, comboName)
	if err != nil {
		return err
	}

	// 如果有有效进度，应用到新动画
	if currentProgress > 0 {
		// 重新获取组件（PlayCombo 可能修改了它）
		comp, ok = ecs.GetComponent[*components.ReanimComponent](s.entityManager, entityID)
		if !ok {
			return nil // PlayCombo 成功，只是无法应用进度
		}

		// 为每个新动画设置相对进度
		for _, animName := range comp.CurrentAnimations {
			if animVisibles, exists := comp.AnimVisiblesMap[animName]; exists {
				visibleCount := countVisibleFrames(animVisibles)
				if visibleCount > 0 {
					newFrameIndex := currentProgress * float64(visibleCount)
					comp.AnimationFrameIndices[animName] = newFrameIndex
					log.Printf("[ReanimSystem] PlayComboWithOptions: 动画 %s 设置进度 %.2f (frame %.1f / %d)",
						animName, currentProgress, newFrameIndex, visibleCount)
				}
			}
		}

		// 同步 CurrentFrame
		if len(comp.CurrentAnimations) > 0 {
			firstAnim := comp.CurrentAnimations[0]
			comp.CurrentFrame = int(comp.AnimationFrameIndices[firstAnim])
		}

		// 标记缓存失效
		comp.LastRenderFrame = -1
	}

	return nil
}
