package systems

import (
	"log"
	"math"
	"reflect"
	"sort"

	"github.com/decker502/pvz/pkg/components"
	"github.com/decker502/pvz/pkg/ecs"
	"github.com/hajimehoshi/ebiten/v2"
)

// RenderSystem 管理所有实体的渲染
type RenderSystem struct {
	entityManager *ecs.EntityManager
	debugPrinted  map[ecs.EntityID]bool // 记录已打印调试信息的实体
}

// NewRenderSystem 创建一个新的渲染系统
func NewRenderSystem(em *ecs.EntityManager) *RenderSystem {
	return &RenderSystem{
		entityManager: em,
		debugPrinted:  make(map[ecs.EntityID]bool),
	}
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
	entities := s.entityManager.GetEntitiesWith(
		reflect.TypeOf(&components.PositionComponent{}),
		reflect.TypeOf(&components.ReanimComponent{}),
	)

	// 第一遍：渲染植物（底层）
	for _, id := range entities {
		// 跳过植物卡片实体（它们由 PlantCardRenderSystem 专门渲染）
		if _, hasPlantCard := s.entityManager.GetComponent(id, reflect.TypeOf(&components.PlantCardComponent{})); hasPlantCard {
			continue
		}

		// 跳过植物预览实体（它们由 PlantPreviewRenderSystem 专门渲染）
		if _, hasPlantPreview := s.entityManager.GetComponent(id, reflect.TypeOf(&components.PlantPreviewComponent{})); hasPlantPreview {
			continue
		}

		// 只渲染植物
		_, isPlant := s.entityManager.GetComponent(id, reflect.TypeOf(&components.PlantComponent{}))
		if !isPlant {
			continue // 跳过非植物实体
		}

		s.drawEntity(screen, id, cameraX)
	}

	// 第二遍：渲染僵尸、子弹、特效（中间层）
	// 需要按Y坐标排序以解决重叠闪烁问题（上方行先渲染，下方行后渲染会遮挡上方）
	zombiesAndProjectiles := make([]ecs.EntityID, 0)
	for _, id := range entities {
		// 跳过植物卡片实体
		if _, hasPlantCard := s.entityManager.GetComponent(id, reflect.TypeOf(&components.PlantCardComponent{})); hasPlantCard {
			continue
		}

		// 跳过植物预览实体
		if _, hasPlantPreview := s.entityManager.GetComponent(id, reflect.TypeOf(&components.PlantPreviewComponent{})); hasPlantPreview {
			continue
		}

		// 跳过植物
		_, isPlant := s.entityManager.GetComponent(id, reflect.TypeOf(&components.PlantComponent{}))
		if isPlant {
			continue
		}

		// 跳过阳光（由 DrawSuns 方法单独渲染）
		_, isSun := s.entityManager.GetComponent(id, reflect.TypeOf(&components.SunComponent{}))
		if isSun {
			continue
		}

		zombiesAndProjectiles = append(zombiesAndProjectiles, id)
	}

	// 按Y坐标排序（从小到大，即从上到下）
	// 这样上方行的僵尸先绘制，下方行的僵尸后绘制会正确遮挡
	sort.Slice(zombiesAndProjectiles, func(i, j int) bool {
		posI, _ := s.entityManager.GetComponent(zombiesAndProjectiles[i], reflect.TypeOf(&components.PositionComponent{}))
		posJ, _ := s.entityManager.GetComponent(zombiesAndProjectiles[j], reflect.TypeOf(&components.PositionComponent{}))
		return posI.(*components.PositionComponent).Y < posJ.(*components.PositionComponent).Y
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
	entities := s.entityManager.GetEntitiesWith(
		reflect.TypeOf(&components.PositionComponent{}),
		reflect.TypeOf(&components.ReanimComponent{}),
	)

	// 只渲染阳光
	for _, id := range entities {
		// 跳过植物卡片实体
		if _, hasPlantCard := s.entityManager.GetComponent(id, reflect.TypeOf(&components.PlantCardComponent{})); hasPlantCard {
			continue
		}

		// 跳过植物预览实体
		if _, hasPlantPreview := s.entityManager.GetComponent(id, reflect.TypeOf(&components.PlantPreviewComponent{})); hasPlantPreview {
			continue
		}

		// 只渲染阳光
		_, isSun := s.entityManager.GetComponent(id, reflect.TypeOf(&components.SunComponent{}))
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
	// Story 6.3: 所有实体都使用 ReanimComponent 渲染
	_, hasReanimComp := s.entityManager.GetComponent(id, reflect.TypeOf(&components.ReanimComponent{}))
	if hasReanimComp {
		s.renderReanimEntity(screen, id, cameraX)
		return
	}

	// 如果没有 ReanimComponent，记录警告（不应该出现这种情况）
	log.Printf("[RenderSystem] 警告: 实体 %d 没有 ReanimComponent，无法渲染", id)
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
// 参数:
//   - reanim: ReanimComponent 包含 AnimVisibles 数组
//   - logicalFrameNum: 逻辑帧号（从 0 开始）
//
// 返回:
//   - 物理帧索引，如果找不到则返回 -1
func (s *RenderSystem) findPhysicalFrameIndex(reanim *components.ReanimComponent, logicalFrameNum int) int {
	if len(reanim.AnimVisibles) == 0 {
		return -1
	}

	// 逻辑帧按区间映射：从第一个0开始到下一个非0之前
	// 如果当前逻辑帧号 n，则寻找第 n 个可见段的起点物理索引
	logicalIndex := 0
	for i := 0; i < len(reanim.AnimVisibles); i++ {
		if reanim.AnimVisibles[i] == 0 {
			if logicalIndex == logicalFrameNum {
				return i
			}
			logicalIndex++
		}
	}

	return -1
}

// renderReanimEntity 渲染使用 ReanimComponent 的实体
// 完全按照参考实现的逻辑：
// 1. 将逻辑帧映射到物理帧
// 2. 按 AnimTracks 顺序遍历轨道（保证 Z-order）
// 3. 使用完整的变换矩阵（不使用 GeoM 链式调用）
// 参数:
//   - screen: 绘制目标屏幕
//   - id: 实体ID
//   - cameraX: 摄像机的世界坐标X位置
func (s *RenderSystem) renderReanimEntity(screen *ebiten.Image, id ecs.EntityID, cameraX float64) {
	// 获取组件
	posComp, hasPosComp := s.entityManager.GetComponent(id, reflect.TypeOf(&components.PositionComponent{}))
	reanimComp, hasReanimComp := s.entityManager.GetComponent(id, reflect.TypeOf(&components.ReanimComponent{}))

	if !hasPosComp || !hasReanimComp {
		return
	}

	// 类型断言
	pos := posComp.(*components.PositionComponent)
	reanim := reanimComp.(*components.ReanimComponent)

	// 如果没有当前动画或动画轨道，跳过
	if reanim.CurrentAnim == "" || len(reanim.AnimTracks) == 0 {
		return
	}

	// 将逻辑帧映射到物理帧索引
	physicalIndex := s.findPhysicalFrameIndex(reanim, reanim.CurrentFrame)
	if physicalIndex < 0 {
		return
	}

	// 将世界坐标转换为屏幕坐标，并应用 Reanim 的中心偏移
	//
	// 坐标系统说明：
	// - PositionComponent(X,Y) 表示格子中心的世界坐标
	// - Reanim 的部件坐标以"原点"为基准，部件图片锚点在左上角
	// - CenterOffset 将绘制原点从 Position 向左上平移，使视觉中心对齐到 Position
	//
	// 例如：豌豆射手的 CenterOffset = (39, 47.7)
	//      渲染时原点 = Position - (39, 47.7)，使得植物视觉上居中显示
	screenX := pos.X - cameraX - reanim.CenterOffsetX
	screenY := pos.Y - reanim.CenterOffsetY

	// 调试：打印第一个植物的位置和部件范围
	if !s.debugPrinted[id] {
		if plantComp, hasPlant := s.entityManager.GetComponent(id, reflect.TypeOf(&components.PlantComponent{})); hasPlant {
			plant := plantComp.(*components.PlantComponent)
			log.Printf("=== 植物 %d (Type=%d) 位置调试 ===", id, plant.PlantType)
			log.Printf("世界坐标: (%.1f, %.1f), 屏幕坐标: (%.1f, %.1f)", pos.X, pos.Y, screenX, screenY)

			// 计算部件坐标范围
			minX, maxX, minY, maxY := 999.0, -999.0, 999.0, -999.0
			for _, track := range reanim.AnimTracks {
				if mergedFrames, ok := reanim.MergedTracks[track.Name]; ok && physicalIndex < len(mergedFrames) {
					frame := mergedFrames[physicalIndex]
					if frame.FrameNum == nil || *frame.FrameNum != -1 {
						if frame.ImagePath != "" {
							x, y := getFloat(frame.X), getFloat(frame.Y)
							if x < minX {
								minX = x
							}
							if x > maxX {
								maxX = x
							}
							if y < minY {
								minY = y
							}
							if y > maxY {
								maxY = y
							}
						}
					}
				}
			}
			log.Printf("部件坐标范围: X[%.1f, %.1f], Y[%.1f, %.1f]", minX, maxX, minY, maxY)
			log.Printf("部件中心: (%.1f, %.1f)", (minX+maxX)/2, (minY+maxY)/2)
			s.debugPrinted[id] = true
		}
	}

	// 按 AnimTracks 顺序渲染部件（保证 Z-order 正确）
	for _, track := range reanim.AnimTracks {
		// 如果设置了 VisibleTracks，只渲染白名单中的轨道
		if reanim.VisibleTracks != nil && len(reanim.VisibleTracks) > 0 {
			if !reanim.VisibleTracks[track.Name] {
				continue
			}
		}

		// 获取该轨道的累积帧数组
		mergedFrames, ok := reanim.MergedTracks[track.Name]
		if !ok || len(mergedFrames) == 0 {
			continue
		}

		// 确保物理索引在范围内
		if physicalIndex >= len(mergedFrames) {
			continue
		}

		// 获取累积后的帧数据
		mergedFrame := mergedFrames[physicalIndex]

		// 如果该帧标记为隐藏（f == -1），跳过绘制
		if mergedFrame.FrameNum != nil && *mergedFrame.FrameNum == -1 {
			continue
		}

		// 必须有图片引用才能绘制
		if mergedFrame.ImagePath == "" {
			continue
		}

		// 使用 IMAGE 引用查找图片
		img, exists := reanim.PartImages[mergedFrame.ImagePath]
		if !exists || img == nil {
			continue
		}

		// 获取图片尺寸
		bounds := img.Bounds()
		w := bounds.Dx()
		h := bounds.Dy()
		fw := float64(w)
		fh := float64(h)

		// 使用 Transform2D 等价矩阵，通过 DrawTriangles 精确绘制
		// 这与参考实现完全一致（test_animation_viewer.go 第 799-856 行）
		//
		// 关键点：Reanim 的变换矩阵假设图片锚点在左上角（0,0）
		// 不需要先移动到中心，直接应用变换即可

		scaleX := 1.0
		scaleY := 1.0
		if mergedFrame.ScaleX != nil {
			scaleX = *mergedFrame.ScaleX
		}
		if mergedFrame.ScaleY != nil {
			scaleY = *mergedFrame.ScaleY
		}

		kx := 0.0
		ky := 0.0
		if mergedFrame.SkewX != nil {
			kx = *mergedFrame.SkewX
		}
		if mergedFrame.SkewY != nil {
			ky = *mergedFrame.SkewY
		}

		// 构建变换矩阵（列主序，与 PopStudio/Godot 一致）
		// Matrix = [a c tx]
		//          [b d ty]
		//          [0 0  1]
		// 其中：
		// a = cos(kx) * scaleX
		// b = sin(kx) * scaleX
		// c = -sin(ky) * scaleY
		// d = cos(ky) * scaleY
		a := math.Cos(kx*math.Pi/180.0) * scaleX
		b := math.Sin(kx*math.Pi/180.0) * scaleX
		c := -math.Sin(ky*math.Pi/180.0) * scaleY
		d := math.Cos(ky*math.Pi/180.0) * scaleY

		// 平移分量（部件位置 + 实体屏幕位置）
		tx := 0.0
		ty := 0.0
		if mergedFrame.X != nil {
			tx = *mergedFrame.X
		}
		if mergedFrame.Y != nil {
			ty = *mergedFrame.Y
		}
		tx += screenX
		ty += screenY

		// 应用变换矩阵到图片的四个角
		// 左上角 (0, 0)
		x0 := tx
		y0 := ty
		// 右上角 (w, 0)
		x1 := a*fw + tx
		y1 := b*fw + ty
		// 左下角 (0, h)
		x2 := c*fh + tx
		y2 := d*fh + ty
		// 右下角 (w, h)
		x3 := a*fw + c*fh + tx
		y3 := b*fw + d*fh + ty

		// 构建顶点数组（两个三角形组成矩形）
		vs := []ebiten.Vertex{
			{DstX: float32(x0), DstY: float32(y0), SrcX: 0, SrcY: 0, ColorR: 1, ColorG: 1, ColorB: 1, ColorA: 1},
			{DstX: float32(x1), DstY: float32(y1), SrcX: float32(w), SrcY: 0, ColorR: 1, ColorG: 1, ColorB: 1, ColorA: 1},
			{DstX: float32(x2), DstY: float32(y2), SrcX: 0, SrcY: float32(h), ColorR: 1, ColorG: 1, ColorB: 1, ColorA: 1},
			{DstX: float32(x3), DstY: float32(y3), SrcX: float32(w), SrcY: float32(h), ColorR: 1, ColorG: 1, ColorB: 1, ColorA: 1},
		}
		is := []uint16{0, 1, 2, 1, 3, 2}
		screen.DrawTriangles(vs, is, img, nil)
	}
}
