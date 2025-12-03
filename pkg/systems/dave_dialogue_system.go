package systems

import (
	"image/color"
	"log"
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

		// 加载第一条对话
		s.loadCurrentDialogue(dialogueComp)

		// 切换到 idle 动画（循环播放）
		s.playAnimation(entityID, "anim_idle", true)

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

	dialogueComp.CurrentText = cleanText
	dialogueComp.CurrentExpressions = expressions

	log.Printf("[DaveDialogueSystem] Loaded dialogue key=%s, text='%s', expressions=%v",
		key, cleanText, expressions)
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
			// 播放 anim_crazy 动画
			dialogueComp.Expression = "SHAKE"
			s.playAnimation(entityID, "anim_crazy", false)

		case "SCREAM":
			// 切换到 MOUTH1 轨道 + 播放 anim_crazy
			dialogueComp.Expression = "SCREAM"
			s.setMouthTrack(entityID, "IMAGE_REANIM_CRAZYDAVE_MOUTH1")
			s.playAnimation(entityID, "anim_crazy", false)

		case "SHOW_WALLNUT":
			// 本 Story 不实现，后续 Story 处理
			log.Printf("[DaveDialogueSystem] Expression SHOW_WALLNUT not implemented yet")

		default:
			log.Printf("[DaveDialogueSystem] Unknown expression: %s", expr)
		}
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
	}

	log.Printf("[DaveDialogueSystem] Entity %d: Playing animation %s (loop=%v)", entityID, animName, loop)
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

		// 渲染对话文本
		s.drawDialogueText(screen, dialogueComp.CurrentText, bubbleX, bubbleY)

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
func (s *DaveDialogueSystem) drawDialogueText(screen *ebiten.Image, textStr string, bubbleX, bubbleY float64) {
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

	// 渲染每一行
	for i, line := range lines {
		lineY := startY + float64(i)*config.DaveDialogueLineHeight

		// 计算水平居中
		lineWidth := s.measureTextWidth(line)
		lineX := textAreaX + (textAreaWidth-lineWidth)/2

		// 绘制文本（黑色）
		op := &text.DrawOptions{}
		op.GeoM.Translate(lineX, lineY)
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
