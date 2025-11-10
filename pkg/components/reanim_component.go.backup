package components

import (
	"github.com/decker502/pvz/internal/reanim"
	"github.com/hajimehoshi/ebiten/v2"
)

// AnimState 动画状态
// 每个动画有自己的时间线，可以独立循环
type AnimState struct {
	// Name 动画名称（如 "anim_cloud1", "anim_grass"）
	Name string

	// LogicalFrame 当前逻辑帧索引（每个动画独立）
	// Story 13.2: 重命名自 Frame，统一使用逻辑帧概念
	LogicalFrame int

	// Accumulator 帧累加器（用于精确 FPS 控制）
	Accumulator float64

	// StartFrame 起始帧索引（循环动画从此帧开始）
	// 用于支持"跳过前面的隐藏帧"场景
	// 例如：anim_grass 的可见帧是 78-102，StartFrame=78
	StartFrame int

	// FrameCount 总帧数（从 StartFrame 开始计算）
	// 例如：anim_grass 的 StartFrame=78, FrameCount=25 (78到102)
	FrameCount int

	// IsLooping 是否循环播放
	IsLooping bool

	// IsActive 是否激活（控制帧推进）
	// true: 动画帧会持续更新
	// false: 动画帧停止推进（锁定在当前帧）
	IsActive bool

	// RenderWhenStopped 停止推进后是否继续渲染
	// true: 即使 IsActive=false，仍然渲染当前帧（适用于静态显示、锁定最后一帧）
	// false: 当 IsActive=false 时，完全隐藏（适用于一次性特效）
	// 默认值：true（保持向后兼容）
	RenderWhenStopped bool

	// DelayTimer 延迟计时器（用于控制周期性动画的间隔）
	// 云朵、草丛等可能需要间隔一段时间才重新播放
	DelayTimer float64

	// DelayDuration 延迟时长（秒）
	// 0 表示无延迟，立即循环
	DelayDuration float64
}

// RenderPartData 存储单个部件的渲染数据缓存（Story 13.4）
// 用于优化 Reanim 渲染性能，避免每帧重复计算
type RenderPartData struct {
	// Img 图片引用（从 PartImages 获取）
	Img *ebiten.Image

	// Frame 帧数据（包含变换信息：位置、缩放、旋转等）
	Frame reanim.Frame

	// OffsetX 父子偏移 X（Story 13.3）
	// 用于实现头部跟随身体摆动等效果
	OffsetX float64

	// OffsetY 父子偏移 Y（Story 13.3）
	OffsetY float64
}

// TrackPlaybackConfig defines playback behavior for an individual track (Story 12.1).
// This allows fine-grained control over track behavior at the business logic level.
type TrackPlaybackConfig struct {
	// PlayOnce indicates the track should play once and then lock at the final frame.
	// When true, the track will stop updating after reaching its last visible frame.
	// Used for one-time animations like tombstone rising or sign dropping.
	PlayOnce bool

	// IsLocked indicates the track has finished playing and is locked at a specific frame.
	// When true, ReanimSystem will not update this track's frame index.
	// Set automatically when PlayOnce track completes.
	IsLocked bool

	// LockedFrame is the frame number where the track is locked.
	// Only used when IsLocked is true.
	LockedFrame int

	// IsPaused indicates the track should temporarily stop updating.
	// Unlike IsLocked, paused tracks can be resumed.
	IsPaused bool
}

// ReanimComponent is a Reanim animation component (pure data, no methods).
// It stores the animation data, part images, and current animation state
// for entities using skeletal animations.
//
// This component follows the ECS architecture principle of data-behavior separation:
// all animation logic is implemented in ReanimSystem.
//
// Fields are organized into three logical groups:
// 1. Animation Definition - The animation data and playback mode
// 2. Playback State - Current runtime state of the animation
// 3. Advanced Features - Blending, caching, and control features
type ReanimComponent struct {
	// ==========================================================================
	// Animation Definition (动画定义)
	// ==========================================================================

	// ReanimName is the name of the Reanim file (without .reanim extension).
	// Used for configuration lookup and debugging.
	// Example: "SelectorScreen", "PeaShooter", "Sun"
	ReanimName string

	// Reanim is the parsed Reanim animation data (from internal/reanim package).
	// Contains FPS and track definitions for the animation.
	Reanim *reanim.ReanimXML

	// PartImages maps image reference names to image objects.
	// Key: image reference name (e.g., "IMAGE_REANIM_PEASHOOTER_HEAD")
	// Value: corresponding Ebitengine image object
	PartImages map[string]*ebiten.Image

	// ==========================================================================
	// Playback State (播放状态)
	// ==========================================================================

	// CurrentAnim is the name of the currently playing animation (e.g., "anim_idle").
	CurrentAnim string

	// CurrentAnimations 当前播放的动画列表（Story 13.2）
	// 支持多动画组合播放（如 ["anim_shooting", "anim_head_idle"]）
	// 单动画时也存储为数组���如 ["anim_idle"]）
	CurrentAnimations []string

	// FrameAccumulator is the frame accumulator for precise FPS control (float64).
	// Accumulates deltaTime until it reaches the time for one animation frame (1.0/fps).
	// This ensures accurate playback speed regardless of game loop framerate.
	FrameAccumulator float64

	// VisibleFrameCount is the number of visible frames in the current animation.
	// Used to determine when to loop the animation (when CurrentFrame >= VisibleFrameCount).
	VisibleFrameCount int

	// IsLooping determines whether the animation should loop.
	// If true, the animation will restart from frame 0 when it reaches the end.
	// If false, the animation will stay at the last frame (used for death animations).
	IsLooping bool

	// IsFinished indicates whether a non-looping animation has completed.
	// This is only set to true for non-looping animations when they reach the last frame.
	// For looping animations, this is always false.
	IsFinished bool

	// IsPaused determines whether the animation should update.
	// If true, ReanimSystem will skip updating CurrentFrame and FrameAccumulator.
	// Used for entities that should remain static (e.g., lawnmowers before trigger).
	IsPaused bool

	// ==========================================================================
	// Advanced Features (高级特性)
	// ==========================================================================

	// AnimVisiblesMap stores time windows for multiple animations (Story 6.5).
	// Key: animation name (e.g., "anim_idle", "anim_shooting")
	// Value: array mapping physical frame index to visibility (0 = visible, -1 = hidden)
	// This supports dual-animation blending where different parts use different animations.
	AnimVisiblesMap map[string][]int

	// MergedTracks is the accumulated frame array for each part track.
	// Key: track name (e.g., "head", "body")
	// Value: array of frames with accumulated transformations (frame inheritance applied)
	// Built by buildMergedTracks when PlayAnimation is called.
	MergedTracks map[string][]reanim.Frame

	// --- Dual Animation Blending (Story 6.5) ---
	// REMOVED: 简化的单动画渲染系统不再需要双动画混合
	// 原因: Reanim 格式中的坐标已经烘焙,单个动画的同一物理帧内所有部件坐标已协调
	// 验证: cmd/verify_single_animation/main.go 证明了 anim_shooting 包含完整的身体+头部数据
	// 移除的字段: IsBlending, PrimaryAnimation, SecondaryAnimation

	// --- Rendering and Caching ---

	// AnimTracks is the list of part tracks to render for the current animation, in rendering order.
	// This preserves the Z-order from the Reanim file.
	// Built by getAnimationTracks when PlayAnimation is called.
	AnimTracks []reanim.Track

	// CenterOffsetX and CenterOffsetY are the offsets to center the animation visually.
	// These values are calculated based on the bounding box of all visible parts
	// in the first frame of the animation, to align the visual center with the entity position.
	CenterOffsetX float64
	CenterOffsetY float64

	// CachedRenderData 渲染数据缓存（Story 13.4）
	// 存储预计算的渲染数据，避免每帧重复计算
	// 包括图片引用、帧数据、父子偏移等
	// 缓存在帧变化时自动更新（通过 LastRenderFrame 检测）
	CachedRenderData []RenderPartData

	// LastRenderFrame 上次渲染的逻辑帧（Story 13.4）
	// 用于检测缓存是否失效：当前逻辑帧 != LastRenderFrame 时需要重新构建缓存
	// 初始化为 -1，表示尚未渲染过
	LastRenderFrame int

	// FixedCenterOffset 是否使用固定的中心偏移量（避免动画切换时的位置跳动）
	// 如果为 true，则 CenterOffsetX/Y 在实体创建时计算一次后固定不变
	// 如果为 false，则每次 PlayAnimation 时重新计算 CenterOffset
	// 用于解决不同动画包围盒大小不同导致的位置跳动问题
	FixedCenterOffset bool

	// BestPreviewFrame is the optimal frame index for generating preview icons (Story 10.3).
	// This frame is automatically calculated when PlayAnimation is called by finding the frame
	// with the most visible parts (highest count of ImagePath != "").
	// Used by RenderPlantIcon to ensure preview shows the most complete representation of the entity.
	// Default: 0 (first frame)
	BestPreviewFrame int

	// --- Visibility Control ---

	// VisibleTracks is a whitelist of track names that should be rendered.
	// If this map is not nil and not empty, ONLY tracks in this map will be rendered.
	// This provides a clear "opt-in" approach for complex entities like zombies.
	// Use ReanimSystem.HideTrack() and ShowTrack() to manage track visibility.
	VisibleTracks map[string]bool

	// PartGroups - REMOVED: 简化的单动画渲染系统不再需要部件分组
	// 原因: 双动画模式已移除,不再需要区分"头部"和"身体"部件
	// 僵尸等实体可以继续使用 VisibleTracks 白名单进行部件控制
	// 移除的方法: HidePartGroup, ShowPartGroup, GetPartGroupImage

	// TrackConfigs stores per-track playback configuration (Story 12.1).
	// Key: track name (e.g., "SelectorScreen_Adventure_button", "Cloud1")
	// Value: playback configuration for this track
	// This allows business logic to control individual track behavior:
	// - Some tracks play once and lock (e.g., tombstone rising)
	// - Some tracks loop continuously (e.g., clouds, grass)
	// - Some tracks can be paused/resumed
	TrackConfigs map[string]*TrackPlaybackConfig

	// --- Independent Animations (独立动画系统) ---

	// AnimStates 存储每个动画的独���状态（Story 13.2 统一命名）
	// Key: 动画名称（如 "anim_idle", "anim_shooting", "anim_head_idle"）
	// Value: 动画状态（包含独立的 LogicalFrame、Accumulator 等）
	//
	// Story 13.2 重构说明：
	// - 移除了同步/异步双模式，所有动画统一使用独立帧
	// - 每个动画维护自己的 LogicalFrame（从 0 开始）
	// - 轨道绑定通过 TrackBindings 实现（Story 13.1）
	AnimStates map[string]*AnimState

	// TrackBindings 定义每个轨道由哪个动画控制（Story 13.1）
	// Key: 轨道名（如 "anim_face", "stalk_bottom"）
	// Value: 控制该轨道的动画名（如 "anim_head_idle", "anim_shooting"）
	//
	// 用途：
	// - 支持"头部用动画A，身体用动画B"的复杂组合
	// - nil 时使用默认行为（所有轨道使用第一个动画）
	//
	// 示例：
	//   TrackBindings["anim_face"] = "anim_head_idle"
	//   TrackBindings["stalk_bottom"] = "anim_shooting"
	TrackBindings map[string]string

	// ParentTracks 定义轨道的父子层级关系（Story 13.3）
	// Key: 子轨道名（如 "anim_face"）
	// Value: 父轨道名（如 "anim_stem"）
	//
	// 用途：
	// - 子轨道渲染时会叠加父轨道的偏移量
	// - 实现头部跟随身体摆动等自然动画效果
	// - nil 时不应用父子偏移（保持向后兼容）
	//
	// 示例：
	//   ParentTracks["anim_face"] = "anim_stem"
	//   表示 anim_face 的父轨道是 anim_stem，渲染时需要叠加 anim_stem 的偏移
	ParentTracks map[string]string
}
