// cmd/animation_showcase/grid_layout.go
// 网格布局管理器 - 管理动画单元的布局和滚动

package main

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// GridLayout 网格布局管理器
type GridLayout struct {
	config *GridConfig

	// 动画单元列表
	cells []*AnimationCell

	// 布局数据
	columns    int
	cellWidth  int
	cellHeight int
	padding    int

	// 滚动
	scrollY      float64
	scrollSpeed  float64
	maxScrollY   float64

	// 窗口大小
	windowWidth  int
	windowHeight int

	// 选中的单元索引
	selectedIndex int

	// 字体
	textFont *text.GoTextFace

	// 重用的渲染对象（避免每帧分配）
	textDrawOpts text.DrawOptions
}

// NewGridLayout 创建网格布局管理器
func NewGridLayout(config *GridConfig, cells []*AnimationCell, windowWidth, windowHeight int, textFont *text.GoTextFace) *GridLayout {
	layout := &GridLayout{
		config:        config,
		cells:         cells,
		columns:       config.Columns,
		cellWidth:     config.CellWidth,
		cellHeight:    config.CellHeight,
		padding:       config.Padding,
		scrollY:       0,
		scrollSpeed:   float64(config.ScrollSpeed),
		windowWidth:   windowWidth,
		windowHeight:  windowHeight,
		selectedIndex: -1,
		textFont:      textFont,
	}

	layout.calculateMaxScroll()

	return layout
}

// calculateMaxScroll 计算最大滚动距离
func (g *GridLayout) calculateMaxScroll() {
	totalRows := (len(g.cells) + g.columns - 1) / g.columns
	totalHeight := totalRows * (g.cellHeight + g.padding)
	g.maxScrollY = float64(totalHeight - g.windowHeight + g.padding)

	if g.maxScrollY < 0 {
		g.maxScrollY = 0
	}
}

// Update 更新布局
func (g *GridLayout) Update() {
	// 更新所有单元的动画（分页模式下不再需要可见性检查）
	for _, cell := range g.cells {
		cell.Update()
	}
}

// Render 渲染网格布局
func (g *GridLayout) Render(screen *ebiten.Image) {
	// 渲染背景网格
	for i, cell := range g.cells {
		x, y := g.getCellPosition(i)

		// 绘制单元背景
		cellColor := color.RGBA{240, 240, 240, 255}
		if i == g.selectedIndex {
			cellColor = color.RGBA{255, 255, 200, 255} // 高亮选中
		}

		vector.DrawFilledRect(
			screen,
			float32(x),
			float32(y),
			float32(g.cellWidth),
			float32(g.cellHeight),
			cellColor,
			false,
		)

		// 绘制边框
		vector.StrokeRect(
			screen,
			float32(x),
			float32(y),
			float32(g.cellWidth),
			float32(g.cellHeight),
			2,
			color.RGBA{200, 200, 200, 255},
			false,
		)

		// 以单元格左上角为原点渲染动画
		cell.Render(screen, x, y)

		// 绘制文本背景（深色半透明）
		textBgY := y + float64(g.cellHeight) - 35
		textBgHeight := 35.0
		vector.DrawFilledRect(
			screen,
			float32(x),
			float32(textBgY),
			float32(g.cellWidth),
			float32(textBgHeight),
			color.RGBA{0, 0, 0, 160}, // 黑色半透明背景
			false,
		)

		// 绘制单元信息（白色文本在深色背景上）
		textY := y + float64(g.cellHeight) - 30

		if g.textFont != nil {
			// 使用中文字体渲染（重用 DrawOptions 对象）
			g.textDrawOpts.GeoM.Reset()
			g.textDrawOpts.GeoM.Translate(x+5, textY)
			g.textDrawOpts.ColorScale.Reset()
			g.textDrawOpts.ColorScale.ScaleWithColor(color.White)
			text.Draw(screen, cell.GetName(), g.textFont, &g.textDrawOpts)

			animInfo := cell.GetCurrentAnimationName()
			g.textDrawOpts.GeoM.Reset()
			g.textDrawOpts.GeoM.Translate(x+5, textY+14)
			text.Draw(screen, animInfo, g.textFont, &g.textDrawOpts)
		} else {
			// 降级到默认字体
			ebitenutil.DebugPrintAt(screen, cell.GetName(), int(x)+5, int(textY))
			animInfo := cell.GetCurrentAnimationName()
			ebitenutil.DebugPrintAt(screen, animInfo, int(x)+5, int(textY)+12)
		}
	}
}

// getCellPosition 获取指定索引单元的位置
func (g *GridLayout) getCellPosition(index int) (float64, float64) {
	row := index / g.columns
	col := index % g.columns

	x := float64(col*(g.cellWidth+g.padding) + g.padding)
	y := float64(row*(g.cellHeight+g.padding) + g.padding)

	return x, y
}

// isCellVisible 检查单元是否在可见区域内
func (g *GridLayout) isCellVisible(index int) bool {
	_, y := g.getCellPosition(index)
	y -= g.scrollY

	return y+float64(g.cellHeight) >= 0 && y <= float64(g.windowHeight)
}

// GetCellAt 获取指定屏幕坐标处的单元索引
func (g *GridLayout) GetCellAt(screenX, screenY int) int {
	for i := range g.cells {
		x, y := g.getCellPosition(i)

		if float64(screenX) >= x && float64(screenX) <= x+float64(g.cellWidth) &&
			float64(screenY) >= y && float64(screenY) <= y+float64(g.cellHeight) {
			return i
		}
	}

	return -1
}

// SetSelectedIndex 设置选中的单元索引
func (g *GridLayout) SetSelectedIndex(index int) {
	if index >= 0 && index < len(g.cells) {
		g.selectedIndex = index
	}
}

// GetSelectedIndex 获取选中的单元索引
func (g *GridLayout) GetSelectedIndex() int {
	return g.selectedIndex
}

// GetCell 获取指定索引的单元
func (g *GridLayout) GetCell(index int) *AnimationCell {
	if index >= 0 && index < len(g.cells) {
		return g.cells[index]
	}
	return nil
}

// GetCellCount 获取单元总数
func (g *GridLayout) GetCellCount() int {
	return len(g.cells)
}
