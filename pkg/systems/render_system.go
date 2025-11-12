package systems

import (
	"image/color"
	"log"
	"math"
	"sort"

	"github.com/decker502/pvz/pkg/components"
	"github.com/decker502/pvz/pkg/config"
	"github.com/decker502/pvz/pkg/ecs"
	"github.com/decker502/pvz/pkg/utils"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

// RenderSystem 管理游戏世界实体的渲染
//
// 职责范围：
//   - 游戏世界实体：植物、僵尸、子弹、阳光、特效等
//   - 所有这些实体使用 ReanimComponent 进行渲染
//   - 支持复杂的多部件骨骼动画和变换效果
//
// 不包括：
//   - UI 元素（植物卡片、按钮等）由专门的渲染系统处理
//   - PlantCardRenderSystem: 处理植物卡片
//   - PlantPreviewRenderSystem: 处理植物预览（虽然预览也使用 ReanimComponent）
//
// 组件策略（Story 6.3）：
//   - 游戏世界实体 → ReanimComponent（支持复杂动画）
//   - UI 元素 → SpriteComponent（简单高效）
//   - 详见：CLAUDE.md#组件使用策略
//
// 架构决策：
//   - 分离游戏逻辑渲染和 UI 渲染，保持关注点分离
//   - ReanimComponent 提供统一的动画渲染管线
//   - 单图片实体（如阳光、子弹）使用 createSimpleReanimComponent 包装
//
// 相关文档：
//   - CLAUDE.md#组件使用策略
//   - docs/stories/6.3.story.md
type RenderSystem struct {
	entityManager     *ecs.EntityManager
	reanimSystem      *ReanimSystem         // ✅ 修复：添加 ReanimSystem 引用以调用 GetRenderData()
	debugPrinted      map[ecs.EntityID]bool // 记录已打印调试信息的实体
	particleVertices  []ebiten.Vertex       // 粒子顶点数组（复用，避免每帧分配）
	particleIndices   []uint16              // 粒子索引数组（复用，避免每帧分配）
	particleDebugOnce bool                  // 粒子调试日志只输出一次
}

// NewRenderSystem 创建一个新的渲染系统
func NewRenderSystem(em *ecs.EntityManager) *RenderSystem {
	return &RenderSystem{
		entityManager:     em,
		debugPrinted:      make(map[ecs.EntityID]bool),
		particleVertices:  make([]ebiten.Vertex, 0, 4000), // 预分配容量：支持 1000 个粒子（每粒子 4 顶点）
		particleIndices:   make([]uint16, 0, 6000),        // 预分配容量：支持 1000 个粒子（每粒子 6 索引）
		particleDebugOnce: true,                           // 启用一次调试日志
	}
}

// SetReanimSystem 设置 ReanimSystem 引用（用于调用 GetRenderData）
// ✅ 修复：Epic 14 引入的问题 - RenderSystem 需要调用 ReanimSystem.GetRenderData() 更新缓存
func (s *RenderSystem) SetReanimSystem(rs *ReanimSystem) {
	s.reanimSystem = rs
}

// DrawEntity 绘制单个实体（公开方法，用于特殊场景如主菜单）
// 参数:
//   - screen: 绘制目标屏幕
//   - id: 实体ID
//   - cameraX: 摄像机的世界坐标X位置
func (s *RenderSystem) DrawEntity(screen *ebiten.Image, id ecs.EntityID, cameraX float64) {
	s.drawEntity(screen, id, cameraX)
}

// Draw 绘制所有拥有位置和精灵组件的实体（包括阳光）
// 渲染顺序（从底到顶）：植物 → 僵尸/子弹 → 阳光
// 注意：此方法包含阳光渲染，如果需要在UI层之后渲染阳光，请使用 DrawGameWorld + DrawSuns
// 参数:
//   - screen: 绘制目标屏幕
//   - cameraX: 摄像机的世界坐标X位置（用于世界坐标到屏幕坐标的转换）
func (s *RenderSystem) Draw(screen *ebiten.Image, cameraX float64) {
	s.DrawGameWorld(screen, cameraX)
	s.DrawSuns(screen, cameraX)
}

// DrawGameWorld 绘制游戏世界实体（植物、僵尸、子弹），不包括阳光
// 用于需要在阳光和UI之间插入其他渲染层的场景
// 参数:
//   - screen: 绘制目标屏幕
//   - cameraX: 摄像机的世界坐标X位置
func (s *RenderSystem) DrawGameWorld(screen *ebiten.Image, cameraX float64) {
	// Story 6.3: 所有实体都使用 ReanimComponent 渲染
	// 查询拥有 PositionComponent 和 ReanimComponent 的实体
	entities := ecs.GetEntitiesWith2[
		*components.PositionComponent,
		*components.ReanimComponent,
	](s.entityManager)

	// 第一遍：渲染植物（底层）
	for _, id := range entities {
		// 跳过植物卡片实体（它们由 PlantCardRenderSystem 专门渲染）
		if _, hasPlantCard := ecs.GetComponent[*components.PlantCardComponent](s.entityManager, id); hasPlantCard {
			continue
		}

		// 跳过植物预览实体（它们由 PlantPreviewRenderSystem 专门渲染）
		if _, hasPlantPreview := ecs.GetComponent[*components.PlantPreviewComponent](s.entityManager, id); hasPlantPreview {
			continue
		}

		// 只渲染植物
		_, isPlant := ecs.GetComponent[*components.PlantComponent](s.entityManager, id)
		if !isPlant {
			continue // 跳过非植物实体
		}

		s.drawEntity(screen, id, cameraX)
	}

	// 第二遍：渲染僵尸、子弹、特效（中间层）
	// 特效包括：SodRoll（草皮卷）、爆炸效果等
	// 需要按Y坐标排序以解决重叠闪烁问题（上方行先渲染，下方行后渲染会遮挡上方）
	zombiesAndProjectiles := make([]ecs.EntityID, 0)
	for _, id := range entities {
		// 跳过植物卡片实体
		if _, hasPlantCard := ecs.GetComponent[*components.PlantCardComponent](s.entityManager, id); hasPlantCard {
			continue
		}

		// 跳过植物预览实体
		if _, hasPlantPreview := ecs.GetComponent[*components.PlantPreviewComponent](s.entityManager, id); hasPlantPreview {
			continue
		}

		// 跳过植物
		_, isPlant := ecs.GetComponent[*components.PlantComponent](s.entityManager, id)
		if isPlant {
			continue
		}

		// 跳过阳光（由 DrawSuns 方法单独渲染）
		_, isSun := ecs.GetComponent[*components.SunComponent](s.entityManager, id)
		if isSun {
			continue
		}

		// 渲染其他所有实体（僵尸、子弹、SodRoll 等特效）
		zombiesAndProjectiles = append(zombiesAndProjectiles, id)
	}

	// 按Y坐标排序（从小到大，即从上到下）
	// 当Y坐标相同时，按X坐标排序（从大到小，即从右到左）
	// 这样可以确保：
	//   1. 上方行的僵尸先绘制，下方行的僵尸后绘制会正确遮挡
	//   2. 同一行中，右侧的僵尸先绘制，左侧的僵尸后绘制会遮挡右侧（符合透视效果）
	//   3. 避免同行僵尸重叠时的渲染闪烁
	sort.Slice(zombiesAndProjectiles, func(i, j int) bool {
		posI, _ := ecs.GetComponent[*components.PositionComponent](s.entityManager, zombiesAndProjectiles[i])
		posJ, _ := ecs.GetComponent[*components.PositionComponent](s.entityManager, zombiesAndProjectiles[j])

		// 主排序：按Y坐标（从小到大）
		if posI.Y != posJ.Y {
			return posI.Y < posJ.Y
		}

		// 二级排序：当Y坐标相同时，按X坐标（从大到小，右侧先渲染）
		return posI.X > posJ.X
	})

	// 按排序后的顺序渲染
	for _, id := range zombiesAndProjectiles {
		s.drawEntity(screen, id, cameraX)
	}
}

// DrawSuns 单独渲染阳光（最顶层）
// 用于确保阳光显示在所有UI元素（包括植物卡片）之上，便于玩家点击收集
// 参数:
//   - screen: 绘制目标屏幕
//   - cameraX: 摄像机的世界坐标X位置
func (s *RenderSystem) DrawSuns(screen *ebiten.Image, cameraX float64) {
	// Story 6.3: 所有实体都使用 ReanimComponent 渲染
	// 查询拥有 PositionComponent 和 ReanimComponent 的实体
	entities := ecs.GetEntitiesWith2[
		*components.PositionComponent,
		*components.ReanimComponent,
	](s.entityManager)

	// 只渲染阳光
	for _, id := range entities {
		// 跳过植物卡片实体
		if _, hasPlantCard := ecs.GetComponent[*components.PlantCardComponent](s.entityManager, id); hasPlantCard {
			continue
		}

		// 跳过植物预览实体
		if _, hasPlantPreview := ecs.GetComponent[*components.PlantPreviewComponent](s.entityManager, id); hasPlantPreview {
			continue
		}

		// 只渲染阳光
		_, isSun := ecs.GetComponent[*components.SunComponent](s.entityManager, id)
		if !isSun {
			continue
		}

		s.drawEntity(screen, id, cameraX)
	}
}

// drawEntity 绘制单个实体
// 参数:
//   - screen: 绘制目标屏幕
//   - id: 实体ID
//   - cameraX: 摄像机的世界坐标X位置
func (s *RenderSystem) drawEntity(screen *ebiten.Image, id ecs.EntityID, cameraX float64) {
	// Story 6.3: 优先使用 ReanimComponent 渲染
	_, hasReanimComp := ecs.GetComponent[*components.ReanimComponent](s.entityManager, id)
	if hasReanimComp {
		s.renderReanimEntity(screen, id, cameraX)
		return
	}

	// 支持简单的 SpriteComponent 渲染（用于工具图标等简单实体）
	spriteComp, hasSpriteComp := ecs.GetComponent[*components.SpriteComponent](s.entityManager, id)
	if hasSpriteComp {
		s.renderSpriteEntity(screen, id, spriteComp, cameraX)
		return
	}

	// 如果既没有 ReanimComponent 也没有 SpriteComponent，记录警告
	log.Printf("[RenderSystem] 警告: 实体 %d 没有可渲染组件（ReanimComponent 或 SpriteComponent）", id)
}

// renderSpriteEntity 渲染简单的 SpriteComponent 实体
func (s *RenderSystem) renderSpriteEntity(screen *ebiten.Image, id ecs.EntityID, sprite *components.SpriteComponent, cameraX float64) {
	if sprite.Image == nil {
		return
	}

	// 获取位置组件
	pos, hasPos := ecs.GetComponent[*components.PositionComponent](s.entityManager, id)
	if !hasPos {
		return
	}

	// 检查是否是 UI 实体（不需要相机偏移）
	_, isUI := ecs.GetComponent[*components.UIComponent](s.entityManager, id)

	// 计算屏幕坐标
	var screenX, screenY float64
	if isUI {
		// UI 实体使用屏幕坐标，不需要相机偏移
		screenX = pos.X
		screenY = pos.Y
	} else {
		// 游戏世界实体使用世界坐标，需要相机偏移
		screenX = pos.X - cameraX
		screenY = pos.Y
	}

	// 绘制选项
	op := &ebiten.DrawImageOptions{}

	// 居中图片
	bounds := sprite.Image.Bounds()
	op.GeoM.Translate(-float64(bounds.Dx())/2, -float64(bounds.Dy())/2)

	// 移动到目标位置
	op.GeoM.Translate(screenX, screenY)

	screen.DrawImage(sprite.Image, op)
}

// getFloat 辅助函数：安全获取 float 指针的值
func getFloat(p *float64) float64 {
	if p == nil {
		return 0.0
	}
	return *p
}

// findPhysicalFrameIndex 将逻辑帧号映射到物理帧索引
// 逻辑帧是可见帧的序号（0, 1, 2, ...），物理帧是 AnimVisibles 数组中的索引
//
// Story 12.1: 如果 AnimVisiblesMap 中当前动画的 AnimVisibles 为空，说明使用 PlayAllFrames 模式，
// CurrentFrame 直接就是物理帧索引，无需映射。
//
// 参数:
//   - reanim: ReanimComponent 包含 AnimVisiblesMap
//   - logicalFrameNum: 逻辑帧号（从 0 开始）
//
// 返回:
//   - 物理帧索引，如果找不到则返回 -1
func (s *RenderSystem) findPhysicalFrameIndex(reanim *components.ReanimComponent, logicalFrameNum int) int {
	// 获取当前动画的 AnimVisibles
	animVisibles := reanim.AnimVisiblesMap[reanim.CurrentAnimations[0]]

	// Story 12.1: PlayAllFrames 模式 - CurrentFrame 直接是物理帧
	// 这适用于 SelectorScreen 等不基于动画定义的复杂动画
	if len(animVisibles) == 0 {
		return logicalFrameNum // 直接返回，无需映射
	}

	// PlayAnimation 模式 - 映射逻辑帧到物理帧
	// 逻辑帧按区间映射：从第一个0开始到下一个非0之前
	// 如果当前逻辑帧号 n，则寻找第 n 个可见段的起点物理索引
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

// renderReanimEntity 渲染使用 ReanimComponent 的实体
// Story 13.8: 完全重写，使用 CachedRenderData 简化实现
//
// 参数:
//   - screen: 绘制目标屏幕
//   - id: 实体ID
//   - cameraX: 摄像机的世界坐标X位置
func (s *RenderSystem) renderReanimEntity(screen *ebiten.Image, id ecs.EntityID, cameraX float64) {
	// 获取组件
	pos, hasPosComp := ecs.GetComponent[*components.PositionComponent](s.entityManager, id)
	reanimComp, hasReanimComp := ecs.GetComponent[*components.ReanimComponent](s.entityManager, id)

	if !hasPosComp || !hasReanimComp {
		return
	}

	// 检查是否是 UI 元素（UI 元素不受摄像机影响）
	_, isUI := ecs.GetComponent[*components.UIComponent](s.entityManager, id)
	effectiveCameraX := cameraX
	if isUI {
		effectiveCameraX = 0 // UI 元素使用屏幕坐标，不应用摄像机偏移
	}

	// ✅ 修复：调用 GetRenderData() 更新渲染缓存
	// Epic 14 之前的错误：直接读取 CachedRenderData 导致缓存从不更新，主菜单黑屏
	var renderData []components.RenderPartData
	if s.reanimSystem != nil {
		renderData = s.reanimSystem.GetRenderData(id)
	} else {
		// 后备：直接读取缓存（兼容旧代码）
		renderData = reanimComp.CachedRenderData
	}

	if renderData == nil || len(renderData) == 0 {
		return // 没有渲染数据
	}

	// 检查是否有闪烁效果组件
	flashIntensity := 0.0
	if flashComp, hasFlash := ecs.GetComponent[*components.FlashEffectComponent](s.entityManager, id); hasFlash && flashComp.IsActive {
		flashIntensity = flashComp.Intensity
	}

	// 计算屏幕坐标（世界坐标 - 摄像机偏移 - 居中偏移）
	// CenterOffsetX/Y 是预先计算好的 bounding box 中心坐标
	// 减去 CenterOffset 使得 bounding box 中心对齐到 Position
	baseScreenX := pos.X - effectiveCameraX - reanimComp.CenterOffsetX
	baseScreenY := pos.Y - reanimComp.CenterOffsetY

	// 渲染每个部件
	for _, partData := range renderData {
		if partData.Img == nil {
			continue
		}

		frame := partData.Frame

		// 跳过隐藏帧（FrameNum == -1）
		if frame.FrameNum != nil && *frame.FrameNum == -1 {
			continue
		}

		// 计算部件位置（相对于实体原点，加上父子偏移）
		partX := getFloat(frame.X) + partData.OffsetX
		partY := getFloat(frame.Y) + partData.OffsetY

		// 获取图片尺寸
		bounds := partData.Img.Bounds()
		w := float64(bounds.Dx())
		h := float64(bounds.Dy())

		// 获取变换参数
		scaleX := getFloat(frame.ScaleX)
		scaleY := getFloat(frame.ScaleY)
		if scaleX == 0 {
			scaleX = 1.0
		}
		if scaleY == 0 {
			scaleY = 1.0
		}

		skewX := getFloat(frame.SkewX)
		skewY := getFloat(frame.SkewY)

		// 构建变换矩阵
		// a, b 控制 X 方向的变换
		// c, d 控制 Y 方向的变换
		//
		// ✅ Story 13.8 Bug Fix #11: 修复 Skew 变换计算
		//   - 参考实现：animation_cell.go:530-546
		//   - Reanim 文件中 SkewX/SkewY 存储的是度数，需要转换为弧度
		//   - 使用正确的 cos/sin 矩阵，而不是 tan
		var a, b, c, d float64
		if skewX == 0 && skewY == 0 {
			// 优化：无倾斜时使用简单的缩放（最常见情况）
			a = scaleX
			b = 0
			c = 0
			d = scaleY
		} else {
			// 有倾斜，使用完整的变换矩阵
			// ✅ 修复 1: 将度数转换为弧度（Reanim 文件存储的是度数）
			skewXRad := skewX * math.Pi / 180.0
			skewYRad := skewY * math.Pi / 180.0

			// ✅ 修复 2: 使用正确的 cos/sin 矩阵
			// 标准 skew 矩阵：
			// [a  c]   [cos(kx)*sx   -sin(ky)*sy]
			// [b  d] = [sin(kx)*sx    cos(ky)*sy]
			a = math.Cos(skewXRad) * scaleX
			b = math.Sin(skewXRad) * scaleX
			c = -math.Sin(skewYRad) * scaleY // 注意负号
			d = math.Cos(skewYRad) * scaleY
		}

		// 计算最终位置（部件位置 + 父子偏移 + 实体屏幕位置）
		tx := partX + baseScreenX
		ty := partY + baseScreenY

		// 应用变换矩阵到图片的四个角
		x0 := tx
		y0 := ty
		x1 := a*w + tx
		y1 := b*w + ty
		x2 := c*h + tx
		y2 := d*h + ty
		x3 := a*w + c*h + tx
		y3 := b*w + d*h + ty

		// 构建顶点数组（应用闪烁效果）
		colorR := float32(1.0 + flashIntensity)
		colorG := float32(1.0 + flashIntensity)
		colorB := float32(1.0 + flashIntensity)
		colorA := float32(1.0)

		vs := []ebiten.Vertex{
			{DstX: float32(x0), DstY: float32(y0), SrcX: 0, SrcY: 0, ColorR: colorR, ColorG: colorG, ColorB: colorB, ColorA: colorA},
			{DstX: float32(x1), DstY: float32(y1), SrcX: float32(w), SrcY: 0, ColorR: colorR, ColorG: colorG, ColorB: colorB, ColorA: colorA},
			{DstX: float32(x2), DstY: float32(y2), SrcX: 0, SrcY: float32(h), ColorR: colorR, ColorG: colorG, ColorB: colorB, ColorA: colorA},
			{DstX: float32(x3), DstY: float32(y3), SrcX: float32(w), SrcY: float32(h), ColorR: colorR, ColorG: colorG, ColorB: colorB, ColorA: colorA},
		}
		is := []uint16{0, 1, 2, 1, 3, 2}
		screen.DrawTriangles(vs, is, partData.Img, nil)
	}
}

// ==================================================================
// Story 6.5: Dual-Animation Rendering Helper Functions
// ==================================================================

// getStemOffset calculates the offset of anim_stem from its initial position.
// This offset is applied to head parts to implement parent-child hierarchy.
//
// The anim_stem track defines the attachment point for the head. In anim_idle,
// it sways with the body. In anim_shooting, it stays static. By applying the
// stem offset to head parts, we make the head follow the body movement.
//
// Parameters:
//   - reanim: the ReanimComponent containing merged tracks
//   - physicalFrame: the physical frame index for the idle animation
//
// Returns:
//   - offsetX, offsetY: the offset from the initial anim_stem position
func (s *RenderSystem) getStemOffset(
	reanim *components.ReanimComponent,
	physicalFrame int,
) (float64, float64) {
	// Get anim_stem merged frames
	stemFrames, ok := reanim.MergedTracks["anim_stem"]
	if !ok || physicalFrame >= len(stemFrames) {
		return 0, 0
	}

	stemFrame := stemFrames[physicalFrame]

	// Get initial stem position from first frame (generalized approach)
	// Instead of hardcoding ReanimStemInitX/Y, use the first frame as reference
	initX, initY := 0.0, 0.0
	if len(stemFrames) > 0 && stemFrames[0].X != nil && stemFrames[0].Y != nil {
		initX = *stemFrames[0].X
		initY = *stemFrames[0].Y
	}

	// Get current stem position
	currentX := initX
	currentY := initY

	if stemFrame.X != nil {
		currentX = *stemFrame.X
	}
	if stemFrame.Y != nil {
		currentY = *stemFrame.Y
	}

	// Calculate offset from initial position
	offsetX := currentX - initX
	offsetY := currentY - initY

	return offsetX, offsetY
}

// mapLogicalFrameToPhysical 将逻辑帧号映射到物理帧索引
//
// 这是一个独立的辅助函数，用于多动画叠加场景下为每个动画独立映射帧索引。
// 与 findPhysicalFrameIndex 类似，但接受 animVisibles 作为参数，不依赖组件状态。
//
// 参数:
//   - logicalFrameNum: 逻辑帧号（从 0 开始）
//   - animVisibles: 动画的可见性数组（0=可见，-1=隐藏）
//
// 返回:
//   - 物理帧索引，如果找不到则返回 -1
func (s *RenderSystem) mapLogicalFrameToPhysical(logicalFrameNum int, animVisibles []int) int {
	// 如果 AnimVisibles 为空，说明使用 PlayAllFrames 模式，直接返回逻辑帧
	if len(animVisibles) == 0 {
		return logicalFrameNum
	}

	// PlayAnimation 模式：映射逻辑帧到物理帧
	// 逻辑帧按可见段映射：从第一个0开始到下一个非0之前
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

// DrawParticles 渲染所有粒子效果
//
// 渲染流程：
// 1. 查询所有拥有 ParticleComponent 和 PositionComponent 的实体
// 2. 按图片和混合模式分组批量渲染（减少 DrawTriangles 调用次数）
// 3. 每个粒子生成 6 个顶点（2 个三角形组成矩形）
// 4. 应用粒子变换：位置、旋转、缩放
// 5. 应用粒子颜色：RGB、Alpha、Brightness
//
// 性能优化：
// - 使用预分配的顶点数组（s.particleVertices），避免每帧内存分配
// - 批量渲染相同图片和混合模式的粒子
//
// 参数:
//   - screen: 绘制目标屏幕
//   - cameraX: 摄像机的世界坐标X位置（用于世界坐标到屏幕坐标的转换）
func (s *RenderSystem) DrawParticles(screen *ebiten.Image, cameraX float64) {
	// DEBUG: 输出摄像机位置（只输出一次避免刷屏）
	if s.particleDebugOnce {
		log.Printf("[RenderSystem] DrawParticles: cameraX=%.1f", cameraX)
		s.particleDebugOnce = false
	}

	// 查询所有拥有 ParticleComponent 和 PositionComponent 的实体
	entities := ecs.GetEntitiesWith2[
		*components.PositionComponent,
		*components.ParticleComponent,
	](s.entityManager)

	if len(entities) == 0 {
		return
	}

	// 过滤出只有 UI 粒子（避免与 DrawGameWorldParticles 重复渲染）
	uiParticleEntities := make([]ecs.EntityID, 0)
	for _, id := range entities {
		_, isUIParticle := ecs.GetComponent[*components.UIComponent](s.entityManager, id)
		if isUIParticle {
			uiParticleEntities = append(uiParticleEntities, id)
		}
	}

	if len(uiParticleEntities) == 0 {
		return
	}

	// DEBUG: 粒子数量日志（每帧打印会刷屏，已注释）
	// log.Printf("[RenderSystem] DrawParticles (UI only): 找到 %d 个 UI 粒子实体", len(uiParticleEntities))

	// 按图片和混合模式分组粒子（用于批量渲染）
	// 以 (image 指针, 混合模式) 作为批次键，避免不同贴图被错误混用
	type renderBatch struct {
		image    *ebiten.Image
		additive bool
		entities []ecs.EntityID
	}

	type batchKey struct {
		img      *ebiten.Image
		additive bool
	}

	batches := make(map[batchKey]*renderBatch)

	for _, id := range uiParticleEntities {
		particle, hasParticle := ecs.GetComponent[*components.ParticleComponent](s.entityManager, id)
		if !hasParticle {
			continue
		}

		if particle.Image == nil {
			continue
		}

		key := batchKey{img: particle.Image, additive: particle.Additive}
		batch, exists := batches[key]
		if !exists {
			batch = &renderBatch{
				image:    particle.Image,
				additive: particle.Additive,
				entities: make([]ecs.EntityID, 0),
			}
			batches[key] = batch
		}
		batch.entities = append(batch.entities, id)
	}

	// 渲染顺序：先 Normal 后 Additive，保证发光效果叠加在上
	// 需要遍历 map 两次以维持顺序
	renderBatches := func(targetAdditive bool) {
		for _, batch := range batches {
			if batch.additive != targetAdditive {
				continue
			}

			// 重置顶点数组（保留容量，避免内存分配）
			s.particleVertices = s.particleVertices[:0]
			s.particleIndices = s.particleIndices[:0]

			// 为批次中的每个粒子生成顶点
			for _, id := range batch.entities {
				pos, hasPos := ecs.GetComponent[*components.PositionComponent](s.entityManager, id)
				particle, hasParticle := ecs.GetComponent[*components.ParticleComponent](s.entityManager, id)

				if !hasPos || !hasParticle {
					continue
				}

				// 检查粒子是否为UI粒子（不需要减去cameraX）
				_, isUIParticle := ecs.GetComponent[*components.UIComponent](s.entityManager, id)
				cameraDelta := cameraX
				if isUIParticle {
					cameraDelta = 0 // UI粒子不受摄像机影响
				}

				// 生成粒子的顶点（4 个顶点，用索引构建 2 个三角形）
				vertices := s.buildParticleVertices(particle, pos, cameraDelta)
				if len(vertices) != 4 {
					continue
				}

				// 添加顶点到批次数组
				baseIndex := uint16(len(s.particleVertices))
				s.particleVertices = append(s.particleVertices, vertices...)

				// 添加索引（两个三角形）
				s.particleIndices = append(s.particleIndices,
					baseIndex+0, baseIndex+1, baseIndex+2, // 第一个三角形
					baseIndex+1, baseIndex+3, baseIndex+2, // 第二个三角形
				)
			}

			// 如果没有顶点，跳过渲染
			if len(s.particleVertices) == 0 {
				continue
			}

			// 配置绘制选项（混合模式）
			op := &ebiten.DrawTrianglesOptions{}

			// Story 7.4 修复：设置 AntiAlias 为 true 以获得更平滑的渲染
			op.AntiAlias = true

			if batch.additive {
				// 加法混合模式（用于发光效果，如爆炸、火焰）
				op.Blend = ebiten.Blend{
					BlendFactorSourceRGB:        ebiten.BlendFactorOne,
					BlendFactorDestinationRGB:   ebiten.BlendFactorOne,
					BlendOperationRGB:           ebiten.BlendOperationAdd,
					BlendFactorSourceAlpha:      ebiten.BlendFactorOne,
					BlendFactorDestinationAlpha: ebiten.BlendFactorOne,
					BlendOperationAlpha:         ebiten.BlendOperationAdd,
				}
			}
			// 如果 additive == false，使用默认混合模式（普通 Alpha 混合）

			// 批量绘制所有粒子（同一批次共享同一贴图）
			screen.DrawTriangles(s.particleVertices, s.particleIndices, batch.image, op)
		}
	}

	// 先绘制 Normal，再绘制 Additive
	renderBatches(false)
	renderBatches(true)
}

// DrawGameWorldParticles 只渲染游戏世界的粒子（过滤掉 UI 粒子）
// 用于 GameScene Layer 6，确保 UI 粒子（如奖励动画）由各自的系统管理
//
// 参数:
//   - screen: 绘制目标屏幕
//   - cameraX: 摄像机的世界坐标X位置
func (s *RenderSystem) DrawGameWorldParticles(screen *ebiten.Image, cameraX float64) {
	// 查询所有拥有 ParticleComponent 和 PositionComponent 的实体
	entities := ecs.GetEntitiesWith2[
		*components.PositionComponent,
		*components.ParticleComponent,
	](s.entityManager)

	if len(entities) == 0 {
		return
	}

	// 过滤掉 UI 粒子
	gameWorldEntities := make([]ecs.EntityID, 0, len(entities))
	uiParticleCount := 0
	for _, id := range entities {
		_, isUIParticle := ecs.GetComponent[*components.UIComponent](s.entityManager, id)
		if !isUIParticle {
			gameWorldEntities = append(gameWorldEntities, id)
		} else {
			uiParticleCount++
		}
	}

	if len(gameWorldEntities) == 0 {
		return
	}

	// 使用相同的批量渲染逻辑
	type renderBatch struct {
		image    *ebiten.Image
		additive bool
		entities []ecs.EntityID
	}

	type batchKey struct {
		img      *ebiten.Image
		additive bool
	}

	batches := make(map[batchKey]*renderBatch)

	for _, id := range gameWorldEntities {
		particle, hasParticle := ecs.GetComponent[*components.ParticleComponent](s.entityManager, id)
		if !hasParticle || particle.Image == nil {
			continue
		}

		key := batchKey{img: particle.Image, additive: particle.Additive}
		batch, exists := batches[key]
		if !exists {
			batch = &renderBatch{
				image:    particle.Image,
				additive: particle.Additive,
				entities: make([]ecs.EntityID, 0),
			}
			batches[key] = batch
		}
		batch.entities = append(batch.entities, id)
	}

	renderBatches := func(targetAdditive bool) {
		for _, batch := range batches {
			if batch.additive != targetAdditive {
				continue
			}

			s.particleVertices = s.particleVertices[:0]
			s.particleIndices = s.particleIndices[:0]

			for _, id := range batch.entities {
				pos, hasPos := ecs.GetComponent[*components.PositionComponent](s.entityManager, id)
				particle, hasParticle := ecs.GetComponent[*components.ParticleComponent](s.entityManager, id)

				if !hasPos || !hasParticle {
					continue
				}

				vertices := s.buildParticleVertices(particle, pos, cameraX)
				if len(vertices) != 4 {
					continue
				}

				baseIndex := uint16(len(s.particleVertices))
				s.particleVertices = append(s.particleVertices, vertices...)
				s.particleIndices = append(s.particleIndices,
					baseIndex+0, baseIndex+1, baseIndex+2,
					baseIndex+1, baseIndex+3, baseIndex+2,
				)
			}

			if len(s.particleVertices) == 0 {
				continue
			}

			op := &ebiten.DrawTrianglesOptions{}
			op.AntiAlias = true

			if batch.additive {
				op.Blend = ebiten.Blend{
					BlendFactorSourceRGB:        ebiten.BlendFactorOne,
					BlendFactorDestinationRGB:   ebiten.BlendFactorOne,
					BlendOperationRGB:           ebiten.BlendOperationAdd,
					BlendFactorSourceAlpha:      ebiten.BlendFactorOne,
					BlendFactorDestinationAlpha: ebiten.BlendFactorOne,
					BlendOperationAlpha:         ebiten.BlendOperationAdd,
				}
			}

			screen.DrawTriangles(s.particleVertices, s.particleIndices, batch.image, op)
		}
	}

	renderBatches(false)
	renderBatches(true)
}

// buildParticleVertices 为单个粒子生成顶点数组
//
// 生成顺序：
// 1. 计算粒子矩形的四个角（未变换，中心对齐）
// 2. 应用旋转变换（旋转矩阵）
// 3. 应用缩放变换
// 4. 平移到世界位置
// 5. 转换为屏幕坐标（减去 cameraX）
// 6. 设置顶点颜色：RGB * Brightness, Alpha
//
// 锚点策略：
// - 粒子图片锚点在中心（与植物、僵尸一致，参见 CLAUDE.md）
// - 因此四个角相对于中心点计算：(-w/2, -h/2) 到 (w/2, h/2)
//
// 精灵图处理（Story 7.4 修复）：
// - 如果 ImageFrames > 1，使用 SubImage() 提取单个帧
// - 帧排列方式：水平排列（从左到右）
// - 例如：96x24 图片，4 帧 = 每帧 24x24
//
// 参数:
//   - particle: 粒子组件（包含旋转、缩放、颜色等属性）
//   - pos: 位置组件（世界坐标）
//   - cameraX: 摄像机X坐标
//
// 返回:
//   - 4 个顶点（左上、右上、左下、右下），用于通过索引数组构建 2 个三角形
func (s *RenderSystem) buildParticleVertices(particle *components.ParticleComponent, pos *components.PositionComponent, cameraX float64) []ebiten.Vertex {
	if particle.Image == nil {
		// Story 7.4 调试：记录图片为 nil 的情况
		log.Printf("[RenderSystem] 警告：粒子图片为 nil，跳过渲染（位置=%.1f,%.1f, Alpha=%.2f）", pos.X, pos.Y, particle.Alpha)
		return nil
	}

	// 获取图片尺寸
	fullBounds := particle.Image.Bounds()
	fullWidth := fullBounds.Dx()
	fullHeight := fullBounds.Dy()

	// 计算粒子尺寸和纹理坐标
	var w, h float64
	var srcX0, srcY0, srcX1, srcY1 float32

	if particle.ImageFrames > 1 {
		// BUG修复：多帧/多行精灵图的正确处理
		// 精灵图布局：cols × rows（例如：IMAGE_DIRTSMALL 是 8 cols × 2 rows）
		//
		// 计算单个帧的尺寸
		cols := particle.ImageFrames
		rows := particle.ImageRows
		if rows == 0 {
			rows = 1 // 默认单行（向后兼容）
		}

		frameWidth := fullWidth / cols
		frameHeight := fullHeight / rows // ✅ 修复：除以行数，而不是使用完整高度

		// 计算当前帧在精灵图中的行列位置
		// frameNum 是 0-based 索引，按行优先顺序（从左到右，从上到下）
		// 例如：8 cols × 2 rows，frameNum=0 → (0,0)，frameNum=7 → (7,0)，frameNum=8 → (0,1)
		frameCol := particle.FrameNum % cols
		frameRow := particle.FrameNum / cols

		// 计算纹理坐标（相对于原始图片）
		frameX := frameCol * frameWidth
		frameY := frameRow * frameHeight // ✅ 修复：考虑行偏移

		srcX0 = float32(fullBounds.Min.X + frameX)
		srcY0 = float32(fullBounds.Min.Y + frameY) // ✅ 修复：从对应行开始
		srcX1 = float32(fullBounds.Min.X + frameX + frameWidth)
		srcY1 = float32(fullBounds.Min.Y + frameY + frameHeight) // ✅ 修复：正确的单帧高度

		w = float64(frameWidth)
		h = float64(frameHeight)

		// DEBUG: 多帧精灵图日志（每个粒子每帧都打印会刷屏，已禁用）
		// log.Printf("[RenderSystem] 精灵图: 总尺寸=%dx%d, 帧数=%dx%d, 当前帧=%d(col=%d,row=%d), 纹理坐标=(%.0f,%.0f)-(%.0f,%.0f), 帧尺寸=%.0fx%.0f",
		// 	fullWidth, fullHeight, cols, rows, particle.FrameNum, frameCol, frameRow, srcX0, srcY0, srcX1, srcY1, w, h)
	} else {
		// 单帧图片：使用整个图片
		srcX0 = float32(fullBounds.Min.X)
		srcY0 = float32(fullBounds.Min.Y)
		srcX1 = float32(fullBounds.Max.X)
		srcY1 = float32(fullBounds.Max.Y)

		w = float64(fullWidth)
		h = float64(fullHeight)
	}

	// 粒子矩形的四个角（未变换，中心对齐）
	// 左上、右上、左下、右下
	corners := [][2]float64{
		{-w / 2, -h / 2}, // 左上
		{w / 2, -h / 2},  // 右上
		{-w / 2, h / 2},  // 左下
		{w / 2, h / 2},   // 右下
	}

	// 旋转角度（度转弧度）
	radians := particle.Rotation * math.Pi / 180.0
	cosTheta := math.Cos(radians)
	sinTheta := math.Sin(radians)

	// 变换后的四个角（世界坐标）
	transformedCorners := [4][2]float64{}
	for i, corner := range corners {
		// 1. 应用旋转（旋转矩阵）
		rotatedX := corner[0]*cosTheta - corner[1]*sinTheta
		rotatedY := corner[0]*sinTheta + corner[1]*cosTheta

		// 2. 应用缩放
		scaledX := rotatedX * particle.Scale
		scaledY := rotatedY * particle.Scale

		// 3. 平移到世界位置
		worldX := pos.X + scaledX
		worldY := pos.Y + scaledY

		// 4. 转换为屏幕坐标
		screenX := worldX - cameraX
		screenY := worldY

		transformedCorners[i] = [2]float64{screenX, screenY}
	}

	// 计算顶点颜色（应用亮度乘数）
	colorR := float32(particle.Red * particle.Brightness)
	colorG := float32(particle.Green * particle.Brightness)
	colorB := float32(particle.Blue * particle.Brightness)
	colorA := float32(particle.Alpha)

	// DEBUG: 粒子渲染调试日志（每个新粒子都打印会刷屏，已禁用）
	// Story 7.4: 如需调试，可以临时启用此日志查看粒子渲染参数
	// if particle.Age < 0.1 {
	// 	log.Printf("[RenderSystem] 新粒子渲染: 位置=(%.0f,%.0f) 屏幕位置=(%.0f,%.0f) 尺寸=%.1fx%.1f Scale=%.2f Alpha=%.2f 颜色RGB=(%.2f,%.2f,%.2f)",
	// 		pos.X, pos.Y, pos.X-cameraX, pos.Y,
	// 		w, h, particle.Scale, particle.Alpha,
	// 		particle.Red, particle.Green, particle.Blue)
	// }

	// 构建顶点数组（4 个顶点，用于 2 个三角形）
	// 三角形 1: 左上、右上、左下
	// 三角形 2: 右上、右下、左下
	vertices := []ebiten.Vertex{
		// 左上
		{
			DstX:   float32(transformedCorners[0][0]),
			DstY:   float32(transformedCorners[0][1]),
			SrcX:   srcX0,
			SrcY:   srcY0,
			ColorR: colorR,
			ColorG: colorG,
			ColorB: colorB,
			ColorA: colorA,
		},
		// 右上
		{
			DstX:   float32(transformedCorners[1][0]),
			DstY:   float32(transformedCorners[1][1]),
			SrcX:   srcX1,
			SrcY:   srcY0,
			ColorR: colorR,
			ColorG: colorG,
			ColorB: colorB,
			ColorA: colorA,
		},
		// 左下
		{
			DstX:   float32(transformedCorners[2][0]),
			DstY:   float32(transformedCorners[2][1]),
			SrcX:   srcX0,
			SrcY:   srcY1,
			ColorR: colorR,
			ColorG: colorG,
			ColorB: colorB,
			ColorA: colorA,
		},
		// 右下（用于第二个三角形）
		{
			DstX:   float32(transformedCorners[3][0]),
			DstY:   float32(transformedCorners[3][1]),
			SrcX:   srcX1,
			SrcY:   srcY1,
			ColorR: colorR,
			ColorG: colorG,
			ColorB: colorB,
			ColorA: colorA,
		},
	}

	// 返回 4 个顶点，在 DrawParticles 中通过索引数组构建 2 个三角形
	return vertices
}

// DrawTutorialText 绘制教学文本（Story 8.2 - TrueType 字体版本）
// 在屏幕底部中央显示教学提示文本，带半透明黑色背景条
// 参数:
//   - screen: 绘制目标屏幕
//   - tutorialFont: 教学字体（SimHei.ttf 或其他 TrueType 字体）
func (s *RenderSystem) DrawTutorialText(screen *ebiten.Image, tutorialFont interface{}) {
	// 查询教学文本实体
	textEntities := ecs.GetEntitiesWith1[*components.TutorialTextComponent](s.entityManager)

	if len(textEntities) == 0 {
		return // 无教学文本实体
	}

	for _, entity := range textEntities {
		textComp, ok := ecs.GetComponent[*components.TutorialTextComponent](s.entityManager, entity)
		if !ok {
			continue
		}

		// 如果文本为空，跳过渲染
		if textComp.Text == "" {
			continue
		}

		// 获取屏幕尺寸
		screenWidth, screenHeight := screen.Bounds().Dx(), screen.Bounds().Dy()

		// 绘制半透明黑色背景条（横贯整个屏幕）
		// 使用配置常量，方便后续手工调整
		bgY := float64(screenHeight) - config.TutorialTextBackgroundOffsetFromBottom
		bgHeight := config.TutorialTextBackgroundHeight
		ebitenutil.DrawRect(screen, 0, bgY, float64(screenWidth), bgHeight,
			color.RGBA{0, 0, 0, uint8(config.TutorialTextBackgroundAlpha)})

		// 计算文本位置（底部中央）
		// 使用配置常量，方便后续手工调整
		textX := float64(screenWidth) / 2
		textY := float64(screenHeight) - config.TutorialTextOffsetFromBottom

		// 检查是否为 TrueType 字体
		if ttFont, ok := tutorialFont.(*text.GoTextFace); ok && ttFont != nil {
			// 使用 TrueType 字体绘制（浅黄色文字 + 黑色描边）
			s.drawCenteredTextTTF(screen, textComp.Text, textX, textY, ttFont)
		} else if bFont, ok := tutorialFont.(*utils.BitmapFont); ok && bFont != nil {
			// 备选：位图字体（不支持中文，已废弃）
			log.Printf("[RenderSystem] WARNING: BitmapFont does not support Chinese, using fallback")
			bFont.DrawText(screen, textComp.Text, textX, textY, "center")
		} else {
			log.Printf("[RenderSystem] ERROR: Unknown font type or nil font!")
		}
	}
}

// drawCenteredTextTTF 使用 TrueType 字体绘制居中文本（带黑色描边）
// 教学文本效果：浅黄色文字 + 黑色描边
// 参数:
//   - screen: 绘制目标屏幕
//   - textStr: 文本内容
//   - centerX: 文本中心X坐标
//   - centerY: 文本中心Y坐标
//   - fontFace: TrueType 字体
func (s *RenderSystem) drawCenteredTextTTF(screen *ebiten.Image, textStr string, centerX, centerY float64, fontFace *text.GoTextFace) {
	// 测量文本宽度
	width, _ := text.Measure(textStr, fontFace, 0)

	// 计算左上角坐标（居中对齐）
	x := centerX - width/2
	y := centerY

	// Step 1: 绘制黑色描边（在8个方向偏移1-2像素）
	strokeColor := color.RGBA{R: 0, G: 0, B: 0, A: 255} // 黑色
	strokeOffsets := []struct{ dx, dy float64 }{
		{-1, -1}, {0, -1}, {1, -1}, // 上
		{-1, 0}, {1, 0}, // 左右
		{-1, 1}, {0, 1}, {1, 1}, // 下
	}

	for _, offset := range strokeOffsets {
		op := &text.DrawOptions{}
		op.GeoM.Translate(x+offset.dx, y+offset.dy)
		op.ColorScale.ScaleWithColor(strokeColor)
		text.Draw(screen, textStr, fontFace, op)
	}

	// Step 2: 绘制浅黄色主文本（在中心）
	// 使用浅黄色 RGB(255, 242, 0)
	textColor := color.RGBA{R: 255, G: 242, B: 0, A: 255}
	op := &text.DrawOptions{}
	op.GeoM.Translate(x, y)
	op.ColorScale.ScaleWithColor(textColor)
	text.Draw(screen, textStr, fontFace, op)
}

// findLastVisibleFrame finds the last visible frame for a given track (Story 12.1).
// Returns the physical frame index where the track is last visible (f != -1).
// Returns -1 if the track has no visible frames or is not found.
//
// This is used for PlayOnce tracks to determine where to lock the track.
