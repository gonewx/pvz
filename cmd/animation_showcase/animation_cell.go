// cmd/animation_showcase/animation_cell.go
// 动画展示单元 - 管理单个 Reanim 动画的加载、播放和渲染
//
// 坐标系统说明（Story 16.3 统一化）：
//   - 使用中心锚点方案（与游戏系统 pkg/systems/render_reanim.go 一致）
//   - CenterOffset 在初始化时计算一次，基于第一帧所有可见部件的 bounding box 中心
//   - 渲染公式：渲染原点 = 中心 - CenterOffset，部件坐标 = 渲染原点 + frame.X
//   - 与游戏系统使用相同的坐标转换逻辑，降低认知负担，便于调试
//
// 参考：
//   - pkg/utils/coordinates.go - 坐标转换工具库
//   - pkg/systems/reanim_helpers.go:calculateCenterOffset - CenterOffset 计算逻辑
//   - docs/architecture/adr/001-coordinate-transformation-library.md - 坐标系统设计文档

package main

import (
	"fmt"
	"image"
	"image/color"
	_ "image/jpeg" // 支持 JPEG 格式图片
	_ "image/png"  // 支持 PNG 格式图片
	"log"
	"math"
	"os"
	"path/filepath"
	"strings"

	"github.com/decker502/pvz/internal/reanim"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

// AnimationCell 动画展示单元
type AnimationCell struct {
	config *AnimationUnitConfig

	// Reanim 数据
	reanimXML    *reanim.ReanimXML
	mergedTracks map[string][]reanim.Frame

	// 图片资源
	partImages map[string]*ebiten.Image

	// 可切换的动画项（包含普通动画和组合动画）
	switchableAnimations  []SwitchableAnimation
	currentAnimationIndex int // 当前播放的动画索引（用于切换）

	// 当前播放状态
	currentFrame     int
	frameAccumulator float64 // 帧累加器，用于控制动画速度
	animationFPS     float64 // 动画播放帧率（从 reanim 文件读取）
	targetTPS        float64 // 目标游戏 TPS（从配置文件读取）

	// 动画播放数据
	visualTracks          []string
	logicalTracks         []string
	currentAnimations     []string          // 当前播放的动画列表
	animVisiblesMap       map[string][]int  // 每个动画的可见性数组
	trackAnimationBinding map[string]string // 轨道到动画的绑定
	parentTracks          map[string]string // 父子关系
	hiddenTracks          map[string]bool   // 隐藏的轨道
	manualHiddenTracks    map[string]bool   // 手动隐藏的轨道（用于单个模式）

	// 渲染缓存（优化性能）
	lastRenderFrame  int
	cachedRenderData []renderPartData

	// 渲染配置
	scale   float64
	centerX float64
	centerY float64

	// 中心锚点偏移（用于对齐游戏系统坐标方案）
	// 在初始化时计算一次，基于第一帧所有可见部件的 bounding box 中心
	// 与游戏系统 pkg/systems/reanim_helpers.go:calculateCenterOffset 的逻辑一致
	centerOffsetX float64
	centerOffsetY float64

	// 详情模式（点击后显示）
	detailMode bool
}

// renderPartData 缓存的渲染数据
type renderPartData struct {
	img              *ebiten.Image
	frame            reanim.Frame
	offsetX, offsetY float64
}

// SwitchableAnimation 可切换的动画项
type SwitchableAnimation struct {
	Name        string   // 动画名称或组合名称
	DisplayName string   // 显示名称
	IsCombo     bool     // 是否为组合动画
	Animations  []string // 如果是组合，包含的动画列表；如果是单个，只有一个元素
}

// NewAnimationCell 创建动画展示单元
// config: 动画单元配置
// globalFPS: 全局默认帧率（仅当 reanim 文件未指定时使用）
// targetTPS: 目标游戏 TPS（从配置文件读取）
func NewAnimationCell(config *AnimationUnitConfig, globalFPS int, targetTPS int) (*AnimationCell, error) {
	// 加载 Reanim 文件
	reanimXML, err := reanim.ParseReanimFile(config.ReanimFile)
	if err != nil {
		return nil, fmt.Errorf("加载 Reanim 文件失败 [%s]: %w", config.ReanimFile, err)
	}

	// 确定动画帧率：优先使用 Reanim 文件中的定义，否则使用全局配置
	animFPS := reanimXML.FPS
	if animFPS <= 0 {
		animFPS = globalFPS
	}
	if animFPS <= 0 {
		animFPS = 12 // 最后的默认值
	}

	// 加载图片资源（支持 JPG+PNG 蒙版）
	partImages := make(map[string]*ebiten.Image)
	for ref, path := range config.Images {
		img, err := loadImageWithMask(path)
		if err != nil {
			log.Printf("  警告: 无法加载图片 %s: %v", path, err)
			continue
		}
		partImages[ref] = img
	}

	// 构建合并轨道
	mergedTracks := reanim.BuildMergedTracks(reanimXML)

	// 分析轨道类型
	visualTracks, logicalTracks := analyzeTrackTypes(reanimXML)

	cell := &AnimationCell{
		config:       config,
		reanimXML:    reanimXML,
		mergedTracks: mergedTracks,
		partImages:   partImages,

		currentFrame:          0,
		currentAnimationIndex: 0,
		frameAccumulator:      0,
		animationFPS:          float64(animFPS),
		targetTPS:             float64(targetTPS),

		visualTracks:       visualTracks,
		logicalTracks:      logicalTracks,
		animVisiblesMap:    make(map[string][]int),
		manualHiddenTracks: make(map[string]bool),

		scale:      config.Scale,
		detailMode: false,
	}

	// 构建可切换的动画列表（包含普通动画和组合动画）
	cell.buildSwitchableAnimations()

	// 计算中心锚点偏移（CenterOffset）
	// 基于第一帧所有可见部件的 bounding box 中心
	// 与游戏系统坐标方案保持一致
	cell.centerOffsetX, cell.centerOffsetY = calculateCenterOffset(
		mergedTracks,
		partImages,
		visualTracks,
	)

	if *verbose {
		log.Printf("[AnimationCell] %s: CenterOffset=(%.1f, %.1f)",
			config.Name, cell.centerOffsetX, cell.centerOffsetY)
	}

	// 设置默认动画并更新索引
	defaultIndex := cell.findAnimationIndex(config.DefaultAnimation)
	if defaultIndex >= 0 {
		cell.currentAnimationIndex = defaultIndex
	}
	cell.SetAnimation(config.DefaultAnimation)

	return cell, nil
}

// findAnimationIndex 查找动画在可切换列表中的索引
func (c *AnimationCell) findAnimationIndex(name string) int {
	for i, anim := range c.switchableAnimations {
		if anim.Name == name {
			return i
		}
	}
	return 0 // 找不到时返回第一个
}

// buildSwitchableAnimations 构建可切换的动画列表
func (c *AnimationCell) buildSwitchableAnimations() {
	c.switchableAnimations = make([]SwitchableAnimation, 0)

	// 添加普通动画
	for _, anim := range c.config.AvailableAnimations {
		c.switchableAnimations = append(c.switchableAnimations, SwitchableAnimation{
			Name:        anim.Name,
			DisplayName: anim.DisplayName,
			IsCombo:     false,
			Animations:  []string{anim.Name},
		})
	}

	// 添加组合动画
	for _, combo := range c.config.AnimationCombos {
		c.switchableAnimations = append(c.switchableAnimations, SwitchableAnimation{
			Name:        combo.Name,
			DisplayName: combo.DisplayName,
			IsCombo:     true,
			Animations:  combo.Animations,
		})
	}
}

// SetAnimation 设置单个动画播放
func (c *AnimationCell) SetAnimation(animName string) {
	c.currentAnimations = []string{animName}
	c.currentFrame = 0
	c.rebuildAnimationData()
}

// SetAnimationCombo 设置多动画组合播放
func (c *AnimationCell) SetAnimationCombo(comboName string) {
	// 查找组合配置
	var combo *AnimationComboConfig
	for i := range c.config.AnimationCombos {
		if c.config.AnimationCombos[i].Name == comboName {
			combo = &c.config.AnimationCombos[i]
			break
		}
	}

	if combo == nil {
		log.Printf("警告: 未找到组合配置 %s", comboName)
		return
	}

	c.currentAnimations = combo.Animations
	c.currentFrame = 0

	// 设置父子关系
	if len(combo.ParentTracks) > 0 {
		c.parentTracks = combo.ParentTracks
	}

	// 设置隐藏轨道
	if len(combo.HiddenTracks) > 0 {
		c.hiddenTracks = make(map[string]bool)
		for _, track := range combo.HiddenTracks {
			c.hiddenTracks[track] = true
		}
	}

	c.rebuildAnimationData()

	// 根据策略构建轨道绑定
	if combo.BindingStrategy == "auto" {
		c.trackAnimationBinding = c.analyzeTrackBinding()
	} else if combo.BindingStrategy == "manual" && len(combo.ManualBindings) > 0 {
		c.trackAnimationBinding = combo.ManualBindings
	}
}

// rebuildAnimationData 重建动画数据（AnimVisiblesMap）
func (c *AnimationCell) rebuildAnimationData() {
	c.animVisiblesMap = make(map[string][]int)

	for _, animName := range c.currentAnimations {
		animVisibles := buildVisiblesArray(c.reanimXML, c.mergedTracks, animName)
		c.animVisiblesMap[animName] = animVisibles
	}
}

// analyzeTrackBinding 自动分析轨道绑定（基于 solution_attack_with_sway 的策略）
func (c *AnimationCell) analyzeTrackBinding() map[string]string {
	binding := make(map[string]string)

	// 1. 分析视觉轨道
	for _, trackName := range c.visualTracks {
		frames, ok := c.mergedTracks[trackName]
		if !ok {
			continue
		}

		var bestAnim string
		var bestScore float64

		for _, animName := range c.currentAnimations {
			animVisibles := c.animVisiblesMap[animName]
			firstVisible, lastVisible := findVisibleWindow(animVisibles)

			if firstVisible < 0 || lastVisible >= len(frames) {
				continue
			}

			// 检查是否有图片
			hasImage := false
			for i := firstVisible; i <= lastVisible && i < len(frames); i++ {
				if frames[i].ImagePath != "" {
					hasImage = true
					break
				}
			}

			if !hasImage {
				continue
			}

			// 计算评分
			variance := calculatePositionVariance(frames, firstVisible, lastVisible)
			score := 1.0 + variance

			if score > bestScore {
				bestScore = score
				bestAnim = animName
			}
		}

		if bestAnim != "" {
			binding[trackName] = bestAnim
		}
	}

	// 2. 分析逻辑轨道
	for _, trackName := range c.logicalTracks {
		frames, ok := c.mergedTracks[trackName]
		if !ok || len(frames) == 0 {
			continue
		}

		var bestAnim string
		var maxVariance float64

		for _, animName := range c.currentAnimations {
			animVisibles := c.animVisiblesMap[animName]
			firstVisible, lastVisible := findVisibleWindow(animVisibles)

			if firstVisible < 0 || lastVisible >= len(frames) {
				continue
			}

			variance := calculatePositionVariance(frames, firstVisible, lastVisible)

			if variance > maxVariance {
				maxVariance = variance
				bestAnim = animName
			}
		}

		if bestAnim != "" && maxVariance > 0.1 {
			binding[trackName] = bestAnim
		}
	}

	return binding
}

// Update 更新动画帧
func (c *AnimationCell) Update() {
	// 使用帧累加器控制动画速度
	// animationFPS: 从 Reanim 文件读取的动画帧率
	// targetTPS: 从配置文件读取的目标游戏 TPS
	//
	// 计算公式：frameAccumulator += animationFPS / targetTPS
	// 例如：animationFPS=12, targetTPS=60 时，每次累加 0.2
	// 累加 5 次（5/60秒）后推进一帧，即每秒推进 12 帧

	c.frameAccumulator += c.animationFPS / c.targetTPS

	if c.frameAccumulator >= 1.0 {
		c.currentFrame++
		c.frameAccumulator -= 1.0
	}
}

// Render 渲染动画到指定画布
// originX, originY: 渲染原点（左上角）坐标
func (c *AnimationCell) Render(canvas *ebiten.Image, originX, originY float64) {
	c.centerX = originX
	c.centerY = originY

	// 使用缓存：如果帧没有变化，使用缓存的渲染数据
	if c.currentFrame != c.lastRenderFrame {
		c.updateRenderCache()
		c.lastRenderFrame = c.currentFrame
	}

	// 快速渲染缓存的部件
	for i := range c.cachedRenderData {
		data := &c.cachedRenderData[i]
		c.drawPart(canvas, data.frame, data.img, c.centerX+data.offsetX, c.centerY+data.offsetY)
	}
}

// updateRenderCache 更新渲染缓存
func (c *AnimationCell) updateRenderCache() {
	// 重用切片避免分配
	c.cachedRenderData = c.cachedRenderData[:0]

	for _, trackName := range c.visualTracks {
		// 检查配置中的隐藏轨道
		if c.hiddenTracks != nil && c.hiddenTracks[trackName] {
			continue
		}
		// 检查手动隐藏的轨道
		if c.manualHiddenTracks != nil && c.manualHiddenTracks[trackName] {
			continue
		}

		controllingAnim, physicalFrame := c.findControllingAnimation(trackName)
		if controllingAnim == "" {
			continue
		}

		mergedFrames, ok := c.mergedTracks[trackName]
		if !ok || physicalFrame >= len(mergedFrames) {
			continue
		}

		frame := mergedFrames[physicalFrame]

		if frame.ImagePath == "" {
			continue
		}

		// 计算父轨道偏移
		offsetX, offsetY := 0.0, 0.0
		if parentTrackName, hasParent := c.parentTracks[trackName]; hasParent {
			childAnimName, _ := c.findControllingAnimation(trackName)
			parentAnimName, _ := c.findControllingAnimation(parentTrackName)

			if childAnimName != parentAnimName && childAnimName != "" && parentAnimName != "" {
				offsetX, offsetY = c.getParentOffset(parentTrackName)
			}
		}

		img, ok := c.partImages[frame.ImagePath]
		if !ok || img == nil {
			continue
		}

		c.cachedRenderData = append(c.cachedRenderData, renderPartData{
			img:     img,
			frame:   frame,
			offsetX: offsetX,
			offsetY: offsetY,
		})
	}
}

// findControllingAnimation 查找控制指定轨道的动画
func (c *AnimationCell) findControllingAnimation(trackName string) (string, int) {
	// 优先使用绑定
	if c.trackAnimationBinding != nil {
		if animName, exists := c.trackAnimationBinding[trackName]; exists {
			animVisibles := c.animVisiblesMap[animName]
			visibleCount := countVisibleFrames(animVisibles)
			if visibleCount > 0 {
				animLogicalFrame := c.currentFrame % visibleCount
				physicalFrame := mapLogicalToPhysical(animLogicalFrame, animVisibles)
				return animName, physicalFrame
			}
		}
	}

	// 默认使用第一个动画
	if len(c.currentAnimations) > 0 {
		animName := c.currentAnimations[0]
		animVisibles := c.animVisiblesMap[animName]
		visibleCount := countVisibleFrames(animVisibles)
		if visibleCount > 0 {
			animLogicalFrame := c.currentFrame % visibleCount
			physicalFrame := mapLogicalToPhysical(animLogicalFrame, animVisibles)
			return animName, physicalFrame
		}
	}

	return "", -1
}

// getParentOffset 获取父轨道的偏移量
func (c *AnimationCell) getParentOffset(parentTrackName string) (float64, float64) {
	parentFrames, ok := c.mergedTracks[parentTrackName]
	if !ok || len(parentFrames) == 0 {
		return 0, 0
	}

	parentAnimName, parentPhysicalFrame := c.findControllingAnimation(parentTrackName)
	if parentAnimName == "" || parentPhysicalFrame < 0 {
		return 0, 0
	}

	parentAnimVisibles := c.animVisiblesMap[parentAnimName]
	firstVisibleFrameIndex := -1
	for i, v := range parentAnimVisibles {
		if v == 0 {
			firstVisibleFrameIndex = i
			break
		}
	}

	if firstVisibleFrameIndex < 0 || firstVisibleFrameIndex >= len(parentFrames) {
		return 0, 0
	}

	initX, initY := 0.0, 0.0
	if parentFrames[firstVisibleFrameIndex].X != nil {
		initX = *parentFrames[firstVisibleFrameIndex].X
	}
	if parentFrames[firstVisibleFrameIndex].Y != nil {
		initY = *parentFrames[firstVisibleFrameIndex].Y
	}

	if parentPhysicalFrame >= len(parentFrames) {
		parentPhysicalFrame = len(parentFrames) - 1
	}
	currentX, currentY := initX, initY
	if parentFrames[parentPhysicalFrame].X != nil {
		currentX = *parentFrames[parentPhysicalFrame].X
	}
	if parentFrames[parentPhysicalFrame].Y != nil {
		currentY = *parentFrames[parentPhysicalFrame].Y
	}

	return currentX - initX, currentY - initY
}

// drawPart 绘制单个部件
// 使用中心锚点方案（与游戏系统一致）：
//   - 渲染原点 = 中心 - CenterOffset * scale
//   - 部件坐标 = 渲染原点 + frame.X * scale
func (ac *AnimationCell) drawPart(canvas *ebiten.Image, frame reanim.Frame, img *ebiten.Image, centerX, centerY float64) {
	bounds := img.Bounds()
	w := bounds.Dx()
	h := bounds.Dy()
	fw := float64(w)
	fh := float64(h)

	// 使用中心锚点方案（与游戏系统坐标方案一致）
	// 渲染原点 = 中心 - CenterOffset
	renderOriginX := centerX - ac.centerOffsetX*ac.scale
	renderOriginY := centerY - ac.centerOffsetY*ac.scale

	// 部件坐标 = 渲染原点 + frame.X
	x := getFloat(frame.X)*ac.scale + renderOriginX
	y := getFloat(frame.Y)*ac.scale + renderOriginY

	scaleX := ac.scale
	scaleY := ac.scale
	if frame.ScaleX != nil {
		scaleX *= *frame.ScaleX
	}
	if frame.ScaleY != nil {
		scaleY *= *frame.ScaleY
	}

	skewX := 0.0
	skewY := 0.0
	if frame.SkewX != nil {
		skewX = *frame.SkewX
	}
	if frame.SkewY != nil {
		skewY = *frame.SkewY
	}

	// 优化：大多数情况下 skew 为 0，避免三角函数计算
	var a, b, c, d float64
	if skewX == 0 && skewY == 0 {
		// 无 skew 的快速路径（最常见）
		a = scaleX
		b = 0
		c = 0
		d = scaleY
	} else {
		// 有 skew 的慢速路径
		skewXRad := skewX * math.Pi / 180.0
		skewYRad := skewY * math.Pi / 180.0
		a = math.Cos(skewXRad) * scaleX
		b = math.Sin(skewXRad) * scaleX
		c = -math.Sin(skewYRad) * scaleY
		d = math.Cos(skewYRad) * scaleY
	}

	x0 := x
	y0 := y
	x1 := a*fw + x
	y1 := b*fw + y
	x2 := c*fh + x
	y2 := d*fh + y
	x3 := a*fw + c*fh + x
	y3 := b*fw + d*fh + y

	vs := []ebiten.Vertex{
		{DstX: float32(x0), DstY: float32(y0), SrcX: 0, SrcY: 0, ColorR: 1, ColorG: 1, ColorB: 1, ColorA: 1},
		{DstX: float32(x1), DstY: float32(y1), SrcX: float32(w), SrcY: 0, ColorR: 1, ColorG: 1, ColorB: 1, ColorA: 1},
		{DstX: float32(x2), DstY: float32(y2), SrcX: 0, SrcY: float32(h), ColorR: 1, ColorG: 1, ColorB: 1, ColorA: 1},
		{DstX: float32(x3), DstY: float32(y3), SrcX: float32(w), SrcY: float32(h), ColorR: 1, ColorG: 1, ColorB: 1, ColorA: 1},
	}
	is := []uint16{0, 1, 2, 1, 3, 2}
	canvas.DrawTriangles(vs, is, img, nil)
}

// NextAnimation 切换到下一个可用的动画
func (c *AnimationCell) NextAnimation() {
	if len(c.switchableAnimations) == 0 {
		return
	}

	c.currentAnimationIndex = (c.currentAnimationIndex + 1) % len(c.switchableAnimations)
	anim := c.switchableAnimations[c.currentAnimationIndex]

	if anim.IsCombo {
		// 播放组合动画
		c.SetAnimationCombo(anim.Name)
		log.Printf("[%s] 切换到组合动画: %s (包含: %v)", c.config.Name, anim.DisplayName, anim.Animations)
	} else {
		// 播放单个动画
		c.SetAnimation(anim.Name)
		log.Printf("[%s] 切换到动画: %s", c.config.Name, anim.DisplayName)
	}
}

// PrevAnimation 切换到上一个动画（循环）
func (c *AnimationCell) PrevAnimation() {
	if len(c.switchableAnimations) == 0 {
		return
	}

	// 向前循环：减1后加上长度再取模，确保结果为正数
	c.currentAnimationIndex = (c.currentAnimationIndex - 1 + len(c.switchableAnimations)) % len(c.switchableAnimations)
	anim := c.switchableAnimations[c.currentAnimationIndex]

	if anim.IsCombo {
		// 播放组合动画
		c.SetAnimationCombo(anim.Name)
		log.Printf("[%s] 切换到组合动画: %s (包含: %v)", c.config.Name, anim.DisplayName, anim.Animations)
	} else {
		// 播放单个动画
		c.SetAnimation(anim.Name)
		log.Printf("[%s] 切换到动画: %s", c.config.Name, anim.DisplayName)
	}
}

// GetCurrentAnimationName 获取当前动画名称
func (c *AnimationCell) GetCurrentAnimationName() string {
	if c.currentAnimationIndex >= 0 && c.currentAnimationIndex < len(c.switchableAnimations) {
		return c.switchableAnimations[c.currentAnimationIndex].DisplayName
	}

	// 降级处理
	if len(c.currentAnimations) == 0 {
		return ""
	}
	return c.currentAnimations[0]
}

// GetName 获取单元名称
func (c *AnimationCell) GetName() string {
	return c.config.Name
}

// ToggleDetailMode 切换详情模式
func (c *AnimationCell) ToggleDetailMode() {
	c.detailMode = !c.detailMode
}

// IsDetailMode 是否处于详情模式
func (c *AnimationCell) IsDetailMode() bool {
	return c.detailMode
}

// GetVisualTracks 获取所有视觉轨道列表
func (c *AnimationCell) GetVisualTracks() []string {
	return c.visualTracks
}

// IsTrackVisible 检查轨道是否可见
func (c *AnimationCell) IsTrackVisible(trackName string) bool {
	// 检查配置中的隐藏轨道
	if c.hiddenTracks != nil && c.hiddenTracks[trackName] {
		return false
	}
	// 检查手动隐藏的轨道
	if c.manualHiddenTracks != nil && c.manualHiddenTracks[trackName] {
		return false
	}
	return true
}

// ToggleTrackVisibility 切换轨道可见性
func (c *AnimationCell) ToggleTrackVisibility(trackName string) {
	if c.manualHiddenTracks == nil {
		c.manualHiddenTracks = make(map[string]bool)
	}
	c.manualHiddenTracks[trackName] = !c.manualHiddenTracks[trackName]
}

// ResetTrackVisibility 重置所有手动隐藏的轨道
func (c *AnimationCell) ResetTrackVisibility() {
	c.manualHiddenTracks = make(map[string]bool)
}

// === 工具函数 ===

// calculateCenterOffset 计算动画的中心偏移量
// 逻辑与游戏系统的 pkg/systems/reanim_helpers.go:calculateCenterOffset 一致
//
// 该函数遍历所有视觉轨道的第一帧，计算所有可见部件的 bounding box，
// 然后返回 bounding box 的中心坐标作为 CenterOffset。
//
// CenterOffset 用于实现中心锚点方案：
//   - 渲染原点 = 中心 - CenterOffset
//   - 部件坐标 = 渲染原点 + frame.X
//
// 参数:
//   - mergedTracks: 合并后的轨道数据
//   - partImages: 图片资源映射
//   - visualTracks: 视觉轨道列表
//
// 返回:
//   - centerOffsetX, centerOffsetY: BoundingBox 中心坐标
func calculateCenterOffset(
	mergedTracks map[string][]reanim.Frame,
	partImages map[string]*ebiten.Image,
	visualTracks []string,
) (float64, float64) {
	// 边界检查
	if mergedTracks == nil || len(visualTracks) == 0 {
		return 0, 0
	}

	// 计算第一帧（索引 0）的 bounding box
	minX, maxX := 9999.0, -9999.0
	minY, maxY := 9999.0, -9999.0
	hasVisibleParts := false

	for _, trackName := range visualTracks {
		frames, ok := mergedTracks[trackName]
		if !ok || len(frames) == 0 {
			continue
		}

		// 获取第一帧
		frame := frames[0]

		// 跳过隐藏帧
		if frame.FrameNum != nil && *frame.FrameNum == -1 {
			continue
		}

		// 跳过无图片的帧
		if frame.ImagePath == "" {
			continue
		}

		// 获取图片
		img, ok := partImages[frame.ImagePath]
		if !ok || img == nil {
			continue
		}

		// 计算部件位置
		partX := getFloat(frame.X)
		partY := getFloat(frame.Y)

		// 获取图片尺寸
		bounds := img.Bounds()
		w := float64(bounds.Dx())
		h := float64(bounds.Dy())

		// 考虑缩放
		scaleX := getFloat(frame.ScaleX)
		scaleY := getFloat(frame.ScaleY)
		if scaleX == 0 {
			scaleX = 1.0
		}
		if scaleY == 0 {
			scaleY = 1.0
		}

		// 计算部件的 bounding box（考虑图片尺寸）
		// 注意：这里使用左上角锚点，因为 frame.X/Y 代表图片左上角
		partMinX := partX
		partMaxX := partX + w*scaleX
		partMinY := partY
		partMaxY := partY + h*scaleY

		// 更新全局 bounding box
		if partMinX < minX {
			minX = partMinX
		}
		if partMaxX > maxX {
			maxX = partMaxX
		}
		if partMinY < minY {
			minY = partMinY
		}
		if partMaxY > maxY {
			maxY = partMaxY
		}

		hasVisibleParts = true
	}

	// 如果没有可见部件，返回 (0, 0)
	if !hasVisibleParts {
		return 0, 0
	}

	// 计算并返回 bounding box 中心
	centerOffsetX := (minX + maxX) / 2.0
	centerOffsetY := (minY + maxY) / 2.0

	return centerOffsetX, centerOffsetY
}

// analyzeTrackTypes 分析轨道类型（视觉轨道和逻辑轨道）
func analyzeTrackTypes(reanimXML *reanim.ReanimXML) (visualTracks []string, logicalTracks []string) {
	animationDefinitionTracks := map[string]bool{
		"anim_idle":      true,
		"anim_shooting":  true,
		"anim_head_idle": true,
		"anim_full_idle": true,
	}

	for _, track := range reanimXML.Tracks {
		// 跳过动画定义轨道
		if animationDefinitionTracks[track.Name] {
			continue
		}

		hasImage := false
		for _, frame := range track.Frames {
			if frame.ImagePath != "" {
				hasImage = true
				break
			}
		}

		if hasImage {
			visualTracks = append(visualTracks, track.Name)
		} else {
			logicalTracks = append(logicalTracks, track.Name)
		}
	}

	return
}

// buildVisiblesArray 构建动画可见性数组
func buildVisiblesArray(reanimXML *reanim.ReanimXML, mergedTracks map[string][]reanim.Frame, animName string) []int {
	var animTrack *reanim.Track
	for i := range reanimXML.Tracks {
		if reanimXML.Tracks[i].Name == animName {
			animTrack = &reanimXML.Tracks[i]
			break
		}
	}

	if animTrack == nil {
		return []int{}
	}

	standardFrameCount := 0
	for _, track := range reanimXML.Tracks {
		if len(track.Frames) > standardFrameCount {
			standardFrameCount = len(track.Frames)
		}
	}

	if standardFrameCount == 0 {
		return []int{}
	}

	visibles := make([]int, standardFrameCount)
	currentValue := 0

	for i := 0; i < standardFrameCount; i++ {
		if i < len(animTrack.Frames) {
			frame := animTrack.Frames[i]
			if frame.FrameNum != nil {
				currentValue = *frame.FrameNum
			}
		}
		visibles[i] = currentValue
	}

	return visibles
}

// countVisibleFrames 计算可见帧数
func countVisibleFrames(animVisibles []int) int {
	count := 0
	for _, visible := range animVisibles {
		if visible == 0 {
			count++
		}
	}
	return count
}

// mapLogicalToPhysical 将逻辑帧号映射到物理帧号
func mapLogicalToPhysical(logicalFrameNum int, animVisibles []int) int {
	if len(animVisibles) == 0 {
		return logicalFrameNum
	}

	logicalIndex := 0
	for i := 0; i < len(animVisibles); i++ {
		if animVisibles[i] == 0 {
			if logicalIndex == logicalFrameNum {
				return i
			}
			logicalIndex++
		}
	}

	return -1
}

// findVisibleWindow 查找动画的可见时间窗口
func findVisibleWindow(animVisibles []int) (int, int) {
	firstVisible, lastVisible := -1, -1
	for i, v := range animVisibles {
		if v == 0 {
			if firstVisible == -1 {
				firstVisible = i
			}
			lastVisible = i
		}
	}
	return firstVisible, lastVisible
}

// calculatePositionVariance 计算位置方差
func calculatePositionVariance(frames []reanim.Frame, startIdx, endIdx int) float64 {
	if startIdx < 0 || endIdx >= len(frames) || startIdx > endIdx {
		return 0.0
	}

	sumX, sumY := 0.0, 0.0
	count := 0
	for i := startIdx; i <= endIdx; i++ {
		if frames[i].X != nil && frames[i].Y != nil {
			sumX += *frames[i].X
			sumY += *frames[i].Y
			count++
		}
	}

	if count == 0 {
		return 0.0
	}

	meanX := sumX / float64(count)
	meanY := sumY / float64(count)

	variance := 0.0
	for i := startIdx; i <= endIdx; i++ {
		if frames[i].X != nil && frames[i].Y != nil {
			dx := *frames[i].X - meanX
			dy := *frames[i].Y - meanY
			variance += dx*dx + dy*dy
		}
	}

	return variance / float64(count)
}

// getFloat 安全获取 float64 指针的值
func getFloat(p *float64) float64 {
	if p == nil {
		return 0.0
	}
	return *p
}

// loadImageWithMask 加载图片并应用蒙版（如果存在）
// 对于 JPG 文件，检查是否有同名的以 _ 结尾的 PNG 文件作为 Alpha 蒙版
func loadImageWithMask(imagePath string) (*ebiten.Image, error) {
	// 检查文件扩展名
	ext := strings.ToLower(filepath.Ext(imagePath))

	// 如果不是 JPG，直接加载
	if ext != ".jpg" && ext != ".jpeg" {
		img, _, err := ebitenutil.NewImageFromFile(imagePath)
		return img, err
	}

	// 检查是否有对应的蒙版文件
	// 例如：image.jpg -> image_.png
	dir := filepath.Dir(imagePath)
	baseName := strings.TrimSuffix(filepath.Base(imagePath), filepath.Ext(imagePath))
	maskPath := filepath.Join(dir, baseName+"_.png")

	// 如果蒙版文件不存在，直接加载原图
	if _, err := os.Stat(maskPath); os.IsNotExist(err) {
		img, _, err := ebitenutil.NewImageFromFile(imagePath)
		return img, err
	}

	// 加载 JPG 图片（使用标准库）
	baseFile, err := os.Open(imagePath)
	if err != nil {
		return nil, fmt.Errorf("加载 JPG 失败: %w", err)
	}
	defer baseFile.Close()

	baseImg, _, err := image.Decode(baseFile)
	if err != nil {
		return nil, fmt.Errorf("解码 JPG 失败: %w", err)
	}

	// 加载蒙版文件（使用标准库）
	maskFile, err := os.Open(maskPath)
	if err != nil {
		log.Printf("  警告: 无法打开蒙版文件 %s: %v (将使用原图)", maskPath, err)
		return ebiten.NewImageFromImage(baseImg), nil
	}
	defer maskFile.Close()

	maskImg, _, err := image.Decode(maskFile)
	if err != nil {
		log.Printf("  警告: 无法解码蒙版文件 %s: %v (将使用原图)", maskPath, err)
		return ebiten.NewImageFromImage(baseImg), nil
	}

	// 应用蒙版（在标准 image.Image 上操作）
	resultImg, err := applyAlphaMask(baseImg, maskImg)
	if err != nil {
		log.Printf("  警告: 应用蒙版失败 %s: %v (将使用原图)", maskPath, err)
		return ebiten.NewImageFromImage(baseImg), nil
	}

	if *verbose {
		log.Printf("  ✓ 应用蒙版: %s <- %s", imagePath, maskPath)
	}

	// 转换为 Ebiten 图像
	return ebiten.NewImageFromImage(resultImg), nil
}

// applyAlphaMask 将 PNG 蒙版应用到 JPG 图片上
// PNG 蒙版是 8-bit 调色板模式，蒙版的非白色部分会让背景图片透明
func applyAlphaMask(baseImage image.Image, maskImage image.Image) (*image.RGBA, error) {
	baseBounds := baseImage.Bounds()
	maskBounds := maskImage.Bounds()

	// 检查尺寸是否匹配
	if baseBounds.Dx() != maskBounds.Dx() || baseBounds.Dy() != maskBounds.Dy() {
		return nil, fmt.Errorf("图片尺寸不匹配: base=%dx%d, mask=%dx%d",
			baseBounds.Dx(), baseBounds.Dy(), maskBounds.Dx(), maskBounds.Dy())
	}

	// 创建新的 RGBA 图像
	width := baseBounds.Dx()
	height := baseBounds.Dy()
	rgba := image.NewRGBA(image.Rect(0, 0, width, height))

	// 逐像素处理
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			// 获取基础图片的颜色
			baseColor := baseImage.At(baseBounds.Min.X+x, baseBounds.Min.Y+y)
			br, bg, bb, _ := baseColor.RGBA()

			// 获取蒙版的颜色（使用红色通道作为 alpha 值）
			maskColor := maskImage.At(maskBounds.Min.X+x, maskBounds.Min.Y+y)
			mr, _, _, _ := maskColor.RGBA()

			// 蒙版值：白色(255) = 不透明，黑色(0) = 透明
			// 转换为 0-255 范围的 alpha 值
			alpha := uint8(mr >> 8)

			// 预乘 alpha 处理（改善边缘质量）
			// 预乘 alpha: RGB 值乘以 alpha 值
			alphaF := float64(alpha) / 255.0
			finalR := uint8(float64(br>>8) * alphaF)
			finalG := uint8(float64(bg>>8) * alphaF)
			finalB := uint8(float64(bb>>8) * alphaF)

			// 设置像素（预乘 alpha 格式）
			rgba.SetRGBA(x, y, color.RGBA{R: finalR, G: finalG, B: finalB, A: alpha})
		}
	}

	return rgba, nil
}
