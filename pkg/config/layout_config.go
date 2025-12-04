package config

// 布局配置常量
// 本文件定义了游戏场景中的布局参数，包括网格系统、UI元素位置等

// Background Configuration (背景配置)
const (
	// BackgroundWidth 背景图片宽度（像素）
	// 用于限制僵尸生成位置和镜头移动范围
	BackgroundWidth = 1400.0

	// BackgroundHeight 背景图片高度（像素）
	BackgroundHeight = 600.0
)

// Camera Configuration (摄像机配置)
const (
	// GameCameraX 是游戏摄像机的X位置（世界坐标）
	// 这是游戏正常运行时的摄像机位置，镜头居中对准草坪
	GameCameraX = 220.0

	// GridScreenStartX 是草坪网格在屏幕坐标系中的起始X坐标
	// 这是草坪相对于屏幕左侧的距离
	GridScreenStartX = 35.0
)

// Lawn Grid Configuration (草坪网格配置)
// 所有坐标使用"世界坐标系"（相对于背景图片左上角）
// 世界坐标是固定的，不随摄像机移动而变化
const (
	// GridWorldStartX 是草坪网格在背景图片中的起始X坐标（世界坐标）
	// 计算方式：屏幕坐标 + 游戏摄像机位置
	GridWorldStartX = GridScreenStartX + GameCameraX

	// GridWorldStartY 是草坪网格在背景图片中的起始Y坐标（世界坐标）
	// Y轴不受摄像机水平移动影响，因此世界坐标等于屏幕坐标
	GridWorldStartY = 78.0

	// GridColumns 是草坪的列数（横向格子数）
	GridColumns = 9

	// GridRows 是草坪的行数（纵向格子数）
	GridRows = 5

	// CellWidth 是每个格子的宽度（像素）
	CellWidth = 80.0

	// CellHeight 是每个格子的高度（像素）
	CellHeight = 100.0

	// GridWorldEndX 是草坪网格在背景图片中的结束X坐标（世界坐标）
	// 计算方式：起始X + 列数 * 格子宽度 = 251 + 9*80 = 971
	// 用于判断实体是否在草坪范围内（如豌豆射手攻击范围检测）
	GridWorldEndX = GridWorldStartX + float64(GridColumns)*CellWidth // 971.0

	// ========== 草皮动画配置参数（可手工调节） ==========

	// SodRoll 动画起点X（相对于网格起点的偏移）
	// 调整此值可以改变草皮卷从哪里开始滚动
	SodRollStartOffsetX = 0.0 // 相对于 GridWorldStartX 的偏移量

	// SodRoll 动画Y偏移（相对于目标行中心）
	// 调整此值可以改变草皮卷的垂直位置
	SodRollOffsetY = -8.0 // 相对于行中心的Y偏移量

	// 草皮叠加图X偏移（相对于网格起点）
	// 调整此值可以改变草皮显示的水平位置
	// SodOverlayOffsetX = -26.0 // 相对于 GridWorldStartX 的偏移量
	SodOverlayOffsetX = -20.0 // 相对于 GridWorldStartX 的偏移量

	// 草皮叠加图Y偏移（相对于目标行中心）
	// 调整此值可以改变草皮显示的垂直位置
	SodOverlayOffsetY = -3.0 // 相对于行中心的Y偏移量

	// ========== 教学文本UI配置参数（可手工调节） ==========

	// TutorialTextBackgroundHeight 教学文本背景条高度（像素）
	TutorialTextBackgroundHeight = 100.0

	// TutorialTextOffsetFromBottom 教学文本距离屏幕底部的偏移（像素）
	// 文字Y坐标 = screenHeight - TutorialTextOffsetFromBottom
	TutorialTextOffsetFromBottom = 140.0

	// TutorialTextBackgroundOffsetFromBottom 教学文本背景条距离屏幕底部的偏移（像素）
	// 背景条顶部Y坐标 = screenHeight - TutorialTextBackgroundOffsetFromBottom
	TutorialTextBackgroundOffsetFromBottom = 180.0

	// ========== 提示性教学文本UI配置参数（Level 1-2 等非强制教学关卡） ==========

	// AdvisoryTutorialTextOffsetFromBottom 提示性教学文本距离屏幕底部的偏移（像素）
	AdvisoryTutorialTextOffsetFromBottom = 85.0

	// AdvisoryTutorialTextBackgroundOffsetFromBottom 提示性教学背景条距离屏幕底部的偏移（像素）
	AdvisoryTutorialTextBackgroundOffsetFromBottom = 125.0

	// ========== Level 1-5 铲子教学文本UI配置参数（Story 19.x） ==========

	// BowlingTutorialTextBackgroundHeight Level 1-5 教学文本背景条高度（像素）
	BowlingTutorialTextBackgroundHeight = 50.0

	// BowlingTutorialTextOffsetFromBottom Level 1-5 教学文本距离屏幕底部的偏移（像素）
	BowlingTutorialTextOffsetFromBottom = 60.0

	// BowlingTutorialTextBackgroundOffsetFromBottom Level 1-5 教学背景条距离屏幕底部的偏移（像素）
	BowlingTutorialTextBackgroundOffsetFromBottom = 70.0

	// BowlingTutorialTextFontSize Level 1-5 教学文本字体大小
	BowlingTutorialTextFontSize = 28.0

	// AdvisoryTutorialTextDisplayDuration 提示性教学文字显示时长（秒）
	AdvisoryTutorialTextDisplayDuration = 5.0

	// TutorialTextMinDisplayTime 教学文字最小显示时长（秒）
	// 在此时间内，状态检测型触发器（如 sunSpawned、enoughSunAndCooldown）不会触发
	// 确保事件触发型文本（如种植成功）有足够时间显示
	TutorialTextMinDisplayTime = 1.5

	// TutorialTextBackgroundAlpha 教学文本背景条的透明度（0-255）
	TutorialTextBackgroundAlpha = 128

	// ========== 僵尸生成位置配置参数（可手工调节） ==========

	// ZombieSpawnMinX 僵尸生成的最小X坐标（世界坐标）
	// 用于开场预览和正常游戏，僵尸在此范围内随机分布
	ZombieSpawnMinX = 1150.0

	// ZombieSpawnMaxX 僵尸生成的最大X坐标（世界坐标）
	// 不能超过背景宽度，留出边距避免僵尸贴边
	// 这是默认值，适用于第3、4、5行（没有特殊配置的行）
	ZombieSpawnMaxX = 1250.0 // 减少范围，让僵尸更快进入画面

	// ZombieSpawnMaxX_Row1 第1行僵尸生成的最大X坐标（世界坐标）
	// 建议值范围：1000.0 - 1350.0
	// 调整此值可以控制第1行僵尸的生成范围
	ZombieSpawnMaxX_Row1 = 1200.0

	// ZombieSpawnMaxX_Row2 第2行僵尸生成的最大X坐标（世界坐标）
	// 建议值范围：1000.0 - 1350.0
	// 调整此值可以控制第2行僵尸的生成范围
	ZombieSpawnMaxX_Row2 = 1200.0

	// OpeningZombiePreviewX 开场动画僵尸预览位置X坐标（已废弃，使用 ZombieSpawnMinX/MaxX 范围）
	// 保留此常量以保持向后兼容
	OpeningZombiePreviewX = 1200.0

	// ========== UI元素位置配置参数（可手工调节） ==========

	// SeedBankX 种子栏X坐标（屏幕坐标，像素）
	SeedBankX = 10

	// SeedBankY 种子栏Y坐标（屏幕坐标，像素）
	SeedBankY = 0

	// SeedBankWidth 种子栏宽度（像素）
	SeedBankWidth = 500

	// SeedBankHeight 种子栏高度（像素）
	SeedBankHeight = 87

	// SunCounterOffsetX 阳光计数器相对于 SeedBank 的 X 偏移量（像素）
	SunCounterOffsetX = 40

	// SunCounterOffsetY 阳光计数器相对于 SeedBank 的 Y 偏移量（像素）
	// 这是文字显示位置
	SunCounterOffsetY = 64

	// SunCounterWidth 阳光计数器宽度（像素）
	SunCounterWidth = 130

	// SunCounterHeight 阳光计数器高度（像素）
	SunCounterHeight = 60

	// SunCounterFontSize 阳光数值字体大小（像素）
	SunCounterFontSize = 18.0

	// SunPoolOffsetX 阳光池图标相对于 SeedBank 的 X 偏移量（像素）
	// 阳光收集目标位置（与文字对齐）
	SunPoolOffsetX = 40

	// SunPoolOffsetY 阳光池图标相对于 SeedBank 的 Y 偏移量（像素）
	// 阳光收集目标位置（在文字上方，池子中心）
	SunPoolOffsetY = 32

	// PlantCardStartOffsetX 第一张植物卡片相对于 SeedBank 的 X 偏移量（像素）
	PlantCardStartOffsetX = 84

	// PlantCardOffsetY 植物卡片相对于 SeedBank 的 Y 偏移量（像素）
	PlantCardOffsetY = 8

	// PlantCardSpacing 卡片槽之间的间距（像素）
	// 包含卡槽边框，每个卡槽约76px宽
	PlantCardSpacing = 60

	// ShovelX 铲子X坐标（屏幕坐标，像素）
	// 已废弃：现在铲子位置根据选择栏图片宽度动态计算
	// 保留此常量作为备用值（当选择栏图片未加载时使用）
	ShovelX = 620

	// ShovelY 铲子Y坐标（屏幕坐标，像素）
	ShovelY = 8

	// ShovelGapFromSeedBank 铲子与选择栏之间的间距（像素）
	// 铲子紧挨选择栏右侧，无间距
	ShovelGapFromSeedBank = 0

	// ShovelWidth 铲子宽度（像素）
	ShovelWidth = 70

	// ShovelHeight 铲子高度（像素）
	ShovelHeight = 74

	// BowlingShovelGapFromMenuButton 保龄球模式铲子与菜单按钮之间的间距（像素）
	// 铲子右边缘到菜单按钮左边缘的距离
	BowlingShovelGapFromMenuButton = 2

	// BowlingShovelY 保龄球模式铲子Y坐标（屏幕坐标，像素）
	BowlingShovelY = 0

	// IntroAnimDuration 开场动画时长（秒）
	IntroAnimDuration = 3.0

	// CameraScrollSpeed 开场动画摄像机滚动速度（像素/秒）
	CameraScrollSpeed = 100

	// MenuButtonOffsetFromRight 菜单按钮距离屏幕右边缘的距离（像素）
	MenuButtonOffsetFromRight = 123.0

	// MenuButtonOffsetFromTop 菜单按钮距离屏幕顶部的距离（像素）
	MenuButtonOffsetFromTop = 0.0

	// MenuButtonTextPadding 菜单按钮文字左右内边距（像素）
	MenuButtonTextPadding = 16.0

	// MenuButtonTextWidth 菜单按钮文字宽度（"菜单"两个字，像素）
	MenuButtonTextWidth = 18.0

	// ProgressBarOffsetFromRight 进度条距离屏幕右边缘的距离（像素）
	ProgressBarOffsetFromRight = 170.0

	// ProgressBarOffsetFromBottom 进度条距离屏幕底部的距离（像素）
	ProgressBarOffsetFromBottom = 60.0

	// ProgressBarZombieHeadOffsetX 僵尸头图标X偏移（相对于进度条左上角，像素）
	ProgressBarZombieHeadOffsetX = 8.0

	// ProgressBarZombieHeadOffsetY 僵尸头图标Y偏移（相对于进度条左上角，像素）
	ProgressBarZombieHeadOffsetY = 2.0

	// ProgressBarFillOffsetX 进度条填充X偏移（相对于进度条左上角，像素）
	ProgressBarFillOffsetX = 35.0

	// ProgressBarFillOffsetY 进度条填充Y偏移（相对于进度条左上角，像素）
	ProgressBarFillOffsetY = 16.0

	// ProgressBarLevelTextOffsetX 关卡编号文字X偏移（相对于进度条背景右边缘，像素）
	ProgressBarLevelTextOffsetX = 5.0

	// ProgressBarLevelTextOffsetY 关卡编号文字Y偏移（相对于进度条左上角，像素）
	ProgressBarLevelTextOffsetY = 8.0

	// ========== 暂停菜单配置参数（Story 10.1）（可手工调节） ==========

	// PauseMenuOverlayAlpha 暂停菜单遮罩透明度（0-255）
	PauseMenuOverlayAlpha = 150

	// PauseMenuBackToGameButtonFontSize "返回游戏"按钮字体大小（可调节）
	PauseMenuBackToGameButtonFontSize = 45.0

	// PauseMenuInnerButtonFontSize "重新开始"和"主菜单"按钮字体大小（可调节）
	PauseMenuInnerButtonFontSize = 20.0

	// PauseMenuBackToGameButtonOffsetY "返回游戏"按钮Y偏移（相对于屏幕中心，像素）
	// 正值向下，负值向上。调整此值控制按钮相对于墓碑背景的垂直位置
	PauseMenuBackToGameButtonOffsetY = 132.0

	// PauseMenuRestartButtonOffsetY "重新开始"按钮Y偏移（相对于屏幕中心，像素）
	// 此按钮在墓碑内部，位于顶部
	// 建议值范围：50.0 - 100.0
	PauseMenuRestartButtonOffsetY = 30.0

	// PauseMenuMainMenuButtonOffsetY "主菜单"按钮Y偏移（相对于屏幕中心，像素）
	// 此按钮在墓碑内部，位于中间偏下
	// 建议值范围：120.0 - 160.0
	PauseMenuMainMenuButtonOffsetY = 75.0

	// ========== 暂停菜单UI元素偏移（Story 10.1）（可手工调节） ==========

	// PauseMenuMusicSliderOffsetX 音乐滑动条X偏移（相对于屏幕中心，像素）
	PauseMenuMusicSliderOffsetX = 0.0

	// PauseMenuMusicSliderOffsetY 音乐滑动条Y偏移（相对于屏幕中心，像素）
	PauseMenuMusicSliderOffsetY = -120.0

	// PauseMenuSoundSliderOffsetX 音效滑动条X偏移（相对于屏幕中心，像素）
	PauseMenuSoundSliderOffsetX = 0.0

	// PauseMenuSoundSliderOffsetY 音效滑动条Y偏移（相对于屏幕中心，像素）
	PauseMenuSoundSliderOffsetY = -90.0

	// PauseMenu3DCheckboxOffsetX 3D加速复选框X偏移（相对于屏幕中心，像素）
	PauseMenu3DCheckboxOffsetX = 60.0

	// PauseMenu3DCheckboxOffsetY 3D加速复选框Y偏移（相对于屏幕中心，像素）
	PauseMenu3DCheckboxOffsetY = -60.0

	// PauseMenuFullscreenCheckboxOffsetX 全屏复选框X偏移（相对于屏幕中心，像素）
	PauseMenuFullscreenCheckboxOffsetX = 60.0

	// PauseMenuFullscreenCheckboxOffsetY 全屏复选框Y偏移（相对于屏幕中心，像素）
	PauseMenuFullscreenCheckboxOffsetY = -30.0

	// PauseMenuLabelFontSize UI元素标签文字字体大小（像素）
	PauseMenuLabelFontSize = 16.0

	// PauseMenuLabelOffsetX 标签文字X偏移（相对于滑动条/复选框位置，像素）
	// 负值表示在左侧
	PauseMenuLabelOffsetX = -80.0

	// PauseMenuLabelOffsetY 标签文字Y偏移（相对于滑动条/复选框位置，像素）
	// 用于垂直居中对齐
	PauseMenuLabelOffsetY = 0.0

	// PauseMenuInnerButtonWidth 墓碑内部按钮总宽度（像素）
	// 包含左右边框，中间部分宽度 = 总宽度 - 左右边框宽度
	PauseMenuInnerButtonWidth = 180.0

	// ========== 除草车配置参数（Story 10.2）（可手工调节） ==========

	// LawnmowerStartX 除草车初始X位置（世界坐标，像素）
	// 除草车放置在草坪左侧台阶上
	// 注意：必须在摄像机视野内（worldX >= GameCameraX = 220）
	// 设置为 230，屏幕坐标 = 230 - 220 = 10（刚好在屏幕左边缘）
	// 原建议值范围：30.0 - 100.0（错误，会在视野外）
	// 正确范围：220.0 - 260.0（在摄像机视野内，草坪左侧）
	LawnmowerStartX = 225.0

	// LawnmowerSpeed 除草车移动速度（像素/秒）
	// 原版除草车快速向右移动的速度
	// 建议值范围：200.0 - 400.0
	LawnmowerSpeed = 300.0

	// LawnmowerTriggerBoundary 除草车触发边界（世界坐标，像素）
	// 僵尸X坐标小于此值时触发除草车
	// 应该在除草车右侧，当僵尸靠近除草车时触发
	// 设置为 240（比除草车位置 230 稍右），给除草车留出启动空间
	LawnmowerTriggerBoundary = 240.0

	// LawnmowerDeletionBoundary 除草车删除边界（世界坐标，像素）
	// 除草车X坐标超过此值时删除实体
	// 通常设置为背景宽度
	LawnmowerDeletionBoundary = BackgroundWidth

	// LawnmowerWidth 除草车宽度（像素）
	// 用于碰撞检测和渲染
	LawnmowerWidth = 80.0

	// LawnmowerHeight 除草车高度（像素）
	// 用于碰撞检测和渲染
	LawnmowerHeight = 60.0

	// LawnmowerCollisionRange 除草车碰撞检测范围（像素）
	// 僵尸在除草车前后此范围内时视为碰撞
	// 建议值范围：30.0 - 80.0
	LawnmowerCollisionRange = 50.0

	// ========== 除草车入场动画配置参数（可手工调节） ==========

	// LawnmowerEnterStartX 除草车入场动画起始X位置（世界坐标，像素）
	// 除草车从屏幕左侧外开始移动，应该在摄像机视野左侧
	// 世界坐标 = 屏幕坐标 + GameCameraX，屏幕左侧外 = 负值
	// 设置为 100，使除草车从左侧约 120 像素外开始进入
	LawnmowerEnterStartX = 100.0

	// LawnmowerEnterSpeed 除草车入场动画移动速度（像素/秒）
	// 除草车开出来的速度，建议比触发后速度慢一些
	// 建议值范围：150.0 - 300.0
	LawnmowerEnterSpeed = 200.0

	// LawnmowerEnterDelayPerLane 每行除草车入场动画延迟间隔（秒）
	// 用于实现错开入场效果，第 N 行延迟 = (N-1) * 此值
	// 建议值范围：0.1 - 0.3
	LawnmowerEnterDelayPerLane = 0.15

	// ========== 开场动画用户名显示配置（可手工调节） ==========

	// OpeningUsernameOffsetFromBottom 用户名文本距离屏幕底部的偏移（像素）
	OpeningUsernameOffsetFromBottom = 100.0

	// OpeningUsernameFontSize 用户名文本字体大小
	OpeningUsernameFontSize = 32.0

	// OpeningUsernameShadowOffsetX 用户名文本阴影X偏移（像素）
	OpeningUsernameShadowOffsetX = 2.0

	// OpeningUsernameShadowOffsetY 用户名文本阴影Y偏移（像素）
	OpeningUsernameShadowOffsetY = 2.0

	// ========== 疯狂戴夫对话系统配置参数（Story 19.1）（可手工调节） ==========
	// 注意：Dave 使用 CenterOffset = (0, 0)，直接按动画定义的坐标渲染
	// Dave 有 UIComponent，不受摄像机影响
	//
	// 动画文件定义了入场/离场的移动轨迹：
	// - anim_enter: X 从 -356.9 移动到 -55.9（动画自带走动效果）
	// - anim_idle: X 约 -55.9（静止位置）
	// - anim_leave: X 从 -55.9 移动到屏幕外
	//
	// 因此 pos 保持 (0, 0)，让动画自己控制移动，无需代码干预

	// DaveTargetX Dave 目标位置X坐标（屏幕坐标，像素）
	// 保持为 0，动画本身会将 Dave 移动到正确位置
	DaveTargetX = 0.0

	// DaveTargetY Dave 目标位置Y坐标（屏幕坐标，像素）
	// 保持为 0，使用动画定义的 Y 坐标
	DaveTargetY = 0.0

	// DaveEnterStartX Dave 入场动画起始X坐标（屏幕坐标，像素）
	// 保持为 0，因为动画本身定义了入场移动轨迹
	DaveEnterStartX = 0.0

	// DaveBubbleOffsetX 对话气泡相对于 Dave 位置的X偏移（像素）
	// Dave 静止位置约 X=-55 到 X=298，中心约 X=121
	// 气泡放在 Dave 右侧
	DaveBubbleOffsetX = 300.0

	// DaveBubbleOffsetY 对话气泡相对于 Dave 位置的Y偏移（像素）
	DaveBubbleOffsetY = 20.0

	// DaveBubblePaddingX 对话气泡内部水平内边距（像素）
	DaveBubblePaddingX = 20.0

	// DaveBubblePaddingY 对话气泡内部垂直内边距（像素）
	DaveBubblePaddingY = 15.0

	// DaveDialogueFontSize 对话文本字体大小
	DaveDialogueFontSize = 18.0

	// DaveDialogueLineHeight 对话文本行高（像素）
	DaveDialogueLineHeight = 24.0

	// DaveDialogueMaxWidth 对话文本最大宽度（像素，用于自动换行）
	// 应小于气泡宽度减去两侧内边距
	DaveDialogueMaxWidth = 240.0

	// DaveContinueTextOffsetY 「点击继续」文字距气泡底部的偏移（像素）
	DaveContinueTextOffsetY = 10.0

	// ========== 强引导教学系统配置参数（Story 19.3）（可手工调节） ==========

	// GuidedTutorialIdleThreshold 空闲触发阈值（秒）
	// 玩家无操作超过此时间后显示浮动箭头提示
	// 建议值范围：3.0 - 10.0
	GuidedTutorialIdleThreshold = 5.0

	// GuidedTutorialArrowOffsetY 箭头与铲子槽位底部的垂直距离（像素）
	// 箭头位置 Y = 铲子槽位底部 + 此偏移量
	// 建议值范围：5.0 - 20.0
	GuidedTutorialArrowOffsetY = 10.0

	// ========== 阶段转场配置参数（Story 19.4）（可手工调节） ==========

	// PhaseTransitionConveyorSlideDuration 传送带滑入时长（秒）
	PhaseTransitionConveyorSlideDuration = 0.5

	// ConveyorBeltStartY 传送带起始 Y 位置（屏幕外）
	ConveyorBeltStartY = -100.0

	// ConveyorBeltTargetY 传送带目标 Y 位置
	// 与铲子卡槽上对齐（BowlingShovelY = 0）
	ConveyorBeltTargetY = 0.0

	// BowlingRedLineColumn 红线位置（第 3 列和第 4 列之间）
	// 红线 X 坐标 = GridWorldStartX + BowlingRedLineColumn * CellWidth
	BowlingRedLineColumn = 3

	// BowlingRedLineOffsetY 红线 Y 偏移（相对于网格起点）
	// 用于调整红线垂直��置
	BowlingRedLineOffsetY = 0.0

	// ========== 传送带配置参数（Story 19.5）（可手工调节） ==========

	// ConveyorBeltWidth 传送带宽度（像素）
	// 容纳 10 张卡片 + 边框
	ConveyorBeltWidth = 450.0

	// ConveyorCardWidth 传送带卡片宽度（像素）
	ConveyorCardWidth = 40.0

	// ConveyorCardHeight 传送带卡片高度（像素）
	ConveyorCardHeight = 60.0

	// ConveyorCardSpacing 卡片间距（像素）
	ConveyorCardSpacing = 2.0

	// ConveyorBeltAnimSpeed 传动动画速度（像素/秒）
	ConveyorBeltAnimSpeed = 50.0

	// ConveyorCardGenerationInterval 默认卡片生成间隔（秒）
	ConveyorCardGenerationInterval = 3.0

	// ConveyorBeltPadding 传送带内边距（像素）
	// 卡片距离传送带边缘的距离
	ConveyorBeltPadding = 5.0

	// ConveyorBeltRowCount 传送带纹理行数
	// ConveyorBelt.png 有 6 行交错纹理
	ConveyorBeltRowCount = 6

	// ConveyorBeltAnimOffsetX 履带纹理相对于背景的 X 偏移（像素）
	// 正值向右移动，负值向左移动
	// 用于调整履带在背景上的水平位置
	ConveyorBeltAnimOffsetX = 0.0

	// ConveyorBeltAnimOffsetY 履带纹理相对于背景的 Y 偏移（像素）
	// 正值向下移动，负值向上移动
	// 背景高度 86px，履带高度 96px（6行x16px）
	// 履带应该在背景的下边部分显示
	ConveyorBeltAnimOffsetY = 60.0

	// ========== 保龄球坚果配置参数（Story 19.6）（可手工调节） ==========

	// BowlingNutRollingStartFrame 滚动动画起始帧（逻辑帧索引）
	// anim_face 轨道中滚动动画的起始逻辑帧
	// 逻辑帧 17 对应物理帧 43（kx=0°）
	BowlingNutRollingStartFrame = 18

	// BowlingNutCollisionWidth 保龄球坚果碰撞盒宽度（像素）
	// 用于 Story 19.7 碰撞检测
	BowlingNutCollisionWidth = 60.0

	// BowlingNutCollisionHeight 保龄球坚果碰撞盒高度（像素）
	// 用于 Story 19.7 碰撞检测
	BowlingNutCollisionHeight = 60.0

	// ConveyorCardScale 传送带卡片缩放比例
	// 用于等比例缩小卡片，基于原始卡片背景尺寸（约 100x140）
	// 建议值范围：0.3 - 0.6
	ConveyorCardScale = 0.5

	// ConveyorCardSelectedOverlayAlpha 选中卡片遮罩透明度（0-255）
	// 选中状态的传送带卡片添加的半透明灰色遮罩
	ConveyorCardSelectedOverlayAlpha = 128

	// ConveyorCardSlideInSpeed 卡片滑入速度（单位/秒）
	// 控制卡片从传送带右侧进入的速度
	// 值越小速度越慢：0.5 = 2秒完成，1.0 = 1秒完成，2.0 = 0.5秒完成
	// 建议值范围：0.3 - 2.0
	ConveyorCardSlideInSpeed = 0.35

	// ConveyorBeltLeftPadding 传送带左侧内边距（像素）
	// 卡片不会进入此区域，防止卡片太靠左遮挡边框
	// 建议值范围：10.0 - 30.0
	ConveyorBeltLeftPadding = 10.0

	// BowlingNutPreviewAlpha 草坪预览透明度（0-255）
	// 悬停在草坪网格时显示的半透明坚果预览
	BowlingNutPreviewAlpha = 150

	// ========== 保龄球坚果碰撞与弹射配置参数（Story 19.7）（可手工调节） ==========

	// BowlingNutBounceSpeed 弹射时的垂直移动速度（像素/秒）
	// 建议值范围：200.0 - 400.0
	BowlingNutBounceSpeed = 300.0

	// BowlingNutCollisionCooldown 碰撞冷却时间（秒）
	// 碰撞后短暂时间内不再检测碰撞，防止重复碰撞
	BowlingNutCollisionCooldown = 0.1

	// ExplosiveNutExplosionRadius 爆炸坚果爆炸范围半径（格子数）
	// Story 19.8: 1.5 格子距离覆盖 3x3 范围内的所有僵尸
	// 实际像素半径 = ExplosiveNutExplosionRadius * CellWidth = 1.5 * 80 = 120 像素
	ExplosiveNutExplosionRadius = 1.5
)

// CalculateBowlingNutSpeed 动态计算保龄球坚果滚动速度
// 使位移速度与动画旋转速度同步，避免"滑步"效果
//
// 计算公式：
//
//	动画周期 = 帧数 / FPS
//	速度 = 周长 / 动画周期 = 周长 * FPS / 帧数
//
// 参数:
//   - fps: 动画帧率（从 ReanimXML.FPS 获取）
//   - rollingFrameCount: 滚动动画帧数（从 AnimVisibles 计算）
//   - circumference: 滚动周长（从动画 x 轨迹计算）
//
// 返回:
//   - float64: 滚动速度（像素/秒）
func CalculateBowlingNutSpeed(fps int, rollingFrameCount int, circumference float64) float64 {
	animationPeriod := float64(rollingFrameCount) / float64(fps)
	return circumference / animationPeriod
}
