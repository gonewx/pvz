package components

import (
	"github.com/decker502/pvz/internal/reanim"
	"github.com/hajimehoshi/ebiten/v2"
)

// RenderPartData 存储单个部件的渲染数据缓存
// 用于优化 Reanim 渲染性能，避免每帧重复计算
type RenderPartData struct {
	// Img 图片引用（从 PartImages 获取）
	Img *ebiten.Image

	// Frame 帧数据（包含变换信息：位置、缩放、旋转等）
	Frame reanim.Frame

	// OffsetX 父子偏移 X（用于实现头部跟随身体摆动等效果）
	OffsetX float64

	// OffsetY 父子偏移 Y
	OffsetY float64
}

// ReanimComponent 是 Reanim 动画组件（纯数据，无方法）
// 基于 animation_showcase/AnimationCell 重写，简化并修复 Epic 13 遗留问题
//
// Story 13.8 重构目标：
// - 字段数量从 30+ 减少到 ~18 个
// - 代码行数从 278 减少到 ~100 行
// - 与 AnimationCell 保持一致的命名和结构
type ReanimComponent struct {
	// ==========================================================================
	// 基础数据 (Basic Data)
	// ==========================================================================

	// ReanimName 是 Reanim 文件名（不含扩展名）
	// 用于配置查找和调试
	// Example: "PeaShooter", "SunFlower", "Zombie"
	ReanimName string

	// ReanimXML 是解析后的 Reanim 动画数据
	// 包含 FPS 和轨道定义
	ReanimXML *reanim.ReanimXML

	// PartImages 图片资源映射
	// Key: 图片引用名（如 "IMAGE_REANIM_PEASHOOTER_HEAD"）
	// Value: 对应的 Ebitengine 图片对象
	PartImages map[string]*ebiten.Image

	// MergedTracks 是每个轨道的累加帧数组
	// Key: 轨道名（如 "anim_stem", "anim_face"）
	// Value: 帧数组（已应用帧继承的变换）
	MergedTracks map[string][]reanim.Frame

	// ==========================================================================
	// 轨道分类 (Track Classification)
	// ==========================================================================

	// VisualTracks 是需要渲染的轨道列表（按 Z-order 排序）
	// 这些轨道包含图片数据，需要在屏幕上显示
	VisualTracks []string

	// LogicalTracks 是逻辑轨道列表（不渲染）
	// 这些轨道只用于父子偏移计算，不包含图片
	LogicalTracks []string

	// ==========================================================================
	// 播放状态 (Playback State)
	// ==========================================================================

	// CurrentFrame 当前逻辑帧索引（所有动画共享）
	// 从 0 开始，随着时间推进递增
	// ⚠️ 注意：如果使用 AnimationFPSOverrides，此字段仅作为后备，
	// 实际每个动画的帧索引存储在 AnimationFrameIndices 中
	CurrentFrame int

	// AnimationFrameIndices 存储每个动画的独立逻辑帧索引
	// Key: 动画名称（如 "anim_open", "anim_cloud1"）
	// Value: 该动画当前的逻辑帧索引（浮点数，支持亚帧精度）
	// 用于支持不同动画以不同速度播放（如开场动画 2 FPS + 云朵动画 12 FPS）
	AnimationFrameIndices map[string]float64

	// FrameAccumulator 帧累加器，用于精确 FPS 控制
	// 累加 deltaTime 直到达到一帧的时间 (1.0/fps)
	FrameAccumulator float64

	// AnimationFPS 动画播放帧率（从 reanim 文件读取）
	// 控制动画播放速度
	AnimationFPS float64

	// CurrentAnimations 当前播放的动画列表
	// 支持多动画组合播放（如 ["anim_shooting", "anim_head_idle"]）
	// 单动画时也存储为数组（如 ["anim_idle"]）
	CurrentAnimations []string

	// AnimationLoopStates 存储每个动画的循环状态
	// Key: 动画名称（如 "anim_open", "anim_cloud1"）
	// Value: true 表示循环播放，false 表示播放一次后停止
	// 如果某个动画不在此 map 中，默认使用全局 IsLooping 值
	AnimationLoopStates map[string]bool

	// AnimationPausedStates 存储每个动画的暂停状态
	// Key: 动画名称（如 "anim_cloud1", "anim_grass"）
	// Value: true 表示暂停（不推进帧，但显示当前帧），false 表示正常播放
	// 如果某个动画不在此 map 中，默认为 false（不暂停）
	// 用于实现"云朵初始帧可见但延迟播放"等效果
	AnimationPausedStates map[string]bool

	// AnimationFPSOverrides 存储每个动画的独立 FPS
	// Key: 动画名称（如 "anim_cloud1", "anim_open"）
	// Value: 该动画的 FPS（如 6.0, 12.0, 24.0）
	// 如果某个动画不在此 map 中，使用全局 AnimationFPS
	// 用于实现不同动画以不同速度播放（如慢速开场动画 + 快速云朵动画）
	AnimationFPSOverrides map[string]float64

	// AnimationSpeedOverrides 存储每个动画的速度倍率
	// Key: 动画名称（如 "anim_cloud1", "anim_grass"）
	// Value: 速度倍率（0.0-1.0+），1.0=正常速度，0.5=50%速度，2.0=200%速度
	// 如果某个动画不在此 map 中，默认为 1.0（正常速度）
	// 用于在保持高 FPS（平滑）的同时控制动画播放速度
	AnimationSpeedOverrides map[string]float64

	// ==========================================================================
	// 动画数据 (Animation Data)
	// ==========================================================================

	// AnimVisiblesMap 存储每个动画的可见性数组
	// Key: 动画名称（如 "anim_idle", "anim_shooting"）
	// Value: 可见性数组（索引为逻辑帧，值为物理帧或 -1 表示隐藏）
	// 用于逻辑帧到物理帧的映射
	AnimVisiblesMap map[string][]int

	// ✅ Story 13.10: TrackAnimationBinding 已删除
	// 新的渲染逻辑不再需要轨道绑定机制，直接从动画遍历到轨道

	// ==========================================================================
	// 配置字段 (Configuration)
	// ==========================================================================

	// ParentTracks 定义轨道的父子层级关系
	// Key: 子轨道名（如 "anim_face"）
	// Value: 父轨道名（如 "anim_stem"）
	// 子轨道渲染时会叠加父轨道的偏移量，实现头部跟随身体摆动等效果
	ParentTracks map[string]string

	// HiddenTracks 隐藏的轨道（黑名单）
	// Key: 轨道名
	// Value: true 表示隐藏，false 或不存在表示显示
	// 渲染时会跳过这些轨道
	// 关键修复：animation_showcase 使用黑名单，而非白名单
	HiddenTracks map[string]bool

	// TrackOffsets 轨道偏移量（用于抖动效果）
	// Key: 轨道名
	// Value: [X, Y] 偏移量
	// 渲染时会在轨道的原始位置上叠加此偏移
	TrackOffsets map[string][2]float64

	// CenterOffsetX/Y 是 bounding box 中心的坐标（相对于 Reanim 原点）
	// 用于渲染时居中对齐：screenPos = worldPos - CenterOffset
	// 在动画初始化时计算一次并缓存，避免每帧重新计算导致位置抖动
	CenterOffsetX float64
	CenterOffsetY float64

	// ==========================================================================
	// 渲染缓存 (Render Cache)
	// ==========================================================================

	// CachedRenderData 渲染数据缓存
	// 存储预计算的渲染数据，避免每帧重复计算
	// 包括图片引用、帧数据、父子偏移等
	CachedRenderData []RenderPartData

	// LastRenderFrame 上次渲染的逻辑帧
	// 用于检测缓存是否失效：CurrentFrame != LastRenderFrame 时需要重新构建缓存
	// 初始化为 -1，表示尚未渲染过
	LastRenderFrame int

	// ==========================================================================
	// 控制标志 (Control Flags)
	// ==========================================================================

	// IsPaused 是否暂停动画
	// 如果为 true，ReanimSystem 将跳过帧推进
	IsPaused bool

	// IsLooping 是否循环播放
	// 如果为 true，动画结束后会重新开始
	// 如果为 false，动画会停留在最后一帧（用于死亡动画等）
	IsLooping bool

	// IsFinished 非循环动画是否已完成
	// 只有非循环动画到达最后一帧时才会设置为 true
	// 循环动画始终为 false
	IsFinished bool
}
