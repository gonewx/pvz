// cmd/animation_showcase/animation_cell.go
// 动画展示单元 - 管理单个 Reanim 动画的加载、播放和渲染

package main

import (
	"fmt"
	_ "image/jpeg" // 支持 JPEG 格式图片
	"log"
	"math"

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
	switchableAnimations []SwitchableAnimation
	currentAnimationIndex int // 当前播放的动画索引（用于切换）

	// 当前播放状态
	currentFrame int
	frameAccumulator float64 // 帧累加器，用于控制动画速度
	animationFPS float64 // 动画播放帧率

	// 动画播放数据
	visualTracks          []string
	logicalTracks         []string
	currentAnimations     []string              // 当前播放的动画列表
	animVisiblesMap       map[string][]int      // 每个动画的可见性数组
	trackAnimationBinding map[string]string     // 轨道到动画的绑定
	parentTracks          map[string]string     // 父子关系
	hiddenTracks          map[string]bool       // 隐藏的轨道

	// 渲染缓存（优化性能）
	lastRenderFrame int
	cachedRenderData []renderPartData

	// 渲染配置
	scale   float64
	centerX float64
	centerY float64

	// 详情模式（点击后显示）
	detailMode bool
}

// renderPartData 缓存的渲染数据
type renderPartData struct {
	img            *ebiten.Image
	frame          reanim.Frame
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
func NewAnimationCell(config *AnimationUnitConfig, globalFPS int) (*AnimationCell, error) {
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

	// 加载图片资源
	partImages := make(map[string]*ebiten.Image)
	for ref, path := range config.Images {
		img, _, err := ebitenutil.NewImageFromFile(path)
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

		visualTracks:  visualTracks,
		logicalTracks: logicalTracks,
		animVisiblesMap: make(map[string][]int),

		scale:      config.Scale,
		detailMode: false,
	}

	// 构建可切换的动画列表（包含普通动画和组合动画）
	cell.buildSwitchableAnimations()

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
	// TPS = 60, animationFPS 从 Reanim 文件读取（通常是 12）
	// 例如：TPS=60, animationFPS=12, 每 5 个游戏帧更新一次动画帧 (60/12 = 5)

	const gameTPS = 60.0 // 游戏运行的 TPS（在 main.go 中设置）

	c.frameAccumulator += c.animationFPS / gameTPS

	if c.frameAccumulator >= 1.0 {
		c.currentFrame++
		c.frameAccumulator -= 1.0
	}
}

// Render 渲染动画到指定画布
func (c *AnimationCell) Render(canvas *ebiten.Image, x, y float64) {
	c.centerX = x
	c.centerY = y

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
		if c.hiddenTracks != nil && c.hiddenTracks[trackName] {
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
func (ac *AnimationCell) drawPart(canvas *ebiten.Image, frame reanim.Frame, img *ebiten.Image, centerX, centerY float64) {
	bounds := img.Bounds()
	w := bounds.Dx()
	h := bounds.Dy()
	fw := float64(w)
	fh := float64(h)

	x := getFloat(frame.X)*ac.scale + centerX
	y := getFloat(frame.Y)*ac.scale + centerY

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

// === 工具函数 ===

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
