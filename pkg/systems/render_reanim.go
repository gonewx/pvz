package systems

import (
	"log"
	"math"

	"github.com/decker502/pvz/pkg/components"
	"github.com/decker502/pvz/pkg/ecs"
	"github.com/decker502/pvz/pkg/utils"
	"github.com/hajimehoshi/ebiten/v2"
)

// render_reanim.go - Reanim 动画渲染相关方法
//
// 本文件包含 RenderSystem 的 Reanim 动画渲染功能：
//   - renderReanimEntity: 核心 Reanim 渲染逻辑
//   - getStemOffset: 计算茎部偏移量（父子关系）
//   - findPhysicalFrameIndex: 查找物理帧索引
//   - mapLogicalFrameToPhysical: 逻辑帧到物理帧的映射
//
// 所有方法都是 RenderSystem 的成员方法（接收者：*RenderSystem）。
// 使用相同的 package systems，可以直接访问 RenderSystem 的私有字段。
func (s *RenderSystem) findPhysicalFrameIndex(reanim *components.ReanimComponent, logicalFrameNum int) int {
	// 获取当前动画的 AnimVisibles
	animVisibles := reanim.AnimVisiblesMap[reanim.CurrentAnimations[0]]

	// PlayAllFrames 模式 - CurrentFrame 直接是物理帧
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
// 完全重写，使用 CachedRenderData 简化实现
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

	// Debug: SodRoll 渲染入口（前 15 帧）
	if reanimComp.ReanimName == "sodroll" && reanimComp.CurrentFrame < 15 {
		log.Printf("[RenderReanim] 🟫 renderReanimEntity 被调用: entity=%d, frame=%d", id, reanimComp.CurrentFrame)
	}

	// DEBUG: 追踪子弹渲染
	if reanimComp.ReanimName == "simple_pea" {
		log.Printf("[RenderSystem] 🎯 开始渲染子弹 %d: pos=(%.1f, %.1f), cameraX=%.1f",
			id, pos.X, pos.Y, cameraX)
	}

	// 之前的错误：直接读取 CachedRenderData 导致缓存从不更新，主菜单黑屏
	var renderData []components.RenderPartData
	if s.reanimSystem != nil {
		renderData = s.reanimSystem.GetRenderData(id)
	} else {
		// 后备：直接读取缓存（兼容旧代码）
		renderData = reanimComp.CachedRenderData
	}

	// DEBUG: 追踪子弹的渲染数据
	if reanimComp.ReanimName == "simple_pea" {
		log.Printf("[RenderSystem] 🎯 子弹 %d GetRenderData: len(renderData)=%d, reanimSystem=%v",
			id, len(renderData), s.reanimSystem != nil)
	}

	if renderData == nil || len(renderData) == 0 {
		// DEBUG: 追踪空渲染数据
		if reanimComp.ReanimName == "simple_pea" {
			log.Printf("[RenderSystem] 🎯 子弹 %d 渲染数据为空，跳过渲染", id)
		}
		return // 没有渲染数据
	}

	// 检查是否有闪烁效果组件
	flashIntensity := 0.0
	if flashComp, hasFlash := ecs.GetComponent[*components.FlashEffectComponent](s.entityManager, id); hasFlash && flashComp.IsActive {
		flashIntensity = flashComp.Intensity
	}

	// 检查是否有悬停高亮组件（持续高亮，不闪烁）
	if hoverComp, hasHover := ecs.GetComponent[*components.HoverHighlightComponent](s.entityManager, id); hasHover && hoverComp.IsActive {
		// 悬停高亮叠加到闪烁强度上
		flashIntensity += hoverComp.Intensity
	}

	// 检查向日葵脸部发光效果
	var sunflowerGlow *components.SunflowerGlowComponent
	if glowComp, hasGlow := ecs.GetComponent[*components.SunflowerGlowComponent](s.entityManager, id); hasGlow {
		sunflowerGlow = glowComp
	}

	// 检查坚果墙被啃食发光效果
	var wallnutHitGlow *components.WallnutHitGlowComponent
	if glowComp, hasGlow := ecs.GetComponent[*components.WallnutHitGlowComponent](s.entityManager, id); hasGlow {
		wallnutHitGlow = glowComp
	}

	// Story 19.8: 检查是否是爆炸坚果（需要红色调色）
	isExplosiveNut := false
	if nutComp, hasNut := ecs.GetComponent[*components.BowlingNutComponent](s.entityManager, id); hasNut && nutComp.IsExplosive {
		isExplosiveNut = true
	}

	// 使用坐标转换工具库计算屏幕坐标
	baseScreenX, baseScreenY, err := utils.GetRenderScreenOrigin(s.entityManager, id, pos, cameraX)
	if err != nil {
		// 实体没有 ReanimComponent（理论上不会到这里，因为前面已经检查过）
		return
	}

	// 渲染每个部件
	for i, partData := range renderData {
		// DEBUG: 追踪子弹部件数据
		if reanimComp.ReanimName == "simple_pea" {
			log.Printf("[RenderSystem] 🎯 子弹 %d 部件[%d]: Img=%v, Frame.X=%v, Frame.Y=%v",
				id, i, partData.Img != nil, partData.Frame.X, partData.Frame.Y)
		}

		if partData.Img == nil {
			if reanimComp.ReanimName == "simple_pea" {
				log.Printf("[RenderSystem] 🎯 子弹 %d 部件[%d] 图片为 nil，跳过", id, i)
			}
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

		// 获取实体级别的缩放（ScaleComponent）
		entityScaleX := 1.0
		entityScaleY := 1.0
		if scaleComp, hasScaleComp := ecs.GetComponent[*components.ScaleComponent](s.entityManager, id); hasScaleComp {
			entityScaleX = scaleComp.ScaleX
			entityScaleY = scaleComp.ScaleY
		}

		// 叠加 ReanimComponent 的整体缩放
		if reanimComp.ScaleX != 0 {
			entityScaleX *= reanimComp.ScaleX
		}
		if reanimComp.ScaleY != 0 {
			entityScaleY *= reanimComp.ScaleY
		}

		// 应用整体缩放（影响部件位置）
		partX *= entityScaleX
		partY *= entityScaleY

		// 应用整体旋转（ReanimComponent.Rotation，影响部件位置）
		if reanimComp.Rotation != 0 {
			rad := reanimComp.Rotation * math.Pi / 180.0
			cosR := math.Cos(rad)
			sinR := math.Sin(rad)

			// 旋转部件位置（相对于实体原点）
			// x' = x*cos - y*sin
			// y' = x*sin + y*cos
			newX := partX*cosR - partY*sinR
			newY := partX*sinR + partY*cosR
			partX = newX
			partY = newY
		}

		// 获取图片尺寸
		bounds := partData.Img.Bounds()
		w := float64(bounds.Dx())
		h := float64(bounds.Dy())

		// DEBUG: 追踪子弹的最终坐标
		if reanimComp.ReanimName == "simple_pea" {
			finalX := partX + baseScreenX
			finalY := partY + baseScreenY
			log.Printf("[RenderSystem] 🎯 子弹 %d 最终坐标: partX=%.1f, partY=%.1f, baseScreenX=%.1f, baseScreenY=%.1f, finalX=%.1f, finalY=%.1f, 图片尺寸=%.0fx%.0f",
				id, partX, partY, baseScreenX, baseScreenY, finalX, finalY, w, h)
		}

		// 获取变换参数
		scaleX := getFloat(frame.ScaleX)
		scaleY := getFloat(frame.ScaleY)
		if scaleX == 0 {
			scaleX = 1.0
		}
		if scaleY == 0 {
			scaleY = 1.0
		}

		// 应用实体级别的缩放（ScaleComponent）到图片大小
		// 最终缩放 = Frame.ScaleX * ScaleComponent.ScaleX
		scaleX *= entityScaleX
		scaleY *= entityScaleY

		skewX := getFloat(frame.SkewX)
		skewY := getFloat(frame.SkewY)

		// Debug: SodRoll 变换数据（前 15 帧）
		if reanimComp.ReanimName == "sodroll" && reanimComp.CurrentFrame < 15 {
			log.Printf("[RenderReanim] 🟫 SodRoll Frame %d Part[%d]: scaleX=%.3f, scaleY=%.3f, skewX=%.1f°, skewY=%.1f°",
				reanimComp.CurrentFrame, i, scaleX, scaleY, skewX, skewY)
		}

		// Debug: Wallnut (保龄球坚果) 变换数据
		if reanimComp.ReanimName == "Wallnut" && frame.ImagePath == "IMAGE_REANIM_WALLNUT_BODY" {
			log.Printf("[RenderReanim] 🥜 Wallnut Frame %d Part[%d]: x=%.1f, y=%.1f, skewX=%.1f°, skewY=%.1f°",
				reanimComp.CurrentFrame, i, getFloat(frame.X), getFloat(frame.Y), skewX, skewY)
		}

		// 构建变换矩阵
		// a, b 控制 X 方向的变换
		// c, d 控制 Y 方向的变换
		//
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
			skewXRad := skewX * math.Pi / 180.0
			skewYRad := skewY * math.Pi / 180.0

			// 标准 skew 矩阵（考虑镜像）：
			// 当 entityScaleX < 0 时，我们需要镜像图片但保持旋转角度的视觉效果
			//
			// 镜像变换的关键：
			// - 水平镜像会反转 X 坐标
			// - 但 skew 角度应该保持视觉上的一致性
			//
			// 使用帧缩放的绝对值来计算 skew，然后单独应用镜像
			frameScaleXAbs := math.Abs(getFloat(frame.ScaleX))
			if frameScaleXAbs == 0 {
				frameScaleXAbs = 1.0
			}
			frameScaleYAbs := math.Abs(getFloat(frame.ScaleY))
			if frameScaleYAbs == 0 {
				frameScaleYAbs = 1.0
			}

			// 先计算基于帧缩放的 skew 矩阵
			cosKx := math.Cos(skewXRad)
			sinKx := math.Sin(skewXRad)
			cosKy := math.Cos(skewYRad)
			sinKy := math.Sin(skewYRad)

			// 应用帧缩放
			a = cosKx * frameScaleXAbs
			b = sinKx * frameScaleXAbs
			c = -sinKy * frameScaleYAbs
			d = cosKy * frameScaleYAbs

			// 然后应用实体缩放（镜像）
			// 镜像只影响水平方向：a 和 c 需要乘以 entityScaleX
			a *= entityScaleX
			c *= entityScaleX
			// b 和 d 保持原样（控制 Y 方向）
			// 但如果 entityScaleY 也是负的，则需要镜像 Y 方向
			b *= entityScaleY
			d *= entityScaleY
		}

		// 应用整体旋转（ReanimComponent.Rotation）到变换矩阵
		if reanimComp.Rotation != 0 {
			rad := reanimComp.Rotation * math.Pi / 180.0
			cosR := math.Cos(rad)
			sinR := math.Sin(rad)

			// M_final = M_rotation * M_current
			// [cos -sin] * [a c] = [cos*a - sin*b,  cos*c - sin*d]
			// [sin  cos]   [b d]   [sin*a + cos*b,  sin*c + cos*d]

			newA := cosR*a - sinR*b
			newB := sinR*a + cosR*b
			newC := cosR*c - sinR*d
			newD := sinR*c + cosR*d

			a = newA
			b = newB
			c = newC
			d = newD
		}

		// 计算最终位置（部件位置 + 父子偏移 + 实体屏幕位置）
		tx := partX + baseScreenX
		ty := partY + baseScreenY

		// 注意：镜像时不需要额外补偿
		// partX *= entityScaleX 已经正确地镜像了部件中心位置
		// scaleX *= entityScaleX 让图片翻转
		// 负缩放时，图片从 tx 向左绘制到 tx - |a|*w
		// 镜像后的部件中心 = tx - |a|*w/2 = -partX_原 + base - |a|*w/2
		// 这正好是原中心 (partX_原 + base + |a|*w/2) 的镜像

		// Debug: LoadBar_sprout 镜像渲染（前 3 帧）
		if reanimComp.ReanimName == "LoadBar_sprout" && reanimComp.CurrentFrame < 3 {
			frameScaleX := getFloat(frame.ScaleX)
			if frameScaleX == 0 {
				frameScaleX = 1.0
			}
			if entityScaleX < 0 {
				log.Printf("[RenderReanim] 🪞 Mirror Part[%d]: entityScaleX=%.1f, frameScaleX=%.3f, scaleX=%.3f, w=%.1f, partX=%.1f, partY=%.1f, tx=%.1f, baseScreenX=%.1f",
					i, entityScaleX, frameScaleX, scaleX, w, partX, partY, tx, baseScreenX)
			} else {
				log.Printf("[RenderReanim] 🌱 Normal Part[%d]: entityScaleX=%.1f, frameScaleX=%.3f, scaleX=%.3f, w=%.1f, partX=%.1f, partY=%.1f, tx=%.1f, baseScreenX=%.1f",
					i, entityScaleX, frameScaleX, scaleX, w, partX, partY, tx, baseScreenX)
			}
		}

		// Debug: 僵尸手掌渲染坐标
		if reanimComp.ReanimName == "Zombie_hand" && i < 3 { // 只打印前3个部件
			log.Printf("[RenderReanim] 🧟 Part %d: partX=%.1f, partY=%.1f, baseScreenX=%.1f, baseScreenY=%.1f → final tx=%.1f, ty=%.1f",
				i, partX, partY, baseScreenX, baseScreenY, tx, ty)
		}

		// 应用变换矩阵到图片的四个角
		x0 := tx
		y0 := ty
		x1 := a*w + tx
		y1 := b*w + ty
		x2 := c*h + tx
		y2 := d*h + ty
		x3 := a*w + c*h + tx
		y3 := b*w + d*h + ty

		// 构建顶点数组（应用闪烁效果、向日葵发光和透明度）
		colorR := float32(1.0 + flashIntensity)
		colorG := float32(1.0 + flashIntensity)
		colorB := float32(1.0 + flashIntensity)
		colorA := float32(1.0)

		// Story 19.8: 爆炸坚果红色调色
		// 应用 ColorM.Scale(1.0, 0.3, 0.3, 1.0) 效果
		if isExplosiveNut {
			colorG *= 0.3
			colorB *= 0.3
		}

		// 应用透明度（Alpha）值
		if frame.Alpha != nil {
			colorA = float32(*frame.Alpha)
		}

		vs := []ebiten.Vertex{
			{DstX: float32(x0), DstY: float32(y0), SrcX: 0, SrcY: 0, ColorR: colorR, ColorG: colorG, ColorB: colorB, ColorA: colorA},
			{DstX: float32(x1), DstY: float32(y1), SrcX: float32(w), SrcY: 0, ColorR: colorR, ColorG: colorG, ColorB: colorB, ColorA: colorA},
			{DstX: float32(x2), DstY: float32(y2), SrcX: 0, SrcY: float32(h), ColorR: colorR, ColorG: colorG, ColorB: colorB, ColorA: colorA},
			{DstX: float32(x3), DstY: float32(y3), SrcX: float32(w), SrcY: float32(h), ColorR: colorR, ColorG: colorG, ColorB: colorB, ColorA: colorA},
		}
		is := []uint16{0, 1, 2, 1, 3, 2}
		screen.DrawTriangles(vs, is, partData.Img, nil)

		// 向日葵脸部发光效果：使用加法混合绘制金色光层
		if sunflowerGlow != nil && sunflowerGlow.Intensity > 0 {
			glowIntensity := float32(sunflowerGlow.Intensity)
			// 金色发光层的颜色（乘以强度实现渐变）
			glowR := glowIntensity * float32(sunflowerGlow.ColorR)
			glowG := glowIntensity * float32(sunflowerGlow.ColorG)
			glowB := glowIntensity * float32(sunflowerGlow.ColorB)
			glowA := glowIntensity * 0.6 // 透明度也随强度衰减

			glowVs := []ebiten.Vertex{
				{DstX: float32(x0), DstY: float32(y0), SrcX: 0, SrcY: 0, ColorR: glowR, ColorG: glowG, ColorB: glowB, ColorA: glowA},
				{DstX: float32(x1), DstY: float32(y1), SrcX: float32(w), SrcY: 0, ColorR: glowR, ColorG: glowG, ColorB: glowB, ColorA: glowA},
				{DstX: float32(x2), DstY: float32(y2), SrcX: 0, SrcY: float32(h), ColorR: glowR, ColorG: glowG, ColorB: glowB, ColorA: glowA},
				{DstX: float32(x3), DstY: float32(y3), SrcX: float32(w), SrcY: float32(h), ColorR: glowR, ColorG: glowG, ColorB: glowB, ColorA: glowA},
			}
			// 使用加法混合模式绘制发光层
			glowOpts := &ebiten.DrawTrianglesOptions{
				Blend: ebiten.Blend{
					BlendFactorSourceRGB:        ebiten.BlendFactorOne,
					BlendFactorDestinationRGB:   ebiten.BlendFactorOne,
					BlendOperationRGB:           ebiten.BlendOperationAdd,
					BlendFactorSourceAlpha:      ebiten.BlendFactorOne,
					BlendFactorDestinationAlpha: ebiten.BlendFactorOne,
					BlendOperationAlpha:         ebiten.BlendOperationAdd,
				},
			}
			screen.DrawTriangles(glowVs, is, partData.Img, glowOpts)
		}

		// 坚果墙被啃食发光效果：使用加法混合绘制白色闪烁光层
		if wallnutHitGlow != nil && wallnutHitGlow.Intensity > 0 {
			glowIntensity := float32(wallnutHitGlow.Intensity)
			// 白色/浅黄色发光层的颜色（乘以强度实现闪烁效果）
			glowR := glowIntensity * float32(wallnutHitGlow.ColorR)
			glowG := glowIntensity * float32(wallnutHitGlow.ColorG)
			glowB := glowIntensity * float32(wallnutHitGlow.ColorB)
			glowA := glowIntensity * 0.5 // 透明度随强度衰减

			glowVs := []ebiten.Vertex{
				{DstX: float32(x0), DstY: float32(y0), SrcX: 0, SrcY: 0, ColorR: glowR, ColorG: glowG, ColorB: glowB, ColorA: glowA},
				{DstX: float32(x1), DstY: float32(y1), SrcX: float32(w), SrcY: 0, ColorR: glowR, ColorG: glowG, ColorB: glowB, ColorA: glowA},
				{DstX: float32(x2), DstY: float32(y2), SrcX: 0, SrcY: float32(h), ColorR: glowR, ColorG: glowG, ColorB: glowB, ColorA: glowA},
				{DstX: float32(x3), DstY: float32(y3), SrcX: float32(w), SrcY: float32(h), ColorR: glowR, ColorG: glowG, ColorB: glowB, ColorA: glowA},
			}
			// 使用加法混合模式绘制发光层
			glowOpts := &ebiten.DrawTrianglesOptions{
				Blend: ebiten.Blend{
					BlendFactorSourceRGB:        ebiten.BlendFactorOne,
					BlendFactorDestinationRGB:   ebiten.BlendFactorOne,
					BlendOperationRGB:           ebiten.BlendOperationAdd,
					BlendFactorSourceAlpha:      ebiten.BlendFactorOne,
					BlendFactorDestinationAlpha: ebiten.BlendFactorOne,
					BlendOperationAlpha:         ebiten.BlendOperationAdd,
				},
			}
			screen.DrawTriangles(glowVs, is, partData.Img, glowOpts)
		}
	}
}

// ==================================================================
// Dual-Animation Rendering Helper Functions
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
