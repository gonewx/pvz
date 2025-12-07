package systems

import (
	"image/color"
	"log"
	"math"
	"math/rand"
	"regexp"
	"strings"

	"github.com/decker502/pvz/pkg/components"
	"github.com/decker502/pvz/pkg/config"
	"github.com/decker502/pvz/pkg/ecs"
	"github.com/decker502/pvz/pkg/game"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

// 文字抖动参数（参考 ZombiesWonPhaseSystem 的屏幕抖动参数）
const (
	TextShakeAmplitude = 3.0  // 振幅（像素）
	TextShakeFrequency = 10.0 // 频率（Hz），相对较低以便可见
)

// Dave 说话动画参数
// 根据文本长度选择不同动画，动画播放到与文本长度匹配的帧数后暂停
const (
	// 动画可见帧范围（24 FPS，从 reanim 文件分析得出）
	// anim_smalltalk: 帧 85-94 可见（9帧）
	// anim_mediumtalk: 帧 130-148 可见（18帧）
	// anim_blahblah: 帧 33-64 可见（31帧）
	DaveSmallTalkStartFrame  = 85
	DaveSmallTalkEndFrame    = 94
	DaveMediumTalkStartFrame = 130
	DaveMediumTalkEndFrame   = 148
	DaveBlahBlahStartFrame   = 33
	DaveBlahBlahEndFrame     = 64

	// 文本长度阈值（按 rune 计数，中文每个字符为 1）
	DaveShortTextThreshold  = 8  // ≤8 字符使用 anim_smalltalk (9帧)
	DaveMediumTextThreshold = 12 // ≤12 字符使用 anim_mediumtalk (18帧)，>12 使用 anim_blahblah (31帧)
)

// Dave 音效 ID 常量
// 根据文本长度和表情指令选择对应的音效
var (
	// 短对话音效（对应 anim_smalltalk）
	DaveShortSounds = []string{
		"SOUND_CRAZYDAVESHORT1",
		"SOUND_CRAZYDAVESHORT2",
		"SOUND_CRAZYDAVESHORT3",
	}
	// 中等对话音效（对应 anim_mediumtalk）
	DaveLongSounds = []string{
		"SOUND_CRAZYDAVELONG1",
		"SOUND_CRAZYDAVELONG2",
		"SOUND_CRAZYDAVELONG3",
	}
	// 长对话音效（对应 anim_blahblah）
	DaveExtraLongSounds = []string{
		"SOUND_CRAZYDAVEEXTRALONG1",
		"SOUND_CRAZYDAVEEXTRALONG2",
		"SOUND_CRAZYDAVEEXTRALONG3",
	}
	// 疯狂音效（对应 {SHAKE} 表情）
	DaveCrazySound = "SOUND_CRAZYDAVECRAZY"
	// 尖叫音效（对应 {SCREAM} 表情）
	DaveScreamSounds = []string{
		"SOUND_CRAZYDAVESCREAM",
		"SOUND_CRAZYDAVESCREAM2",
	}
)

// DaveDialogueSystem 疯狂戴夫对话系统
// 管理 Dave 的出场动画、对话显示、表情切换和离场动画
//
// 职责：
//   - 处理状态机：Hidden → Entering → Talking → Leaving → Hidden
//   - 解析表情指令（{MOUTH_SMALL_OH} 等）
//   - 处理点击输入推进对话
//   - 渲染对话气泡和文本
type DaveDialogueSystem struct {
	entityManager   *ecs.EntityManager
	gameState       *game.GameState
	resourceManager *game.ResourceManager

	// 对话气泡资源
	bubbleImage *ebiten.Image

	// 字体
	dialogueFont *text.GoTextFace

	// 表情指令正则表达式（预编译优化性能）
	expressionRegex *regexp.Regexp
}

// NewDaveDialogueSystem 创建疯狂戴夫对话系统
// 参数：
//   - em: EntityManager 实例
//   - gs: GameState 实例（用于访问 LawnStrings）
//   - rm: ResourceManager 实例（用于加载图片和字体）
//
// 返回：
//   - *DaveDialogueSystem: 系统实例
func NewDaveDialogueSystem(
	em *ecs.EntityManager,
	gs *game.GameState,
	rm *game.ResourceManager,
) *DaveDialogueSystem {
	// 加载对话气泡图片
	bubbleImage, err := rm.LoadImage("assets/images/Store_SpeechBubble2.png")
	if err != nil {
		log.Printf("[DaveDialogueSystem] Warning: Failed to load speech bubble image: %v", err)
	}

	// 加载对话字体（18号）
	dialogueFont, err := rm.LoadFont("assets/fonts/SimHei.ttf", config.DaveDialogueFontSize)
	if err != nil {
		log.Printf("[DaveDialogueSystem] Warning: Failed to load dialogue font: %v", err)
	}

	// 预编译表情指令正则表达式
	// 匹配 {WORD} 格式的指令，如 {MOUTH_SMALL_OH}、{SCREAM} 等
	expressionRegex := regexp.MustCompile(`\{([A-Z_]+)\}`)

	log.Printf("[DaveDialogueSystem] Initialized")

	return &DaveDialogueSystem{
		entityManager:   em,
		gameState:       gs,
		resourceManager: rm,
		bubbleImage:     bubbleImage,
		dialogueFont:    dialogueFont,
		expressionRegex: expressionRegex,
	}
}

// Update 更新对话系统状态
// 处理状态机转换、动画完成检测、点击输入
func (s *DaveDialogueSystem) Update(dt float64) {
	// 查询所有 Dave 对话实体
	daveEntities := ecs.GetEntitiesWith2[
		*components.DaveDialogueComponent,
		*components.PositionComponent,
	](s.entityManager)

	for _, entityID := range daveEntities {
		dialogueComp, _ := ecs.GetComponent[*components.DaveDialogueComponent](s.entityManager, entityID)
		posComp, _ := ecs.GetComponent[*components.PositionComponent](s.entityManager, entityID)
		reanimComp, hasReanimComp := ecs.GetComponent[*components.ReanimComponent](s.entityManager, entityID)

		switch dialogueComp.State {
		case components.DaveStateEntering:
			s.updateEnteringState(entityID, dialogueComp, posComp, reanimComp, hasReanimComp, dt)

		case components.DaveStateTalking:
			s.updateTalkingState(entityID, dialogueComp, posComp, dt)

		case components.DaveStateLeaving:
			s.updateLeavingState(entityID, dialogueComp, posComp, reanimComp, hasReanimComp, dt)

		case components.DaveStateHidden:
			// Hidden 状态下，检查是否需要销毁实体
			// 通常在离场动画完成后已经触发回调，这里不需要额外处理
		}
	}
}

// updateEnteringState 处理入场状态
// 动画文件 CrazyDave.reanim 已定义入场移动轨迹（anim_enter: X 从 -356.9 到 -55.9）
// 因此不需要手动移动位置，只需检测动画完成状态
func (s *DaveDialogueSystem) updateEnteringState(
	entityID ecs.EntityID,
	dialogueComp *components.DaveDialogueComponent,
	posComp *components.PositionComponent,
	reanimComp *components.ReanimComponent,
	hasReanimComp bool,
	dt float64,
) {
	// 检测入场动画是否完成
	// anim_enter 是非循环动画，播放完成后 IsFinished 会被设置为 true
	if hasReanimComp && reanimComp.IsFinished {
		// 切换到 Talking 状态
		dialogueComp.State = components.DaveStateTalking
		dialogueComp.IsVisible = true

		// 加载第一条对话（会计算动画类型和目标帧）
		s.loadCurrentDialogue(dialogueComp)

		// Bug Fix: 应用第一条对话的表情指令
		// 之前只在 advanceDialogue 中调用 applyExpressions，导致第一条对话的表情不生效
		s.applyExpressions(entityID, dialogueComp)

		log.Printf("[DaveDialogueSystem] Entity %d: Entering → Talking, anim_enter finished", entityID)
	}
}

// updateTalkingState 处理对话状态
func (s *DaveDialogueSystem) updateTalkingState(
	entityID ecs.EntityID,
	dialogueComp *components.DaveDialogueComponent,
	posComp *components.PositionComponent,
	dt float64,
) {
	// 更新文字抖动计时器
	if dialogueComp.TextShaking {
		dialogueComp.TextShakeTime += dt
	}

	// 监控说话动画帧，到达目标帧后暂停
	if dialogueComp.TalkAnimationStarted && !dialogueComp.IsHoldingItem {
		reanimComp, ok := ecs.GetComponent[*components.ReanimComponent](s.entityManager, entityID)
		if ok && !reanimComp.IsPaused && len(reanimComp.CurrentAnimations) > 0 {
			animName := reanimComp.CurrentAnimations[0]
			// 检查是否是说话动画
			if animName == dialogueComp.CurrentTalkAnimation {
				// 获取当前帧
				currentFrame := reanimComp.CurrentFrame
				if frameIdx, exists := reanimComp.AnimationFrameIndices[animName]; exists {
					currentFrame = int(frameIdx)
				}

				// 到达目标帧后暂停动画
				if currentFrame >= dialogueComp.TalkAnimationTargetFrame {
					reanimComp.IsPaused = true
					log.Printf("[DaveDialogueSystem] Entity %d: Talk animation paused at frame %d (target=%d)",
						entityID, currentFrame, dialogueComp.TalkAnimationTargetFrame)
				}
			}
		}
	}

	// 检测点击输入
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		s.advanceDialogue(entityID, dialogueComp)
	}
}

// updateLeavingState 处理离场状态
// 动画文件 CrazyDave.reanim 已定义离场移动轨迹（anim_leave: X 从 -55.9 移动到屏幕外）
// 因此不需要手动移动位置，只需检测动画完成状态
func (s *DaveDialogueSystem) updateLeavingState(
	entityID ecs.EntityID,
	dialogueComp *components.DaveDialogueComponent,
	posComp *components.PositionComponent,
	reanimComp *components.ReanimComponent,
	hasReanimComp bool,
	dt float64,
) {
	// 检测离场动画是否完成
	// anim_leave 是非循环动画，播放完成后 IsFinished 会被设置为 true
	if hasReanimComp && reanimComp.IsFinished {
		// 切换到 Hidden 状态
		dialogueComp.State = components.DaveStateHidden

		// 触发完成回调
		if dialogueComp.OnCompleteCallback != nil {
			dialogueComp.OnCompleteCallback()
		}

		// 销毁实体
		s.entityManager.DestroyEntity(entityID)

		log.Printf("[DaveDialogueSystem] Entity %d: Leaving → Hidden, anim_leave finished, entity destroyed", entityID)
	}
}

// advanceDialogue 推进对话到下一条
func (s *DaveDialogueSystem) advanceDialogue(
	entityID ecs.EntityID,
	dialogueComp *components.DaveDialogueComponent,
) {
	// 检查是否还有下一条对话
	if dialogueComp.CurrentLineIndex < len(dialogueComp.DialogueKeys)-1 {
		// 推进到下一条
		dialogueComp.CurrentLineIndex++
		s.loadCurrentDialogue(dialogueComp)

		// 应用表情指令
		s.applyExpressions(entityID, dialogueComp)

		log.Printf("[DaveDialogueSystem] Entity %d: Advanced to dialogue %d/%d",
			entityID, dialogueComp.CurrentLineIndex+1, len(dialogueComp.DialogueKeys))
	} else {
		// 最后一条对话，切换到离场状态
		dialogueComp.State = components.DaveStateLeaving
		dialogueComp.IsVisible = false

		// 播放离场动画
		s.playAnimation(entityID, "anim_leave", false)

		log.Printf("[DaveDialogueSystem] Entity %d: Talking → Leaving", entityID)
	}
}

// loadCurrentDialogue 加载当前对话文本
func (s *DaveDialogueSystem) loadCurrentDialogue(dialogueComp *components.DaveDialogueComponent) {
	if dialogueComp.CurrentLineIndex >= len(dialogueComp.DialogueKeys) {
		return
	}

	key := dialogueComp.DialogueKeys[dialogueComp.CurrentLineIndex]
	rawText := s.gameState.LawnStrings.GetString(key)

	// 解析表情指令
	cleanText, expressions := s.parseExpressionCommands(rawText)

	// 合并 key 行上的标签（如 {SHAKE}, {SHOW_WALLNUT}）
	// 这些标签在 [KEY] {TAG} 格式中定义，需要优先应用
	keyTags := s.gameState.LawnStrings.GetKeyTags(key)
	if len(keyTags) > 0 {
		// key 行标签插入到表情列表前面，优先处理
		expressions = append(keyTags, expressions...)
	}

	dialogueComp.CurrentText = cleanText
	dialogueComp.CurrentExpressions = expressions

	// 重置文字抖动状态（新对话开始时）
	dialogueComp.TextShaking = false
	dialogueComp.TextShakeTime = 0.0

	// 计算说话动画类型和目标帧
	s.calculateTalkAnimation(dialogueComp, cleanText)

	log.Printf("[DaveDialogueSystem] Loaded dialogue key=%s, text='%s', expressions=%v, keyTags=%v, anim=%s, targetFrame=%d",
		key, cleanText, expressions, keyTags, dialogueComp.CurrentTalkAnimation, dialogueComp.TalkAnimationTargetFrame)
}

// calculateTalkAnimation 根据文本长度计算说话动画类型和目标停止帧
func (s *DaveDialogueSystem) calculateTalkAnimation(dialogueComp *components.DaveDialogueComponent, text string) {
	textLen := len([]rune(text))

	// 根据文本长度选择动画和计算目标停止帧
	var animName string
	var startFrame, endFrame int

	if textLen <= DaveShortTextThreshold {
		animName = "anim_smalltalk"
		startFrame = DaveSmallTalkStartFrame
		endFrame = DaveSmallTalkEndFrame
	} else if textLen <= DaveMediumTextThreshold {
		animName = "anim_mediumtalk"
		startFrame = DaveMediumTalkStartFrame
		endFrame = DaveMediumTalkEndFrame
	} else {
		animName = "anim_blahblah"
		startFrame = DaveBlahBlahStartFrame
		endFrame = DaveBlahBlahEndFrame
	}

	// 动画帧范围
	frameRange := endFrame - startFrame

	// 计算目标停止帧：根据文本长度在帧范围内线性映射
	// 文本越长，播放到越接近 endFrame 的位置
	var targetFrame int
	if textLen <= DaveShortTextThreshold {
		// 短文本：在 smalltalk 范围内按比例
		ratio := float64(textLen) / float64(DaveShortTextThreshold)
		targetFrame = startFrame + int(float64(frameRange)*ratio)
	} else if textLen <= DaveMediumTextThreshold {
		// 中等文本：在 mediumtalk 范围内按比例
		ratio := float64(textLen-DaveShortTextThreshold) / float64(DaveMediumTextThreshold-DaveShortTextThreshold)
		targetFrame = startFrame + int(float64(frameRange)*ratio)
	} else {
		// 长文本：在 blahblah 范围内按比例，最大 30 字符映射到完整范围
		maxLongText := 30
		effectiveLen := textLen - DaveMediumTextThreshold
		if effectiveLen > maxLongText-DaveMediumTextThreshold {
			effectiveLen = maxLongText - DaveMediumTextThreshold
		}
		ratio := float64(effectiveLen) / float64(maxLongText-DaveMediumTextThreshold)
		targetFrame = startFrame + int(float64(frameRange)*ratio)
	}

	// 确保目标帧在有效范围内
	if targetFrame < startFrame {
		targetFrame = startFrame
	}
	if targetFrame > endFrame {
		targetFrame = endFrame
	}

	dialogueComp.CurrentTalkAnimation = animName
	dialogueComp.TalkAnimationTargetFrame = targetFrame
	dialogueComp.TalkAnimationStarted = false
}

// parseExpressionCommands 解析并移除表情指令
// 输入: "开始挖吧！{MOUTH_SMALL_OH} {SCREAM}"
// 输出: ("开始挖吧！", ["MOUTH_SMALL_OH", "SCREAM"])
func (s *DaveDialogueSystem) parseExpressionCommands(text string) (string, []string) {
	var expressions []string

	// 提取所有表情指令
	matches := s.expressionRegex.FindAllStringSubmatch(text, -1)
	for _, match := range matches {
		if len(match) >= 2 {
			expressions = append(expressions, match[1])
		}
	}

	// 移除表情指令并清理多余空格
	cleanText := s.expressionRegex.ReplaceAllString(text, "")
	cleanText = strings.TrimSpace(cleanText)
	// 替换多个连续空格为单个空格
	cleanText = regexp.MustCompile(`\s+`).ReplaceAllString(cleanText, " ")

	return cleanText, expressions
}

// applyExpressions 应用表情指令
func (s *DaveDialogueSystem) applyExpressions(
	entityID ecs.EntityID,
	dialogueComp *components.DaveDialogueComponent,
) {
	// 重置手持物品状态
	wasHoldingItem := dialogueComp.IsHoldingItem
	dialogueComp.IsHoldingItem = false
	dialogueComp.HeldItemType = ""

	// 检查是否需要特殊动画和特殊音效
	hasSpecialAnimation := false
	hasSpecialSound := false

	for _, expr := range dialogueComp.CurrentExpressions {
		switch expr {
		case "MOUTH_SMALL_OH":
			// 切换到 MOUTH6 轨道（小嘴 O 形）
			dialogueComp.Expression = "MOUTH_SMALL_OH"
			s.setMouthTrack(entityID, "IMAGE_REANIM_CRAZYDAVE_MOUTH6")

		case "MOUTH_SMALL_SMILE":
			// 切换到 MOUTH2 轨道（微笑线条）
			dialogueComp.Expression = "MOUTH_SMALL_SMILE"
			s.setMouthTrack(entityID, "IMAGE_REANIM_CRAZYDAVE_MOUTH2")

		case "MOUTH_BIG_SMILE":
			// 切换到 MOUTH5 轨道（大笑露齿）
			dialogueComp.Expression = "MOUTH_BIG_SMILE"
			s.setMouthTrack(entityID, "IMAGE_REANIM_CRAZYDAVE_MOUTH5")

		case "SHAKE":
			// 播放 anim_crazy 动画 + 启用文字抖动效果 + 播放疯狂音效
			dialogueComp.Expression = "SHAKE"
			dialogueComp.TextShaking = true
			dialogueComp.TextShakeTime = 0.0
			s.playAnimation(entityID, "anim_crazy", false)
			s.playDaveSound(DaveCrazySound)
			hasSpecialAnimation = true
			hasSpecialSound = true

		case "SCREAM":
			// 切换到 MOUTH1 轨道 + 播放 anim_crazy + 播放尖叫音效
			dialogueComp.Expression = "SCREAM"
			s.setMouthTrack(entityID, "IMAGE_REANIM_CRAZYDAVE_MOUTH1")
			s.playAnimation(entityID, "anim_crazy", false)
			s.playRandomDaveSound(DaveScreamSounds)
			hasSpecialAnimation = true
			hasSpecialSound = true

		case "SHOW_WALLNUT":
			// 显示拿坚果的动画
			dialogueComp.IsHoldingItem = true
			dialogueComp.HeldItemType = "wallnut"
			// 播放拿物品说话的动画（非循环，与其他特殊动画保持一致）
			s.playAnimation(entityID, "anim_talk_handing", false)
			// 设置坚果图片到手部轨道
			s.setHandingItemImage(entityID, "assets/reanim/Wallnut_body.png")
			hasSpecialAnimation = true
			log.Printf("[DaveDialogueSystem] SHOW_WALLNUT: Playing anim_talk_handing with wallnut")

		default:
			log.Printf("[DaveDialogueSystem] Unknown expression: %s", expr)
		}
	}

	// 如果没有特殊动画指令，播放根据文本长度选择的说话动画（非循环）
	if !hasSpecialAnimation {
		if wasHoldingItem {
			// 清除手持物品图片
			s.clearHandingItemImage(entityID)
		}
		// 播放计算好的说话动画（非循环，到达目标帧后暂停）
		s.playTalkAnimation(entityID, dialogueComp)
	}

	// 如果没有特殊音效，根据文本长度播放对应的说话音效
	if !hasSpecialSound {
		s.playTalkSound(dialogueComp)
	}
}

// setMouthTrack 设置嘴型轨道（通过 ImageOverrides）
func (s *DaveDialogueSystem) setMouthTrack(entityID ecs.EntityID, mouthImageKey string) {
	reanimComp, ok := ecs.GetComponent[*components.ReanimComponent](s.entityManager, entityID)
	if !ok {
		return
	}

	// 获取目标嘴型图片
	mouthImage, exists := reanimComp.PartImages[mouthImageKey]
	if !exists {
		log.Printf("[DaveDialogueSystem] Mouth image not found: %s", mouthImageKey)
		return
	}

	// 使用 ImageOverrides 覆盖当前嘴型
	// 需要找到当前正在显示的嘴型轨道并覆盖
	if reanimComp.ImageOverrides == nil {
		reanimComp.ImageOverrides = make(map[string]*ebiten.Image)
	}

	// 覆盖所有嘴型图片到目标嘴型
	mouthKeys := []string{
		"IMAGE_REANIM_CRAZYDAVE_MOUTH1",
		"IMAGE_REANIM_CRAZYDAVE_MOUTH2",
		"IMAGE_REANIM_CRAZYDAVE_MOUTH3",
		"IMAGE_REANIM_CRAZYDAVE_MOUTH4",
		"IMAGE_REANIM_CRAZYDAVE_MOUTH5",
		"IMAGE_REANIM_CRAZYDAVE_MOUTH6",
	}

	for _, key := range mouthKeys {
		if key != mouthImageKey {
			reanimComp.ImageOverrides[key] = mouthImage
		}
	}

	log.Printf("[DaveDialogueSystem] Entity %d: Set mouth track to %s", entityID, mouthImageKey)
}

// setHandingItemImage 设置手持物品图片
// 将指定图片设置到 Dave 的手部轨道上
func (s *DaveDialogueSystem) setHandingItemImage(entityID ecs.EntityID, imagePath string) {
	reanimComp, ok := ecs.GetComponent[*components.ReanimComponent](s.entityManager, entityID)
	if !ok {
		return
	}

	// 加载物品图片
	itemImage, err := s.resourceManager.LoadImage(imagePath)
	if err != nil {
		log.Printf("[DaveDialogueSystem] Failed to load handing item image %s: %v", imagePath, err)
		return
	}

	// 使用 ImageOverrides 将手部图片覆盖为物品图片
	// handinghand3 是手掌上方的图层，适合放置物品
	if reanimComp.ImageOverrides == nil {
		reanimComp.ImageOverrides = make(map[string]*ebiten.Image)
	}

	// 将物品图片设置到 handinghand3 轨道（手掌上方）
	reanimComp.ImageOverrides["IMAGE_REANIM_CRAZYDAVE_HANDINGHAND3"] = itemImage

	log.Printf("[DaveDialogueSystem] Entity %d: Set handing item image to %s", entityID, imagePath)
}

// clearHandingItemImage 清除手持物品图片
func (s *DaveDialogueSystem) clearHandingItemImage(entityID ecs.EntityID) {
	reanimComp, ok := ecs.GetComponent[*components.ReanimComponent](s.entityManager, entityID)
	if !ok {
		return
	}

	if reanimComp.ImageOverrides != nil {
		delete(reanimComp.ImageOverrides, "IMAGE_REANIM_CRAZYDAVE_HANDINGHAND3")
	}

	log.Printf("[DaveDialogueSystem] Entity %d: Cleared handing item image", entityID)
}

// playAnimation 播放动画
func (s *DaveDialogueSystem) playAnimation(entityID ecs.EntityID, animName string, loop bool) {
	// 使用 AnimationCommandComponent 触发动画（符合 ECS 架构）
	ecs.AddComponent(s.entityManager, entityID, &components.AnimationCommandComponent{
		UnitID:    "crazydave",
		ComboName: animName,
		Processed: false,
	})

	// 更新 ReanimComponent 的循环设置
	reanimComp, ok := ecs.GetComponent[*components.ReanimComponent](s.entityManager, entityID)
	if ok {
		reanimComp.IsLooping = loop
		reanimComp.IsPaused = false // 确保动画不是暂停状态
	}

	log.Printf("[DaveDialogueSystem] Entity %d: Playing animation %s (loop=%v)", entityID, animName, loop)
}

// playTalkAnimation 播放根据文本长度选择的说话动画
// 动画为非循环，到达目标帧后由 updateTalkingState 暂停
func (s *DaveDialogueSystem) playTalkAnimation(entityID ecs.EntityID, dialogueComp *components.DaveDialogueComponent) {
	animName := dialogueComp.CurrentTalkAnimation
	if animName == "" {
		animName = "anim_smalltalk" // 默认使用短动画
	}

	// 播放动画（非循环）
	s.playAnimation(entityID, animName, false)

	// 标记动画已开始
	dialogueComp.TalkAnimationStarted = true

	log.Printf("[DaveDialogueSystem] Entity %d: Playing talk animation %s, target frame=%d",
		entityID, animName, dialogueComp.TalkAnimationTargetFrame)
}

// Draw 渲染对话气泡和文本
func (s *DaveDialogueSystem) Draw(screen *ebiten.Image) {
	// 查询所有 Dave 对话实体
	daveEntities := ecs.GetEntitiesWith2[
		*components.DaveDialogueComponent,
		*components.PositionComponent,
	](s.entityManager)

	for _, entityID := range daveEntities {
		dialogueComp, _ := ecs.GetComponent[*components.DaveDialogueComponent](s.entityManager, entityID)
		posComp, _ := ecs.GetComponent[*components.PositionComponent](s.entityManager, entityID)

		// 只在对话可见时渲染
		if !dialogueComp.IsVisible {
			continue
		}

		// 计算气泡位置
		bubbleX := posComp.X + dialogueComp.BubbleOffsetX
		bubbleY := posComp.Y + dialogueComp.BubbleOffsetY

		// 渲染对话气泡背景
		s.drawBubble(screen, bubbleX, bubbleY)

		// 渲染对话文本（传递抖动状态）
		s.drawDialogueText(screen, dialogueComp.CurrentText, bubbleX, bubbleY,
			dialogueComp.TextShaking, dialogueComp.TextShakeTime)

		// 渲染「点击继续」提示
		s.drawContinueHint(screen, bubbleX, bubbleY)
	}
}

// drawBubble 渲染对话气泡背景
func (s *DaveDialogueSystem) drawBubble(screen *ebiten.Image, x, y float64) {
	if s.bubbleImage == nil {
		return
	}

	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(x, y)
	screen.DrawImage(s.bubbleImage, op)
}

// drawDialogueText 渲染对话文本
// shaking: 是否启用文字抖动效果
// shakeTime: 抖动计时器（秒）
func (s *DaveDialogueSystem) drawDialogueText(screen *ebiten.Image, textStr string, bubbleX, bubbleY float64, shaking bool, shakeTime float64) {
	if s.dialogueFont == nil || textStr == "" {
		return
	}

	// 获取气泡尺寸
	var bubbleWidth, bubbleHeight float64 = 280, 183
	if s.bubbleImage != nil {
		bounds := s.bubbleImage.Bounds()
		bubbleWidth = float64(bounds.Dx())
		bubbleHeight = float64(bounds.Dy())
	}

	// 计算文本区域（气泡内部，扣除边距和尖角）
	textAreaX := bubbleX + config.DaveBubblePaddingX
	textAreaY := bubbleY + config.DaveBubblePaddingY
	textAreaWidth := bubbleWidth - 2*config.DaveBubblePaddingX
	textAreaHeight := bubbleHeight - 2*config.DaveBubblePaddingY - 30 // 预留底部「点击继续」空间

	// 自动换行处理
	lines := s.wrapText(textStr, textAreaWidth)

	// 计算垂直居中偏移
	totalTextHeight := float64(len(lines)) * config.DaveDialogueLineHeight
	startY := textAreaY + (textAreaHeight-totalTextHeight)/2

	// 计算抖动偏移量
	var shakeOffsetX, shakeOffsetY float64
	if shaking {
		// 使用正弦波计算抖动偏移（水平+垂直同时抖动，相位差90度）
		shakeOffsetX = TextShakeAmplitude * math.Sin(2*math.Pi*TextShakeFrequency*shakeTime)
		shakeOffsetY = TextShakeAmplitude * math.Sin(2*math.Pi*TextShakeFrequency*shakeTime+math.Pi/2)
	}

	// 渲染每一行
	for i, line := range lines {
		lineY := startY + float64(i)*config.DaveDialogueLineHeight

		// 计算水平居中
		lineWidth := s.measureTextWidth(line)
		lineX := textAreaX + (textAreaWidth-lineWidth)/2

		// 应用抖动偏移
		finalX := lineX + shakeOffsetX
		finalY := lineY + shakeOffsetY

		// 绘制文本（黑色）
		op := &text.DrawOptions{}
		op.GeoM.Translate(finalX, finalY)
		op.ColorScale.ScaleWithColor(color.Black)
		text.Draw(screen, line, s.dialogueFont, op)
	}
}

// drawContinueHint 渲染「点击继续」提示
func (s *DaveDialogueSystem) drawContinueHint(screen *ebiten.Image, bubbleX, bubbleY float64) {
	if s.dialogueFont == nil {
		return
	}

	hintText := "点击继续"

	// 获取气泡尺寸
	var bubbleWidth, bubbleHeight float64 = 280, 183
	if s.bubbleImage != nil {
		bounds := s.bubbleImage.Bounds()
		bubbleWidth = float64(bounds.Dx())
		bubbleHeight = float64(bounds.Dy())
	}

	// 计算位置（气泡底部居中，调整距离底部的偏移）
	hintWidth := s.measureTextWidth(hintText)
	hintX := bubbleX + (bubbleWidth-hintWidth)/2
	// Story 19.x QA: 调整位置，减小距离底部的偏移量，使其更靠近文本区域
	hintY := bubbleY + bubbleHeight - config.DaveContinueTextOffsetY - config.DaveDialogueFontSize - 20

	// 绘制文本（黑色，Story 19.x QA: 修复颜色问题）
	op := &text.DrawOptions{}
	op.GeoM.Translate(hintX, hintY)
	op.ColorScale.ScaleWithColor(color.Black)
	text.Draw(screen, hintText, s.dialogueFont, op)
}

// wrapText 自动换行处理
func (s *DaveDialogueSystem) wrapText(textStr string, maxWidth float64) []string {
	if s.dialogueFont == nil {
		return []string{textStr}
	}

	var lines []string
	var currentLine string

	runes := []rune(textStr)
	for _, r := range runes {
		testLine := currentLine + string(r)
		testWidth := s.measureTextWidth(testLine)

		if testWidth > maxWidth && currentLine != "" {
			lines = append(lines, currentLine)
			currentLine = string(r)
		} else {
			currentLine = testLine
		}
	}

	if currentLine != "" {
		lines = append(lines, currentLine)
	}

	return lines
}

// measureTextWidth 测量文本宽度
func (s *DaveDialogueSystem) measureTextWidth(textStr string) float64 {
	if s.dialogueFont == nil {
		return 0
	}

	width, _ := text.Measure(textStr, s.dialogueFont, config.DaveDialogueLineHeight)
	return width
}

// ==========================================================================
// 音效播放方法 (Sound Playback Methods)
// ==========================================================================

// playTalkSound 根据文本长度播放对应的说话音效
// 短文本 → DaveShortSounds
// 中等文本 → DaveLongSounds
// 长文本 → DaveExtraLongSounds
func (s *DaveDialogueSystem) playTalkSound(dialogueComp *components.DaveDialogueComponent) {
	var soundList []string

	switch dialogueComp.CurrentTalkAnimation {
	case "anim_smalltalk":
		soundList = DaveShortSounds
	case "anim_mediumtalk":
		soundList = DaveLongSounds
	case "anim_blahblah":
		soundList = DaveExtraLongSounds
	default:
		// 默认使用短音效
		soundList = DaveShortSounds
	}

	s.playRandomDaveSound(soundList)
}

// playRandomDaveSound 从音效列表中随机选择一个播放
func (s *DaveDialogueSystem) playRandomDaveSound(soundList []string) {
	if len(soundList) == 0 {
		return
	}

	soundID := soundList[rand.Intn(len(soundList))]
	s.playDaveSound(soundID)
}

// playDaveSound 播放指定的 Dave 音效
func (s *DaveDialogueSystem) playDaveSound(soundID string) {
	if audioManager := s.gameState.GetAudioManager(); audioManager != nil {
		audioManager.PlaySound(soundID)
		log.Printf("[DaveDialogueSystem] Playing sound: %s", soundID)
	} else {
		log.Printf("[DaveDialogueSystem] Warning: AudioManager not available for %s", soundID)
	}
}
