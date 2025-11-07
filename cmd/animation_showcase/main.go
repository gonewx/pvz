// cmd/animation_showcase/main.go
// 动画展示系统主程序
//
// 用法：
//   go run cmd/animation_showcase/*.go --config=cmd/animation_showcase/config.yaml

package main

import (
	"bytes"
	"flag"
	"fmt"
	"image/color"
	_ "image/jpeg" // 支持 JPEG 格式图片
	"log"
	"os"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

var (
	configPath = flag.String("config", "cmd/animation_showcase/config.yaml", "配置文件路径")
	verbose    = flag.Bool("verbose", false, "详细日志")
)

// HelpPosition 帮助面板位置
type HelpPosition int

const (
	HelpTopRight HelpPosition = iota
	HelpTopLeft
	HelpBottomRight
	HelpBottomLeft
)

// Game 主游戏结构
type Game struct {
	config *ShowcaseConfig
	layout *GridLayout

	// UI 状态
	showHelp     bool
	helpPosition HelpPosition // 帮助面板位置

	// 分页
	allAnimConfigs []AnimationUnitConfig // 所有动画配置
	currentPage    int                   // 当前页码（从0开始）
	totalPages     int                   // 总页数
	cellsPerPage   int                   // 每页单元数

	// 字体
	textFont *text.GoTextFace // 中文字体

	// 重用的渲染对象（避免每帧分配）
	textDrawOpts text.DrawOptions
}

// NewGame 创建游戏实例
func NewGame(configPath string) (*Game, error) {
	// 加载配置
	config, err := LoadConfig(configPath)
	if err != nil {
		return nil, fmt.Errorf("加载配置失败: %w", err)
	}

	log.Printf("✓ 加载配置成功: %d 个动画单元", len(config.Animations))

	// 计算分页参数（从配置读取）
	rowsPerPage := config.Global.Grid.RowsPerPage
	cellsPerPage := rowsPerPage * config.Global.Grid.Columns
	totalPages := (len(config.Animations) + cellsPerPage - 1) / cellsPerPage

	log.Printf("✓ 分页配置: 每页 %d 行 × %d 列 = %d 个单元, 共 %d 页",
		rowsPerPage, config.Global.Grid.Columns, cellsPerPage, totalPages)

	// 加载中文字体
	font, err := loadFont("assets/fonts/SimHei.ttf", 14)
	if err != nil {
		log.Printf("警告: 无法加载中文字体: %v (将使用默认字体)", err)
	} else {
		log.Printf("✓ 加载中文字体: SimHei.ttf (14px)")
	}

	game := &Game{
		config:         config,
		allAnimConfigs: config.Animations,
		currentPage:    0,
		totalPages:     totalPages,
		cellsPerPage:   cellsPerPage,
		showHelp:       true,
		helpPosition:   HelpTopRight, // 默认右上角
		textFont:       font,
	}

	// 加载第一页
	if err := game.loadPage(0); err != nil {
		return nil, err
	}

	return game, nil
}

// loadPage 加载指定页的动画
func (g *Game) loadPage(pageNum int) error {
	if pageNum < 0 || pageNum >= g.totalPages {
		return fmt.Errorf("页码超出范围: %d (总页数: %d)", pageNum, g.totalPages)
	}

	log.Printf("=== 加载第 %d/%d 页 ===", pageNum+1, g.totalPages)

	// 计算当前页的动画范围
	startIdx := pageNum * g.cellsPerPage
	endIdx := startIdx + g.cellsPerPage
	if endIdx > len(g.allAnimConfigs) {
		endIdx = len(g.allAnimConfigs)
	}

	// 创建当前页的动画单元
	cells := make([]*AnimationCell, 0, endIdx-startIdx)
	for i := startIdx; i < endIdx; i++ {
		cell, err := NewAnimationCell(&g.allAnimConfigs[i], g.config.Global.Playback.FPS)
		if err != nil {
			log.Printf("  警告: 无法加载动画单元 [%s]: %v", g.allAnimConfigs[i].Name, err)
			continue
		}
		cells = append(cells, cell)
		if *verbose {
			log.Printf("  ✓ 加载: %s (%s)", cell.GetName(), cell.GetCurrentAnimationName())
		}
	}

	if len(cells) == 0 {
		return fmt.Errorf("没有成功加载任何动画单元")
	}

	log.Printf("✓ 成功加载 %d 个动画单元", len(cells))

	// 创建网格布局
	g.layout = NewGridLayout(
		&g.config.Global.Grid,
		cells,
		g.config.Global.Window.Width,
		g.config.Global.Window.Height,
		g.textFont,
	)

	g.currentPage = pageNum

	return nil
}

// Update 更新游戏状态
func (g *Game) Update() error {
	// 处理帮助切换
	if inpututil.IsKeyJustPressed(ebiten.KeyH) {
		g.showHelp = !g.showHelp
	}

	// 处理帮助位置切换（Tab 键）
	if inpututil.IsKeyJustPressed(ebiten.KeyTab) {
		g.helpPosition = (g.helpPosition + 1) % 4
		if *verbose {
			positions := []string{"右上角", "左上角", "右下角", "左下角"}
			log.Printf("帮助面板移动到: %s", positions[g.helpPosition])
		}
	}

	// 处理翻页（只用 PageDown/PageUp）
	if inpututil.IsKeyJustPressed(ebiten.KeyPageDown) {
		// 下一页
		if g.currentPage < g.totalPages-1 {
			if err := g.loadPage(g.currentPage + 1); err != nil {
				log.Printf("警告: 加载下一页失败: %v", err)
			}
		}
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyPageUp) {
		// 上一页
		if g.currentPage > 0 {
			if err := g.loadPage(g.currentPage - 1); err != nil {
				log.Printf("警告: 加载上一页失败: %v", err)
			}
		}
	}

	// 处理方向键切换选中单元的动画
	if inpututil.IsKeyJustPressed(ebiten.KeyRight) {
		// 右方向键：切换到下一个动画
		selectedIndex := g.layout.GetSelectedIndex()
		if selectedIndex >= 0 {
			cell := g.layout.GetCell(selectedIndex)
			if cell != nil {
				cell.NextAnimation()
				if *verbose {
					log.Printf("→ 切换动画: %s -> %s", cell.GetName(), cell.GetCurrentAnimationName())
				}
			}
		}
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyLeft) {
		// 左方向键：切换到上一个动画（循环）
		selectedIndex := g.layout.GetSelectedIndex()
		if selectedIndex >= 0 {
			cell := g.layout.GetCell(selectedIndex)
			if cell != nil {
				cell.PrevAnimation()
				if *verbose {
					log.Printf("← 切换动画: %s -> %s", cell.GetName(), cell.GetCurrentAnimationName())
				}
			}
		}
	}

	// 数字键快速跳转
	for key := ebiten.Key0; key <= ebiten.Key9; key++ {
		if inpututil.IsKeyJustPressed(key) {
			pageNum := int(key - ebiten.Key0)
			if key == ebiten.Key0 {
				pageNum = 10 // 0 键代表第 10 页
			}
			pageNum-- // 转换为从 0 开始的索引
			if pageNum >= 0 && pageNum < g.totalPages {
				if err := g.loadPage(pageNum); err != nil {
					log.Printf("警告: 跳转到第 %d 页失败: %v", pageNum+1, err)
				}
			}
		}
	}

	// 处理鼠标点击
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		x, y := ebiten.CursorPosition()
		cellIndex := g.layout.GetCellAt(x, y)

		if cellIndex >= 0 {
			cell := g.layout.GetCell(cellIndex)
			if cell != nil {
				// 左键点击：切换动画
				cell.NextAnimation()
				g.layout.SetSelectedIndex(cellIndex)

				if *verbose {
					log.Printf("点击单元 #%d: %s -> %s", cellIndex, cell.GetName(), cell.GetCurrentAnimationName())
				}
			}
		}
	}

	// 处理右键点击（切换详情模式 - 为未来扩展保留）
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonRight) {
		x, y := ebiten.CursorPosition()
		cellIndex := g.layout.GetCellAt(x, y)

		if cellIndex >= 0 {
			cell := g.layout.GetCell(cellIndex)
			if cell != nil {
				cell.ToggleDetailMode()
				if *verbose {
					log.Printf("切换详情模式: %s -> %v", cell.GetName(), cell.IsDetailMode())
				}
			}
		}
	}

	// 更新布局和动画
	g.layout.Update()

	return nil
}

// Draw 绘制游戏画面
func (g *Game) Draw(screen *ebiten.Image) {
	// 清空屏幕
	screen.Fill(color.RGBA{50, 50, 50, 255})

	// 渲染网格布局
	g.layout.Render(screen)

	// 渲染顶部信息栏
	g.drawInfoBar(screen)

	// 渲染帮助信息
	if g.showHelp {
		g.drawHelp(screen)
	}
}

// drawInfoBar 绘制顶部信息栏
func (g *Game) drawInfoBar(screen *ebiten.Image) {
	info := fmt.Sprintf("FPS: %.1f | 第 %d/%d 页 | 当前页: %d 个动画 | 选中: ",
		ebiten.ActualTPS(), g.currentPage+1, g.totalPages, g.layout.GetCellCount())

	selectedIndex := g.layout.GetSelectedIndex()
	if selectedIndex >= 0 {
		cell := g.layout.GetCell(selectedIndex)
		if cell != nil {
			info += fmt.Sprintf("%s (%s)", cell.GetName(), cell.GetCurrentAnimationName())
		}
	} else {
		info += "无"
	}

	// 绘制半透明背景
	bgWidth := 800
	bgHeight := 25
	bgImage := ebiten.NewImage(bgWidth, bgHeight)
	bgImage.Fill(color.RGBA{0, 0, 0, 160})
	screen.DrawImage(bgImage, &ebiten.DrawImageOptions{})

	// 绘制文本（重用 DrawOptions）
	if g.textFont != nil {
		g.textDrawOpts.GeoM.Reset()
		g.textDrawOpts.GeoM.Translate(10, 10)
		g.textDrawOpts.ColorScale.Reset()
		g.textDrawOpts.ColorScale.ScaleWithColor(color.White)
		text.Draw(screen, info, g.textFont, &g.textDrawOpts)
	} else {
		ebitenutil.DebugPrintAt(screen, info, 10, 10)
	}
}

// drawHelp 绘制帮助信息
func (g *Game) drawHelp(screen *ebiten.Image) {
	helpLines := []string{
		"操作说明:",
		"  PageDown    - 下一页",
		"  PageUp      - 上一页",
		"  1-9 数字键  - 快速跳转页面",
		"  →/← 方向键  - 切换选中单元的动画",
		"  左键点击    - 选中并切换动画",
		"  H          - 显示/隐藏帮助",
		"  Tab        - 切换帮助位置",
		"  ESC        - 退出",
	}

	// 计算帮助面板大小
	helpWidth := 370
	helpHeight := 195

	// 根据位置计算坐标
	var helpX, helpY int
	switch g.helpPosition {
	case HelpTopRight:
		helpX = g.config.Global.Window.Width - helpWidth - 20
		helpY = 50
	case HelpTopLeft:
		helpX = 20
		helpY = 50
	case HelpBottomRight:
		helpX = g.config.Global.Window.Width - helpWidth - 20
		helpY = g.config.Global.Window.Height - helpHeight - 20
	case HelpBottomLeft:
		helpX = 20
		helpY = g.config.Global.Window.Height - helpHeight - 20
	}

	// 绘制半透明背景
	bgImage := ebiten.NewImage(helpWidth, helpHeight)
	bgImage.Fill(color.RGBA{0, 0, 0, 180})

	opts := &ebiten.DrawImageOptions{}
	opts.GeoM.Translate(float64(helpX), float64(helpY))
	screen.DrawImage(bgImage, opts)

	// 绘制帮助文本（重用 DrawOptions）
	if g.textFont != nil {
		for i, line := range helpLines {
			g.textDrawOpts.GeoM.Reset()
			g.textDrawOpts.GeoM.Translate(float64(helpX+10), float64(helpY+15+i*18))
			g.textDrawOpts.ColorScale.Reset()
			g.textDrawOpts.ColorScale.ScaleWithColor(color.White)
			text.Draw(screen, line, g.textFont, &g.textDrawOpts)
		}
	} else {
		// 降级到默认字体
		help := ""
		for _, line := range helpLines {
			help += line + "\n"
		}
		ebitenutil.DebugPrintAt(screen, help, helpX+10, helpY+10)
	}
}

// Layout 设置窗口布局
func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return g.config.Global.Window.Width, g.config.Global.Window.Height
}

func main() {
	flag.Parse()

	if *verbose {
		log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
	}

	log.Println("=== 动画展示系统启动 ===")

	game, err := NewGame(*configPath)
	if err != nil {
		log.Fatalf("初始化失败: %v", err)
	}

	// 设置窗口
	ebiten.SetWindowSize(game.config.Global.Window.Width, game.config.Global.Window.Height)
	ebiten.SetWindowTitle(game.config.Global.Window.Title)
	// 注意：TPS 应该保持 60 以确保流畅的输入响应和渲染
	// 配置文件中的 fps 参数用于控制动画播放速度，而非游戏更新频率
	ebiten.SetTPS(60)

	log.Printf("✓ 窗口配置: %dx%d @ 60 TPS (动画速度: %d FPS)",
		game.config.Global.Window.Width,
		game.config.Global.Window.Height,
		game.config.Global.Playback.FPS,
	)
	log.Println("=== 启动完成，开始运行 ===")

	// 运行游戏
	if err := ebiten.RunGame(game); err != nil {
		log.Fatal(err)
	}
}

// loadFont 加载字体文件
func loadFont(path string, size float64) (*text.GoTextFace, error) {
	// 读取字体文件
	fontData, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("无法读取字体文件 %s: %w", path, err)
	}

	// 创建字体源
	source, err := text.NewGoTextFaceSource(bytes.NewReader(fontData))
	if err != nil {
		return nil, fmt.Errorf("无法创建字体源 %s: %w", path, err)
	}

	// 创建字体 face
	goTextFace := &text.GoTextFace{
		Source:    source,
		Size:      size,
		Direction: text.DirectionLeftToRight,
	}

	return goTextFace, nil
}
