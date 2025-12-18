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
	"github.com/gonewx/pvz/pkg/modules"
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
	plantID    = flag.String("plant", "", "植物ID (使用 --list 查看所有可用植物)")
	noteID     = flag.String("note", "", "来信ID (zombienote1, zombienote2, zombienote3, zombienote4)")
	listPlants = flag.Bool("list", false, "列出所有可用植物")
	verbose    = flag.Bool("verbose", false, "显示详细调试信息")

	// 当前测试类型
	panelType string // "plant" 或 "note"
	panelID   string
)

// VerifyPanelGame 奖励面板验证游戏
// 支持植物介绍面板和僵尸来信面板
type VerifyPanelGame struct {
	entityManager         *ecs.EntityManager
	gameState             *game.GameState
	resourceManager       *game.ResourceManager
	audioManager          *game.AudioManager               // 音频管理器
	reanimSystem          *systems.ReanimSystem            // Reanim 动画系统（必须每帧更新）
	panelRenderSystem     *systems.RewardPanelRenderSystem // 植物面板渲染系统
	plantCardRenderSystem *systems.PlantCardRenderSystem   // 植物卡片渲染系统
	buttonSystem          *systems.ButtonSystem            // 按钮系统
	buttonRenderSystem    *systems.ButtonRenderSystem      // 按钮渲染系统

	// 僵尸来信面板模块
	notePanelModule *modules.ZombieNotePanelModule

	debugFont *text.GoTextFace // 中文调试字体

	panelEntity ecs.EntityID // 植物面板实体（仅 plant 模式）
}

// NewVerifyPanelGame 创建验证游戏实例
func NewVerifyPanelGame() (*VerifyPanelGame, error) {
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

	// 加载 LawnStrings
	lawnStrings, err := game.NewLawnStrings("assets/properties/LawnStrings.txt")
	if err != nil {
		return nil, fmt.Errorf("failed to load LawnStrings: %w", err)
	}
	gs.LawnStrings = lawnStrings

	// 创建音频管理器并设置到 GameState
	audioManager := game.NewAudioManager(rm, nil)
	gs.SetAudioManager(audioManager)

	// 创建 Reanim 系统（用于渲染植物）
	reanimSystem := systems.NewReanimSystem(em)
	reanimSystem.SetConfigManager(reanimConfigManager)

	// 创建面板渲染系统（需要 ReanimSystem 来渲染植物）
	panelRenderSystem := systems.NewRewardPanelRenderSystem(em, gs, rm, reanimSystem)

	// 创建植物卡片渲染系统（渲染卡片实体）
	sunFont, err := rm.LoadFont("assets/fonts/SimHei.ttf", config.PlantCardSunCostFontSize)
	if err != nil {
		log.Printf("Warning: Failed to load sun cost font: %v", err)
		sunFont = nil
	}
	plantCardRenderSystem := systems.NewPlantCardRenderSystem(em, sunFont)

	// 创建按钮系统（用于来信面板的按钮交互）
	buttonSystem := systems.NewButtonSystem(em)
	buttonRenderSystem := systems.NewButtonRenderSystem(em)

	// 加载中文调试字体
	debugFont, err := rm.LoadFont("assets/fonts/SimHei.ttf", 14)
	if err != nil {
		log.Printf("Warning: Failed to load debug font: %v", err)
		debugFont = nil
	}

	vpg := &VerifyPanelGame{
		entityManager:         em,
		gameState:             gs,
		resourceManager:       rm,
		audioManager:          audioManager,
		reanimSystem:          reanimSystem,
		panelRenderSystem:     panelRenderSystem,
		plantCardRenderSystem: plantCardRenderSystem,
		buttonSystem:          buttonSystem,
		buttonRenderSystem:    buttonRenderSystem,
		debugFont:             debugFont,
	}

	// 根据类型创建不同的面板
	if panelType == "note" {
		log.Println("[VerifyPanelGame] 僵尸来信面板验证程序已启动")
		log.Printf("[VerifyPanelGame] 测试来信: %s", panelID)

		// 创建按钮渲染回调函数
		drawButtonFunc := func(screen *ebiten.Image, buttonEntity ecs.EntityID) {
			vpg.buttonRenderSystem.DrawButton(screen, buttonEntity)
		}

		// 创建僵尸来信面板模块
		notePanelModule, err := modules.NewZombieNotePanelModule(
			em,
			rm,
			gs,
			drawButtonFunc,
			screenWidth,
			screenHeight,
			panelID,
			func() {
				// 下一关回调
				log.Println("[VerifyPanelGame] 点击了下一关按钮")
			},
			func() {
				// 主菜单回调
				log.Println("[VerifyPanelGame] 点击了主菜单按钮")
			},
		)
		if err != nil {
			return nil, fmt.Errorf("failed to create ZombieNotePanelModule: %w", err)
		}

		vpg.notePanelModule = notePanelModule
		vpg.notePanelModule.Show()

		log.Println("[VerifyPanelGame] 来信面板已创建并显示")
	} else {
		log.Println("[VerifyPanelGame] 奖励植物介绍面板验证程序已启动")
		log.Printf("[VerifyPanelGame] 测试植物: %s", panelID)

		// 直接创建并显示植物面板实体
		panelEntity := em.CreateEntity()

		// 使用统一的植物信息获取方法
		plantName, plantDesc := game.GetPlantInfoWithStrings(panelID, lawnStrings)
		sunCost := game.GetPlantSunCost(panelID)

		// 添加面板组件
		ecs.AddComponent(em, panelEntity, &components.RewardPanelComponent{
			PlantID:            panelID,
			IsVisible:          true,
			FadeAlpha:          1.0,
			PlantIconTexture:   nil,
			PlantName:          plantName,
			PlantDescription:   plantDesc,
			SunCost:            sunCost,
			CardScale:          1.0,
			ShowMainMenuButton: true, // 与 verify_reward_animation 保持一致
		})

		vpg.panelEntity = panelEntity
		log.Println("[VerifyPanelGame] 植物面板已创建并显示")
	}

	return vpg, nil
}

// Update 更新游戏逻辑
func (vpg *VerifyPanelGame) Update() error {
	// 快捷键：Q 键退出
	if inpututil.IsKeyJustPressed(ebiten.KeyQ) {
		log.Println("[VerifyPanelGame] 退出验证程序")
		return fmt.Errorf("quit")
	}

	mouseX, mouseY := ebiten.CursorPosition()

	if panelType == "note" {
		// 来信面板模式
		vpg.notePanelModule.Update(1.0 / 60.0)
		vpg.buttonSystem.Update(1.0 / 60.0)

		// 检测悬停状态
		if vpg.notePanelModule.IsHoveringNextButton() || vpg.notePanelModule.IsHoveringMainMenuButton() {
			ebiten.SetCursorShape(ebiten.CursorShapePointer)
		} else {
			ebiten.SetCursorShape(ebiten.CursorShapeDefault)
		}
	} else {
		// 植物面板模式
		if vpg.isNextLevelButtonClicked(float64(mouseX), float64(mouseY)) {
			ebiten.SetCursorShape(ebiten.CursorShapePointer)
		} else {
			ebiten.SetCursorShape(ebiten.CursorShapeDefault)
		}

		// 检测鼠标点击"下一关"按钮
		if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
			if vpg.isNextLevelButtonClicked(float64(mouseX), float64(mouseY)) {
				if vpg.audioManager != nil {
					vpg.audioManager.PlaySound("SOUND_TAP2")
				}
				log.Println("[VerifyPanelGame] 点击了下一关按钮")
			}
		}
	}

	// 更新 Reanim 动画系统
	vpg.reanimSystem.Update(1.0 / 60.0)

	return nil
}

// isNextLevelButtonClicked 检查鼠标点击是否在"下一关"按钮范围内（植物面板模式）
func (vpg *VerifyPanelGame) isNextLevelButtonClicked(mouseX, mouseY float64) bool {
	bgWidth := config.RewardPanelBackgroundWidth
	bgHeight := config.RewardPanelBackgroundHeight
	offsetX := (screenWidth - bgWidth) / 2
	offsetY := (screenHeight - bgHeight) / 2

	buttonX := offsetX + bgWidth/2
	buttonY := offsetY + bgHeight*config.RewardPanelButtonY

	buttonWidth := 155.0
	buttonHeight := 50.0

	halfWidth := buttonWidth / 2
	halfHeight := buttonHeight / 2

	return mouseX >= buttonX-halfWidth && mouseX <= buttonX+halfWidth &&
		mouseY >= buttonY-halfHeight && mouseY <= buttonY+halfHeight
}

// Draw 绘制游戏画面
func (vpg *VerifyPanelGame) Draw(screen *ebiten.Image) {
	// 清空屏幕
	screen.Fill(color.RGBA{0, 0, 0, 255})

	// 手动绘制背景
	backgroundImg := vpg.resourceManager.GetImageByID("IMAGE_BACKGROUND1")
	if backgroundImg != nil {
		opts := &ebiten.DrawImageOptions{}
		opts.GeoM.Translate(-vpg.gameState.CameraX, 0)
		screen.DrawImage(backgroundImg, opts)
	}

	if panelType == "note" {
		// 绘制来信面板
		vpg.notePanelModule.Draw(screen)
	} else {
		// 绘制植物面板
		if *verbose {
			panelComp, _ := ecs.GetComponent[*components.RewardPanelComponent](vpg.entityManager, vpg.panelEntity)
			if panelComp != nil {
				log.Printf("[Draw] PlantID=%s, Visible=%v, Alpha=%.2f, CardScale=%.2f (位置自动居中)",
					panelComp.PlantID, panelComp.IsVisible, panelComp.FadeAlpha, panelComp.CardScale)
			}
		}
		vpg.panelRenderSystem.Draw(screen)
	}

	// 绘制调试信息
	vpg.drawDebugInfo(screen)
}

// Layout 设置屏幕布局
func (vpg *VerifyPanelGame) Layout(outsideWidth, outsideHeight int) (int, int) {
	return screenWidth, screenHeight
}

// drawDebugInfo 绘制调试信息
func (vpg *VerifyPanelGame) drawDebugInfo(screen *ebiten.Image) {
	var debugText string

	if panelType == "note" {
		// 来信面板模式
		var activeStatus string
		if vpg.notePanelModule != nil && vpg.notePanelModule.IsActive() {
			activeStatus = "显示中"
		} else {
			activeStatus = "隐藏"
		}

		debugText = fmt.Sprintf(`僵尸来信面板验证
来信ID: %s
状态: %s

快捷键: Q=退出`, panelID, activeStatus)
	} else {
		// 植物面板模式
		panelComp, ok := ecs.GetComponent[*components.RewardPanelComponent](vpg.entityManager, vpg.panelEntity)
		if !ok {
			return
		}

		debugText = fmt.Sprintf(`奖励植物介绍面板验证
植物: %s
显示状态: %v
透明度: %.2f
卡片缩放: %.2f

快捷键: Q=退出`, panelComp.PlantName, panelComp.IsVisible, panelComp.FadeAlpha, panelComp.CardScale)
	}

	// 使用中文字体渲染调试信息
	if vpg.debugFont != nil {
		lines := splitLines(debugText)
		textHeight := float64(len(lines)) * 18
		textWidth := 450.0

		bgImg := ebiten.NewImage(int(textWidth), int(textHeight)+10)
		bgImg.Fill(color.RGBA{0, 0, 0, 180})
		bgOp := &ebiten.DrawImageOptions{}
		bgOp.GeoM.Translate(5, 5)
		screen.DrawImage(bgImg, bgOp)

		y := 10.0
		for _, line := range lines {
			op := &text.DrawOptions{}
			op.GeoM.Translate(10, y)
			op.ColorScale.ScaleWithColor(color.White)
			text.Draw(screen, line, vpg.debugFont, op)
			y += 18
		}
	} else {
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

	// 列出所有可用植物
	if *listPlants {
		allPlants := config.GetAllPlants()
		fmt.Println("可用植物ID:")
		for _, plant := range allPlants {
			fmt.Printf("  %s\n", plant.ID)
		}
		fmt.Println("\n可用来信ID:")
		fmt.Println("  zombienote1")
		fmt.Println("  zombienote2")
		fmt.Println("  zombienote3")
		fmt.Println("  zombienote4")
		os.Exit(0)
	}

	// 验证参数
	validNotes := map[string]bool{
		"zombienote1": true,
		"zombienote2": true,
		"zombienote3": true,
		"zombienote4": true,
	}

	// 检查参数互斥
	if *plantID != "" && *noteID != "" {
		fmt.Fprintln(os.Stderr, "错误: 不能同时指定 --plant 和 --note")
		os.Exit(1)
	}

	if *plantID != "" {
		// 使用植物注册表验证植物ID
		if config.GetPlantByID(*plantID) == nil {
			fmt.Fprintf(os.Stderr, "错误: 无效的植物ID '%s'\n", *plantID)
			fmt.Fprintln(os.Stderr, "使用 --list 查看所有可用植物")
			os.Exit(1)
		}
		panelType = "plant"
		panelID = *plantID
	} else if *noteID != "" {
		if !validNotes[*noteID] {
			fmt.Fprintf(os.Stderr, "错误: 无效的来信ID '%s'\n", *noteID)
			fmt.Fprintln(os.Stderr, "有效的来信ID: zombienote1, zombienote2, zombienote3, zombienote4")
			os.Exit(1)
		}
		panelType = "note"
		panelID = *noteID
	} else {
		// 默认测试向日葵
		panelType = "plant"
		panelID = "sunflower"
	}

	// 创建游戏实例
	verifyGame, err := NewVerifyPanelGame()
	if err != nil {
		log.Fatalf("Failed to create verify game: %v", err)
	}

	// 设置窗口标题
	var title string
	if panelType == "note" {
		title = fmt.Sprintf("僵尸来信面板验证 - %s", panelID)
	} else {
		title = fmt.Sprintf("奖励植物介绍面板验证 - %s", panelID)
	}
	ebiten.SetWindowTitle(title)
	ebiten.SetWindowSize(screenWidth, screenHeight)

	// 运行游戏
	if err := ebiten.RunGame(verifyGame); err != nil {
		log.Fatal(err)
	}
}
