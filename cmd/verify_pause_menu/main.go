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
	"github.com/decker502/pvz/pkg/modules"
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
	verbose = flag.Bool("verbose", false, "显示详细调试信息")
)

// VerifyPauseMenuGame 暂停菜单验证游戏
// 直接使用 PauseMenuModule 显示暂停菜单UI
type VerifyPauseMenuGame struct {
	entityManager   *ecs.EntityManager
	gameState       *game.GameState
	resourceManager *game.ResourceManager
	pauseMenuModule *modules.PauseMenuModule // 暂停菜单模块

	debugFont *text.GoTextFace // 中文调试字体

	menuVisible bool // 菜单是否显示
}

// NewVerifyPauseMenuGame 创建验证游戏实例
func NewVerifyPauseMenuGame() (*VerifyPauseMenuGame, error) {
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

	// 获取游戏状态单例
	gs := game.GetGameState()
	gs.CameraX = config.GameCameraX // 设置摄像机位置

	// 创建按钮系统（暂停菜单需要）
	buttonSystem := systems.NewButtonSystem(em)
	buttonRenderSystem := systems.NewButtonRenderSystem(em)

	// 创建暂停菜单模块
	log.Println("Creating pause menu module...")
	pauseMenuModule, err := modules.NewPauseMenuModule(
		em,
		gs,
		rm,
		buttonSystem,
		buttonRenderSystem,
		nil, // settingsManager（测试场景使用 nil）
		screenWidth,
		screenHeight,
		modules.PauseMenuCallbacks{
			OnContinue: func() {
				log.Println("[Callback] Continue button clicked")
			},
			OnRestart: func() {
				log.Println("[Callback] Restart button clicked")
			},
			OnMainMenu: func() {
				log.Println("[Callback] Main menu button clicked")
			},
			OnPauseMusic:  nil,
			OnResumeMusic: nil,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create pause menu module: %w", err)
	}

	// 加载中文调试字体
	debugFont, err := rm.LoadFont("assets/fonts/SimHei.ttf", 14)
	if err != nil {
		log.Printf("Warning: Failed to load debug font: %v", err)
		debugFont = nil
	}

	log.Println("[VerifyPauseMenuGame] 暂停菜单验证程序已启动")

	// 默认显示菜单
	pauseMenuModule.Show()

	return &VerifyPauseMenuGame{
		entityManager:   em,
		gameState:       gs,
		resourceManager: rm,
		pauseMenuModule: pauseMenuModule,
		debugFont:       debugFont,
		menuVisible:     true,
	}, nil
}

// Update 更新游戏逻辑
func (vpg *VerifyPauseMenuGame) Update() error {
	// 快捷键：Q 键退出
	if inpututil.IsKeyJustPressed(ebiten.KeyQ) {
		log.Println("[VerifyPauseMenuGame] 退出验证程序")
		return fmt.Errorf("quit")
	}

	// 快捷键：ESC 键切换菜单显示/隐藏
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		vpg.pauseMenuModule.Toggle()
		vpg.menuVisible = vpg.pauseMenuModule.IsActive()
		log.Printf("[VerifyPauseMenuGame] 菜单状态切换: %v", vpg.menuVisible)
	}

	// 快捷键：1-4 键调试UI元素位置
	// 注意：config 常量不可修改，这里仅用于演示
	// 实际调整需要在 pkg/config/layout_config.go 中修改
	if inpututil.IsKeyJustPressed(ebiten.Key1) {
		log.Printf("[Debug] 提示: 请在 pkg/config/layout_config.go 中调整 PauseMenuMusicSliderOffsetY")
	}
	if inpututil.IsKeyJustPressed(ebiten.Key2) {
		log.Printf("[Debug] 当前音乐滑动条Y偏移: %.1f", config.PauseMenuMusicSliderOffsetY)
	}
	if inpututil.IsKeyJustPressed(ebiten.Key3) {
		log.Printf("[Debug] 当前音效滑动条Y偏移: %.1f", config.PauseMenuSoundSliderOffsetY)
	}
	if inpututil.IsKeyJustPressed(ebiten.Key4) {
		log.Printf("[Debug] 提示: 请在 pkg/config/layout_config.go 中调整相关偏移值")
	}

	// 更新暂停菜单（同步状态）
	vpg.pauseMenuModule.Update(1.0 / 60.0)

	return nil
}

// Draw 绘制游戏画面
func (vpg *VerifyPauseMenuGame) Draw(screen *ebiten.Image) {
	// 清空屏幕
	screen.Fill(color.RGBA{50, 100, 50, 255}) // 深绿色背景

	// 手动绘制背景
	backgroundImg := vpg.resourceManager.GetImageByID("IMAGE_BACKGROUND1")
	if backgroundImg != nil {
		opts := &ebiten.DrawImageOptions{}
		opts.GeoM.Translate(-vpg.gameState.CameraX, 0)
		screen.DrawImage(backgroundImg, opts)
	}

	// 绘制暂停菜单
	vpg.pauseMenuModule.Draw(screen)

	// 绘制调试信息
	vpg.drawDebugInfo(screen)
}

// Layout 设置屏幕布局
func (vpg *VerifyPauseMenuGame) Layout(outsideWidth, outsideHeight int) (int, int) {
	return screenWidth, screenHeight
}

// drawDebugInfo 绘制调试信息
func (vpg *VerifyPauseMenuGame) drawDebugInfo(screen *ebiten.Image) {
	// 获取"返回游戏"按钮的位置信息
	buttonInfo := "按钮未找到"
	if vpg.pauseMenuModule != nil {
		entities := vpg.pauseMenuModule.GetButtonEntities()
		if len(entities) > 0 {
			entity := entities[0]
			if pos, ok := ecs.GetComponent[*components.PositionComponent](vpg.entityManager, entity); ok {
				if btn, ok := ecs.GetComponent[*components.ButtonComponent](vpg.entityManager, entity); ok {
					buttonInfo = fmt.Sprintf("Pos:(%.0f,%.0f) Img:%v",
						pos.X, pos.Y, btn.NormalImage != nil)
				}
			}
		}
	}

	// 显示状态信息
	debugText := fmt.Sprintf(`暂停菜单验证程序
菜单状态: %v
游戏暂停: %v
返回游戏按钮: %s

UI元素偏移（查看配置）:
音乐滑动条Y: %.1f
音效滑动条Y: %.1f
3D复选框Y: %.1f
全屏复选框Y: %.1f

快捷键:
ESC = 切换菜单
1-4 = 查看偏移值
Q = 退出`, vpg.menuVisible, vpg.gameState.IsPaused,
		buttonInfo,
		config.PauseMenuMusicSliderOffsetY,
		config.PauseMenuSoundSliderOffsetY,
		config.PauseMenu3DCheckboxOffsetY,
		config.PauseMenuFullscreenCheckboxOffsetY)

	// 使用中文字体渲染调试信息
	if vpg.debugFont != nil {
		// 分行渲染
		lines := splitLines(debugText)

		// 计算文本背景区域大小
		textHeight := float64(len(lines)) * 18
		textWidth := 200.0 // 固定宽度（减少一半）

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

	// 创建游戏实例
	verifyGame, err := NewVerifyPauseMenuGame()
	if err != nil {
		log.Fatalf("Failed to create verify game: %v", err)
	}

	// 设置窗口标题
	ebiten.SetWindowTitle("暂停菜单验证 - Plants vs Zombies")
	ebiten.SetWindowSize(screenWidth, screenHeight)

	// 运行游戏
	if err := ebiten.RunGame(verifyGame); err != nil {
		log.Fatal(err)
	}
}
