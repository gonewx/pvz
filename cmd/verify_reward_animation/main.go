package main

import (
	"flag"
	"fmt"
	"image/color"
	"log"
	"os"

	"github.com/decker502/pvz/pkg/components"
	"github.com/decker502/pvz/pkg/config"
	"github.com/decker502/pvz/pkg/ecs"
	"github.com/decker502/pvz/pkg/game"
	"github.com/decker502/pvz/pkg/systems"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

const (
	screenWidth  = 800
	screenHeight = 600
)

var (
	// 命令行参数
	plantID = flag.String("plant", "sunflower", "植物ID (sunflower, peashooter, cherrybomb, wallnut)")
	verbose = flag.Bool("verbose", false, "显示详细调试信息")
)

// VerifyRewardAnimationGame 完整奖励动画流程验证游戏
// 包含卡片包动画（Phase 1-3）和面板显示（Phase 4）
type VerifyRewardAnimationGame struct {
	entityManager         *ecs.EntityManager
	gameState             *game.GameState
	resourceManager       *game.ResourceManager
	reanimSystem          *systems.ReanimSystem
	particleSystem        *systems.ParticleSystem        // 粒子系统（用于光晕效果）
	rewardSystem          *systems.RewardAnimationSystem // 奖励动画系统（Story 8.4重构：完全封装）
	renderSystem          *systems.RenderSystem
	plantCardRenderSystem *systems.PlantCardRenderSystem // 植物卡片渲染系统（用于渲染Phase 1-3的卡片包）

	debugFont *text.GoTextFace // 中文调试字体

	triggered bool // 是否已触发奖励
	completed bool // 是否已完成验证（所有阶段完成）
}

// NewVerifyRewardAnimationGame 创建验证游戏实例
func NewVerifyRewardAnimationGame() (*VerifyRewardAnimationGame, error) {
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

	// 加载奖励面板资源（延迟加载组）
	log.Println("Loading reward panel resources...")
	if err := rm.LoadResourceGroup("DelayLoad_AwardScreen"); err != nil {
		log.Printf("Warning: Failed to load reward panel resources: %v", err)
	}

	// 加载 Reanim 资源（用于植物动画显示）
	log.Println("Loading Reanim resources...")
	if err := rm.LoadReanimResources(); err != nil {
		log.Fatal("Failed to load Reanim resources:", err)
	}

	// 获取游戏状态单例
	gs := game.GetGameState()
	gs.CameraX = config.GameCameraX // 设置摄像机位置

	// 创建系统
	reanimSystem := systems.NewReanimSystem(em)
	particleSystem := systems.NewParticleSystem(em, rm) // 粒子系统用于光晕效果
	renderSystem := systems.NewRenderSystem(em)

	// 创建植物卡片渲染系统（用于渲染 Phase 1-3 的卡片包）
	sunFont, err := rm.LoadFont("assets/fonts/SimHei.ttf", config.PlantCardSunCostFontSize)
	if err != nil {
		log.Printf("Warning: Failed to load sun cost font: %v", err)
		sunFont = nil
	}
	plantCardRenderSystem := systems.NewPlantCardRenderSystem(em, sunFont)

	// Story 8.4重构：RewardAnimationSystem完全封装面板渲染逻辑
	// 内部自动创建和管理RewardPanelRenderSystem
	rewardSystem := systems.NewRewardAnimationSystem(em, gs, rm, reanimSystem, particleSystem)

	// 加载中文调试字体
	debugFont, err := rm.LoadFont("assets/fonts/SimHei.ttf", 14)
	if err != nil {
		log.Printf("Warning: Failed to load debug font: %v", err)
		debugFont = nil
	}

	log.Println("╔════════════════════════════════════════════════════════╗")
	log.Println("║      完整奖励动画流程验证程序 (Story 8.3 + 8.4)         ║")
	log.Println("╚════════════════════════════════════════════════════════╝")
	log.Printf("[VerifyRewardAnimation] 测试植物: %s", *plantID)
	log.Println()
	log.Println("【验证流程】")
	log.Println("  Phase 1: appearing     - 卡片包弹出动画 (0.3s)")
	log.Println("  Phase 2: waiting       - 等待用户点击 (手动触发)")
	log.Println("  Phase 3: expanding     - 卡片包移动+展开动画 (2s)")
	log.Println("  Phase 3.5: pausing     - 短暂停顿+Award粒子 (0.5s)")
	log.Println("  Phase 3.6: disappearing - 卡片包渐渐消失 (0.3s)")
	log.Println("  Phase 4: showing       - 显示奖励面板 (持续)")
	log.Println()
	log.Println("【快捷键】")
	log.Println("  Space/Click - 展开卡片包 (Phase 2)")
	log.Println("  R - 重启验证")
	log.Println("  Q - 退出程序")
	log.Println("════════════════════════════════════════════════════════")

	game := &VerifyRewardAnimationGame{
		entityManager:         em,
		gameState:             gs,
		resourceManager:       rm,
		reanimSystem:          reanimSystem,
		particleSystem:        particleSystem,
		rewardSystem:          rewardSystem,
		renderSystem:          renderSystem,
		plantCardRenderSystem: plantCardRenderSystem,
		debugFont:             debugFont,
		triggered:             false,
		completed:             false,
	}

	// 自动触发奖励动画（无需手动按T键）
	log.Println("[VerifyRewardAnimation] 自动触发奖励动画")
	rewardSystem.TriggerReward(*plantID)
	game.triggered = true

	return game, nil
}

// Update 更新游戏逻辑
func (vg *VerifyRewardAnimationGame) Update() error {
	// 快捷键：T 键手动触发奖励（如果未触发）
	if inpututil.IsKeyJustPressed(ebiten.KeyT) && !vg.triggered {
		log.Println("[VerifyRewardAnimation] 手动触发奖励动画")
		vg.rewardSystem.TriggerReward(*plantID)
		vg.triggered = true
	}

	// 快捷键：R 键重启
	if inpututil.IsKeyJustPressed(ebiten.KeyR) {
		log.Println("[VerifyRewardAnimation] 重启验证")
		vg.reset()
		return nil
	}

	// 快捷键：Q 键退出
	if inpututil.IsKeyJustPressed(ebiten.KeyQ) {
		log.Println("[VerifyRewardAnimation] 退出验证程序")
		return fmt.Errorf("quit")
	}

	// 更新系统
	dt := 1.0 / 60.0
	vg.reanimSystem.Update(dt)
	vg.particleSystem.Update(dt) // 更新粒子系统

	// 更新奖励系统（包含完整的 4 个阶段）
	vg.rewardSystem.Update(dt)

	// 检查是否完成所有阶段
	if vg.triggered && !vg.completed {
		rewardComp, ok := ecs.GetComponent[*components.RewardAnimationComponent](
			vg.entityManager,
			vg.rewardSystem.GetEntity(),
		)
		if ok && rewardComp.Phase == "showing" && rewardComp.ElapsedTime > 1.0 {
			// Phase 4 (showing) 持续 1 秒后标记为完成
			if !vg.completed {
				log.Println("╔════════════════════════════════════════════════════════╗")
				log.Println("║           ✅ 完整奖励动画流程验证完成！               ║")
				log.Println("╚════════════════════════════════════════════════════════╝")
				log.Println()
				log.Println("【验证成果】")
				log.Println("  ✅ Phase 1: appearing     - 卡片包弹出 (完成)")
				log.Println("  ✅ Phase 2: waiting       - 等待点击 (完成)")
				log.Println("  ✅ Phase 3: expanding     - 移动+展开动画 (完成)")
				log.Println("  ✅ Phase 3.5: pausing     - 短暂停顿+粒子 (完成)")
				log.Println("  ✅ Phase 3.6: disappearing - 卡片包消失 (完成)")
				log.Println("  ✅ Phase 4: showing       - 面板显示 (完成)")
				log.Println()
				log.Println("按 R 重启或 Q 退出")
				log.Println("════════════════════════════════════════════════════════")
				vg.completed = true
			}
		}
	}

	return nil
}

// Draw 绘制游戏画面
func (vg *VerifyRewardAnimationGame) Draw(screen *ebiten.Image) {
	// 清空屏幕
	screen.Fill(color.RGBA{0, 0, 0, 255})

	// 手动绘制背景
	backgroundImg := vg.resourceManager.GetImageByID("IMAGE_BACKGROUND1")
	if backgroundImg != nil {
		opts := &ebiten.DrawImageOptions{}
		opts.GeoM.Translate(-vg.gameState.CameraX, 0)
		screen.DrawImage(backgroundImg, opts)
	}

	// 渲染顺序（从下往上）：
	// 1. Reanim 实体（背景层）
	// 2. 植物卡片（Phase 1-3 的卡片包）
	// 3. 粒子效果（光晕，装饰层）
	// 4. 奖励面板（Phase 4，最上层）
	
	cameraOffsetX := vg.gameState.CameraX
	
	// 1. 绘制 Reanim 实体
	vg.renderSystem.Draw(screen, cameraOffsetX)
	
	// 2. 绘制植物卡片（Phase 1-3 的卡片包）
	vg.plantCardRenderSystem.Draw(screen)

	// 3. 绘制粒子效果（光晕）
	vg.renderSystem.DrawParticles(screen, cameraOffsetX)

	// 4. 绘制奖励面板（Phase 4）
	vg.rewardSystem.Draw(screen)

	// 绘制调试信息
	vg.drawDebugInfo(screen)
}

// Layout 设置屏幕布局
func (vg *VerifyRewardAnimationGame) Layout(outsideWidth, outsideHeight int) (int, int) {
	return screenWidth, screenHeight
}

// reset 重置验证程序
func (vg *VerifyRewardAnimationGame) reset() {
	// 清理旧的奖励实体
	if vg.rewardSystem.GetEntity() != 0 {
		vg.entityManager.DestroyEntity(vg.rewardSystem.GetEntity())
	}
	vg.entityManager.RemoveMarkedEntities()

	// 重新创建奖励系统
	vg.rewardSystem = systems.NewRewardAnimationSystem(
		vg.entityManager,
		vg.gameState,
		vg.resourceManager,
		vg.reanimSystem,
		vg.particleSystem,
	)

	vg.triggered = false
	vg.completed = false

	// 自动触发
	log.Println("[VerifyRewardAnimation] 重新触发奖励动画")
	vg.rewardSystem.TriggerReward(*plantID)
	vg.triggered = true
}

// drawDebugInfo 绘制调试信息
func (vg *VerifyRewardAnimationGame) drawDebugInfo(screen *ebiten.Image) {
	rewardEntity := vg.rewardSystem.GetEntity()

	// Phase 4 (showing) 时不显示调试信息，避免遮挡奖励面板
	if rewardEntity != 0 {
		rewardComp, ok := ecs.GetComponent[*components.RewardAnimationComponent](vg.entityManager, rewardEntity)
		if ok && rewardComp.Phase == "showing" {
			// 只显示简短提示
			if vg.debugFont != nil {
				hintText := "Phase 4: 显示奖励面板 - 按 Space 关闭"
				op := &text.DrawOptions{}
				op.GeoM.Translate(10, 10)
				op.ColorScale.ScaleWithColor(color.White)
				text.Draw(screen, hintText, vg.debugFont, op)
			}
			return
		}
	}

	var debugText string

	if rewardEntity == 0 {
		debugText = `完整奖励动画流程验证程序

完整游戏场景已加载（背景 + 植物）

验证流程:
  Phase 1: appearing     - 卡片包弹出 (0.3s)
  Phase 2: waiting       - 等待点击
  Phase 3: expanding     - 移动+展开 (2s)
  Phase 3.5: pausing     - 短暂停顿+粒子 (0.5s)
  Phase 3.6: disappearing - 卡片包消失 (0.3s)
  Phase 4: showing       - 显示面板

快捷键:
  T - 触发奖励动画
  Space/Click - 展开卡片包
  R - 重启
  Q - 退出

按 T 键开始验证...`
	} else {
		// 获取奖励组件信息
		rewardComp, ok := ecs.GetComponent[*components.RewardAnimationComponent](vg.entityManager, rewardEntity)
		if !ok {
			return
		}

		posComp, _ := ecs.GetComponent[*components.PositionComponent](vg.entityManager, rewardEntity)

		// 显示状态信息
		debugText = fmt.Sprintf(`完整奖励动画流程验证 (Story 8.3 + 8.4)
植物: %s
当前阶段: %s (%.2fs)
缩放: %.2f
`, *plantID, rewardComp.Phase, rewardComp.ElapsedTime, rewardComp.Scale)

		if posComp != nil {
			debugText += fmt.Sprintf("位置: (%.1f, %.1f)\n", posComp.X, posComp.Y)
		}

		// 阶段说明
		phaseDesc := map[string]string{
			"appearing":    "Phase 1: 卡片包弹出 (0.3s) 🎁",
			"waiting":      "Phase 2: 等待点击 - 按 Space ⏳",
			"expanding":    "Phase 3: 移动+展开动画 (2s) ✨",
			"pausing":      "Phase 3.5: 短暂停顿+粒子 (0.5s) 💫",
			"disappearing": "Phase 3.6: 卡片包消失 (0.3s) 🌟",
			"showing":      "Phase 4: 显示奖励面板 ✅",
		}

		if desc, exists := phaseDesc[rewardComp.Phase]; exists {
			debugText += "\n" + desc + "\n"
		}

		// 完成状态
		if vg.completed {
			debugText += "\n【验证完成】所有阶段已完成！\n"
		}

		debugText += "\n快捷键: Space=展开 R=重启 Q=退出"
	}

	// 使用中文字体渲染调试信息
	if vg.debugFont != nil {
		// 分行渲染
		lines := splitLines(debugText)

		// 计算文本背景区域大小
		textHeight := float64(len(lines)) * 18
		textWidth := 500.0 // 固定宽度

		// 绘制半透明黑色背景
		bgImg := ebiten.NewImage(int(textWidth), int(textHeight)+10)
		bgImg.Fill(color.RGBA{0, 0, 0, 180}) // 半透明黑色 (alpha=180)
		bgOp := &ebiten.DrawImageOptions{}
		bgOp.GeoM.Translate(5, 5)
		screen.DrawImage(bgImg, bgOp)

		// 绘制文字
		y := 10.0
		for _, line := range lines {
			op := &text.DrawOptions{}
			op.GeoM.Translate(10, y)
			op.ColorScale.ScaleWithColor(color.White)
			text.Draw(screen, line, vg.debugFont, op)
			y += 18 // 行高
		}
	} else {
		// 回退到默认字体（不支持中文）
		ebitenutil.DebugPrint(screen, debugText)
	}
}

// splitLines 将文本按换行符分割成行
func splitLines(text string) []string {
	lines := []string{}
	currentLine := ""
	for _, ch := range text {
		if ch == '\n' {
			lines = append(lines, currentLine)
			currentLine = ""
		} else {
			currentLine += string(ch)
		}
	}
	if currentLine != "" {
		lines = append(lines, currentLine)
	}
	return lines
}

func main() {
	flag.Parse()

	// 设置日志输出
	if !*verbose {
		log.SetOutput(os.Stdout)
	}

	// 验证植物ID
	validPlants := map[string]bool{
		"sunflower":  true,
		"peashooter": true,
		"cherrybomb": true,
		"wallnut":    true,
	}

	if !validPlants[*plantID] {
		fmt.Fprintf(os.Stderr, "错误: 无效的植物ID '%s'\n", *plantID)
		fmt.Fprintln(os.Stderr, "有效的植物ID: sunflower, peashooter, cherrybomb, wallnut")
		os.Exit(1)
	}

	// 创建游戏实例
	verifyGame, err := NewVerifyRewardAnimationGame()
	if err != nil {
		log.Fatalf("Failed to create verify game: %v", err)
	}

	// 设置窗口标题
	ebiten.SetWindowTitle(fmt.Sprintf("完整奖励动画流程验证 - %s - Story 8.3 + 8.4", *plantID))
	ebiten.SetWindowSize(screenWidth, screenHeight)

	// 运行游戏
	if err := ebiten.RunGame(verifyGame); err != nil {
		log.Fatal(err)
	}
}
