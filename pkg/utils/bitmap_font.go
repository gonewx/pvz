package utils

import (
	"fmt"
	"image"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
)

// BitmapFont 位图字体
// 用于渲染原版 PVZ 的位图字体（如 HouseofTerror28）
type BitmapFont struct {
	Image      *ebiten.Image     // 字体图集
	CharMap    map[rune]CharInfo // 字符 → 字符信息映射
	LineHeight int               // 行高（像素）
}

// CharInfo 单个字符的信息
type CharInfo struct {
	Rect  image.Rectangle // 字符在图集中的矩形区域
	Width int             // 字符宽度（像素）
}

// LoadBitmapFont 加载位图字体
// 参数：
//   - imagePath: PNG 图集路径（如 "assets/data/HouseofTerror28.png"）
//   - metaPath: TXT 元数据路径（如 "assets/data/HouseofTerror28.txt"）
//
// 返回：
//   - *BitmapFont: 字体实例
//   - error: 加载失败时返回错误
func LoadBitmapFont(imagePath, metaPath string) (*BitmapFont, error) {
	// 加载图集
	img, err := loadImage(imagePath)
	if err != nil {
		return nil, fmt.Errorf("failed to load font image: %w", err)
	}

	// 解析元数据
	charList, widthList, rectList, err := parseFontMeta(metaPath)
	if err != nil {
		return nil, fmt.Errorf("failed to parse font meta: %w", err)
	}

	// 验证数据一致性
	// 注意：WidthList 和 RectList 的第一个元素对应空字符，可能比 CharList 多一个
	// 如果 widthList 比 charList 多 1个，删除第一个元素
	if len(widthList) == len(charList)+1 && len(rectList) == len(charList)+1 {
		widthList = widthList[1:]
		rectList = rectList[1:]
	}

	if len(charList) != len(widthList) || len(charList) != len(rectList) {
		return nil, fmt.Errorf("inconsistent font data: chars=%d, widths=%d, rects=%d",
			len(charList), len(widthList), len(rectList))
	}

	// 构建字符映射
	charMap := make(map[rune]CharInfo)
	for i, char := range charList {
		charMap[char] = CharInfo{
			Rect:  rectList[i],
			Width: widthList[i],
		}
	}

	// 计算行高（使用第一个非空字符的高度）
	lineHeight := 54 // 默认值
	if len(rectList) > 1 {
		lineHeight = rectList[1].Dy()
	}

	return &BitmapFont{
		Image:      img,
		CharMap:    charMap,
		LineHeight: lineHeight,
	}, nil
}

// DrawText 绘制文本
// 参数：
//   - screen: 绘制目标
//   - text: 文本内容
//   - x, y: 文本位置（左上角）
//   - align: 对齐方式（"left", "center", "right"）
func (bf *BitmapFont) DrawText(screen *ebiten.Image, text string, x, y float64, align string) {
	// 计算文本总宽度（用于居中对齐）
	totalWidth := bf.MeasureText(text)

	// 根据对齐调整起始X坐标
	startX := x
	switch align {
	case "center":
		startX = x - float64(totalWidth)/2
	case "right":
		startX = x - float64(totalWidth)
	}

	log.Printf("[BitmapFont] DrawText: text='%s', align=%s, x=%.0f, y=%.0f, startX=%.0f, totalWidth=%d",
		text, align, x, y, startX, totalWidth)

	// 逐字符绘制
	currentX := startX
	for i, char := range text {
		charInfo, ok := bf.CharMap[char]
		if !ok {
			// 字符不存在，跳过
			log.Printf("[BitmapFont] Character '%c' (U+%04X) not found in font", char, char)
			continue
		}

		// 从图集中提取字符图像
		charImg := bf.Image.SubImage(charInfo.Rect).(*ebiten.Image)

		// 绘制字符
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(currentX, y)
		screen.DrawImage(charImg, op)

		if i == 0 {
			log.Printf("[BitmapFont] First char '%c': rect=%v, width=%d, drawing at (%.0f, %.0f)",
				char, charInfo.Rect, charInfo.Width, currentX, y)
		}

		// 移动到下一个字符位置
		currentX += float64(charInfo.Width)
	}

	log.Printf("[BitmapFont] Drew %d characters", len([]rune(text)))
}

// MeasureText 测量文本宽度
// 参数：
//   - text: 要测量的文本
//
// 返回：
//   - int: 文本宽度（像素）
func (bf *BitmapFont) MeasureText(text string) int {
	totalWidth := 0
	for _, char := range text {
		if charInfo, ok := bf.CharMap[char]; ok {
			totalWidth += charInfo.Width
		}
	}
	return totalWidth
}

// loadImage 加载图像文件
func loadImage(path string) (*ebiten.Image, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	img, _, err := image.Decode(file)
	if err != nil {
		return nil, err
	}

	return ebiten.NewImageFromImage(img), nil
}

// parseFontMeta 解析字体元数据文件
func parseFontMeta(path string) ([]rune, []int, []image.Rectangle, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, nil, err
	}

	content := string(data)

	// 解析 CharList
	charList, err := parseCharList(content)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to parse CharList: %w", err)
	}

	// 解析 WidthList
	widthList, err := parseIntList(content, "WidthList")
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to parse WidthList: %w", err)
	}

	// 解析 RectList
	rectList, err := parseRectList(content)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to parse RectList: %w", err)
	}

	return charList, widthList, rectList, nil
}

// parseCharList 解析字符列表
// 格式：('A', 'B', 'C', ...)
func parseCharList(content string) ([]rune, error) {
	// 查找 Define CharList 部分（使用 (?s) 开启单行模式，. 匹配换行符）
	re := regexp.MustCompile(`(?s)Define CharList\s*\(\s*(.+?)\);`)
	matches := re.FindStringSubmatch(content)
	if len(matches) < 2 {
		return nil, fmt.Errorf("CharList not found")
	}

	charData := matches[1]

	// 提取所有字符（单引号包围）
	charRe := regexp.MustCompile(`'([^']*)'`)
	charMatches := charRe.FindAllStringSubmatch(charData, -1)

	chars := make([]rune, 0, len(charMatches))
	for _, match := range charMatches {
		charStr := match[1]
		if charStr == "" {
			// 空字符（空格）
			chars = append(chars, ' ')
		} else {
			// 取第一个字符
			for _, r := range charStr {
				chars = append(chars, r)
				break
			}
		}
	}

	return chars, nil
}

// parseIntList 解析整数列表
// 格式：(1, 2, 3, ...)
func parseIntList(content, listName string) ([]int, error) {
	// 查找指定列表（支持跨行）
	re := regexp.MustCompile(`(?s)` + listName + `\s*\(\s*(.+?)\);`)
	matches := re.FindStringSubmatch(content)
	if len(matches) < 2 {
		return nil, fmt.Errorf("%s not found", listName)
	}

	listData := matches[1]

	// 分割并解析整数
	parts := strings.Split(listData, ",")
	ints := make([]int, 0, len(parts))

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		val, err := strconv.Atoi(part)
		if err != nil {
			return nil, fmt.Errorf("invalid integer: %s", part)
		}
		ints = append(ints, val)
	}

	return ints, nil
}

// parseRectList 解析矩形列表
// 格式：((x, y, w, h), ...)
func parseRectList(content string) ([]image.Rectangle, error) {
	// 查找 Define RectList 部分
	re := regexp.MustCompile(`Define RectList\s*\(\s*([^;]+)\);`)
	matches := re.FindStringSubmatch(content)
	if len(matches) < 2 {
		return nil, fmt.Errorf("RectList not found")
	}

	rectData := matches[1]

	// 提取所有矩形 (x, y, w, h)
	rectRe := regexp.MustCompile(`\(\s*(\d+),\s*(\d+),\s*(\d+),\s*(\d+)\)`)
	rectMatches := rectRe.FindAllStringSubmatch(rectData, -1)

	rects := make([]image.Rectangle, 0, len(rectMatches))
	for _, match := range rectMatches {
		x, _ := strconv.Atoi(match[1])
		y, _ := strconv.Atoi(match[2])
		w, _ := strconv.Atoi(match[3])
		h, _ := strconv.Atoi(match[4])

		rects = append(rects, image.Rect(x, y, x+w, y+h))
	}

	return rects, nil
}
