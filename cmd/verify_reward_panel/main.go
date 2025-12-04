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
	plantID = flag.String("plant", "sunflower", "植物ID (sunflower, peashooter, cherrybomb, wallnut, potatomine)")
	verbose = flag.Bool("verbose", false, "显示详细调试信息")
)

// VerifyPanelGame 奖励植物介绍面板验证游戏
// 直接使用 RewardPanelRenderSystem 显示面板
type VerifyPanelGame struct {
	entityManager         *ecs.EntityManager
	gameState             *game.GameState
	resourceManager       *game.ResourceManager
	reanimSystem          *systems.ReanimSystem            // Reanim 动画系统（必须每帧更新）
	panelRenderSystem     *systems.RewardPanelRenderSystem // 面板渲染系统
	plantCardRenderSystem *systems.PlantCardRenderSystem   // 植物卡片渲染系统（新增）

	debugFont *text.GoTextFace // 中文调试字体

	panelEntity ecs.EntityID // 面板实体
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

	// 创建 Reanim 系统（用于渲染植物）
	reanimSystem := systems.NewReanimSystem(em)
	reanimSystem.SetConfigManager(reanimConfigManager)

	// 创建面板渲染系统（需要 ReanimSystem 来渲染植物）
	panelRenderSystem := systems.NewRewardPanelRenderSystem(em, gs, rm, reanimSystem)

	// 创建植物卡片渲染系统（渲染卡片实体）
	// Story 8.4: 使用简化接口，所有内部配置从 config.plant_card_config.go 读取
	sunFont, err := rm.LoadFont("assets/fonts/SimHei.ttf", config.PlantCardSunCostFontSize)
	if err != nil {
		log.Printf("Warning: Failed to load sun cost font: %v", err)
		sunFont = nil
	}
	plantCardRenderSystem := systems.NewPlantCardRenderSystem(em, sunFont)

	// 加载中文调试字体
	debugFont, err := rm.LoadFont("assets/fonts/SimHei.ttf", 14)
	if err != nil {
		log.Printf("Warning: Failed to load debug font: %v", err)
		debugFont = nil
	}

	log.Println("[VerifyPanelGame] 奖励植物介绍面板验证程序已启动")
	log.Printf("[VerifyPanelGame] 测试植物: %s", *plantID)

	// 直接创建并显示面板实体
	panelEntity := em.CreateEntity()

	// 获取植物信息
	plantName := "向日葵"
	plantDesc := "提供你额外的阳光"
	sunCost := 50

	switch *plantID {
	case "sunflower":
		plantName = "向日葵"
		plantDesc = "提供你额外的阳光"
		sunCost = 50
	case "peashooter":
		plantName = "豌豆射手"
		plantDesc = "发射豌豆攻击僵尸"
		sunCost = 100
	case "cherrybomb":
		plantName = "樱桃炸弹"
		plantDesc = "炸毁一定范围内的所有僵尸"
		sunCost = 150
	case "wallnut":
		plantName = "坚果墙"
		plantDesc = "阻挡僵尸前进"
		sunCost = 50
	case "potatomine":
		plantName = "土豆雷"
		plantDesc = "埋在地里等待僵尸踩上去后爆炸"
		sunCost = 25
	}

	// 添加面板组件（PlantID 字段很重要，用于自动加载图标）
	ecs.AddComponent(em, panelEntity, &components.RewardPanelComponent{
		PlantID:          *plantID, // 设置 PlantID，让渲染系统自动加载图标
		IsVisible:        true,
		FadeAlpha:        1.0,
		PlantIconTexture: nil, // 设为 nil，让渲染系统根据 PlantID 加载
		PlantName:        plantName,
		PlantDescription: plantDesc,
		SunCost:          sunCost,
		CardScale:        1.0, // Story 8.4: 卡片位置由 RewardPanelRenderSystem 自动计算
	})

	log.Println("[VerifyPanelGame] 面板已创建并显示")

	return &VerifyPanelGame{
		entityManager:         em,
		gameState:             gs,
		resourceManager:       rm,
		reanimSystem:          reanimSystem, // 保存 ReanimSystem 引用
		panelRenderSystem:     panelRenderSystem,
		plantCardRenderSystem: plantCardRenderSystem, // 植物卡片渲染系统
		debugFont:             debugFont,
		panelEntity:           panelEntity,
	}, nil
}

// Update 更新游戏逻辑
func (vpg *VerifyPanelGame) Update() error {
	// 快捷键：Q 键退出
	if inpututil.IsKeyJustPressed(ebiten.KeyQ) {
		log.Println("[VerifyPanelGame] 退出验证程序")
		return fmt.Errorf("quit")
	}

	// 【关键修复】更新 Reanim 动画系统，使植物动画播放
	// 使用固定时间步长 1.0/60.0 秒（60 FPS）
	vpg.reanimSystem.Update(1.0 / 60.0)

	return nil
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

	// 【调试】打印渲染前的状态
	if *verbose {
		panelComp, _ := ecs.GetComponent[*components.RewardPanelComponent](vpg.entityManager, vpg.panelEntity)
		if panelComp != nil {
			log.Printf("[Draw] PlantID=%s, Visible=%v, Alpha=%.2f, CardScale=%.2f (位置自动居中)",
				panelComp.PlantID, panelComp.IsVisible, panelComp.FadeAlpha, panelComp.CardScale)
		}
	}

	// 绘制奖励面板
	vpg.panelRenderSystem.Draw(screen)

	// 绘制调试信息
	vpg.drawDebugInfo(screen)
}

// Layout 设置屏幕布局
func (vpg *VerifyPanelGame) Layout(outsideWidth, outsideHeight int) (int, int) {
	return screenWidth, screenHeight
}

// drawDebugInfo 绘制调试信息
func (vpg *VerifyPanelGame) drawDebugInfo(screen *ebiten.Image) {
	// 获取面板组件信息
	panelComp, ok := ecs.GetComponent[*components.RewardPanelComponent](vpg.entityManager, vpg.panelEntity)
	if !ok {
		return
	}

	// 显示状态信息
	debugText := fmt.Sprintf(`奖励植物介绍面板验证
植物: %s
显示状态: %v
透明度: %.2f
卡片缩放: %.2f

快捷键: Q=退出`, panelComp.PlantName, panelComp.IsVisible, panelComp.FadeAlpha, panelComp.CardScale)

	// 使用中文字体渲染调试信息
	if vpg.debugFont != nil {
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
			text.Draw(screen, line, vpg.debugFont, op)
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
	verifyGame, err := NewVerifyPanelGame()
	if err != nil {
		log.Fatalf("Failed to create verify game: %v", err)
	}

	// 设置窗口标题
	ebiten.SetWindowTitle(fmt.Sprintf("奖励植物介绍面板验证 - %s", *plantID))
	ebiten.SetWindowSize(screenWidth, screenHeight)

	// 运行游戏
	if err := ebiten.RunGame(verifyGame); err != nil {
		log.Fatal(err)
	}
}
