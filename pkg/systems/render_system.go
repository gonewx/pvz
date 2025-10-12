package systems

import (
	"reflect"
	"sort"

	"github.com/decker502/pvz/pkg/components"
	"github.com/decker502/pvz/pkg/ecs"
	"github.com/hajimehoshi/ebiten/v2"
)

// RenderSystem 管理所有实体的渲染
type RenderSystem struct {
	entityManager *ecs.EntityManager
}

// NewRenderSystem 创建一个新的渲染系统
func NewRenderSystem(em *ecs.EntityManager) *RenderSystem {
	return &RenderSystem{
		entityManager: em,
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
	// 查询所有拥有 PositionComponent 和 SpriteComponent 的实体
	entities := s.entityManager.GetEntitiesWith(
		reflect.TypeOf(&components.PositionComponent{}),
		reflect.TypeOf(&components.SpriteComponent{}),
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
	// 查询所有拥有 PositionComponent 和 SpriteComponent 的实体
	entities := s.entityManager.GetEntitiesWith(
		reflect.TypeOf(&components.PositionComponent{}),
		reflect.TypeOf(&components.SpriteComponent{}),
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
	// 获取组件
	posComp, _ := s.entityManager.GetComponent(id, reflect.TypeOf(&components.PositionComponent{}))
	spriteComp, _ := s.entityManager.GetComponent(id, reflect.TypeOf(&components.SpriteComponent{}))

	// 类型断言
	pos := posComp.(*components.PositionComponent)
	sprite := spriteComp.(*components.SpriteComponent)

	// 如果没有图片,跳过
	if sprite.Image == nil {
		return
	}

	// 获取图像尺寸
	bounds := sprite.Image.Bounds()
	imageWidth := float64(bounds.Dx())
	imageHeight := float64(bounds.Dy())

	// 判断实体类型，决定锚点位置
	// 检查是否有BehaviorComponent（植物、僵尸、子弹都有）
	_, hasBehavior := s.entityManager.GetComponent(id, reflect.TypeOf(&components.BehaviorComponent{}))

	// 将世界坐标转换为屏幕坐标
	// screenX = worldX - cameraX (摄像机向右移动时，实体在屏幕上向左移动)
	screenX := pos.X - cameraX
	screenY := pos.Y // Y轴不受摄像机水平移动影响

	var drawX, drawY float64
	if hasBehavior {
		// 游戏单位（植物、僵尸、子弹）：图像中心对齐到位置坐标
		// 这样确保同一行的所有单位视觉上在同一高度
		drawX = screenX - imageWidth/2
		drawY = screenY - imageHeight/2
	} else {
		// 其他实体（如阳光）：图像左上角对齐到位置坐标
		drawX = screenX
		drawY = screenY
	}

	// 创建绘制选项
	op := &ebiten.DrawImageOptions{}

	// 设置位置平移
	op.GeoM.Translate(drawX, drawY)

	// 绘制到屏幕
	screen.DrawImage(sprite.Image, op)
}
