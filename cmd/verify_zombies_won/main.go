package main

import (
	"flag"
	"fmt"
	"image/color"
	"log"
	"os"

	"github.com/gonewx/pvz/pkg/components"
	"github.com/gonewx/pvz/pkg/config"
	"github.com/gonewx/pvz/pkg/ecs"
	"github.com/gonewx/pvz/pkg/entities"
	"github.com/gonewx/pvz/pkg/game"
	"github.com/gonewx/pvz/pkg/systems"
	"github.com/gonewx/pvz/pkg/systems/behavior"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

const (
	screenWidth  = 800
	screenHeight = 600
)

var (
	// 命令行参数
	skipPhase1 = flag.Bool("skip-phase1", false, "跳过 Phase 1（游戏冻结阶段）")
	skipPhase2 = flag.Bool("skip-phase2", false, "跳过 Phase 2（僵尸入侵阶段）")
	skipPhase3 = flag.Bool("skip-phase3", false, "跳过 Phase 3（惨叫动画阶段）")
	fastMode   = flag.Bool("fast", false, "快速模式（缩短所有阶段时间）")
	verbose    = flag.Bool("verbose", false, "显示详细调试信息")
)

// VerifyZombiesWonGame 僵尸获胜流程验证游戏
// 包含四个阶段：游戏冻结、僵尸入侵、惨叫动画、游戏结束对话框
type VerifyZombiesWonGame struct {
	entityManager     *ecs.EntityManager
	gameState         *game.GameState
	resourceManager   *game.ResourceManager
	reanimSystem      *systems.ReanimSystem
	renderSystem      *systems.RenderSystem
	behaviorSystem    *behavior.BehaviorSystem // 行为系统（处理僵尸移动）
	zombiesWonSystem  *systems.ZombiesWonPhaseSystem
	dialogSystem      *systems.DialogRenderSystem // 对话框渲染系统
	dialogInputSystem *systems.DialogInputSystem  // 对话框输入系统

	debugFont *text.GoTextFace // 中文调试字体

	// 测试实体
	zombieID       ecs.EntityID // 触发失败的僵尸
	plantID        ecs.EntityID // 测试用植物（验证冻结效果）
	bulletID       ecs.EntityID // 测试用子弹（验证冻结时消失）
	flowID         ecs.EntityID // 流程控制实体
	zombiesWonAnim ecs.EntityID // ZombiesWon 动画实体
	dialogID       ecs.EntityID // 游戏结束对话框实体

	triggered bool // 是否已触发流程
	completed bool // 是否已完成所有阶段
}

// NewVerifyZombiesWonGame 创建验证游戏实例
func NewVerifyZombiesWonGame() (*VerifyZombiesWonGame, error) {
	// 创建 ECS 管理器
	em := ecs.NewEntityManager()

	// 创建音频上下文
	audioContext := audio.NewContext(48000)

	// 创建资源管理器
	rm := game.NewResourceManager(audioContext)

	// 加载资源配置
	if err := rm.LoadResourceConfig("assets/config/resources.yaml"); err != nil {
		return nil, fmt.Errorf("failed to load resource config: %w", err)
	}

	// 加载所有资源组
	log.Println("Loading all resources...")
	if err := rm.LoadAllResources(); err != nil {
		log.Fatal("Failed to load resources:", err)
	}

	// 加载 Reanim 资源
	log.Println("Loading Reanim resources...")
	if err := rm.LoadReanimResources(); err != nil {
		log.Fatal("Failed to load Reanim resources:", err)
	}

	// 加载 Reanim 配置
	log.Println("Loading Reanim config...")
	reanimConfigManager, err := config.NewReanimConfigManager("data/reanim_config")
	if err != nil {
		log.Fatal("Failed to load Reanim config:", err)
	}

	// 获取游戏状态单例
	gs := game.GetGameState()
	gs.CameraX = config.GameCameraX // 设置摄像机位置

	// 创建并设置音频管理器（Story 10.9 统一音效管理）
	audioManager := game.NewAudioManager(rm, nil)
	gs.SetAudioManager(audioManager)

	// 创建系统
	reanimSystem := systems.NewReanimSystem(em)
	reanimSystem.SetConfigManager(reanimConfigManager)
	renderSystem := systems.NewRenderSystem(em)
	renderSystem.SetReanimSystem(reanimSystem) // 设置 ReanimSystem 引用以支持 Reanim 动画渲染
	renderSystem.SetResourceManager(rm)        // 设置 ResourceManager 引用以支持房门渲染 (Story 8.8 - Task 6)

	// 创建行为系统（用于处理僵尸移动）
	// 注意：验证程序不需要 LawnmowerSystem 和 LawnGridSystem，传入 nil 即可
	// 因为在 Freeze 状态下，BehaviorSystem 只处理触发僵尸的移动，不会访问这些依赖
	behaviorSystem := behavior.NewBehaviorSystem(em, rm, gs, nil, 0, reanimSystem)

	// 加载中文调试字体
	debugFont, err := rm.LoadFont("assets/fonts/SimHei.ttf", 16)
	if err != nil {
		log.Printf("Warning: Failed to load debug font: %v", err)
		debugFont = nil
	}

	// 加载对话框字体
	titleFont, err := rm.LoadFont("assets/fonts/SimHei.ttf", 24)
	if err != nil {
		log.Printf("Warning: Failed to load title font: %v", err)
		titleFont = debugFont
	}

	messageFont, err := rm.LoadFont("assets/fonts/SimHei.ttf", 18)
	if err != nil {
		log.Printf("Warning: Failed to load message font: %v", err)
		messageFont = debugFont
	}

	buttonFont, err := rm.LoadFont("assets/fonts/SimHei.ttf", 20)
	if err != nil {
		log.Printf("Warning: Failed to load button font: %v", err)
		buttonFont = debugFont
	}

	// 创建对话框和输入系统
	dialogSystem := systems.NewDialogRenderSystem(em, screenWidth, screenHeight, titleFont, messageFont, buttonFont)
	dialogInputSystem := systems.NewDialogInputSystem(em)

	log.Println("╔════════════════════════════════════════════════════════╗")
	log.Println("║      僵尸获胜流程验证程序 (Story 8.8)                 ║")
	log.Println("╚════════════════════════════════════════════════════════╝")
	log.Println()
	log.Println("【验证流程】")
	log.Println("  Phase 1: 游戏冻结 (1.5s)")
	log.Println("    - 植物停止攻击动画")
	log.Println("    - 子弹消失")
	log.Println("    - UI 元素隐藏")
	log.Println("    - 背景音乐淡出")
	log.Println("  Phase 2: 僵尸入侵 (动态)")
	log.Println("    - 僵尸继续行走至屏幕外 (X < -100)")
	log.Println("    - 摄像机平滑左移至世界坐标 0")
	log.Println("  Phase 3: 惨叫与动画 (3-4s)")
	log.Println("    - 播放惨叫音效 (scream.ogg)")
	log.Println("    - 延迟 0.5s 播放咀嚼音效 (chomp_soft.ogg)")
	log.Println("    - 显示 ZombiesWon 动画")
	log.Println("    - 屏幕轻微抖动 (±5 像素, 10Hz)")
	log.Println("  Phase 4: 游戏结束对话框 (等待点击或 3-5s 超时)")
	log.Println("    - 显示游戏结束对话框")
	log.Println("    - 按钮：\"再次尝试\" / \"返回主菜单\"")
	log.Println()
	log.Println("【快捷键】")
	log.Println("  Space - 启动僵尸获胜流程")
	log.Println("  1-4   - 跳转到指定阶段")
	log.Println("  R     - 重启验证")
	log.Println("  Q     - 退出程序")
	log.Println("════════════════════════════════════════════════════════")

	game := &VerifyZombiesWonGame{
		entityManager:     em,
		gameState:         gs,
		resourceManager:   rm,
		reanimSystem:      reanimSystem,
		renderSystem:      renderSystem,
		behaviorSystem:    behaviorSystem,
		dialogSystem:      dialogSystem,
		dialogInputSystem: dialogInputSystem,
		debugFont:         debugFont,
		triggered:         false,
		completed:         false,
	}

	// 创建测试场景
	game.setupTestScene()

	// 【注释掉启动时自动触发，改为当僵尸到达边界时自动触发】
	// log.Println("[VerifyZombiesWon] 自动触发僵尸获胜流程...")
	// game.triggerFlow()
	log.Println("[VerifyZombiesWon] 等待僵尸到达失败边界 (X < 250) 自动触发流程...")
	log.Println("[VerifyZombiesWon] 或按 Space 键手动触发流程")

	return game, nil
}

// setupTestScene 创建测试场景
func (vg *VerifyZombiesWonGame) setupTestScene() {
	// 创建测试用僵尸（从屏幕右侧开始）
	var err error
	vg.zombieID, err = entities.NewZombieEntity(vg.entityManager, vg.resourceManager, 0, 300.0)
	if err != nil {
		log.Printf("Warning: Failed to create zombie entity: %v", err)
		// 创建简化版僵尸
		vg.zombieID = vg.entityManager.CreateEntity()
		ecs.AddComponent(vg.entityManager, vg.zombieID, &components.PositionComponent{
			X: 150.0,
			Y: 300.0,
		})
	}

	// 设置僵尸向左移动（激活状态）
	vel, ok := ecs.GetComponent[*components.VelocityComponent](vg.entityManager, vg.zombieID)
	if ok {
		vel.VX = -30.0 // 标准僵尸移动速度
	}

	// 播放行走动画
	ecs.AddComponent(vg.entityManager, vg.zombieID, &components.AnimationCommandComponent{
		UnitID:    "zombie",
		ComboName: "walk",
		Processed: false,
	})

	// 创建测试用植物（验证冻结时停止动画）
	vg.plantID = vg.entityManager.CreateEntity()
	ecs.AddComponent(vg.entityManager, vg.plantID, &components.PositionComponent{
		X: 400.0,
		Y: 300.0,
	})
	ecs.AddComponent(vg.entityManager, vg.plantID, &components.BehaviorComponent{
		Type: components.BehaviorPeashooter,
	})

	// 创建测试用子弹（验证冻结时消失）
	vg.bulletID = vg.entityManager.CreateEntity()
	ecs.AddComponent(vg.entityManager, vg.bulletID, &components.PositionComponent{
		X: 300.0,
		Y: 300.0,
	})
	ecs.AddComponent(vg.entityManager, vg.bulletID, &components.VelocityComponent{
		VX: 300.0, // 子弹向右移动
		VY: 0.0,
	})
	ecs.AddComponent(vg.entityManager, vg.bulletID, &components.BehaviorComponent{
		Type: components.BehaviorPeaProjectile,
	})

	log.Println("[VerifyZombiesWon] 测试场景创建完成")
	log.Printf("  - 僵尸 ID: %d (位置: X=600, VX=-150)", vg.zombieID)
	log.Printf("  - 植物 ID: %d (位置: X=400)", vg.plantID)
	log.Printf("  - 子弹 ID: %d (位置: X=300)", vg.bulletID)
}

// triggerFlow 触发僵尸获胜流程
func (vg *VerifyZombiesWonGame) triggerFlow() {
	if vg.triggered {
		log.Println("[VerifyZombiesWon] 流程已触发，跳过")
		return
	}

	log.Println("[VerifyZombiesWon] 🚀 触发僵尸获胜流程")

	// 创建 ZombiesWonPhaseSystem（如果尚未创建）
	if vg.zombiesWonSystem == nil {
		vg.zombiesWonSystem = systems.NewZombiesWonPhaseSystem(
			vg.entityManager,
			vg.resourceManager,
			vg.gameState,
			screenWidth,
			screenHeight,
		)
		// 设置"再次尝试"回调
		vg.zombiesWonSystem.SetRetryCallback(func() {
			vg.onRetryClicked()
		})
	}

	// 使用业务逻辑接口启动流程
	vg.flowID = systems.StartZombiesWonFlow(vg.entityManager, vg.zombieID)

	// 确定初始阶段（根据跳过参数）
	initialPhase := 1
	if *skipPhase1 && *skipPhase2 && *skipPhase3 {
		initialPhase = 4
		log.Println("[VerifyZombiesWon] ⏭️  跳过所有阶段，直接进入 Phase 4")
	} else if *skipPhase1 && *skipPhase2 {
		initialPhase = 3
		log.Println("[VerifyZombiesWon] ⏭️  跳过 Phase 1和2，从 Phase 3 开始")
	} else if *skipPhase1 {
		initialPhase = 2
		log.Println("[VerifyZombiesWon] ⏭️  跳过 Phase 1，从 Phase 2 开始")
	}

	// 如果需要跳过阶段，修改组件状态
	if initialPhase > 1 {
		if phaseComp, ok := ecs.GetComponent[*components.ZombiesWonPhaseComponent](vg.entityManager, vg.flowID); ok {
			phaseComp.CurrentPhase = initialPhase
			// 补充设置 InitialCameraX，防止直接跳转导致数据缺失
			phaseComp.InitialCameraX = vg.gameState.CameraX
		}
	}

	vg.triggered = true
	log.Printf("[VerifyZombiesWon] 流程已启动，当前阶段: Phase %d", initialPhase)
}

// onRetryClicked "再次尝试"按钮点击回调
func (vg *VerifyZombiesWonGame) onRetryClicked() {
	log.Println("[VerifyZombiesWon] 🔄 点击了\"再次尝试\"按钮")
	log.Println("  - 模拟重新加载关卡...")
	log.Println("  - 验证程序将重启")
	log.Println()
	log.Println("╔════════════════════════════════════════════════════════╗")
	log.Println("║           ✅ 僵尸获胜流程验证完成！                   ║")
	log.Println("╚════════════════════════════════════════════════════════╝")
	log.Println()
	log.Println("【验证成果】")
	log.Println("  ✅ Phase 1: 游戏冻结 (完成)")
	log.Println("  ✅ Phase 2: 僵尸入侵 (完成)")
	log.Println("  ✅ Phase 3: 惨叫动画 (完成)")
	log.Println("  ✅ Phase 4: 游戏结束对话框 (完成)")
	log.Println("  ✅ 按钮交互: 鼠标悬停、按下、点击 (完成)")
	log.Println()
	log.Println("【提示】")
	log.Println("  - 按 R 重启验证 | 按 Q 退出")
	log.Println("════════════════════════════════════════════════════════")

	// 清空对话框
	if vg.dialogID != 0 {
		vg.entityManager.DestroyEntity(vg.dialogID)
		vg.dialogID = 0
	}
	vg.entityManager.RemoveMarkedEntities()

	// 标记完成
	vg.completed = true

	// 重启验证程序
	vg.reset()
}

// Update 更新游戏逻辑
func (vg *VerifyZombiesWonGame) Update() error {
	// 快捷键：Space 键触发流程
	if inpututil.IsKeyJustPressed(ebiten.KeySpace) {
		vg.triggerFlow()
	}

	// 快捷键：1-4 跳转到指定阶段
	if inpututil.IsKeyJustPressed(ebiten.Key1) {
		vg.jumpToPhase(1)
	}
	if inpututil.IsKeyJustPressed(ebiten.Key2) {
		vg.jumpToPhase(2)
	}
	if inpututil.IsKeyJustPressed(ebiten.Key3) {
		vg.jumpToPhase(3)
	}
	if inpututil.IsKeyJustPressed(ebiten.Key4) {
		vg.jumpToPhase(4)
	}

	// 快捷键：R 键重启
	if inpututil.IsKeyJustPressed(ebiten.KeyR) {
		log.Println("[VerifyZombiesWon] 🔄 重启验证")
		vg.reset()
		return nil
	}

	// 快捷键：Q 键退出
	if inpututil.IsKeyJustPressed(ebiten.KeyQ) {
		log.Println("[VerifyZombiesWon] 👋 退出验证程序")
		return fmt.Errorf("quit")
	}

	// 更新系统
	dt := 1.0 / 60.0

	// 如果开启快速模式，加速时间流逝
	if *fastMode {
		dt *= 3.0
	}

	// 【新增】自动检测僵尸到达失败边界
	if !vg.triggered {
		pos, ok := ecs.GetComponent[*components.PositionComponent](vg.entityManager, vg.zombieID)
		if ok && pos.X < systems.DefeatBoundaryX {
			log.Printf("[VerifyZombiesWon] ⚠️  僵尸到达失败边界 (X=%.2f < %.2f)，自动触发僵尸获胜流程", pos.X, systems.DefeatBoundaryX)
			vg.triggerFlow()
		}
	}

	// 【新增】流程触发前，僵尸继续向左移动
	if !vg.triggered {
		pos, ok := ecs.GetComponent[*components.PositionComponent](vg.entityManager, vg.zombieID)
		if ok {
			vel, ok := ecs.GetComponent[*components.VelocityComponent](vg.entityManager, vg.zombieID)
			if ok {
				pos.X += vel.VX * dt
				pos.Y += vel.VY * dt
			}
		}
	}

	// 更新 Reanim 系统（处理动画播放）
	vg.reanimSystem.Update(dt)

	// 更新资源管理器（处理背景音乐淡出）
	vg.resourceManager.UpdateBGMFade(dt)

	// 更新行为系统（处理僵尸移动）
	// 替代原有的模拟逻辑 updateZombieMovement
	// 在 Freeze 状态下，BehaviorSystem 会专门处理触发僵尸的移动
	vg.behaviorSystem.Update(dt)

	// 更新僵尸获胜流程系统
	if vg.zombiesWonSystem != nil {
		vg.zombiesWonSystem.Update(dt)
	}

	// 更新对话框输入系统（处理鼠标交互）
	if vg.dialogInputSystem != nil {
		vg.dialogInputSystem.Update(dt)
	}

	// 更新光标形状（鼠标悬停在按钮上时显示手形）
	vg.updateCursorShape()

	return nil
}

// updateCursorShape 更新光标形状（鼠标悬停在对话框按钮上时显示手形）
func (vg *VerifyZombiesWonGame) updateCursorShape() {
	cursorShape := ebiten.CursorShapeDefault

	// 查询所有对话框实体
	dialogEntities := ecs.GetEntitiesWith2[*components.DialogComponent, *components.PositionComponent](vg.entityManager)

	for _, entityID := range dialogEntities {
		dialogComp, ok := ecs.GetComponent[*components.DialogComponent](vg.entityManager, entityID)
		if !ok {
			continue
		}

		// 如果有任何按钮悬停，显示手形光标
		if dialogComp.HoveredButtonIdx >= 0 {
			cursorShape = ebiten.CursorShapePointer
			break
		}
	}

	// 设置光标形状
	ebiten.SetCursorShape(cursorShape)
}

// isPhase1Complete 检查 Phase 1 是否完成
func (vg *VerifyZombiesWonGame) isPhase1Complete() bool {
	phaseComp, ok := ecs.GetComponent[*components.ZombiesWonPhaseComponent](vg.entityManager, vg.flowID)
	if !ok {
		return false
	}
	return phaseComp.CurrentPhase > 1
}

// jumpToPhase 跳转到指定阶段
func (vg *VerifyZombiesWonGame) jumpToPhase(phase int) {
	if !vg.triggered {
		log.Printf("[VerifyZombiesWon] ⚠️  流程未触发，先按 Space 键触发流程")
		return
	}

	phaseComp, ok := ecs.GetComponent[*components.ZombiesWonPhaseComponent](vg.entityManager, vg.flowID)
	if !ok {
		log.Println("[VerifyZombiesWon] ❌ 未找到阶段组件")
		return
	}

	if phase < 1 || phase > 4 {
		log.Printf("[VerifyZombiesWon] ⚠️  无效的阶段: %d（有效范围: 1-4）", phase)
		return
	}

	log.Printf("[VerifyZombiesWon] ⏭️  跳转到 Phase %d", phase)
	phaseComp.CurrentPhase = phase
	phaseComp.PhaseTimer = 0.0

	// 根据阶段设置必要的状态
	switch phase {
	case 2:
		// Phase 2: 确保僵尸在合适的位置
		pos, _ := ecs.GetComponent[*components.PositionComponent](vg.entityManager, vg.zombieID)
		if pos != nil && pos.X > 0 {
			pos.X = 50.0 // 将僵尸移动到接近边界的位置
		}
	case 3:
		// Phase 3: 确保摄像机已到达目标位置，僵尸已到达目标位置
		phaseComp.CameraMovedToTarget = true
		phaseComp.ZombieReachedTarget = true
		phaseComp.ZombieStartedWalking = true
	case 4:
		// Phase 4: 标记动画已准备好
		phaseComp.AnimationReady = true
	}
}

// Draw 绘制游戏画面
func (vg *VerifyZombiesWonGame) Draw(screen *ebiten.Image) {
	// 清空屏幕
	screen.Fill(color.RGBA{20, 20, 30, 255})

	// 手动绘制背景
	backgroundImg := vg.resourceManager.GetImageByID("IMAGE_BACKGROUND1")
	if backgroundImg != nil {
		opts := &ebiten.DrawImageOptions{}
		opts.GeoM.Translate(-vg.gameState.CameraX, 0)
		screen.DrawImage(backgroundImg, opts)
	}

	// 绘制游戏世界元素（僵尸、植物等 - 不包括 UI）
	vg.renderSystem.DrawGameWorld(screen, vg.gameState.CameraX)

	// 绘制 UI 元素（ZombiesWon 动画等）
	vg.drawUIElements(screen)

	// 绘制调试信息
	vg.drawDebugInfo(screen)
}

// drawUIElements 绘制 UI 元素（ZombiesWon 动画、对话框等）
func (vg *VerifyZombiesWonGame) drawUIElements(screen *ebiten.Image) {
	// 使用 RenderSystem 的公开方法渲染所有 UI 元素
	vg.renderSystem.DrawUIElements(screen)

	// 绘制对话框
	if vg.dialogSystem != nil {
		vg.dialogSystem.Draw(screen)
	}
}

// drawDebugInfo 绘制调试信息
func (vg *VerifyZombiesWonGame) drawDebugInfo(screen *ebiten.Image) {
	if vg.debugFont == nil {
		return
	}

	y := 20.0

	// 标题
	vg.drawText(screen, "僵尸获胜流程验证 - Story 8.8", 10, y, color.RGBA{255, 255, 0, 255})
	y += 25

	// 快捷键提示
	if !vg.triggered {
		vg.drawText(screen, "按 Space 键触发流程", 10, y, color.RGBA{0, 255, 0, 255})
		y += 20
	}

	// 阶段信息
	if vg.triggered {
		phaseComp, ok := ecs.GetComponent[*components.ZombiesWonPhaseComponent](vg.entityManager, vg.flowID)
		if ok {
			phaseText := fmt.Sprintf("当前阶段: Phase %d (计时: %.2fs)", phaseComp.CurrentPhase, phaseComp.PhaseTimer)
			vg.drawText(screen, phaseText, 10, y, color.RGBA{255, 255, 255, 255})
			y += 20

			// 阶段详细信息
			switch phaseComp.CurrentPhase {
			case 1:
				vg.drawText(screen, "  - 游戏冻结中...", 10, y, color.RGBA{200, 200, 200, 255})
				y += 18
			case 2:
				pos, _ := ecs.GetComponent[*components.PositionComponent](vg.entityManager, vg.zombieID)
				if pos != nil {
					zombieText := fmt.Sprintf("  - 僵尸位置: X=%.1f (目标: X < -100)", pos.X)
					vg.drawText(screen, zombieText, 10, y, color.RGBA{200, 200, 200, 255})
					y += 18
				}
				cameraText := fmt.Sprintf("  - 摄像机位置: X=%.1f (目标: X=0)", vg.gameState.CameraX)
				vg.drawText(screen, cameraText, 10, y, color.RGBA{200, 200, 200, 255})
				y += 18
			case 3:
				vg.drawText(screen, "  - 播放惨叫和动画中...", 10, y, color.RGBA{200, 200, 200, 255})
				y += 18
				if phaseComp.ScreamPlayed {
					vg.drawText(screen, "    ✅ 惨叫音效已播放", 10, y, color.RGBA{150, 255, 150, 255})
					y += 18
				}
				if phaseComp.ChompPlayed {
					vg.drawText(screen, "    ✅ 咀嚼音效已播放", 10, y, color.RGBA{150, 255, 150, 255})
					y += 18
				}
			case 4:
				vg.drawText(screen, "  - 等待显示对话框...", 10, y, color.RGBA{200, 200, 200, 255})
				y += 18
			}
		}
	}

	// 完成状态
	if vg.completed {
		y += 10
		vg.drawText(screen, "✅ 验证完成！", 10, y, color.RGBA{0, 255, 0, 255})
		y += 20
		vg.drawText(screen, "按 R 重启 | 按 Q 退出", 10, y, color.RGBA{200, 200, 200, 255})
	}

	// 底部快捷键提示
	vg.drawText(screen, "快捷键: 1-4=跳转阶段 | R=重启 | Q=退出", 10, screenHeight-25, color.RGBA{150, 150, 150, 255})
}

// drawText 绘制文本
func (vg *VerifyZombiesWonGame) drawText(screen *ebiten.Image, str string, x, y float64, clr color.Color) {
	if vg.debugFont == nil {
		return
	}

	opts := &text.DrawOptions{}
	opts.GeoM.Translate(x, y)
	opts.ColorScale.ScaleWithColor(clr)
	text.Draw(screen, str, vg.debugFont, opts)
}

// Layout 设置屏幕布局
func (vg *VerifyZombiesWonGame) Layout(outsideWidth, outsideHeight int) (int, int) {
	return screenWidth, screenHeight
}

// reset 重置验证程序
func (vg *VerifyZombiesWonGame) reset() {
	// 清理旧的流程实体
	if vg.flowID != 0 {
		vg.entityManager.DestroyEntity(vg.flowID)
	}

	// 清理 ZombiesWon 动画实体（如果存在）
	if vg.zombiesWonAnim != 0 {
		vg.entityManager.DestroyEntity(vg.zombiesWonAnim)
	}

	// 清理对话框实体（如果存在）
	if vg.dialogID != 0 {
		vg.entityManager.DestroyEntity(vg.dialogID)
	}

	vg.entityManager.RemoveMarkedEntities()

	// 重置游戏状态
	vg.gameState.CameraX = config.GameCameraX

	// 重新创建测试场景
	vg.setupTestScene()

	// 重置标志
	vg.triggered = false
	vg.completed = false
	vg.flowID = 0
	vg.zombiesWonAnim = 0
	vg.dialogID = 0

	log.Println("[VerifyZombiesWon] 🔄 验证程序已重置")
}

func main() {
	flag.Parse()

	// 设置日志输出
	if !*verbose {
		log.SetOutput(os.Stdout)
	}

	// 创建游戏实例
	verifyGame, err := NewVerifyZombiesWonGame()
	if err != nil {
		log.Fatalf("Failed to create verify game: %v", err)
	}

	// 设置窗口标题
	ebiten.SetWindowTitle("僵尸获胜流程验证 - Story 8.8")
	ebiten.SetWindowSize(screenWidth, screenHeight)

	// 运行游戏
	if err := ebiten.RunGame(verifyGame); err != nil {
		log.Fatal(err)
	}
}
