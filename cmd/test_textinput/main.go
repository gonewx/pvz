// Package main 提供文本输入框测试工具
// 用于验证 TextInputComponent 和相关系统的功能
//
// 用法:
//
//	go run cmd/test_textinput/main.go
//
// 功能:
//   - 显示"新用户"对话框
//   - 支持文本输入（中英文）
//   - 支持光标移动、退格、删除等操作
//   - 点击"确定"后显示输入的用户名
//   - 点击"取消"或按ESC退出
package main

import (
	"fmt"
	"image/color"
	"log"

	"github.com/gonewx/pvz/pkg/ecs"
	"github.com/gonewx/pvz/pkg/entities"
	"github.com/gonewx/pvz/pkg/game"
	"github.com/gonewx/pvz/pkg/systems"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

const (
	screenWidth  = 800
	screenHeight = 600
)

// TestGame 测试游戏结构
type TestGame struct {
	resourceManager *game.ResourceManager
	entityManager   *ecs.EntityManager

	// 系统
	textInputSystem       *systems.TextInputSystem
	textInputRenderSystem *systems.TextInputRenderSystem
	dialogRenderSystem    *systems.DialogRenderSystem
	dialogInputSystem     *systems.DialogInputSystem

	// 实体
	dialogEntity   ecs.EntityID
	inputBoxEntity ecs.EntityID

	// 状态
	resultReceived bool
	resultText     string
}

// NewTestGame 创建测试游戏
func NewTestGame() (*TestGame, error) {
	// 初始化音频上下文
	audioContext := audio.NewContext(48000)

	// 初始化资源管理器
	rm := game.NewResourceManager(audioContext)

	// 加载 YAML 配置
	if err := rm.LoadResourceConfig("assets/config/resources.yaml"); err != nil {
		return nil, fmt.Errorf("加载资源配置失败: %w", err)
	}

	// 加载初始资源组
	if err := rm.LoadResourceGroup("init"); err != nil {
		return nil, fmt.Errorf("加载初始资源组失败: %w", err)
	}

	// 初始化实体管理器
	em := ecs.NewEntityManager()

	// 加载字体
	font, err := rm.LoadFont("assets/fonts/SimHei.ttf", 18)
	if err != nil {
		return nil, fmt.Errorf("加载字体失败: %w", err)
	}

	titleFont, err := rm.LoadFont("assets/fonts/SimHei.ttf", 22)
	if err != nil {
		return nil, fmt.Errorf("加载标题字体失败: %w", err)
	}

	buttonFont, err := rm.LoadFont("assets/fonts/SimHei.ttf", 20)
	if err != nil {
		return nil, fmt.Errorf("加载按钮字体失败: %w", err)
	}

	// 初始化系统
	textInputSystem := systems.NewTextInputSystem(em)
	textInputRenderSystem := systems.NewTextInputRenderSystem(em, font)
	dialogRenderSystem := systems.NewDialogRenderSystem(em, screenWidth, screenHeight, titleFont, font, buttonFont)
	dialogInputSystem := systems.NewDialogInputSystem(em)

	game := &TestGame{
		resourceManager:       rm,
		entityManager:         em,
		textInputSystem:       textInputSystem,
		textInputRenderSystem: textInputRenderSystem,
		dialogRenderSystem:    dialogRenderSystem,
		dialogInputSystem:     dialogInputSystem,
		resultReceived:        false,
	}

	// 创建"新用户"对话框
	if err := game.createNewUserDialog(); err != nil {
		return nil, fmt.Errorf("创建对话框失败: %w", err)
	}

	return game, nil
}

// createNewUserDialog 创建新用户对话框
func (g *TestGame) createNewUserDialog() error {
	dialogID, inputBoxID, err := entities.NewNewUserDialogEntity(
		g.entityManager,
		g.resourceManager,
		screenWidth,
		screenHeight,
		func(result entities.NewUserDialogResult) {
			if result.Confirmed {
				g.resultText = fmt.Sprintf("欢迎, %s!", result.Username)
				log.Printf("用户确认，输入的名字: %s", result.Username)
			} else {
				g.resultText = "已取消"
				log.Printf("用户取消")
			}
			g.resultReceived = true

			// 关闭对话框
			g.entityManager.DestroyEntity(g.dialogEntity)
			g.entityManager.DestroyEntity(g.inputBoxEntity)
		},
	)

	if err != nil {
		return err
	}

	g.dialogEntity = dialogID
	g.inputBoxEntity = inputBoxID

	log.Printf("新用户对话框已创建 (dialog=%d, inputBox=%d)", dialogID, inputBoxID)
	return nil
}

// Update 更新游戏逻辑
func (g *TestGame) Update() error {
	// 处理 ESC 键退出
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		return ebiten.Termination
	}

	// 如果已收到结果，显示3秒后退出
	if g.resultReceived {
		log.Println(g.resultText)
		return ebiten.Termination
	}

	// 更新系统
	deltaTime := 1.0 / 60.0 // 60 FPS
	g.textInputSystem.Update(deltaTime)
	g.dialogInputSystem.Update(deltaTime)

	// 清理已标记删除的实体
	g.entityManager.RemoveMarkedEntities()

	return nil
}

// Draw 绘制游戏画面
func (g *TestGame) Draw(screen *ebiten.Image) {
	// 清屏（深蓝色背景）
	screen.Fill(color.RGBA{R: 25, G: 25, B: 112, A: 255})

	// 绘制对话框
	g.dialogRenderSystem.Draw(screen)

	// 绘制输入框
	g.textInputRenderSystem.Draw(screen)
}

// Layout 设置窗口布局
func (g *TestGame) Layout(outsideWidth, outsideHeight int) (int, int) {
	return screenWidth, screenHeight
}

func main() {
	// 创建测试游戏
	testGame, err := NewTestGame()
	if err != nil {
		log.Fatalf("创建测试游戏失败: %v", err)
	}

	// 设置窗口
	ebiten.SetWindowSize(screenWidth, screenHeight)
	ebiten.SetWindowTitle("文本输入框测试 - Text Input Test")
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)

	// 运行游戏
	if err := ebiten.RunGame(testGame); err != nil {
		log.Fatalf("运行游戏失败: %v", err)
	}
}
