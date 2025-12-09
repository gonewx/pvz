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
	"github.com/gonewx/pvz/pkg/game"
	"github.com/gonewx/pvz/pkg/systems"
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
	plantID = flag.String("plant", "sunflower", "植物ID (sunflower, peashooter, cherrybomb, wallnut, potatomine)")
	verbose = flag.Bool("verbose", false, "显示详细调试信息")
)

// VerifyGame 奖励卡片包动画验证游戏
// 包含完整的游戏背景场景（草坪、植物等）
type VerifyGame struct {
	entityManager   *ecs.EntityManager
	gameState       *game.GameState
	resourceManager *game.ResourceManager
	reanimSystem    *systems.ReanimSystem
	particleSystem  *systems.ParticleSystem        // 粒子系统（用于光晕效果）
	rewardSystem    *systems.RewardAnimationSystem // 奖励动画系统（封装所有渲染逻辑）
	renderSystem    *systems.RenderSystem          // Reanim 渲染系统

	debugFont *text.GoTextFace // 中文调试字体

	triggered bool // 是否已触发奖励
	completed bool // 是否已完成验证（Phase 3 结束）
}

// NewVerifyGame 创建验证游戏实例
func NewVerifyGame() (*VerifyGame, error) {
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

	// 加载 Reanim 资源（用于植物动画显示）
	log.Println("Loading Reanim resources...")
	if err := rm.LoadReanimResources(); err != nil {
		log.Fatal("Failed to load Reanim resources:", err)
	}

	// 加载 Reanim 配置管理器
	log.Println("Loading Reanim config...")
	reanimConfigManager, err := config.NewReanimConfigManager("data/reanim_config")
	if err != nil {
		return nil, fmt.Errorf("failed to load reanim config: %w", err)
	}

	// 获取游戏状态单例
	gs := game.GetGameState()
	gs.CameraX = config.GameCameraX // 设置摄像机位置

	// 创建音频管理器并设置到 GameState
	audioManager := game.NewAudioManager(rm, nil)
	gs.SetAudioManager(audioManager)

	// 创建系统
	reanimSystem := systems.NewReanimSystem(em)
	reanimSystem.SetConfigManager(reanimConfigManager)
	particleSystem := systems.NewParticleSystem(em, rm) // 粒子系统用于光晕效果
	renderSystem := systems.NewRenderSystem(em)
	rewardSystem := systems.NewRewardAnimationSystem(em, gs, rm, nil, reanimSystem, particleSystem, renderSystem)

	// 加载中文调试字体
	debugFont, err := rm.LoadFont("assets/fonts/SimHei.ttf", 14)
	if err != nil {
		log.Printf("Warning: Failed to load debug font: %v", err)
		debugFont = nil
	}

	log.Println("[VerifyGame] 奖励卡片包动画验证程序已启动")
	log.Printf("[VerifyGame] 测试植物: %s", *plantID)
	log.Println("[VerifyGame] 快捷键: Space/Click=展开卡片, R=重启, Q=退出")

	game := &VerifyGame{
		entityManager:   em,
		gameState:       gs,
		resourceManager: rm,
		reanimSystem:    reanimSystem,
		particleSystem:  particleSystem,
		rewardSystem:    rewardSystem,
		renderSystem:    renderSystem,
		debugFont:       debugFont,
		triggered:       false,
		completed:       false,
	}

	// 自动触发奖励动画（无需手动按T键）
	log.Println("[VerifyGame] 自动触发奖励动画")
	rewardSystem.TriggerReward("plant", *plantID)
	game.triggered = true

	return game, nil
}

// Update 更新游戏逻辑
func (vg *VerifyGame) Update() error {
	// 快捷键：T 键触发奖励
	if inpututil.IsKeyJustPressed(ebiten.KeyT) && !vg.triggered {
		log.Println("[VerifyGame] 手动触发奖励动画")
		vg.rewardSystem.TriggerReward("plant", *plantID)
		vg.triggered = true
	}

	// 快捷键：R 键重启
	if inpututil.IsKeyJustPressed(ebiten.KeyR) {
		log.Println("[VerifyGame] 重启验证")
		vg.reset()
		return nil
	}

	// 快捷键：Q 键退出
	if inpututil.IsKeyJustPressed(ebiten.KeyQ) {
		log.Println("[VerifyGame] 退出验证程序")
		return fmt.Errorf("quit")
	}

	// 更新系统
	dt := 1.0 / 60.0
	vg.reanimSystem.Update(dt)
	vg.particleSystem.Update(dt) // 更新粒子系统

	// 只在未完成验证时更新奖励系统
	if !vg.completed {
		vg.rewardSystem.Update(dt)
	}

	// 更新鼠标光标（奖励图标和按钮悬停时显示手形）
	cursorShape := vg.rewardSystem.GetCursorShape()
	ebiten.SetCursorShape(cursorShape)

	// 检查是否完成（到 Phase 4 showing 时停止，不显示面板）
	if vg.triggered && !vg.completed {
		rewardComp, ok := ecs.GetComponent[*components.RewardAnimationComponent](
			vg.entityManager,
			vg.rewardSystem.GetEntity(),
		)
		if ok && rewardComp.Phase == "showing" {
			// Phase 4 (showing) 时，本验证程序标记为完成
			// 停止更新奖励系统，防止创建面板
			log.Println("[VerifyGame] Phase 3 (expanding) 完成，卡片包验证结束")
			log.Println("[VerifyGame] 按 R 重启或 Q 退出")
			vg.completed = true
		}
	}

	return nil
}

// Draw 绘制游戏画面
func (vg *VerifyGame) Draw(screen *ebiten.Image) {
	// 清空屏幕
	screen.Fill(color.RGBA{0, 0, 0, 255})

	// 手动绘制背景（参考 verify_opening）
	backgroundImg := vg.resourceManager.GetImageByID("IMAGE_BACKGROUND1")
	if backgroundImg != nil {
		opts := &ebiten.DrawImageOptions{}
		opts.GeoM.Translate(-vg.gameState.CameraX, 0)
		screen.DrawImage(backgroundImg, opts)
	}

	// 绘制奖励动画（包括植物卡片包和粒子效果）
	// RewardAnimationSystem.Draw() 完全封装了所有渲染逻辑：
	// - Phase 1-3: 卡片包（PlantCardComponent）+ 粒子效果
	// - Phase 4: 奖励面板
	vg.rewardSystem.Draw(screen)

	// 绘制调试信息
	vg.drawDebugInfo(screen)
}

// Layout 设置屏幕布局
func (vg *VerifyGame) Layout(outsideWidth, outsideHeight int) (int, int) {
	return screenWidth, screenHeight
}

// reset 重置验证程序
func (vg *VerifyGame) reset() {
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
		nil, // sceneManager（测试程序不需要）
		vg.reanimSystem,
		vg.particleSystem,
		vg.renderSystem,
	)

	vg.triggered = false
	vg.completed = false
}

// drawDebugInfo 绘制调试信息
func (vg *VerifyGame) drawDebugInfo(screen *ebiten.Image) {
	rewardEntity := vg.rewardSystem.GetEntity()

	var debugText string

	if rewardEntity == 0 {
		debugText = `奖励卡片包动画验证程序

完整游戏场景已加载（背景 + 植物）

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
		debugText = fmt.Sprintf(`奖励卡片包验证 - 完整场景模式
植物: %s
阶段: %s (%.2fs)
缩放: %.2f
`, *plantID, rewardComp.Phase, rewardComp.ElapsedTime, rewardComp.Scale)

		if posComp != nil {
			debugText += fmt.Sprintf("位置: (%.1f, %.1f)\n", posComp.X, posComp.Y)

			// 计算卡片在屏幕上的实际可见位置
			screenX := posComp.X - vg.gameState.CameraX
			debugText += fmt.Sprintf("屏幕位置: (%.1f, %.1f)\n", screenX, posComp.Y)

			// 显示卡片中心位置（用于验证居中）
			if cardComp, ok := ecs.GetComponent[*components.PlantCardComponent](vg.entityManager, rewardEntity); ok && cardComp.BackgroundImage != nil {
				cardWidth := float64(cardComp.BackgroundImage.Bounds().Dx()) * cardComp.CardScale
				cardHeight := float64(cardComp.BackgroundImage.Bounds().Dy()) * cardComp.CardScale
				centerX := posComp.X + cardWidth/2.0
				centerY := posComp.Y + cardHeight/2.0
				screenCenterX := centerX - vg.gameState.CameraX
				debugText += fmt.Sprintf("卡片尺寸: %.1fx%.1f (缩放: %.2f)\n", cardWidth, cardHeight, cardComp.CardScale)
				debugText += fmt.Sprintf("卡片中心世界坐标: (%.1f, %.1f)\n", centerX, centerY)
				debugText += fmt.Sprintf("卡片中心屏幕坐标: (%.1f, %.1f)\n", screenCenterX, centerY)
				debugText += fmt.Sprintf("草坪中心世界X: %.1f, 屏幕宽度/2: %.1f\n",
					config.GridWorldStartX+float64(config.GridColumns)*config.CellWidth/2.0,
					float64(screenWidth)/2.0)
			}
		}

		// 阶段说明
		phaseDesc := map[string]string{
			"appearing": "Phase 1: 卡片弹出 (0.3s)",
			"waiting":   "Phase 2: 等待点击 - 按 Space",
			"expanding": "Phase 3: 展开动画 (2s)",
			"showing":   "验证完成！",
		}

		if desc, exists := phaseDesc[rewardComp.Phase]; exists {
			debugText += "\n" + desc + "\n"
		}

		debugText += "\n快捷键: T=触发 Space=展开 R=重启 Q=退出"
	}

	// 使用中文字体渲染调试信息
	if vg.debugFont != nil {
		// 分行渲染
		lines := splitLines(debugText)

		// 计算文本背景区域大小
		textHeight := float64(len(lines)) * 18
		textWidth := 450.0 // 固定宽度

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
		"potatomine": true,
	}

	if !validPlants[*plantID] {
		fmt.Fprintf(os.Stderr, "错误: 无效的植物ID '%s'\n", *plantID)
		fmt.Fprintln(os.Stderr, "有效的植物ID: sunflower, peashooter, cherrybomb, wallnut, potatomine")
		os.Exit(1)
	}

	// 创建游戏实例
	verifyGame, err := NewVerifyGame()
	if err != nil {
		log.Fatalf("Failed to create verify game: %v", err)
	}

	// 设置窗口标题
	ebiten.SetWindowTitle(fmt.Sprintf("奖励卡片包验证 - 完整场景 - %s", *plantID))
	ebiten.SetWindowSize(screenWidth, screenHeight)

	// 运行游戏
	if err := ebiten.RunGame(verifyGame); err != nil {
		log.Fatal(err)
	}
}
