// render_particle.go - 粒子渲染相关方法
//
// 本文件包含 RenderSystem 的粒子渲染功能：
//   - DrawParticles: 渲染 UI 粒子效果
//   - DrawGameWorldParticles: 渲染游戏世界粒子效果
//   - buildParticleVertices: 为单个粒子生成顶点数组
//
// 所有方法都是 RenderSystem 的成员方法（接收者：*RenderSystem）。
// 使用相同的 package systems，可以直接访问 RenderSystem 的私有字段。

package systems

import (
	"log"
	"math"

	"github.com/gonewx/pvz/pkg/components"
	"github.com/gonewx/pvz/pkg/ecs"
	"github.com/hajimehoshi/ebiten/v2"
)

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
	// 如需调试，可以临时启用此日志查看粒子渲染参数
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
