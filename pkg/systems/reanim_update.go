package systems

import (
	"fmt"
	"log"

	"github.com/decker502/pvz/pkg/components"
	"github.com/decker502/pvz/pkg/ecs"
)

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
		if cmd.UnitID != "" && cmd.AnimationName != "" && cmd.ComboName == "" {
			// 模式 3: 单动画模式（带配置）- 从 unitID 配置中读取 loop 设置
			log.Printf("[ReanimSystem] 执行单动画命令（带配置）: entity=%d, unit=%s, anim=%s", id, cmd.UnitID, cmd.AnimationName)
			err = s.PlayAnimationWithConfig(id, cmd.UnitID, cmd.AnimationName)
		} else if cmd.AnimationName != "" {
			// 模式 1: 单动画模式（无配置）- 默认循环
			log.Printf("[ReanimSystem] 执行单动画命令: entity=%d, anim=%s", id, cmd.AnimationName)
			err = s.PlayAnimation(id, cmd.AnimationName)
		} else if cmd.UnitID != "" {
			// 模式 2: 配置组合模式
			log.Printf("[ReanimSystem] 执行组合命令: entity=%d, unit=%s, combo=%s, preserveProgress=%v",
				id, cmd.UnitID, cmd.ComboName, cmd.PreserveProgress)
			err = s.PlayComboWithOptions(id, cmd.UnitID, cmd.ComboName, cmd.PreserveProgress)
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
// 逻辑说明:
//   - currentFrame 无限增长，不在 Update 中做循环检查
//   - 循环逻辑由各动画的 AnimationFrameIndices 独立处理
//   - 支持多动画组合（不同动画可以有独立的帧索引）
func (s *ReanimSystem) Update(deltaTime float64) {
	s.processAnimationCommands()

	// Story 8.8: 检查游戏是否冻结（僵尸获胜流程期间）
	// Phase 1: 所有实体动画暂停（包括触发僵尸）
	// Phase 2+: 只有触发僵尸的动画继续，其他实体暂停
	freezeEntities := ecs.GetEntitiesWith1[*components.GameFreezeComponent](s.entityManager)
	isFrozen := len(freezeEntities) > 0
	var triggerZombieID ecs.EntityID = 0
	var currentPhase int = 0

	if isFrozen {
		// 获取触发僵尸的ID和当前阶段
		phaseEntities := ecs.GetEntitiesWith1[*components.ZombiesWonPhaseComponent](s.entityManager)
		for _, phaseEntityID := range phaseEntities {
			phaseComp, ok := ecs.GetComponent[*components.ZombiesWonPhaseComponent](s.entityManager, phaseEntityID)
			if ok {
				triggerZombieID = phaseComp.TriggerZombieID
				currentPhase = phaseComp.CurrentPhase
				break
			}
		}
	}

	entities := ecs.GetEntitiesWith1[*components.ReanimComponent](s.entityManager)

	for _, id := range entities {
		comp, exists := ecs.GetComponent[*components.ReanimComponent](s.entityManager, id)
		if !exists {
			continue
		}

		// Story 8.8: 游戏冻结时的动画暂停逻辑
		// Phase 1: 所有非UI实体动画暂停（包括触发僵尸）
		// Phase 2+: 只有触发僵尸的动画继续，其他非UI实体暂停
		if isFrozen {
			// 检查是否是 UI 元素
			_, isUI := ecs.GetComponent[*components.UIComponent](s.entityManager, id)

			if !isUI {
				// Phase 1: 所有非UI实体动画暂停
				if currentPhase == 1 {
					continue
				}

				// Phase 2+: 只有触发僵尸的动画继续
				if triggerZombieID != 0 && id != triggerZombieID {
					continue
				}
			}
			// UI 元素继续更新（不跳过）
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
			// 🔍 调试：打印 CrazyDave 动画处理
			if comp.ReanimName == "crazydave" || comp.ReanimName == "CrazyDave" {
				log.Printf("[ReanimSystem] 🎩 CrazyDave 处理动画: %s, 帧索引: %.2f, FPS: %.1f",
					animName, comp.AnimationFrameIndices[animName], comp.AnimationFPS)
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
					// 🔍 调试：打印 CrazyDave 非循环动画检查
					if (comp.ReanimName == "crazydave" || comp.ReanimName == "CrazyDave") && int(currentFrame) < 5 {
						log.Printf("[ReanimSystem] 🎩 CrazyDave 非循环检查: anim=%s, frame=%.2f, visibleCount=%d",
							animName, currentFrame, visibleCount)
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
				if speed, hasOverride := comp.AnimationSpeedOverrides[animName]; hasOverride {
					animSpeed = speed // 允许 speed = 0 来完全禁用自动推进
				}
			}

			// 推进该动画的帧索引（应用速度倍率）
			// frameIncrement = (FPS / targetTPS) * speedMultiplier
			// 例如：FPS=12, TPS=60, speed=0.2 → increment = (12/60) * 0.2 = 0.04 帧/tick
			frameIncrement := (animFPS / s.targetTPS) * animSpeed
			comp.AnimationFrameIndices[animName] += frameIncrement

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
			} else {
				// 非循环动画：不需要强制限制在最后一帧
				// 前面的逻辑（visibleCount > 0 && int(currentFrame) >= visibleCount）会负责停止更新
				// 让 indices 自然保持在 >= visibleCount 的状态，以便 IsFinished 可以被触发
				// 如果强制拉回 visibleCount-1，会导致 CurrentFrame 永远小于 visibleCount，IsFinished 永远为 false
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

			// 对于非循环动画，即使已完成也要更新一次 CurrentFrame
			// 这样 CurrentFrame 才能达到 maxVisibleFrames，触发 IsFinished
			if !isLooping {
				// 检查该动画是否已完成
				if animVisibles, exists := comp.AnimVisiblesMap[animName]; exists {
					visibleCount := countVisibleFrames(animVisibles)
					currentFrame := comp.AnimationFrameIndices[animName]
					// 修复：允许 CurrentFrame 达到 visibleCount（而不是跳过）
					// 只有当帧索引远超过 visibleCount 时才跳过（例如 > visibleCount + 1）
					if visibleCount > 0 && int(currentFrame) > visibleCount {
						// 非循环动画已完成且 CurrentFrame 已更新过，跳过
						if comp.ReanimName == "SelectorScreen" {
							log.Printf("[ReanimSystem] ⏭️  跳过已完成的动画 %s（帧 %.2f > %d）", animName, currentFrame, visibleCount)
						}
						continue
					}
				}
			}

			// 使用这个活跃动画的帧索引更新 CurrentFrame
			comp.CurrentFrame = int(comp.AnimationFrameIndices[animName])
			foundActiveAnim = true
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

		// 更新叠加动画帧（如旗帜僵尸的旗杆动画）
		if comp.OverlayReanimXML != nil {
			overlayFPS := float64(comp.OverlayReanimXML.FPS)
			if overlayFPS <= 0 {
				overlayFPS = 12.0
			}

			// 推进叠加动画帧
			comp.OverlayFrameAccumulator += deltaTime
			frameTime := 1.0 / overlayFPS
			if comp.OverlayFrameAccumulator >= frameTime {
				comp.OverlayCurrentFrame++
				comp.OverlayFrameAccumulator -= frameTime

				// 循环播放
				if comp.OverlayMergedTracks != nil {
					// 获取第一个轨道的帧数作为总帧数
					for _, track := range comp.OverlayReanimXML.Tracks {
						if frames, ok := comp.OverlayMergedTracks[track.Name]; ok && len(frames) > 0 {
							if comp.OverlayCurrentFrame >= len(frames) {
								comp.OverlayCurrentFrame = 0
							}
							break
						}
					}
				}
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
