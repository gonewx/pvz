package systems

import (
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/decker502/pvz/pkg/components"
	"github.com/decker502/pvz/pkg/config"
	"github.com/decker502/pvz/pkg/ecs"
	"github.com/decker502/pvz/pkg/game"
	"github.com/hajimehoshi/ebiten/v2"
)

// newMockGameState 创建用于测试的 GameState
func newMockGameState() *game.GameState {
	gs := &game.GameState{}
	ls := &game.LawnStrings{}
	gs.LawnStrings = ls
	return gs
}

// newMockSystemResourceManager 创建用于测试的 ResourceManager
// 使用 nil audio context，因为测试不需要音频功能
func newMockSystemResourceManager() *game.ResourceManager {
	return game.NewResourceManager(nil)
}

// TestParseExpressionCommands 测试表情指令解析
func TestParseExpressionCommands(t *testing.T) {
	// 创建一个简单的系统实例用于测试
	sys := &DaveDialogueSystem{}
	sys.expressionRegex = compileExpressionRegex()

	tests := []struct {
		name         string
		input        string
		wantText     string
		wantCmdsLen  int
		wantFirstCmd string
	}{
		{
			name:         "单个表情指令",
			input:        "开始挖吧！{MOUTH_SMALL_OH}",
			wantText:     "开始挖吧！",
			wantCmdsLen:  1,
			wantFirstCmd: "MOUTH_SMALL_OH",
		},
		{
			name:         "多个表情指令",
			input:        "开始挖吧！{MOUTH_SMALL_OH} {SCREAM}",
			wantText:     "开始挖吧！",
			wantCmdsLen:  2,
			wantFirstCmd: "MOUTH_SMALL_OH",
		},
		{
			name:         "无表情指令",
			input:        "你好，我的邻居！",
			wantText:     "你好，我的邻居！",
			wantCmdsLen:  0,
			wantFirstCmd: "",
		},
		{
			name:         "中间有表情指令",
			input:        "因为我发~~~疯了！！！！！{MOUTH_BIG_SMILE} {SHAKE}",
			wantText:     "因为我发~~~疯了！！！！！",
			wantCmdsLen:  2,
			wantFirstCmd: "MOUTH_BIG_SMILE",
		},
		{
			name:         "空文本",
			input:        "",
			wantText:     "",
			wantCmdsLen:  0,
			wantFirstCmd: "",
		},
		{
			name:         "只有表情指令",
			input:        "{SCREAM}",
			wantText:     "",
			wantCmdsLen:  1,
			wantFirstCmd: "SCREAM",
		},
		{
			name:         "带空格的表情指令",
			input:        "现在出发！ {MOUTH_BIG_SMILE}  {SCREAM} ",
			wantText:     "现在出发！",
			wantCmdsLen:  2,
			wantFirstCmd: "MOUTH_BIG_SMILE",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotText, gotCmds := sys.parseExpressionCommands(tt.input)

			if gotText != tt.wantText {
				t.Errorf("parseExpressionCommands() text = %q, want %q", gotText, tt.wantText)
			}

			if len(gotCmds) != tt.wantCmdsLen {
				t.Errorf("parseExpressionCommands() cmds len = %d, want %d", len(gotCmds), tt.wantCmdsLen)
			}

			if tt.wantCmdsLen > 0 && len(gotCmds) > 0 {
				if gotCmds[0] != tt.wantFirstCmd {
					t.Errorf("parseExpressionCommands() first cmd = %q, want %q", gotCmds[0], tt.wantFirstCmd)
				}
			}
		})
	}
}

// TestParseExpressionCommands_AllExpressions 测试所有支持的表情指令
func TestParseExpressionCommands_AllExpressions(t *testing.T) {
	sys := &DaveDialogueSystem{}
	sys.expressionRegex = compileExpressionRegex()

	expressions := []string{
		"MOUTH_SMALL_OH",
		"MOUTH_SMALL_SMILE",
		"MOUTH_BIG_SMILE",
		"SHAKE",
		"SCREAM",
		"SHOW_WALLNUT", // 本 Story 不实现，但应该能解析
	}

	for _, expr := range expressions {
		input := "测试文本{" + expr + "}"
		_, cmds := sys.parseExpressionCommands(input)

		if len(cmds) != 1 {
			t.Errorf("Expected 1 command for %s, got %d", expr, len(cmds))
		}

		if len(cmds) > 0 && cmds[0] != expr {
			t.Errorf("Expected command %s, got %s", expr, cmds[0])
		}
	}
}

// compileExpressionRegex 辅助函数，编译表情指令正则表达式
func compileExpressionRegex() *regexp.Regexp {
	return regexp.MustCompile(`\{([A-Z_]+)\}`)
}

// TestDaveDialogueSystem_StateTransitions 测试状态机完整流程
func TestDaveDialogueSystem_StateTransitions(t *testing.T) {
	em := ecs.NewEntityManager()
	gs := newMockGameState()
	rm := newMockSystemResourceManager()

	sys := NewDaveDialogueSystem(em, gs, rm)
	if sys == nil {
		t.Fatal("Failed to create DaveDialogueSystem")
	}

	// 创建一个 Dave 实体（Entering 状态）
	entityID := em.CreateEntity()
	em.AddComponent(entityID, &components.PositionComponent{
		X: 0, // 位置由动画控制，初始值设为 0
		Y: config.DaveTargetY,
	})
	em.AddComponent(entityID, &components.DaveDialogueComponent{
		DialogueKeys:     []string{"CRAZY_DAVE_2400", "CRAZY_DAVE_2401"},
		CurrentLineIndex: 0,
		State:            components.DaveStateEntering,
		IsVisible:        false,
		BubbleOffsetX:    config.DaveBubbleOffsetX,
		BubbleOffsetY:    config.DaveBubbleOffsetY,
	})
	em.AddComponent(entityID, &components.ReanimComponent{
		CurrentAnimations: []string{"anim_enter"},
		IsLooping:         false,
		IsFinished:        false, // 动画尚未完成
	})

	// 测试 Entering → Talking
	t.Run("Entering to Talking transition", func(t *testing.T) {
		// 更新一次，状态应该保持 Entering（动画未完成）
		sys.Update(0.1)

		dialogueComp, _ := ecs.GetComponent[*components.DaveDialogueComponent](em, entityID)
		if dialogueComp.State != components.DaveStateEntering {
			t.Errorf("Expected state Entering before animation finishes, got %v", dialogueComp.State)
		}

		// 模拟动画完成
		reanimComp, _ := ecs.GetComponent[*components.ReanimComponent](em, entityID)
		reanimComp.IsFinished = true

		// 再次更新，应该触发状态转换
		sys.Update(0.1)

		dialogueComp, _ = ecs.GetComponent[*components.DaveDialogueComponent](em, entityID)

		// 验证状态转换
		if dialogueComp.State != components.DaveStateTalking {
			t.Errorf("Expected state Talking after animation finishes, got %v", dialogueComp.State)
		}

		// 验证对话可见
		if !dialogueComp.IsVisible {
			t.Error("Expected dialogue to be visible")
		}

		// 验证加载了第一条对话
		if dialogueComp.CurrentText == "" {
			t.Error("Expected current text to be loaded")
		}
	})
}

// TestDaveDialogueSystem_DialogueAdvance 测试对话推进逻辑
func TestDaveDialogueSystem_DialogueAdvance(t *testing.T) {
	em := ecs.NewEntityManager()
	gs := newMockGameState()
	rm := newMockSystemResourceManager()

	sys := NewDaveDialogueSystem(em, gs, rm)

	// 创建 Dave 实体（Talking 状态）
	entityID := em.CreateEntity()
	em.AddComponent(entityID, &components.PositionComponent{
		X: config.DaveTargetX,
		Y: config.DaveTargetY,
	})
	em.AddComponent(entityID, &components.DaveDialogueComponent{
		DialogueKeys:     []string{"TEST_1", "TEST_2", "TEST_3"},
		CurrentLineIndex: 0,
		State:            components.DaveStateTalking,
		IsVisible:        true,
		BubbleOffsetX:    config.DaveBubbleOffsetX,
		BubbleOffsetY:    config.DaveBubbleOffsetY,
	})
	em.AddComponent(entityID, &components.ReanimComponent{
		CurrentAnimations: []string{"anim_idle"},
	})

	t.Run("Advance to next dialogue", func(t *testing.T) {
		// 加载第一条对话
		sys.Update(0.016)

		dialogueComp, _ := ecs.GetComponent[*components.DaveDialogueComponent](em, entityID)

		// 推进到下一条
		sys.advanceDialogue(entityID, dialogueComp)

		if dialogueComp.CurrentLineIndex != 1 {
			t.Errorf("Expected CurrentLineIndex 1, got %d", dialogueComp.CurrentLineIndex)
		}

		if dialogueComp.State != components.DaveStateTalking {
			t.Errorf("Expected state Talking, got %v", dialogueComp.State)
		}
	})

	t.Run("Last dialogue triggers leaving", func(t *testing.T) {
		dialogueComp, _ := ecs.GetComponent[*components.DaveDialogueComponent](em, entityID)

		// 推进到最后一条
		dialogueComp.CurrentLineIndex = len(dialogueComp.DialogueKeys) - 1
		sys.advanceDialogue(entityID, dialogueComp)

		if dialogueComp.State != components.DaveStateLeaving {
			t.Errorf("Expected state Leaving, got %v", dialogueComp.State)
		}

		if dialogueComp.IsVisible {
			t.Error("Expected dialogue to be hidden when leaving")
		}
	})
}

// TestDaveDialogueSystem_LeavingToHidden 测试离场到隐藏状态
func TestDaveDialogueSystem_LeavingToHidden(t *testing.T) {
	em := ecs.NewEntityManager()
	gs := newMockGameState()
	rm := newMockSystemResourceManager()

	sys := NewDaveDialogueSystem(em, gs, rm)

	callbackCalled := false
	onComplete := func() {
		callbackCalled = true
	}

	// 创建 Dave 实体（Leaving 状态）
	entityID := em.CreateEntity()
	em.AddComponent(entityID, &components.PositionComponent{
		X: 0, // 位置由动画控制
		Y: config.DaveTargetY,
	})
	em.AddComponent(entityID, &components.DaveDialogueComponent{
		DialogueKeys:       []string{"TEST_1"},
		CurrentLineIndex:   0,
		State:              components.DaveStateLeaving,
		IsVisible:          false,
		OnCompleteCallback: onComplete,
	})
	em.AddComponent(entityID, &components.ReanimComponent{
		CurrentAnimations: []string{"anim_leave"},
		IsLooping:         false,
		IsFinished:        false, // 动画尚未完成
	})

	// 更新一次，状态应该保持 Leaving（动画未完成）
	sys.Update(0.1)

	dialogueComp, _ := ecs.GetComponent[*components.DaveDialogueComponent](em, entityID)
	if dialogueComp.State != components.DaveStateLeaving {
		t.Errorf("Expected state Leaving before animation finishes, got %v", dialogueComp.State)
	}

	// 模拟动画完成
	reanimComp, _ := ecs.GetComponent[*components.ReanimComponent](em, entityID)
	reanimComp.IsFinished = true

	// 再次更新，应该触发状态转换和实体销毁
	sys.Update(0.1)

	// DestroyEntity 只标记实体待销毁，需要调用 RemoveMarkedEntities 才真正删除
	em.RemoveMarkedEntities()

	// 验证实体被销毁
	_, exists := ecs.GetComponent[*components.DaveDialogueComponent](em, entityID)
	if exists {
		t.Error("Expected entity to be destroyed")
	}

	// 验证回调被调用
	if !callbackCalled {
		t.Error("Expected OnCompleteCallback to be called")
	}
}

// TestDaveDialogueSystem_ExpressionApplication 测试表情指令应用
func TestDaveDialogueSystem_ExpressionApplication(t *testing.T) {
	em := ecs.NewEntityManager()
	gs := newMockGameState()
	rm := newMockSystemResourceManager()

	sys := NewDaveDialogueSystem(em, gs, rm)

	// 创建 Dave 实体
	entityID := em.CreateEntity()
	em.AddComponent(entityID, &components.PositionComponent{
		X: config.DaveTargetX,
		Y: config.DaveTargetY,
	})
	em.AddComponent(entityID, &components.DaveDialogueComponent{
		CurrentExpressions: []string{"MOUTH_BIG_SMILE"},
	})

	// 创建带有嘴型图片的 ReanimComponent
	mouthImages := make(map[string]*ebiten.Image)
	for i := 1; i <= 6; i++ {
		key := fmt.Sprintf("IMAGE_REANIM_CRAZYDAVE_MOUTH%d", i)
		mouthImages[key] = ebiten.NewImage(10, 10)
	}

	em.AddComponent(entityID, &components.ReanimComponent{
		CurrentAnimations: []string{"anim_idle"},
		PartImages:        mouthImages,
		ImageOverrides:    make(map[string]*ebiten.Image),
	})

	tests := []struct {
		expression string
		wantMouth  string
	}{
		{"MOUTH_SMALL_OH", "IMAGE_REANIM_CRAZYDAVE_MOUTH6"},
		{"MOUTH_SMALL_SMILE", "IMAGE_REANIM_CRAZYDAVE_MOUTH2"},
		{"MOUTH_BIG_SMILE", "IMAGE_REANIM_CRAZYDAVE_MOUTH5"},
		{"SCREAM", "IMAGE_REANIM_CRAZYDAVE_MOUTH1"},
	}

	for _, tt := range tests {
		t.Run(tt.expression, func(t *testing.T) {
			dialogueComp, _ := ecs.GetComponent[*components.DaveDialogueComponent](em, entityID)
			dialogueComp.CurrentExpressions = []string{tt.expression}

			sys.applyExpressions(entityID, dialogueComp)

			if dialogueComp.Expression != tt.expression {
				t.Errorf("Expected expression %s, got %s", tt.expression, dialogueComp.Expression)
			}

			// 验证嘴型轨道已设置
			reanimComp, _ := ecs.GetComponent[*components.ReanimComponent](em, entityID)
			if len(reanimComp.ImageOverrides) == 0 {
				t.Errorf("Expected ImageOverrides to be set for %s", tt.expression)
			}
		})
	}
}

// TestDaveDialogueSystem_AnimationTriggers 测试动画触发
func TestDaveDialogueSystem_AnimationTriggers(t *testing.T) {
	em := ecs.NewEntityManager()
	gs := newMockGameState()
	rm := newMockSystemResourceManager()

	sys := NewDaveDialogueSystem(em, gs, rm)

	entityID := em.CreateEntity()
	em.AddComponent(entityID, &components.PositionComponent{
		X: config.DaveTargetX,
		Y: config.DaveTargetY,
	})
	em.AddComponent(entityID, &components.DaveDialogueComponent{
		CurrentExpressions: []string{"SHAKE"},
	})
	em.AddComponent(entityID, &components.ReanimComponent{
		CurrentAnimations: []string{"anim_idle"},
	})

	tests := []struct {
		expression   string
		expectedAnim string
	}{
		{"SHAKE", "anim_crazy"},
		{"SCREAM", "anim_crazy"},
	}

	for _, tt := range tests {
		t.Run(tt.expression, func(t *testing.T) {
			dialogueComp, _ := ecs.GetComponent[*components.DaveDialogueComponent](em, entityID)
			dialogueComp.CurrentExpressions = []string{tt.expression}

			sys.applyExpressions(entityID, dialogueComp)

			// 验证 AnimationCommandComponent 被添加
			animCmd, hasAnimCmd := ecs.GetComponent[*components.AnimationCommandComponent](em, entityID)
			if !hasAnimCmd {
				t.Errorf("Expected AnimationCommandComponent to be added for %s", tt.expression)
			}

			if animCmd.ComboName != tt.expectedAnim {
				t.Errorf("Expected animation %s, got %s", tt.expectedAnim, animCmd.ComboName)
			}
		})
	}
}

// TestDaveDialogueSystem_LoadCurrentDialogue 测试对话加载
func TestDaveDialogueSystem_LoadCurrentDialogue(t *testing.T) {
	em := ecs.NewEntityManager()
	gs := newMockGameState()
	rm := newMockSystemResourceManager()

	sys := NewDaveDialogueSystem(em, gs, rm)

	dialogueComp := &components.DaveDialogueComponent{
		DialogueKeys:     []string{"CRAZY_DAVE_2406"}, // 包含表情指令的对话
		CurrentLineIndex: 0,
	}

	sys.loadCurrentDialogue(dialogueComp)

	// 验证文本被加载（即使是默认值 [key] 格式）
	if dialogueComp.CurrentText == "" {
		t.Error("Expected CurrentText to be loaded")
	}

	// 验证文本中表情指令被移除（如果有的话）
	if strings.Contains(dialogueComp.CurrentText, "{") && strings.Contains(dialogueComp.CurrentText, "}") {
		// 如果文本包含 {} 格式的内容，验证它不是表情指令格式
		matched, _ := regexp.MatchString(`\{[A-Z_]+\}`, dialogueComp.CurrentText)
		if matched {
			t.Error("Expected expression commands to be removed from text")
		}
	}
}

// TestDaveDialogueSystem_HiddenStateDoesNothing 测试 Hidden 状态不执行任何操作
func TestDaveDialogueSystem_HiddenStateDoesNothing(t *testing.T) {
	em := ecs.NewEntityManager()
	gs := newMockGameState()
	rm := newMockSystemResourceManager()

	sys := NewDaveDialogueSystem(em, gs, rm)

	// 创建处于 Hidden 状态的 Dave 实体
	entityID := em.CreateEntity()
	initialX := 100.0
	em.AddComponent(entityID, &components.PositionComponent{
		X: initialX,
		Y: config.DaveTargetY,
	})
	em.AddComponent(entityID, &components.DaveDialogueComponent{
		DialogueKeys: []string{"TEST_1"},
		State:        components.DaveStateHidden,
		IsVisible:    false,
	})
	em.AddComponent(entityID, &components.ReanimComponent{
		CurrentAnimations: []string{},
	})

	// 更新系统
	sys.Update(0.1)

	// 验证位置没有改变
	posComp, _ := ecs.GetComponent[*components.PositionComponent](em, entityID)
	if posComp.X != initialX {
		t.Errorf("Expected X position to remain %f, got %f", initialX, posComp.X)
	}

	// 验证实体仍然存在
	_, exists := ecs.GetComponent[*components.DaveDialogueComponent](em, entityID)
	if !exists {
		t.Error("Expected entity to still exist in Hidden state")
	}
}

// TestDaveDialogueSystem_NilCallbackSafe 测试空回调安全性
func TestDaveDialogueSystem_NilCallbackSafe(t *testing.T) {
	em := ecs.NewEntityManager()
	gs := newMockGameState()
	rm := newMockSystemResourceManager()

	sys := NewDaveDialogueSystem(em, gs, rm)

	// 创建没有回调的 Dave 实体（Leaving 状态）
	entityID := em.CreateEntity()
	em.AddComponent(entityID, &components.PositionComponent{
		X: 0, // 位置由动画控制
		Y: config.DaveTargetY,
	})
	em.AddComponent(entityID, &components.DaveDialogueComponent{
		DialogueKeys:       []string{"TEST_1"},
		State:              components.DaveStateLeaving,
		OnCompleteCallback: nil, // 无回调
	})
	em.AddComponent(entityID, &components.ReanimComponent{
		CurrentAnimations: []string{"anim_leave"},
		IsLooping:         false,
		IsFinished:        false, // 动画尚未完成
	})

	// 更新系统让 Dave 移动出屏幕（不应该 panic）
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Nil callback caused panic: %v", r)
		}
	}()

	// 更新一次，动画未完成
	sys.Update(0.1)

	// 模拟动画完成
	reanimComp, _ := ecs.GetComponent[*components.ReanimComponent](em, entityID)
	reanimComp.IsFinished = true

	// 再次更新，应该触发状态转换和实体销毁（无 panic）
	sys.Update(0.1)

	// DestroyEntity 只标记实体待销毁，需要调用 RemoveMarkedEntities 才真正删除
	em.RemoveMarkedEntities()

	// 验证实体被销毁
	_, exists := ecs.GetComponent[*components.DaveDialogueComponent](em, entityID)
	if exists {
		t.Error("Expected entity to be destroyed")
	}
}

// TestDaveDialogueSystem_EmptyDialogueKeys 测试空对话键列表
func TestDaveDialogueSystem_EmptyDialogueKeys(t *testing.T) {
	em := ecs.NewEntityManager()
	gs := newMockGameState()
	rm := newMockSystemResourceManager()

	sys := NewDaveDialogueSystem(em, gs, rm)

	// 创建对话键为空的组件
	dialogueComp := &components.DaveDialogueComponent{
		DialogueKeys:     []string{},
		CurrentLineIndex: 0,
	}

	// 尝试加载对话（不应该 panic）
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Empty dialogue keys caused panic: %v", r)
		}
	}()

	sys.loadCurrentDialogue(dialogueComp)

	// 验证文本仍为空
	if dialogueComp.CurrentText != "" {
		t.Errorf("Expected CurrentText to remain empty, got %q", dialogueComp.CurrentText)
	}
}

// TestDaveDialogueSystem_OutOfBoundsDialogueIndex 测试超出范围的对话索引
func TestDaveDialogueSystem_OutOfBoundsDialogueIndex(t *testing.T) {
	em := ecs.NewEntityManager()
	gs := newMockGameState()
	rm := newMockSystemResourceManager()

	sys := NewDaveDialogueSystem(em, gs, rm)

	// 创建索引超出范围的组件
	dialogueComp := &components.DaveDialogueComponent{
		DialogueKeys:     []string{"TEST_1"},
		CurrentLineIndex: 5, // 超出范围
	}

	// 尝试加载对话（不应该 panic）
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Out of bounds index caused panic: %v", r)
		}
	}()

	sys.loadCurrentDialogue(dialogueComp)
}

// TestDaveDialogueSystem_UnknownExpression 测试未知表情指令
func TestDaveDialogueSystem_UnknownExpression(t *testing.T) {
	em := ecs.NewEntityManager()
	gs := newMockGameState()
	rm := newMockSystemResourceManager()

	sys := NewDaveDialogueSystem(em, gs, rm)

	entityID := em.CreateEntity()
	em.AddComponent(entityID, &components.PositionComponent{
		X: config.DaveTargetX,
		Y: config.DaveTargetY,
	})
	em.AddComponent(entityID, &components.DaveDialogueComponent{
		CurrentExpressions: []string{"UNKNOWN_EXPRESSION"},
	})
	em.AddComponent(entityID, &components.ReanimComponent{
		CurrentAnimations: []string{"anim_idle"},
	})

	// 应用未知表情（不应该 panic）
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Unknown expression caused panic: %v", r)
		}
	}()

	dialogueComp, _ := ecs.GetComponent[*components.DaveDialogueComponent](em, entityID)
	sys.applyExpressions(entityID, dialogueComp)

	// 验证表情未被设置（因为是未知的）
	if dialogueComp.Expression == "UNKNOWN_EXPRESSION" {
		t.Error("Unknown expression should not be applied")
	}
}

// TestDaveDialogueSystem_WrapText 测试文本自动换行
func TestDaveDialogueSystem_WrapText(t *testing.T) {
	em := ecs.NewEntityManager()
	gs := newMockGameState()
	rm := newMockSystemResourceManager()

	sys := NewDaveDialogueSystem(em, gs, rm)

	tests := []struct {
		name     string
		text     string
		maxWidth float64
	}{
		{
			name:     "短文本",
			text:     "你好",
			maxWidth: 100.0,
		},
		{
			name:     "长文本",
			text:     "这是一段很长的文本，需要自动换行显示",
			maxWidth: 100.0,
		},
		{
			name:     "空文本",
			text:     "",
			maxWidth: 100.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lines := sys.wrapText(tt.text, tt.maxWidth)

			// 验证不会 panic 并返回合理结果
			if tt.text == "" {
				// 空文本时，wrapText 可能返回空切片或单元素切片（取决于实现）
				// 只验证不会 panic
				_ = lines
			} else {
				if len(lines) == 0 {
					t.Error("Expected at least one line for non-empty text")
				}
			}
		})
	}
}

// TestDaveDialogueSystem_MeasureTextWidth 测试文本宽度测量
func TestDaveDialogueSystem_MeasureTextWidth(t *testing.T) {
	em := ecs.NewEntityManager()
	gs := newMockGameState()
	rm := newMockSystemResourceManager()

	sys := NewDaveDialogueSystem(em, gs, rm)

	// 测试空文本
	width := sys.measureTextWidth("")
	if width != 0 {
		t.Errorf("Expected 0 width for empty text, got %f", width)
	}

	// 测试非空文本（如果字体加载成功）
	// 注意：在测试环境中字体可能未加载，所以这里只验证不会 panic
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("measureTextWidth caused panic: %v", r)
		}
	}()

	_ = sys.measureTextWidth("测试文本")
}

// TestDaveDialogueSystem_PlayAnimation 测试动画播放
func TestDaveDialogueSystem_PlayAnimation(t *testing.T) {
	em := ecs.NewEntityManager()
	gs := newMockGameState()
	rm := newMockSystemResourceManager()

	sys := NewDaveDialogueSystem(em, gs, rm)

	entityID := em.CreateEntity()
	em.AddComponent(entityID, &components.PositionComponent{
		X: config.DaveTargetX,
		Y: config.DaveTargetY,
	})
	em.AddComponent(entityID, &components.ReanimComponent{
		CurrentAnimations: []string{"anim_idle"},
	})

	tests := []struct {
		name     string
		animName string
		loop     bool
	}{
		{"播放入场动画", "anim_enter", false},
		{"播放空闲动画（循环）", "anim_idle", true},
		{"播放疯狂动画", "anim_crazy", false},
		{"播放离场动画", "anim_leave", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sys.playAnimation(entityID, tt.animName, tt.loop)

			// 验证 AnimationCommandComponent 被添加
			animCmd, hasAnimCmd := ecs.GetComponent[*components.AnimationCommandComponent](em, entityID)
			if !hasAnimCmd {
				t.Error("Expected AnimationCommandComponent to be added")
			}

			if animCmd.ComboName != tt.animName {
				t.Errorf("Expected animation %s, got %s", tt.animName, animCmd.ComboName)
			}

			// 验证循环设置
			reanimComp, _ := ecs.GetComponent[*components.ReanimComponent](em, entityID)
			if reanimComp.IsLooping != tt.loop {
				t.Errorf("Expected IsLooping=%v, got %v", tt.loop, reanimComp.IsLooping)
			}
		})
	}
}

// TestDaveDialogueSystem_SetMouthTrack 测试嘴型轨道设置
func TestDaveDialogueSystem_SetMouthTrack(t *testing.T) {
	em := ecs.NewEntityManager()
	gs := newMockGameState()
	rm := newMockSystemResourceManager()

	sys := NewDaveDialogueSystem(em, gs, rm)

	entityID := em.CreateEntity()

	// 创建带有嘴型图片的 ReanimComponent
	mouthImages := make(map[string]*ebiten.Image)
	for i := 1; i <= 6; i++ {
		key := fmt.Sprintf("IMAGE_REANIM_CRAZYDAVE_MOUTH%d", i)
		mouthImages[key] = ebiten.NewImage(10, 10)
	}

	em.AddComponent(entityID, &components.ReanimComponent{
		CurrentAnimations: []string{"anim_idle"},
		PartImages:        mouthImages,
		ImageOverrides:    make(map[string]*ebiten.Image),
	})

	// 设置嘴型轨道
	sys.setMouthTrack(entityID, "IMAGE_REANIM_CRAZYDAVE_MOUTH5")

	// 验证 ImageOverrides 被设置
	reanimComp, _ := ecs.GetComponent[*components.ReanimComponent](em, entityID)
	if len(reanimComp.ImageOverrides) == 0 {
		t.Error("Expected ImageOverrides to be set")
	}
}

// TestDaveDialogueSystem_SetMouthTrackMissingImage 测试缺失嘴型图片的情况
func TestDaveDialogueSystem_SetMouthTrackMissingImage(t *testing.T) {
	em := ecs.NewEntityManager()
	gs := newMockGameState()
	rm := newMockSystemResourceManager()

	sys := NewDaveDialogueSystem(em, gs, rm)

	entityID := em.CreateEntity()
	em.AddComponent(entityID, &components.ReanimComponent{
		CurrentAnimations: []string{"anim_idle"},
		PartImages:        make(map[string]*ebiten.Image), // 空的图片映射
		ImageOverrides:    make(map[string]*ebiten.Image),
	})

	// 尝试设置不存在的嘴型（不应该 panic）
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Missing mouth image caused panic: %v", r)
		}
	}()

	sys.setMouthTrack(entityID, "IMAGE_REANIM_CRAZYDAVE_MOUTH5")
}

// TestDaveDialogueSystem_SetMouthTrackNoReanimComponent 测试没有 ReanimComponent 的情况
func TestDaveDialogueSystem_SetMouthTrackNoReanimComponent(t *testing.T) {
	em := ecs.NewEntityManager()
	gs := newMockGameState()
	rm := newMockSystemResourceManager()

	sys := NewDaveDialogueSystem(em, gs, rm)

	entityID := em.CreateEntity()
	em.AddComponent(entityID, &components.PositionComponent{
		X: 0,
		Y: 0,
	})
	// 不添加 ReanimComponent

	// 尝试设置嘴型（不应该 panic）
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Missing ReanimComponent caused panic: %v", r)
		}
	}()

	sys.setMouthTrack(entityID, "IMAGE_REANIM_CRAZYDAVE_MOUTH5")
}

// TestDaveDialogueSystem_CompleteStateMachineFlow 测试完整状态机流程
func TestDaveDialogueSystem_CompleteStateMachineFlow(t *testing.T) {
	em := ecs.NewEntityManager()
	gs := newMockGameState()
	rm := newMockSystemResourceManager()

	sys := NewDaveDialogueSystem(em, gs, rm)

	callbackCalled := false
	onComplete := func() {
		callbackCalled = true
	}

	// 创建 Dave 实体（Entering 状态）
	entityID := em.CreateEntity()
	em.AddComponent(entityID, &components.PositionComponent{
		X: 0, // 位置由动画控制
		Y: config.DaveTargetY,
	})
	em.AddComponent(entityID, &components.DaveDialogueComponent{
		DialogueKeys:       []string{"TEST_1", "TEST_2"},
		CurrentLineIndex:   0,
		State:              components.DaveStateEntering,
		IsVisible:          false,
		BubbleOffsetX:      config.DaveBubbleOffsetX,
		BubbleOffsetY:      config.DaveBubbleOffsetY,
		OnCompleteCallback: onComplete,
	})
	em.AddComponent(entityID, &components.ReanimComponent{
		CurrentAnimations: []string{"anim_enter"},
		IsLooping:         false,
		IsFinished:        false, // 动画尚未完成
	})

	// 阶段 1: Entering → Talking
	// 更新一次，动画未完成，状态保持 Entering
	sys.Update(0.1)

	dialogueComp, exists := ecs.GetComponent[*components.DaveDialogueComponent](em, entityID)
	if !exists {
		t.Fatal("Entity should exist")
	}
	if dialogueComp.State != components.DaveStateEntering {
		t.Errorf("Expected state Entering before animation finishes, got %v", dialogueComp.State)
	}

	// 模拟入场动画完成
	reanimComp, _ := ecs.GetComponent[*components.ReanimComponent](em, entityID)
	reanimComp.IsFinished = true

	// 再次更新，触发状态转换
	sys.Update(0.1)

	dialogueComp, exists = ecs.GetComponent[*components.DaveDialogueComponent](em, entityID)
	if !exists {
		t.Fatal("Entity should exist after entering")
	}
	if dialogueComp.State != components.DaveStateTalking {
		t.Errorf("Expected state Talking after entering, got %v", dialogueComp.State)
	}

	// 阶段 2: Talking - 推进对话
	sys.advanceDialogue(entityID, dialogueComp)
	if dialogueComp.State != components.DaveStateTalking {
		t.Errorf("Expected state Talking after first advance, got %v", dialogueComp.State)
	}
	if dialogueComp.CurrentLineIndex != 1 {
		t.Errorf("Expected CurrentLineIndex 1, got %d", dialogueComp.CurrentLineIndex)
	}

	// 阶段 3: Talking → Leaving
	sys.advanceDialogue(entityID, dialogueComp)
	if dialogueComp.State != components.DaveStateLeaving {
		t.Errorf("Expected state Leaving after last dialogue, got %v", dialogueComp.State)
	}

	// 阶段 4: Leaving → Hidden (实体销毁)
	// 重置 IsFinished（playAnimation 切换到 anim_leave 后需要等动画完成）
	reanimComp, _ = ecs.GetComponent[*components.ReanimComponent](em, entityID)
	reanimComp.IsFinished = false

	// 更新一次，动画未完成
	sys.Update(0.1)

	dialogueComp, _ = ecs.GetComponent[*components.DaveDialogueComponent](em, entityID)
	if dialogueComp.State != components.DaveStateLeaving {
		t.Errorf("Expected state Leaving before anim_leave finishes, got %v", dialogueComp.State)
	}

	// 模拟离场动画完成
	reanimComp.IsFinished = true

	// 再次更新，触发实体销毁
	sys.Update(0.1)

	// DestroyEntity 只标记实体待销毁，需要调用 RemoveMarkedEntities 才真正删除
	em.RemoveMarkedEntities()

	_, exists = ecs.GetComponent[*components.DaveDialogueComponent](em, entityID)
	if exists {
		t.Error("Expected entity to be destroyed after leaving")
	}

	if !callbackCalled {
		t.Error("Expected OnCompleteCallback to be called")
	}
}
