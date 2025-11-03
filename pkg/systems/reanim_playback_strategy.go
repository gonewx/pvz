package systems

import (
	"log"

	"github.com/decker502/pvz/internal/reanim"
	"github.com/decker502/pvz/pkg/components"
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

	// GetVisibleTracks 返回当前帧应该渲染的轨道列表
	// 参数：
	//   - comp: ReanimComponent 组件
	//   - frame: 当前逻辑帧
	// 返回：
	//   - map[trackName]bool: 轨道名 -> 是否可见
	GetVisibleTracks(comp *components.ReanimComponent, frame int) map[string]bool
}

// ==================================================================
// 模式识别算法
// ==================================================================

// detectPlaybackMode 自动检测 Reanim 文件的播放模式
//
// 算法基于以下特征：
// 1. 轨道数量
// 2. f 值存在性
// 3. 图片资源分布
// 4. 动画定义轨道数量
//
// 参数：
//   - reanimData: 解析后的 Reanim XML 数据
//
// 返回：
//   - PlaybackMode: 检测到的播放模式
func detectPlaybackMode(reanimData *reanim.ReanimXML) PlaybackMode {
	if reanimData == nil || len(reanimData.Tracks) == 0 {
		return ModeSimple
	}

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
	// 规则 1：如果有多个动画定义轨道（3+），且部件轨道也多，则是混合模式
	if animDefCount >= 3 && partTrackCount >= 3 {
		log.Printf("[detectPlaybackMode] Detected Blended mode: animDefCount=%d, partTrackCount=%d",
			animDefCount, partTrackCount)
		return ModeBlended
	}

	// 规则 2：如果轨道数量超过 50，则是复杂场景动画
	if trackCount >= 50 {
		log.Printf("[detectPlaybackMode] Detected ComplexScene mode: trackCount=%d", trackCount)
		return ModeComplexScene
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

// GetVisibleTracks 返回所有轨道（简单模式下所有轨道都可见）
func (s *SimplePlaybackStrategy) GetVisibleTracks(comp *components.ReanimComponent, frame int) map[string]bool {
	visible := make(map[string]bool)

	// 简单模式：所有有图片的轨道都可见
	for trackName, frames := range comp.MergedTracks {
		if len(frames) > 0 && frames[0].ImagePath != "" {
			visible[trackName] = true
		}
	}

	return visible
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

// GetVisibleTracks 返回所有轨道（骨骼动画所有部件都可见）
func (s *SkeletonPlaybackStrategy) GetVisibleTracks(comp *components.ReanimComponent, frame int) map[string]bool {
	visible := make(map[string]bool)

	// 骨骼动画：所有有图片的轨道都可见
	for trackName, frames := range comp.MergedTracks {
		if len(frames) > 0 && frames[0].ImagePath != "" {
			visible[trackName] = true
		}
	}

	return visible
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

// GetVisibleTracks 返回当前帧应该显示的轨道
func (s *SequencePlaybackStrategy) GetVisibleTracks(comp *components.ReanimComponent, frame int) map[string]bool {
	visible := make(map[string]bool)

	// 序列动画：根据 f 值时间窗口判断可见性
	for trackName, frames := range comp.MergedTracks {
		if frame >= len(frames) {
			continue
		}

		currentFrame := frames[frame]

		// 检查是否有图片
		if currentFrame.ImagePath == "" {
			continue
		}

		// 检查 f 值（FrameNum）
		// f=0 表示显示，f=-1 表示隐藏
		if currentFrame.FrameNum != nil {
			if *currentFrame.FrameNum == 0 {
				visible[trackName] = true
			}
		} else {
			// 如果没有 FrameNum，默认显示
			visible[trackName] = true
		}
	}

	return visible
}

// ==================================================================
// 策略实现 4: 复杂场景动画 (ComplexScenePlaybackStrategy)
// ==================================================================

// ComplexScenePlaybackStrategy 实现复杂场景动画播放
//
// 特征：
// - 50-500+ 轨道
// - 独立时间线
// - 使用 VisibleTracks 白名单控制
//
// 示例：SelectorScreen.reanim（500+ 轨道，34033 行）
type ComplexScenePlaybackStrategy struct{}

// Update 更新复杂场景动画状态
func (s *ComplexScenePlaybackStrategy) Update(comp *components.ReanimComponent, deltaTime float64) {
	// 复杂场景动画：每个轨道独立更新
	// 使用 TrackConfigs 控制每个轨道的播放行为
	// 默认逻辑由 ReanimSystem.Update 实现
}

// GetVisibleTracks 返回 VisibleTracks 白名单中的轨道
func (s *ComplexScenePlaybackStrategy) GetVisibleTracks(comp *components.ReanimComponent, frame int) map[string]bool {
	// 复杂场景动画使用 VisibleTracks 白名单
	// 如果白名单存在，只渲染白名单中的轨道
	if comp.VisibleTracks != nil && len(comp.VisibleTracks) > 0 {
		return comp.VisibleTracks
	}

	// 如果没有白名单，渲染所有有图片的轨道
	visible := make(map[string]bool)
	for trackName, frames := range comp.MergedTracks {
		if len(frames) > 0 && frames[0].ImagePath != "" {
			visible[trackName] = true
		}
	}

	return visible
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
	// 特殊处理在 GetVisibleTracks 和渲染时进行
}

// GetVisibleTracks 返回当前状态下应该显示的部件轨道
func (s *BlendedPlaybackStrategy) GetVisibleTracks(comp *components.ReanimComponent, frame int) map[string]bool {
	visible := make(map[string]bool)

	// 混合模式：
	// 1. 检查当前动画的时间窗口（AnimVisiblesMap）
	// 2. 渲染部件轨道（非动画定义轨道、非逻辑轨道）

	// 如果有 AnimVisiblesMap，使用时间窗口控制
	outsideTimeWindow := false
	if comp.AnimVisiblesMap != nil && comp.CurrentAnim != "" {
		animVisibles := comp.AnimVisiblesMap[comp.CurrentAnim]
		if animVisibles != nil && frame < len(animVisibles) {
			// 检查当前帧是否在时间窗口外
			if animVisibles[frame] == -1 {
				outsideTimeWindow = true
			}
		}
	}

	// 如果在时间窗口外，隐藏所有部件
	if outsideTimeWindow {
		return visible
	}

	// 渲染所有部件轨道
	for trackName, frames := range comp.MergedTracks {
		// 跳过动画定义轨道
		if AnimationDefinitionTracks[trackName] {
			continue
		}

		// 跳过逻辑轨道
		if LogicalTracks[trackName] {
			continue
		}

		// 检查是否有图片
		if frame >= len(frames) {
			continue
		}

		currentFrame := frames[frame]
		if currentFrame.ImagePath == "" {
			continue
		}

		// 检查混合轨道的 f 值
		// 如果 f=-1，检查时间窗口；如果 f=0，显示
		if currentFrame.FrameNum != nil {
			if *currentFrame.FrameNum == -1 {
				// f=-1：使用动画定义轨道的时间窗口
				// 如果时间窗口内（animVisibles[frame] == 0），显示
				if comp.AnimVisiblesMap != nil && comp.CurrentAnim != "" {
					animVisibles := comp.AnimVisiblesMap[comp.CurrentAnim]
					if animVisibles != nil && frame < len(animVisibles) {
						if animVisibles[frame] == 0 {
							visible[trackName] = true
						}
					}
				}
			} else if *currentFrame.FrameNum == 0 {
				// f=0：显示
				visible[trackName] = true
			}
		} else {
			// 没有 f 值：默认显示
			visible[trackName] = true
		}
	}

	return visible
}
