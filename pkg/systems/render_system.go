package systems

import (
	"log"
	"math"
	"sort"

	"github.com/gonewx/pvz/pkg/components"
	"github.com/gonewx/pvz/pkg/config"
	"github.com/gonewx/pvz/pkg/ecs"
	"github.com/hajimehoshi/ebiten/v2"
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
	entityManager   *ecs.EntityManager
	reanimSystem    *ReanimSystem // ✅ 修复：添加 ReanimSystem 引用以调用 GetRenderData()
	resourceManager interface {
		GetImageByID(string) *ebiten.Image
		GetShadowImage() *ebiten.Image // Story 10.7: 添加获取阴影贴图的方法
	} // 资源管理器（用于加载房门图片、阴影贴图等）
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
func (s *RenderSystem) SetReanimSystem(rs *ReanimSystem) {
	s.reanimSystem = rs
}

// SetResourceManager 设置 ResourceManager 引用（用于加载房门图片、阴影贴图等）
// Story 10.7: 扩展接口以支持 GetShadowImage()
func (s *RenderSystem) SetResourceManager(rm interface {
	GetImageByID(string) *ebiten.Image
	GetShadowImage() *ebiten.Image
}) {
	s.resourceManager = rm
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
	// 检查游戏是否冻结（僵尸获胜流程期间）
	// Story 8.8 - Task 6: 冻结时隐藏除草车
	freezeEntities := ecs.GetEntitiesWith1[*components.GameFreezeComponent](s.entityManager)
	isFrozen := len(freezeEntities) > 0

	// Story 8.8 - Task 6: 检测僵尸获胜阶段（用于房门渲染）
	phaseEntities := ecs.GetEntitiesWith1[*components.ZombiesWonPhaseComponent](s.entityManager)
	var currentPhase int = 0
	if len(phaseEntities) > 0 {
		if phaseComp, ok := ecs.GetComponent[*components.ZombiesWonPhaseComponent](s.entityManager, phaseEntities[0]); ok {
			currentPhase = phaseComp.CurrentPhase
		}
	}

	// 所有实体都使用 ReanimComponent 渲染
	// 查询拥有 PositionComponent 和 ReanimComponent 的实体
	entities := ecs.GetEntitiesWith2[
		*components.PositionComponent,
		*components.ReanimComponent,
	](s.entityManager)

	// Story 10.7: 第一遍A：渲染植物阴影（底层-阴影层）
	s.drawPlantShadows(screen, entities, cameraX)

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

		// 冻结时隐藏除草车（Story 8.8）
		if isFrozen {
			if ecs.HasComponent[*components.LawnmowerComponent](s.entityManager, id) {
				continue
			}
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

		// 冻结时隐藏除草车（Story 8.8）
		if isFrozen {
			if ecs.HasComponent[*components.LawnmowerComponent](s.entityManager, id) {
				continue
			}
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

		// 跳过 UI 实体（由 DrawUIElements 单独渲染）
		// 这包括 ZombiesWon 动画，确保它不会被房门 Overlay 遮挡
		_, isUI := ecs.GetComponent[*components.UIComponent](s.entityManager, id)
		if isUI {
			continue
		}

		// 渲染其他所有实体（僵尸、子弹、SodRoll 等特效）
		// DEBUG: 追踪哪些实体被添加到渲染列表
		if reanim, ok := ecs.GetComponent[*components.ReanimComponent](s.entityManager, id); ok {
			if reanim.ReanimName == "simple_pea" {
				log.Printf("[RenderSystem] 🎯 子弹 %d 被添加到 zombiesAndProjectiles 渲染列表", id)
			}
		}
		zombiesAndProjectiles = append(zombiesAndProjectiles, id)
	}

	// 按Y坐标排序（从小到大，即从上到下）
	// 当Y坐标相同时，按网格列排序（从大到小，即从右到左）
	// 这样可以确保：
	//   1. 上方行的僵尸先绘制，下方行的僵尸后绘制会正确遮挡
	//   2. 同一行中，右侧的僵尸先绘制，左侧的僵尸后绘制会遮挡右侧（符合透视效果）
	//   3. 使用网格列而非连续坐标，避免同列僵尸因微小位置差异导致的渲染闪烁
	sort.Slice(zombiesAndProjectiles, func(i, j int) bool {
		posI, _ := ecs.GetComponent[*components.PositionComponent](s.entityManager, zombiesAndProjectiles[i])
		posJ, _ := ecs.GetComponent[*components.PositionComponent](s.entityManager, zombiesAndProjectiles[j])

		// Story 10.6: 修正 Y 坐标排序逻辑
		// 如果实体正在被压扁，应该使用其 OriginalPosY 参与排序
		// 否则动画过程中的 Y 位移（被铲起）会导致排序错误
		yI := posI.Y
		yJ := posJ.Y

		_, isSquashedI := ecs.GetComponent[*components.SquashAnimationComponent](s.entityManager, zombiesAndProjectiles[i])
		if isSquashedI {
			if squashI, ok := ecs.GetComponent[*components.SquashAnimationComponent](s.entityManager, zombiesAndProjectiles[i]); ok {
				yI = squashI.OriginalPosY
			}
		}

		_, isSquashedJ := ecs.GetComponent[*components.SquashAnimationComponent](s.entityManager, zombiesAndProjectiles[j])
		if isSquashedJ {
			if squashJ, ok := ecs.GetComponent[*components.SquashAnimationComponent](s.entityManager, zombiesAndProjectiles[j]); ok {
				yJ = squashJ.OriginalPosY
			}
		}

		// 主排序：按Y坐标（从小到大）
		// 使用 epsilon 处理浮点数误差，确保同一行的实体（即使有微小差异）进入二级排序
		if math.Abs(yI-yJ) > 1.0 {
			return yI < yJ
		}

		// Story 10.6: 压扁僵尸优先渲染（在底层），除草车后渲染（在上层）
		if isSquashedI != isSquashedJ {
			// 如果 i 被压扁 (true) 而 j 没有 (false)，i 应该先渲染 (return true)
			// 这样 i 就在 j 的下面（被 j 遮挡）
			// 此时 i 是僵尸，j 是除草车，符合"车碾过僵尸"的视觉效果
			return isSquashedI
		}

		// 二级排序：当Y坐标相同时，按网格列排序（从大到小，右侧先渲染）
		// 使用网格列而非连续坐标，避免同列内僵尸因微小位置差异导致渲染顺序抖动
		colI := int((posI.X - config.GridWorldStartX) / config.CellWidth)
		colJ := int((posJ.X - config.GridWorldStartX) / config.CellWidth)
		if colI != colJ {
			return colI > colJ
		}
		// 同一网格列内，按 X 坐标排序（右侧先渲染）
		return posI.X > posJ.X
	})

	// Story 10.7: 第二遍A：渲染僵尸阴影（中间层-阴影层）
	// 当僵尸进入房子时（Phase 2+），阴影也需要被门框剪裁
	if currentPhase >= 2 {
		s.drawZombieShadowsWithClipping(screen, zombiesAndProjectiles, cameraX, config.GameOverDoorMaskX)
	} else {
		s.drawZombieShadows(screen, zombiesAndProjectiles, cameraX)
	}

	// Story 8.8 - Task 6: Phase 2+ 时渲染房门图片
	// 渲染顺序：阴影层（underlay）→ 僵尸 → 门板层（mask）→ ZombiesWon动画（UI层）
	// 这样确保：门板遮挡僵尸，僵尸遮挡阴影
	if currentPhase >= 2 && s.resourceManager != nil {
		s.drawGameOverDoorUnderlay(screen, cameraX) // 阴影层（在僵尸下方）
	}

	// 按排序后的顺序渲染僵尸和子弹
	// Story 8.8 - Task 6: 如果在 Phase 2+，门板层会渲染在僵尸上方进行遮挡
	// 当僵尸完全走进门内（超过门板左边缘）时，才需要剪裁
	for _, id := range zombiesAndProjectiles {
		if currentPhase >= 2 {
			// 计算门板左边界的世界坐标
			// 僵尸超过此边界的部分将被完全隐藏（因为已经进入房子内部）
			doorLeftBoundary := config.GameOverDoorMaskX

			// 渲染僵尸时应用剪裁
			s.drawEntityWithClipping(screen, id, cameraX, doorLeftBoundary)
		} else {
			s.drawEntity(screen, id, cameraX)
		}
	}

	// 渲染房门上层图片（门板），遮挡僵尸
	// 注意：必须在僵尸之后、UI元素（ZombiesWon动画）之前渲染
	if currentPhase >= 2 && s.resourceManager != nil {
		s.drawGameOverDoorOverlay(screen, cameraX) // 门板层（在僵尸上方）
	}
}

// DrawSuns 单独渲染阳光（最顶层）
// 用于确保阳光显示在所有UI元素（包括植物卡片）之上，便于玩家点击收集
// 参数:
//   - screen: 绘制目标屏幕
//   - cameraX: 摄像机的世界坐标X位置
func (s *RenderSystem) DrawSuns(screen *ebiten.Image, cameraX float64) {
	// 所有实体都使用 ReanimComponent 渲染
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
	// 优先使用 ReanimComponent 渲染
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

// DrawUIElements 绘制所有 UI 元素（公开方法，供验证程序使用）
// 渲染所有标记为 UIComponent 的实体
//
// 参数:
//   - screen: 绘制目标屏幕
func (s *RenderSystem) DrawUIElements(screen *ebiten.Image) {
	// 查询所有有 UIComponent 和 PositionComponent 的实体
	uiEntities := ecs.GetEntitiesWith2[
		*components.PositionComponent,
		*components.UIComponent,
	](s.entityManager)

	// 渲染所有 UI 实体（UI 元素不受摄像机影响，cameraX = 0）
	for _, entityID := range uiEntities {
		// 跳过奖励动画实体（由 RewardAnimationSystem.Draw() 单独处理）
		// 奖励动画需要特殊的缩放处理，不能使用通用的 drawEntity
		if _, hasRewardAnim := ecs.GetComponent[*components.RewardAnimationComponent](s.entityManager, entityID); hasRewardAnim {
			continue
		}
		s.drawEntity(screen, entityID, 0)
	}
}
