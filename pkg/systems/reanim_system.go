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
// 基于 animation_showcase/AnimationCell 重写，简化并修复 遗留问题
//
// - API 数量从 50+ 减少到 2 个核心 API
// - 代码行数从 2808 减少到 ~1000 行
// - 与 AnimationCell 保持一致的逻辑
type ReanimSystem struct {
	entityManager *ecs.EntityManager
	configManager *config.ReanimConfigManager

	// 游戏 TPS（用于帧推进计算）
	targetTPS float64

	enableCommandCleanup bool    // 是否启用自动清理
	cleanupInterval      float64 // 清理间隔（秒）
	cleanupTimer         float64 // 清理计时器
}

// NewReanimSystem 创建新的 Reanim 动画系统
func NewReanimSystem(em *ecs.EntityManager) *ReanimSystem {
	return &ReanimSystem{
		entityManager:        em,
		targetTPS:            60.0, // 默认 60 TPS
		enableCommandCleanup: false,
		cleanupInterval:      1.0, // 每秒清理一次
		cleanupTimer:         0.0,
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

// SetCommandCleanup 设置命令清理策略（可选 API）
// 用于配置动画命令组件的自动清理
func (s *ReanimSystem) SetCommandCleanup(enable bool, interval float64) {
	s.enableCommandCleanup = enable
	s.cleanupInterval = interval
	log.Printf("[ReanimSystem] 命令清理配置: enable=%v, interval=%.2f秒", enable, interval)
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

	// 原因：zombie_factory 等调用者只设置 ReanimXML 和 PartImages
	// rebuildAnimationData 需要 MergedTracks 存在
	if comp.MergedTracks == nil {
		comp.MergedTracks = reanim.BuildMergedTracks(comp.ReanimXML)
		comp.VisualTracks, comp.LogicalTracks = s.analyzeTrackTypes(comp.ReanimXML)
		comp.AnimationFPS = float64(comp.ReanimXML.FPS)
		comp.IsLooping = true
		comp.LastRenderFrame = -1
	}

	// 单个动画模式下，不使用 HiddenTracks, ParentTracks
	// 这些都依赖 Reanim 文件本身的定义
	comp.HiddenTracks = nil
	comp.ParentTracks = nil

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

	// 新的渲染逻辑直接从动画遍历到轨道，无需绑定关系

	// 计算并缓存 CenterOffset（基于第一帧）
	s.calculateCenterOffset(comp)

	comp.LastRenderFrame = -1

	return nil
}

// ==================================================================
// 系统更新 (System Update)
// ==================================================================

// processAnimationCommands 处理所有待执行的动画命令
//
// 组件驱动的动画命令处理机制
//
// 设计说明：
//   - 在 Update() 开头调用，优先处理命令
//   - 查询所有带有 AnimationCommandComponent 的实体
//   - 执行未处理的命令（Processed == false）
//   - 标记为已处理（Processed = true）
//   - 可选：定期清理已处理的命令组件
//
// 执行逻辑：
//  1. 如果 AnimationName 非空 → 调用 PlayAnimation()
//  2. 否则 → 调用 PlayCombo(UnitID, ComboName)
//
// 错误处理：
//   - 记录错误日志但不中断处理流程
//   - 即使执行失败也标记 Processed = true（避免无限重试）
//
// 性能优化：
//   - 使用泛型 ECS API (GetEntitiesWith1)
//   - 跳过已处理的命令
//   - 批量处理（一次 Update 处理多个命令）
func (s *ReanimSystem) processAnimationCommands() {
	// 1. 查询所有带有 AnimationCommand 的实体
	entities := ecs.GetEntitiesWith1[*components.AnimationCommandComponent](s.entityManager)

	// 2. 统计信息（用于调试）
	processedCount := 0
	errorCount := 0

	// 3. 处理每个命令
	for _, id := range entities {
		cmd, ok := ecs.GetComponent[*components.AnimationCommandComponent](s.entityManager, id)
		if !ok {
			continue
		}

		// 跳过已处理的命令
		if cmd.Processed {
			continue
		}

		// 执行命令
		var err error
		if cmd.AnimationName != "" {
			// 模式 1: 单动画模式
			log.Printf("[ReanimSystem] 执行单动画命令: entity=%d, anim=%s", id, cmd.AnimationName)
			err = s.PlayAnimation(id, cmd.AnimationName)
		} else if cmd.UnitID != "" {
			// 模式 2: 配置组合模式
			log.Printf("[ReanimSystem] 执行组合命令: entity=%d, unit=%s, combo=%s", id, cmd.UnitID, cmd.ComboName)
			err = s.PlayCombo(id, cmd.UnitID, cmd.ComboName)
		} else {
			// 错误：无效命令
			log.Printf("[ReanimSystem] 无效命令: entity=%d, UnitID 和 AnimationName 都为空", id)
			err = fmt.Errorf("invalid command: both UnitID and AnimationName are empty")
		}

		// 处理错误
		if err != nil {
			log.Printf("[ReanimSystem] 命令执行失败: entity=%d, unit=%s, combo=%s, anim=%s, err=%v",
				id, cmd.UnitID, cmd.ComboName, cmd.AnimationName, err)
			errorCount++
		} else {
			processedCount++
		}

		// 标记为已处理（即使失败也标记，避免无限重试）
		cmd.Processed = true
	}

	// 4. 日志统计（仅在有命令时输出）
	if processedCount > 0 || errorCount > 0 {
		log.Printf("[ReanimSystem] 命令处理完成: 成功=%d, 失败=%d", processedCount, errorCount)
	}
}

// Update 更新所有 Reanim 组件的动画帧
// 基于 AnimationCell.Update() 的逻辑
//   - currentFrame 无限增长，不在 Update 中做循环检查
//   - 循环逻辑完全由 findControllingAnimation 的取模处理
//   - 支持多动画组合（不同轨道可以有不同的帧数）
func (s *ReanimSystem) Update(deltaTime float64) {
	s.processAnimationCommands()

	entities := ecs.GetEntitiesWith1[*components.ReanimComponent](s.entityManager)

	// Debug: 输出 SelectorScreen 的更新信息（前 5 次）
	for _, id := range entities {
		comp, exists := ecs.GetComponent[*components.ReanimComponent](s.entityManager, id)
		if exists && comp.ReanimName == "SelectorScreen" && comp.CurrentFrame < 5 {
			log.Printf("[ReanimSystem] 🔍 Update: SelectorScreen entity %d, frame=%d, animations=%v",
				id, comp.CurrentFrame, comp.CurrentAnimations)
		}
	}

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

		// 初始化 AnimationFrameIndices（如果尚未初始化）
		if comp.AnimationFrameIndices == nil {
			comp.AnimationFrameIndices = make(map[string]float64)
			for _, animName := range comp.CurrentAnimations {
				comp.AnimationFrameIndices[animName] = 0.0
			}
		}

		// 为每个动画独立推进帧
		for _, animName := range comp.CurrentAnimations {
			// 🔍 调试：打印所有动画的处理情况
			if comp.ReanimName == "SelectorScreen" && (animName == "anim_idle" || animName == "anim_grass") {
				log.Printf("[ReanimSystem] 🔍 处理动画: %s, 帧索引: %.2f", animName, comp.AnimationFrameIndices[animName])
			}

			// 检查是否暂停
			if comp.AnimationPausedStates != nil {
				if isPaused, exists := comp.AnimationPausedStates[animName]; exists && isPaused {
					if comp.ReanimName == "SelectorScreen" && (animName == "anim_idle" || animName == "anim_grass") {
						log.Printf("[ReanimSystem] ⏸️  动画 %s 已暂停，跳过", animName)
					}
					continue // 跳过暂停的动画
				}
			}

			// 如果该动画是非循环的，检查是否已完成
			isLooping := comp.IsLooping // 默认使用全局循环状态
			if comp.AnimationLoopStates != nil {
				if loopState, hasState := comp.AnimationLoopStates[animName]; hasState {
					isLooping = loopState
				}
			}

			// 🔍 调试：打印循环状态
			if comp.ReanimName == "SelectorScreen" && (animName == "anim_idle" || animName == "anim_grass") {
				log.Printf("[ReanimSystem] 🔍 动画 %s 循环状态: isLooping=%v", animName, isLooping)
			}
			if !isLooping {
				// 检查该动画是否已完成
				if animVisibles, exists := comp.AnimVisiblesMap[animName]; exists {
					visibleCount := countVisibleFrames(animVisibles)
					currentFrame := comp.AnimationFrameIndices[animName]

					// 🔍 调试：打印 SelectorScreen 的 anim_open 帧信息
					if comp.ReanimName == "SelectorScreen" && animName == "anim_open" && int(currentFrame) < 15 {
						log.Printf("[ReanimSystem] 🔍 检查 anim_open: currentFrame=%.2f, visibleCount=%d, isLooping=%v",
							currentFrame, visibleCount, isLooping)
					}

					if visibleCount > 0 && int(currentFrame) >= visibleCount {
						// 非循环动画已完成，停止更新帧
						if comp.ReanimName == "SelectorScreen" && animName == "anim_open" {
							log.Printf("[ReanimSystem] anim_open 已完成，停止更新帧")
						}
						continue
					}
				}
			}

			// 获取该动画的 FPS
			animFPS := comp.AnimationFPS // 默认使用全局 FPS
			if comp.AnimationFPSOverrides != nil {
				if fps, hasOverride := comp.AnimationFPSOverrides[animName]; hasOverride {
					animFPS = fps
				}
			}

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
			oldFrameIndex := comp.AnimationFrameIndices[animName]
			comp.AnimationFrameIndices[animName] += frameIncrement

			// Debug: 豌豆射手的帧推进（前10帧）
			if (comp.ReanimName == "peashooter" || comp.ReanimName == "peashootersingle") && int(oldFrameIndex) < 10 {
				log.Printf("[ReanimSystem] 帧推进: anim=%s, %.2f -> %.2f (increment=%.4f, FPS=%.1f, speed=%.2f)",
					animName, oldFrameIndex, comp.AnimationFrameIndices[animName], frameIncrement, animFPS, animSpeed)
			}

			if isLooping {
				if animVisibles, exists := comp.AnimVisiblesMap[animName]; exists {
					visibleCount := countVisibleFrames(animVisibles)
					if visibleCount > 0 && comp.AnimationFrameIndices[animName] >= float64(visibleCount) {
						// 对循环动画取模，保持在有效范围内
						comp.AnimationFrameIndices[animName] = float64(int(comp.AnimationFrameIndices[animName]) % visibleCount)

						// 🔍 调试：记录循环重置
						if comp.ReanimName == "SelectorScreen" && (animName == "anim_idle" || animName == "anim_grass") {
							log.Printf("[ReanimSystem] 🔁 动画 %s 循环重置到 %.2f", animName, comp.AnimationFrameIndices[animName])
						}
					}
				}
			}
		}

		// 同步更新 CurrentFrame（用于后备和非循环动画检测）
		// 使用第一个**活跃的**（正在播放的）动画的帧索引
		foundActiveAnim := false
		for _, animName := range comp.CurrentAnimations {
			// 跳过暂停的动画
			if comp.AnimationPausedStates != nil {
				if isPaused, exists := comp.AnimationPausedStates[animName]; exists && isPaused {
					continue
				}
			}

			isLooping := comp.IsLooping
			if comp.AnimationLoopStates != nil {
				if loopState, hasState := comp.AnimationLoopStates[animName]; hasState {
					isLooping = loopState
				}
			}
			if !isLooping {
				// 检查该动画是否已完成
				if animVisibles, exists := comp.AnimVisiblesMap[animName]; exists {
					visibleCount := countVisibleFrames(animVisibles)
					currentFrame := comp.AnimationFrameIndices[animName]
					if visibleCount > 0 && int(currentFrame) >= visibleCount {
						// 非循环动画已完成，跳过
						if comp.ReanimName == "SelectorScreen" {
							log.Printf("[ReanimSystem] ⏭️  跳过已完成的动画 %s（帧 %.2f >= %d）", animName, currentFrame, visibleCount)
						}
						continue
					}
				}
			}

			// 使用这个活跃动画的帧索引更新 CurrentFrame
			comp.CurrentFrame = int(comp.AnimationFrameIndices[animName])
			foundActiveAnim = true
			// Debug: 豌豆射手的帧更新（前10帧）
			if (comp.ReanimName == "peashooter" || comp.ReanimName == "peashootersingle") && comp.CurrentFrame < 10 {
				log.Printf("[ReanimSystem] CurrentFrame更新: anim=%s, frameIndex=%.2f, CurrentFrame=%d",
					animName, comp.AnimationFrameIndices[animName], comp.CurrentFrame)
			}
			if comp.ReanimName == "SelectorScreen" && comp.CurrentFrame < 5 {
				log.Printf("[ReanimSystem] 使用动画 %s 更新 CurrentFrame = %d", animName, comp.CurrentFrame)
			}
			break
		}

		// 🔍 调试：如果没有找到活跃动画，记录一下
		if !foundActiveAnim && comp.ReanimName == "SelectorScreen" {
			log.Printf("[ReanimSystem]  没有找到活跃动画，CurrentFrame 保持不变 = %d", comp.CurrentFrame)
		}

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
							animFrame := comp.AnimationFrameIndices[animName]
							// 检查该动画是否完成
							if visibleCount > 0 && int(animFrame) < visibleCount {
								allNonLoopingComplete = false
								if comp.ReanimName == "SelectorScreen" {
									log.Printf("[ReanimSystem] 🔍 非循环动画 %s 尚未完成: 帧 %.2f < %d", animName, animFrame, visibleCount)
								}
								break
							} else if comp.ReanimName == "SelectorScreen" {
								log.Printf("[ReanimSystem] 非循环动画 %s 已完成: 帧 %.2f >= %d", animName, animFrame, visibleCount)
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

	s.cleanupProcessedCommands(deltaTime)
}

// cleanupProcessedCommands 清理已处理的命令组件（可选功能）
//
// 命令清理机制
//
// 设计说明：
//   - 定期调用（如每秒一次）以释放内存
//   - 仅在调试模式下保留命令历史
//   - 可配置清理策略
//
// 调用时机：
//   - 在 Update() 结尾调用
//   - 使用计时器控制频率（避免每帧都清理）
func (s *ReanimSystem) cleanupProcessedCommands(deltaTime float64) {
	// 检查是否启用清理（可通过配置控制）
	if !s.enableCommandCleanup {
		return
	}

	// 更新清理计时器
	s.cleanupTimer += deltaTime
	if s.cleanupTimer < s.cleanupInterval {
		return // 未到清理时间
	}
	s.cleanupTimer = 0

	// 查询并移除已处理的命令
	entities := ecs.GetEntitiesWith1[*components.AnimationCommandComponent](s.entityManager)
	removedCount := 0

	for _, id := range entities {
		cmd, ok := ecs.GetComponent[*components.AnimationCommandComponent](s.entityManager, id)
		if ok && cmd.Processed {
			// 移除组件（使用泛型 API）
			ecs.RemoveComponent[*components.AnimationCommandComponent](s.entityManager, id)
			removedCount++
		}
	}

	if removedCount > 0 {
		log.Printf("[ReanimSystem] 清理已处理命令: 移除=%d", removedCount)
	}
}

// ==================================================================
// 渲染缓存 (Render Cache)
// ==================================================================

// prepareRenderCache 准备渲染缓存
// 新逻辑：外层循环动画，内层循环轨道，后面的动画自然覆盖前面的动画
func (s *ReanimSystem) prepareRenderCache(comp *components.ReanimComponent) {
	// Debug: 无条件打印向日葵和 SodRoll 的缓存准备
	if comp.ReanimName == "sunflower" && comp.CurrentFrame < 3 {
		log.Printf("[ReanimSystem] 🌻 prepareRenderCache 被调用: frame=%d", comp.CurrentFrame)
	}
	if comp.ReanimName == "sodroll" && comp.CurrentFrame < 3 {
		log.Printf("[ReanimSystem] 🟫 SodRoll prepareRenderCache 被调用: frame=%d, VisualTracks=%d",
			comp.CurrentFrame, len(comp.VisualTracks))
	}
	if comp.ReanimName == "SelectorScreen" && comp.CurrentFrame < 30 {
		log.Printf("[ReanimSystem] 🎬 SelectorScreen prepareRenderCache 被调用: frame=%d, animations=%v",
			comp.CurrentFrame, comp.CurrentAnimations)
	}

	// 重用切片避免分配
	comp.CachedRenderData = comp.CachedRenderData[:0]

	visibleCount := 0
	skippedHidden := 0
	skippedPaused := 0
	skippedNoFrames := 0
	skippedNoImage := 0

	trackRenderSource := make(map[string]string)

	// 这样可以确保云朵轨道（Track 16-21）在按钮轨道（Track 27+）之前添加到 CachedRenderData
	// 从而在渲染时云朵在下面，按钮在上面
	for _, trackName := range comp.VisualTracks {
		// Debug: SelectorScreen 打印轨道处理情况（前10帧）
		if comp.ReanimName == "SelectorScreen" && comp.CurrentFrame < 10 {
			log.Printf("[ReanimSystem] 🎨 处理轨道: %s", trackName)
		}

		// 检查隐藏轨道（黑名单）
		if comp.HiddenTracks != nil && comp.HiddenTracks[trackName] {
			skippedHidden++
			continue
		}

		// 获取该轨道的合并帧数据
		mergedFrames, ok := comp.MergedTracks[trackName]
		if !ok {
			skippedNoFrames++
			continue
		}

		// 用于存储该轨道的最终选中数据（后面的动画会覆盖前面的）
		var selectedFrame reanim.Frame
		var selectedImg *ebiten.Image
		var selectedOffsetX, selectedOffsetY float64
		var hasValidSelection bool

		// 内层循环：遍历所有动画，找到最后一个有效的数据
		for _, animName := range comp.CurrentAnimations {
			// 检查动画是否暂停
			if comp.AnimationPausedStates != nil {
				if isPaused, exists := comp.AnimationPausedStates[animName]; exists && isPaused {
					skippedPaused++
					continue
				}
			}

			// 获取该动画的当前逻辑帧（支持独立帧索引）
			var logicalFrame float64
			if comp.AnimationFrameIndices != nil {
				if frame, exists := comp.AnimationFrameIndices[animName]; exists {
					logicalFrame = frame
				} else {
					logicalFrame = float64(comp.CurrentFrame)
				}
			} else {
				logicalFrame = float64(comp.CurrentFrame)
			}

			// 获取动画的可见性数组
			animVisibles, ok := comp.AnimVisiblesMap[animName]
			if !ok {
				if comp.ReanimName == "simple_pea" {
					log.Printf("[ReanimSystem] simple_pea: AnimVisiblesMap[%s] 不存在, VisualTracks=%v, CurrentAnimations=%v",
						animName, comp.VisualTracks, comp.CurrentAnimations)
				}
				continue
			}

			// 映射逻辑帧到物理帧
			physicalFrame := mapLogicalToPhysical(int(logicalFrame), animVisibles)
			if physicalFrame < 0 || physicalFrame >= len(mergedFrames) {
				continue
			}

			// 检查动画定义轨道是否可见（f != -1）
			animDefTrack, ok := comp.MergedTracks[animName]
			if !ok || physicalFrame >= len(animDefTrack) {
				continue
			}

			defFrame := animDefTrack[physicalFrame]
			if defFrame.FrameNum != nil && *defFrame.FrameNum == -1 {
				// 动画隐藏，跳过整个动画
				continue
			}

			// 检查视觉轨道在该帧是否被隐藏（f=-1）
			currentTrackFrame := mergedFrames[physicalFrame]
			if currentTrackFrame.FrameNum != nil && *currentTrackFrame.FrameNum == -1 {
				// 视觉轨道在该帧被隐藏，跳过
				skippedHidden++
				if comp.ReanimName == "SelectorScreen" && comp.CurrentFrame < 10 {
					log.Printf("[ReanimSystem]   - 动画 %s: 轨道被隐藏 (f=-1)", animName)
				}
				continue
			}

			// 使用帧插值获取平滑的帧数据
			frame := s.getInterpolatedFrame(animName, logicalFrame, animVisibles, mergedFrames)

			// 图片继承逻辑：如果插值后的帧没有图片，向前搜索最近的有图片的帧
			hasValidImage := false
			if frame.ImagePath == "" {
				// 向前搜索有图片的帧（只搜索当前动画的可见帧范围）
				for i := physicalFrame - 1; i >= 0; i-- {
					isFrameVisible := false
					for _, visibleFrame := range animVisibles {
						if visibleFrame == i {
							isFrameVisible = true
							break
						}
					}
					if !isFrameVisible {
						break
					}

					if i < len(mergedFrames) && mergedFrames[i].ImagePath != "" {
						frame.ImagePath = mergedFrames[i].ImagePath
						hasValidImage = true
						break
					}
				}
			} else {
				hasValidImage = true
			}

			// 如果当前动画在这个轨道没有有效图片，跳过
			if !hasValidImage {
				skippedNoImage++
				if comp.ReanimName == "SelectorScreen" && comp.CurrentFrame < 10 {
					log.Printf("[ReanimSystem]   - 动画 %s: 无有效图片", animName)
				}
				continue
			}

			// 获取图片
			img, ok := comp.PartImages[frame.ImagePath]
			if !ok || img == nil {
				if comp.ReanimName == "simple_pea" {
					partImagesKeys := make([]string, 0, len(comp.PartImages))
					for k := range comp.PartImages {
						partImagesKeys = append(partImagesKeys, k)
					}
					log.Printf("[ReanimSystem] simple_pea: PartImages[%s] 不存在或为 nil (ok=%v, img==nil=%v), PartImages keys=%v",
						frame.ImagePath, ok, (img == nil), partImagesKeys)
				}
				continue
			}

			// 计算父轨道偏移
			offsetX, offsetY := 0.0, 0.0
			if parentTrackName, hasParent := comp.ParentTracks[trackName]; hasParent {
				offsetX, offsetY = s.getParentOffsetForAnimation(comp, parentTrackName, animName)
				// Debug: 豌豆射手的父偏移（前10帧）
				if (comp.ReanimName == "peashooter" || comp.ReanimName == "peashootersingle") && comp.CurrentFrame < 10 {
					log.Printf("[ReanimSystem] ParentOffset: track=%s, parent=%s, anim=%s, offset=(%.2f, %.2f)",
						trackName, parentTrackName, animName, offsetX, offsetY)
				}
			}

			// 更新选中数据（后面的动画会覆盖前面的）
			selectedFrame = frame
			selectedImg = img
			selectedOffsetX = offsetX
			selectedOffsetY = offsetY
			hasValidSelection = true
			trackRenderSource[trackName] = animName

			// Debug: SelectorScreen 记录选中的动画
			if comp.ReanimName == "SelectorScreen" && comp.CurrentFrame < 10 {
				log.Printf("[ReanimSystem]   - 动画 %s: 有效数据，选中", animName)
			}
		}

		// 如果该轨道有有效选中数据，添加到缓存
		if hasValidSelection {
			comp.CachedRenderData = append(comp.CachedRenderData, components.RenderPartData{
				Img:     selectedImg,
				Frame:   selectedFrame,
				OffsetX: selectedOffsetX,
				OffsetY: selectedOffsetY,
			})
			visibleCount++
		}
	}

	if comp.ReanimName == "SelectorScreen" && comp.CurrentFrame < 10 {
		log.Printf("[ReanimSystem] 📊 Frame %d 渲染统计 (总计: %d 个轨道):", comp.CurrentFrame, visibleCount)
		for _, trackName := range comp.VisualTracks {
			if source, ok := trackRenderSource[trackName]; ok {
				log.Printf("    - 轨道 %s: 来自动画 %s", trackName, source)
			}
		}
	}

	// Debug: 只在有变化时输出日志（避免刷屏）
	// 特殊调试：向日葵每帧都打印（前 10 帧）
	if comp.ReanimName == "sunflower" && comp.CurrentFrame < 10 {
		log.Printf("[ReanimSystem] 🔍 SunFlower frame %d → %d visible parts (skipped: hidden=%d, paused=%d, noFrames=%d, noImage=%d)",
			comp.CurrentFrame, visibleCount, skippedHidden, skippedPaused, skippedNoFrames, skippedNoImage)
	} else if len(comp.CachedRenderData) > 0 && comp.CurrentFrame%30 == 0 {
		log.Printf("[ReanimSystem] prepareRenderCache: %s frame %d → %d visible parts (skipped: hidden=%d, paused=%d, noFrames=%d, noImage=%d)",
			comp.ReanimName, comp.CurrentFrame, visibleCount, skippedHidden, skippedPaused, skippedNoFrames, skippedNoImage)
	}
}

// GetRenderData 获取渲染数据（供 RenderSystem 使用）
// 如果缓存失效，会自动重建缓存
func (s *ReanimSystem) GetRenderData(entityID ecs.EntityID) []components.RenderPartData {
	comp, ok := ecs.GetComponent[*components.ReanimComponent](s.entityManager, entityID)
	if !ok {
		return nil
	}

	// 问题：使用整数 CurrentFrame 判断缓存失效，导致慢速动画（如 speed=0.05）
	//       的插值帧被忽略（帧 0.05、0.10...0.95 都被当作帧 0）
	// 解决：检查任意动画的浮点帧索引是否改变，确保插值生效
	needRebuild := false

	// 方法 1: 检查 AnimationFrameIndices 中是否有任何帧索引发生变化
	if comp.AnimationFrameIndices != nil && len(comp.AnimationFrameIndices) > 0 {
		// 计算所有动画的帧索引之和（作为缓存键）
		currentFrameSum := 0.0
		for _, frameIdx := range comp.AnimationFrameIndices {
			currentFrameSum += frameIdx
		}

		// 如果帧索引和发生变化，或者是首次渲染
		if comp.LastRenderFrame == -1 || float64(comp.LastRenderFrame) != currentFrameSum {
			needRebuild = true
			comp.LastRenderFrame = int(currentFrameSum * 1000) // 使用千分之一精度作为缓存键
		}
	} else {
		// 后备逻辑：使用整数 CurrentFrame（兼容旧代码）
		if comp.LastRenderFrame != comp.CurrentFrame {
			needRebuild = true
			comp.LastRenderFrame = comp.CurrentFrame
		}
	}

	// Debug: SelectorScreen 前30帧打印
	if comp.ReanimName == "SelectorScreen" && comp.CurrentFrame < 30 {
		log.Printf("[ReanimSystem] 🎨 GetRenderData: frame=%d, lastRenderFrame=%d, needRebuild=%v",
			comp.CurrentFrame, comp.LastRenderFrame, needRebuild)
	}

	// 重建缓存
	if needRebuild {
		s.prepareRenderCache(comp)
	}

	return comp.CachedRenderData
}

// ==================================================================
// 辅助方法 (Helper Methods)
// ==================================================================

// rebuildAnimationData 重建动画数据（AnimVisiblesMap）
// 基于 AnimationCell.rebuildAnimationData()
func (s *ReanimSystem) rebuildAnimationData(comp *components.ReanimComponent) {
	if comp.ReanimName == "simple_pea" {
		log.Printf("[ReanimSystem] 🔍 rebuildAnimationData 被调用: ReanimName=%s, CurrentAnimations=%v, VisualTracks=%v",
			comp.ReanimName, comp.CurrentAnimations, comp.VisualTracks)
	}

	comp.AnimVisiblesMap = make(map[string][]int)

	// 1. 为当前播放的动画创建可见性数组
	for _, animName := range comp.CurrentAnimations {
		animVisibles := buildVisiblesArray(comp.ReanimXML, comp.MergedTracks, animName)
		comp.AnimVisiblesMap[animName] = animVisibles

		if comp.ReanimName == "simple_pea" {
			log.Printf("[ReanimSystem] 🔍 buildVisiblesArray(%s) = %v (len=%d)", animName, animVisibles, len(animVisibles))
		}
	}

	// 为 ParentTracks 中的父轨道创建可见性数组
	// 父轨道不在 CurrentAnimations 中，但计算父偏移时需要它们的可见性数组
	if comp.ParentTracks != nil {
		for _, parentTrackName := range comp.ParentTracks {
			// 如果该父轨道还没有可见性数组，创建一个
			if _, exists := comp.AnimVisiblesMap[parentTrackName]; !exists {
				animVisibles := buildVisiblesArray(comp.ReanimXML, comp.MergedTracks, parentTrackName)
				comp.AnimVisiblesMap[parentTrackName] = animVisibles
			}
		}
	}
}

// 新的渲染逻辑不再需要轨道绑定机制，直接从动画到轨道渲染

// getInterpolatedFrame 获取插值后的帧数据
// 参数：
//   - animName: 动画名称
//   - logicalFrame: 浮点逻辑帧索引（如 2.7 表示第 2 帧和第 3 帧之间，插值因子 0.7）
//   - animVisibles: 动画可见性数组
//   - mergedFrames: 轨道的累加帧数组
//
// 返回：插值后的帧数据
func (s *ReanimSystem) getInterpolatedFrame(
	animName string,
	logicalFrame float64,
	animVisibles []int,
	mergedFrames []reanim.Frame,
) reanim.Frame {
	// 1. 获取整数部分和小数部分
	frame1Index := int(logicalFrame)         // 当前帧索引
	frame2Index := frame1Index + 1           // 下一帧索引
	t := logicalFrame - float64(frame1Index) // 插值因子 (0.0 - 1.0)

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
// 简化版本，使用配置驱动的方式
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
	// 替代旧的 CenterOffset 机制
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
