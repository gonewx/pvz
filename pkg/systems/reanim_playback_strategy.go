package systems

import (
	"log"

	"github.com/decker502/pvz/internal/reanim"
	"github.com/decker502/pvz/pkg/components"
	"github.com/decker502/pvz/pkg/config"
)

// ==================================================================
// Story 6.6: Reanim Playback Mode System (Reanim 播放模式系统)
// ==================================================================
//
// 本文件实现了 5 种通用 Reanim 播放模式，用于替代之前的硬编码逻辑。
//
// **背景**：
// - 原系统为豌豆射手硬编码了双动画叠加
// - 为 SelectorScreen 实现了特殊的 PlayAllFrames() 方法
// - 60% 的植物使用混合模式，但系统缺少通用支持
//
// **解决方案**：
// 通过分析 127 个 Reanim 文件，识别出 5 种明确的动画模式：
// 1. 简单动画（5%）- 单轨道循环
// 2. 骨骼动画（20%）- 多轨道同步
// 3. 序列动画（15%）- 时间窗口控制
// 4. 复杂场景动画（5%）- 独立时间线
// 5. 混合模式（60%）- 状态机 + 骨骼动画
//
// **参考文档**：
// - docs/qa/sprint-change-proposal-reanim-system-redesign.md
// - docs/reanim/reanim-format-guide.md
// - docs/stories/6.6.story.md
//
// ==================================================================

// PlaybackMode 定义 Reanim 动画的播放模式
type PlaybackMode int

const (
	// ModeSimple 是单轨道简单动画模式（5% 的文件）
	// 特征：1-2 个轨道，无 f 值，简单变换
	// 示例：Lilypad.reanim
	// 实现：直接循环播放
	ModeSimple PlaybackMode = iota

	// ModeSkeleton 是多轨道骨骼动画模式（20% 的文件）
	// 特征：4-30 个轨道，每个轨道代表一个部件，同步播放
	// 示例：Sun.reanim（10 轨道），SunFlower.reanim（30 轨道）
	// 实现：所有轨道同步更新帧索引
	ModeSkeleton

	// ModeSequence 是序列动画模式（15% 的文件）
	// 特征：使用 f=-1（隐藏）和 f=0（显示）控制时间窗口
	// 示例：StartReadySetPlant.reanim
	// 实现：构建 AnimVisiblesMap，根据时间窗口控制可见性
	ModeSequence

	// ModeComplexScene 是复杂场景动画模式（5% 的文件）
	// 特征：50-500+ 轨道，独立时间线，文件行数 10000-40000+
	// 示例：SelectorScreen.reanim（34033 行），CrazyDave.reanim（15065 行）
	// 实现：支持 VisibleTracks 白名单，管理大量独立轨道
	ModeComplexScene

	// ModeBlended 是混合模式（60% 的文件）⭐
	// 特征：包含状态定义轨道（只有 f 值）和部件轨道（图片 + 变换）
	// 示例：PotatoMine.reanim, Squash.reanim, PeaShooter.reanim
	// 实现：
	//   - 第一层：状态机（查询 f 值时间窗口）
	//   - 第二层：渲染器（播放部件轨道）
	//   - 关键：动态识别父骨骼（如 anim_stem），计算偏移量
	ModeBlended
)

// String 返回播放模式的字符串表示（用于日志）
func (m PlaybackMode) String() string {
	switch m {
	case ModeSimple:
		return "Simple"
	case ModeSkeleton:
		return "Skeleton"
	case ModeSequence:
		return "Sequence"
	case ModeComplexScene:
		return "ComplexScene"
	case ModeBlended:
		return "Blended"
	default:
		return "Unknown"
	}
}

// parseModeFromString 将配置文件的字符串模式转换为 PlaybackMode 枚举
//
// 参数：
//   - modeStr: 模式字符串（"Simple", "Skeleton", "Sequence", "ComplexScene", "Blended"）
//
// 返回：
//   - PlaybackMode: 对应的枚举值，如果无法识别则返回 ModeSimple
func parseModeFromString(modeStr string) PlaybackMode {
	switch modeStr {
	case "Simple":
		return ModeSimple
	case "Skeleton":
		return ModeSkeleton
	case "Sequence":
		return ModeSequence
	case "ComplexScene":
		return ModeComplexScene
	case "Blended":
		return ModeBlended
	default:
		log.Printf("[parseModeFromString] Warning: Unknown mode string '%s', defaulting to Simple", modeStr)
		return ModeSimple
	}
}

// PlaybackStrategy 定义动画播放策略接口
//
// 每种播放模式实现各自的策略类，通过策略模式实现多态播放。
// 这消除了硬编码逻辑，使系统可扩展。
type PlaybackStrategy interface {
	// Update 更新动画状态（帧推进、循环等）
	// 参数：
	//   - comp: ReanimComponent 组件
	//   - deltaTime: 时间增量（秒）
	Update(comp *components.ReanimComponent, deltaTime float64)
}

// ==================================================================
// 模式识别算法
// ==================================================================

// detectPlaybackMode 自动检测 Reanim 文件的播放模式
//
// 采用**配置优先 + 启发式后备**策略：
// 1. 优先查询 data/reanim_playback_config.yaml 配置文件
// 2. 如果配置存在，使用配置指定的模式（100% 精确）
// 3. 如果配置不存在，使用启发式算法自动检测（后备方案）
//
// 参数：
//   - reanimName: Reanim 文件名（不含 .reanim 后缀，如 "SelectorScreen"）
//   - reanimData: 解析后的 Reanim XML 数据
//
// 返回：
//   - PlaybackMode: 检测到的播放模式
func detectPlaybackMode(reanimName string, reanimData *reanim.ReanimXML) PlaybackMode {
	if reanimData == nil || len(reanimData.Tracks) == 0 {
		return ModeSimple
	}

	// ==================================================================
	// 步骤 1: 查询配置文件（配置优先策略）
	// ==================================================================
	if configMode, found := config.GetAnimationMode(reanimName); found {
		// 将配置的字符串模式转换为 PlaybackMode 枚举
		mode := parseModeFromString(configMode)
		log.Printf("[detectPlaybackMode] ✅ Using configured mode for '%s': %s (source: config)",
			reanimName, mode.String())
		return mode
	}

	// ==================================================================
	// 步骤 2: 启发式算法（后备方案）
	// ==================================================================
	log.Printf("[detectPlaybackMode] ⚠️  No config found for '%s', using heuristic algorithm", reanimName)

	trackCount := len(reanimData.Tracks)
	animDefCount := 0   // 动画定义轨道数量（只有 f 值）
	partTrackCount := 0 // 部件轨道数量（有图片）
	hasFrameNum := false

	// 统计轨道类型
	for _, track := range reanimData.Tracks {
		isAnimDef := true
		hasPart := false
		hasF := false

		for _, frame := range track.Frames {
			if frame.ImagePath != "" {
				hasPart = true
				isAnimDef = false
			}
			if frame.FrameNum != nil {
				hasF = true
				hasFrameNum = true
			}
		}

		if isAnimDef && hasF {
			animDefCount++
		}
		if hasPart {
			partTrackCount++
		}
	}

	// 决策树
	// 规则 1：优先检查复杂场景动画（修复 SelectorScreen 误判为 Blended 的问题）
	// 特征：轨道数量 >= 40 或 动画定义轨道 >= 10
	// 示例：SelectorScreen.reanim（48 轨道，14 动画定义）
	if trackCount >= 40 || animDefCount >= 10 {
		log.Printf("[detectPlaybackMode] Detected ComplexScene mode: trackCount=%d, animDefCount=%d",
			trackCount, animDefCount)
		return ModeComplexScene
	}

	// 规则 2：如果有多个动画定义轨道（3+），且部件轨道也多，则是混合模式
	// 注意：此规则优先级低于 ComplexScene，避免误判
	if animDefCount >= 3 && partTrackCount >= 3 {
		log.Printf("[detectPlaybackMode] Detected Blended mode: animDefCount=%d, partTrackCount=%d",
			animDefCount, partTrackCount)
		return ModeBlended
	}

	// 规则 3：如果有 f 值且轨道数 10-30，则是序列动画
	if hasFrameNum && trackCount >= 10 && trackCount < 50 {
		log.Printf("[detectPlaybackMode] Detected Sequence mode: trackCount=%d, hasFrameNum=%t",
			trackCount, hasFrameNum)
		return ModeSequence
	}

	// 规则 4：如果轨道数 3-30，无动画定义轨道，则是骨骼动画
	if trackCount >= 3 && trackCount <= 30 && animDefCount == 0 {
		log.Printf("[detectPlaybackMode] Detected Skeleton mode: trackCount=%d", trackCount)
		return ModeSkeleton
	}

	// 规则 5：如果有 2+ 动画定义轨道，则是混合模式
	if animDefCount >= 2 {
		log.Printf("[detectPlaybackMode] Detected Blended mode (by animDefCount): animDefCount=%d",
			animDefCount)
		return ModeBlended
	}

	// 默认：简单模式
	log.Printf("[detectPlaybackMode] Detected Simple mode (default): trackCount=%d", trackCount)
	return ModeSimple
}

// ==================================================================
// 策略实现 1: 简单动画 (SimplePlaybackStrategy)
// ==================================================================

// SimplePlaybackStrategy 实现单轨道简单动画播放
//
// 特征：
// - 1-2 个轨道
// - 无 f 值
// - 简单循环播放
//
// 示例：Lilypad.reanim
type SimplePlaybackStrategy struct{}

// Update 更新简单动画状态
func (s *SimplePlaybackStrategy) Update(comp *components.ReanimComponent, deltaTime float64) {
	// 使用默认的帧推进逻辑（在 ReanimSystem.Update 中已实现）
	// 简单模式无需特殊处理
}

// ==================================================================
// 策略实现 2: 骨骼动画 (SkeletonPlaybackStrategy)
// ==================================================================

// SkeletonPlaybackStrategy 实现多轨道骨骼动画播放
//
// 特征：
// - 4-30 个轨道
// - 每个轨道代表一个部件
// - 所有轨道同步播放
//
// 示例：Sun.reanim, SunFlower.reanim
type SkeletonPlaybackStrategy struct{}

// Update 更新骨骼动画状态
func (s *SkeletonPlaybackStrategy) Update(comp *components.ReanimComponent, deltaTime float64) {
	// 使用默认的帧推进逻辑
	// 骨骼动画所有部件同步，无需特殊处理
}

// ==================================================================
// 策略实现 3: 序列动画 (SequencePlaybackStrategy)
// ==================================================================

// SequencePlaybackStrategy 实现序列动画播放
//
// 特征：
// - 使用 f=-1（隐藏）和 f=0（显示）控制时间窗口
// - 轨道依次出现
//
// 示例：StartReadySetPlant.reanim（"Ready", "Set", "Plant!" 依次显示）
type SequencePlaybackStrategy struct{}

// Update 更新序列动画状态
func (s *SequencePlaybackStrategy) Update(comp *components.ReanimComponent, deltaTime float64) {
	// 使用默认的帧推进逻辑
}

// ==================================================================
// 策略实现 4: 复杂场景动画 (ComplexScenePlaybackStrategy)
// ==================================================================

// ComplexScenePlaybackStrategy 实现复杂场景动画播放
//
// 特征：
// - 50-500+ 轨道
// - 独立时间线（每个动画独立循环）
// - 支持 IndependentAnims（如 SelectorScreen 的云朵、草丛、花朵各自独立播放）
//
// 示例：SelectorScreen.reanim（48 轨道，14 个独立动画定义）
//
// 工作原理：
// 1. IndependentAnims 存储每个独立动画（如 "anim_cloud1"）的状态
// 2. Update 方法更新所有独立动画的帧索引
// 3. GetVisibleTracks 基于每个独立动画的当前帧，查询对应视觉轨道的可见性
type ComplexScenePlaybackStrategy struct{}

// Update 更新复杂场景动画状态
//
// 对于每个独立动画：
// 1. 更新 FrameAccumulator
// 2. 推进 CurrentFrame
// 3. 处理循环/延迟
func (s *ComplexScenePlaybackStrategy) Update(comp *components.ReanimComponent, deltaTime float64) {
	if comp.IndependentAnims == nil || len(comp.IndependentAnims) == 0 {
		// 如果没有独立动画，使用默认的全局帧推进（向后兼容）
		return
	}

	// 计算帧时长
	frameTime := 1.0 / float64(comp.Reanim.FPS)

	// 更新每个独立动画
	for _, state := range comp.IndependentAnims {
		// 跳过未激活的动画
		if !state.IsActive {
			// 更新延迟计时器
			if state.DelayDuration > 0 {
				state.DelayTimer += deltaTime
				if state.DelayTimer >= state.DelayDuration {
					// 延迟结束，激活动画
					state.IsActive = true
					state.DelayTimer = 0
					state.CurrentFrame = state.StartFrame       // 跳回起始帧（不是 0）
					state.FrameAccumulator = 0
				}
			}
			continue
		}

		// 更新帧累加器
		state.FrameAccumulator += deltaTime

		// 检查是否应该推进到下一帧
		for state.FrameAccumulator >= frameTime {
			state.FrameAccumulator -= frameTime
			state.CurrentFrame++

			// 计算结束帧（StartFrame + FrameCount）
			endFrame := state.StartFrame + state.FrameCount

			// 检查是否到达动画末尾
			if state.CurrentFrame >= endFrame {
				if state.IsLooping {
					// 循环播放：重置到起始帧，继续播放
					state.CurrentFrame = state.StartFrame

					// 注意：即使没有延迟，也保持 IsActive = true，让动画持续循环
					// 只有当有延迟时，才在每次循环后暂停
					if state.DelayDuration > 0 {
						state.IsActive = false
						state.DelayTimer = 0
					}
					// 如果没有延迟，继续循环（IsActive 保持 true）
				} else {
					// 非循环动画：停在最后一帧
					state.CurrentFrame = endFrame - 1
					state.IsActive = false
					log.Printf("[ComplexScene] Animation '%s' stopped at frame %d", state.AnimName, state.CurrentFrame)
					break
				}
			}
		}
	}
}

// ==================================================================
// 策略实现 5: 混合模式 (BlendedPlaybackStrategy)
// ==================================================================

// BlendedPlaybackStrategy 实现混合模式播放
//
// 特征：
// - 包含状态定义轨道（只有 f 值）
// - 包含部件轨道（图片 + 变换）
// - 两层结构：
//   - 第一层：状态机（查询 f 值时间窗口）
//   - 第二层：渲染器（播放部件轨道）
//
// 示例：PotatoMine.reanim, Squash.reanim, PeaShooter.reanim（60% 的植物）
//
// 关键逻辑：
// 1. 动态识别父骨骼轨道（如 anim_stem）
// 2. 计算父骨骼偏移量
// 3. 头部部件叠加偏移量
type BlendedPlaybackStrategy struct{}

// Update 更新混合模式动画状态
func (s *BlendedPlaybackStrategy) Update(comp *components.ReanimComponent, deltaTime float64) {
	// 混合模式使用默认的帧推进逻辑
	// 特殊处理在渲染时进行
}
