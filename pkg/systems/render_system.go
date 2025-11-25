package systems

import (
	"image"
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
	// 当Y坐标相同时，按X坐标排序（从大到小，即从右到左）
	// 这样可以确保：
	//   1. 上方行的僵尸先绘制，下方行的僵尸后绘制会正确遮挡
	//   2. 同一行中，右侧的僵尸先绘制，左侧的僵尸后绘制会遮挡右侧（符合透视效果）
	//   3. 避免同行僵尸重叠时的渲染闪烁
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

		// 二级排序：当Y坐标相同时，按X坐标（从大到小，右侧先渲染）
		return posI.X > posJ.X
	})

	// Story 10.7: 第二遍A：渲染僵尸阴影（中间层-阴影层）
	s.drawZombieShadows(screen, zombiesAndProjectiles, cameraX)

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

// drawEntityWithClipping 绘制单个实体并应用剪裁
// Story 8.8 - Task 6: 用于僵尸走入房子时，剪裁超出门板左边界的部分
//
// 剪裁逻辑：
//   - 僵尸从右向左走进门
//   - 门板在左侧，遮挡僵尸
//   - 当僵尸的左边缘超过门板左边界时，需要剪裁僵尸的左侧部分
//   - 保留僵尸在门板右边（可见）的部分
//
// 参数:
//   - screen: 绘制目标屏幕
//   - id: 实体ID
//   - cameraX: 摄像机的世界坐标X位置
//   - clipLeftWorldX: 剪裁左边界的世界坐标（僵尸超过此边界的左侧部分将被隐藏）
func (s *RenderSystem) drawEntityWithClipping(screen *ebiten.Image, id ecs.EntityID, cameraX float64, clipLeftWorldX float64) {
	// 获取实体位置
	pos, hasPos := ecs.GetComponent[*components.PositionComponent](s.entityManager, id)
	if !hasPos {
		return
	}

	// 只对僵尸应用剪裁（检查是否有 BehaviorComponent 且是僵尸类型）
	behaviorComp, hasBehavior := ecs.GetComponent[*components.BehaviorComponent](s.entityManager, id)
	isZombie := hasBehavior && (behaviorComp.Type == components.BehaviorZombieBasic ||
		behaviorComp.Type == components.BehaviorZombieEating ||
		behaviorComp.Type == components.BehaviorZombieDying ||
		behaviorComp.Type == components.BehaviorZombieSquashing ||
		behaviorComp.Type == components.BehaviorZombieConehead ||
		behaviorComp.Type == components.BehaviorZombieBuckethead)

	if !isZombie {
		// 非僵尸实体正常渲染
		s.drawEntity(screen, id, cameraX)
		return
	}

	// 获取僵尸的 ReanimComponent 来估算宽度
	reanimComp, hasReanim := ecs.GetComponent[*components.ReanimComponent](s.entityManager, id)
	if !hasReanim {
		s.drawEntity(screen, id, cameraX)
		return
	}

	// 估算僵尸的渲染宽度（使用默认值，因为没有 BoundingBox 字段）
	zombieWidth := 150.0 // 默认僵尸宽度

	// 计算僵尸边缘的世界坐标
	zombieLeftWorldX := pos.X - reanimComp.CenterOffsetX
	zombieRightWorldX := zombieLeftWorldX + zombieWidth

	// 判断僵尸是否需要剪裁（三种情况）
	// 1. 僵尸完全在门板左侧（完全进入房子）：不渲染
	if zombieRightWorldX <= clipLeftWorldX {
		// 调试：记录僵尸被完全遮挡的情况
		log.Printf("[RenderSystem] Zombie fully hidden behind door: pos.X=%.2f, rightWorldX=%.2f, clipBoundary=%.2f",
			pos.X, zombieRightWorldX, clipLeftWorldX)
		return // 完全被遮挡，不渲染
	}

	// 2. 僵尸完全在门板右侧（未触碰到门）：正常渲染
	if zombieLeftWorldX >= clipLeftWorldX {
		s.drawEntity(screen, id, cameraX)
		return
	}

	// 3. 僵尸部分重叠（需要剪裁）
	// 创建临时图像来渲染僵尸
	// 临时图像尺寸需要足够大以容纳整个僵尸（包括负偏移的部件）
	leftPadding := 100.0                              // 左边距，防止部件渲染到负坐标外
	tempWidth := int(zombieWidth + leftPadding + 100) // 给足够的空间

	// BUG 修复：临时图像高度需要足够容纳整个僵尸（包括脚部）
	// 僵尸可能有 200 像素高，加上 CenterOffsetY 可能在 300+ 位置
	// 为了安全，使用更大的高度（600 像素）
	tempHeight := 600 // 足够的高度（原来是 400，导致脚部被裁剪）
	tempImg := ebiten.NewImage(tempWidth, tempHeight)
	defer tempImg.Dispose()

	// 计算僵尸在临时图像中的渲染位置
	// renderReanimEntity 使用公式: screenY = pos.Y - CenterOffsetY
	// 为了让僵尸渲染到临时图像的顶部附近，我们需要记录其在临时图像中的实际 Y 位置
	// 当前 pos.Y=347.74, CenterOffsetY=66.15，所以 screenY=281.59
	// 这会导致僵尸渲染到临时图像的 Y=281.59 位置
	zombieTopInTempImg := pos.Y - reanimComp.CenterOffsetY

	// 将僵尸渲染到临时图像
	// renderReanimEntity 使用公式: screenX = pos.X - cameraX - CenterOffsetX
	// 我们希望僵尸左边缘渲染到临时图像的 x=leftPadding 位置
	// screenX = leftPadding → cameraX = pos.X - CenterOffsetX - leftPadding = zombieLeftWorldX - leftPadding
	tempCameraX := zombieLeftWorldX - leftPadding
	s.renderReanimEntity(tempImg, id, tempCameraX)

	// 计算剪裁区域
	// 僵尸左边缘在临时图像中的位置是 leftPadding
	// 剪裁边界在临时图像中的位置是 (clipLeftWorldX - tempCameraX)
	// 我们要保留剪裁边界右侧的部分，剪掉左侧的部分
	clipInTempX := clipLeftWorldX - tempCameraX
	clipStartX := int(clipInTempX)
	if clipStartX < 0 {
		clipStartX = 0
	}

	// 调试：记录剪裁渲染的详细信息
	log.Printf("[RenderSystem] Clipping zombie: pos.X=%.2f, pos.Y=%.2f, leftWorldX=%.2f, clipBoundary=%.2f, clipStartX=%d, tempCameraX=%.2f, leftPadding=%.0f, CenterOffsetY=%.2f, zombieTopInTempImg=%.2f",
		pos.X, pos.Y, zombieLeftWorldX, clipLeftWorldX, clipStartX, tempCameraX, leftPadding, reanimComp.CenterOffsetY, zombieTopInTempImg)

	// 获取剪裁后的子图像
	// 保留从 clipStartX 到图像右边缘的部分（即门板右侧可见的部分）
	// Y 方向：从僵尸实际渲染的顶部开始，避免包含空白区域
	tempBounds := tempImg.Bounds()
	if clipStartX < tempBounds.Dx() {
		// 计算僵尸在临时图像中的 Y 范围
		zombieTopY := int(zombieTopInTempImg)
		if zombieTopY < 0 {
			zombieTopY = 0
		}

		// BUG 修复：SubImage 应该从僵尸实际渲染的 Y 位置开始，而不是从 0 开始
		// 否则会包含 0~zombieTopY 的空白区域，导致最终绘制时僵尸向下偏移
		clippedImg := tempImg.SubImage(image.Rect(
			clipStartX, zombieTopY, // 从僵尸顶部开始剪裁
			tempBounds.Dx(), tempBounds.Dy(),
		)).(*ebiten.Image)

		// 绘制剪裁后的图像到屏幕
		// X 坐标：剪裁后图像的左边缘对应门板左边界的世界坐标
		// Y 坐标：应该与未剪裁的僵尸渲染位置一致
		op := &ebiten.DrawImageOptions{}
		screenX := clipLeftWorldX - cameraX
		// Y 坐标：使用僵尸实际位置，减去 CenterOffsetY（与正常渲染一致）
		screenY := pos.Y - reanimComp.CenterOffsetY
		op.GeoM.Translate(screenX, screenY)

		// 调试：记录最终绘制位置
		log.Printf("[RenderSystem] Drawing clipped zombie at screenX=%.2f, screenY=%.2f (clipped from Y=%d in temp image)", screenX, screenY, zombieTopY)

		screen.DrawImage(clippedImg, op)
	}
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
// 如果 AnimVisiblesMap 中当前动画的 AnimVisibles 为空，说明使用 PlayAllFrames 模式，
// CurrentFrame 直接就是物理帧索引，无需映射。
//
// 参数:
//   - reanim: ReanimComponent 包含 AnimVisiblesMap
//   - logicalFrameNum: 逻辑帧号（从 0 开始）
//
// 返回:
//   - 物理帧索引，如果找不到则返回 -1

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
		s.drawEntity(screen, entityID, 0)
	}
}

// drawGameOverDoorUnderlay 渲染房门下层图片（阴影/右半部分）
// Story 8.8 - Task 6: 僵尸获胜流程期间显示房门打开效果
// 此图片在僵尸之前绘制，作为阴影层
//
// 参数:
//   - screen: 绘制目标屏幕
//   - cameraX: 摄像机X坐标
func (s *RenderSystem) drawGameOverDoorUnderlay(screen *ebiten.Image, cameraX float64) {
	if s.resourceManager == nil {
		return
	}

	// 加载房门下层图片（阴影）
	underlayImg := s.resourceManager.GetImageByID("IMAGE_BACKGROUND1_GAMEOVER_INTERIOR_OVERLAY")
	if underlayImg == nil {
		log.Printf("[RenderSystem] 警告：无法加载房门下层图片 IMAGE_BACKGROUND1_GAMEOVER_INTERIOR_OVERLAY")
		return
	}

	// 绘制房门下层图片
	// 坐标使用配置常量（可在 pkg/config/gameover_door_config.go 中调整）
	op := &ebiten.DrawImageOptions{}

	// 图片位置：世界坐标转换为屏幕坐标
	worldX := config.GameOverDoorInteriorOverlayX
	worldY := config.GameOverDoorInteriorOverlayY
	screenX := worldX - cameraX
	screenY := worldY
	op.GeoM.Translate(screenX, screenY)

	screen.DrawImage(underlayImg, op)
}

// drawGameOverDoorOverlay 渲染房门上层图片（门板/左半部分）
// Story 8.8 - Task 6: 僵尸获胜流程期间显示房门打开效果
// 此图片在僵尸之后绘制，遮挡僵尸以模拟进屋效果
//
// 参数:
//   - screen: 绘制目标屏幕
//   - cameraX: 摄像机X坐标
func (s *RenderSystem) drawGameOverDoorOverlay(screen *ebiten.Image, cameraX float64) {
	if s.resourceManager == nil {
		return
	}

	// 加载房门上层图片（门板）
	overlayImg := s.resourceManager.GetImageByID("IMAGE_BACKGROUND1_GAMEOVER_MASK")
	if overlayImg == nil {
		log.Printf("[RenderSystem] 警告：无法加载房门上层图片 IMAGE_BACKGROUND1_GAMEOVER_MASK")
		return
	}

	// 绘制房门上层图片
	// 坐标使用配置常量（可在 pkg/config/gameover_door_config.go 中调整）
	op := &ebiten.DrawImageOptions{}

	// 图片位置：世界坐标转换为屏幕坐标
	worldX := config.GameOverDoorMaskX
	worldY := config.GameOverDoorMaskY
	screenX := worldX - cameraX
	screenY := worldY
	op.GeoM.Translate(screenX, screenY)

	screen.DrawImage(overlayImg, op)
}

// drawPlantShadows 渲染植物阴影
// Story 10.7: 为植物添加阴影效果以增加场景深度感
//
// 阴影定位策略：
//   - 植物的 pos 是格子中心，阴影应该在格子底部中心（脚底位置）
//   - 格子底部 Y = pos.Y + CellHeight/2
//   - 阴影稍微上移一点，让它看起来在脚下而不是脚后面
//
// 参数:
//   - screen: 绘制目标屏幕
//   - entities: 所有实体的ID列表
//   - cameraX: 摄像机X坐标
func (s *RenderSystem) drawPlantShadows(screen *ebiten.Image, entities []ecs.EntityID, cameraX float64) {
	if s.resourceManager == nil {
		return
	}

	// 加载阴影贴图
	shadowImg := s.resourceManager.GetShadowImage()
	if shadowImg == nil {
		return // 阴影贴图加载失败，不渲染阴影
	}

	// 获取阴影贴图的原始尺寸
	shadowImgBounds := shadowImg.Bounds()
	shadowImgWidth := float64(shadowImgBounds.Dx())
	shadowImgHeight := float64(shadowImgBounds.Dy())

	// 遍历所有植物实体，渲染阴影
	for _, id := range entities {
		// 跳过非植物实体
		_, isPlant := ecs.GetComponent[*components.PlantComponent](s.entityManager, id)
		if !isPlant {
			continue
		}

		// 获取位置组件
		pos, hasPos := ecs.GetComponent[*components.PositionComponent](s.entityManager, id)
		if !hasPos {
			continue
		}

		// 计算阴影位置：格子底部中心，稍微上移让阴影在脚下
		// 格子底部 Y = pos.Y + CellHeight/2
		// 阴影上移偏移量，让阴影看起来在植物脚下而不是脚后面
		shadowOffsetY := config.PlantShadowOffsetY // 可配置的偏移量
		footY := pos.Y + config.CellHeight/2 + shadowOffsetY
		screenX := pos.X - shadowImgWidth/2 - cameraX
		screenY := footY - shadowImgHeight/2

		// 应用变换和透明度
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(screenX, screenY)
		op.ColorScale.ScaleAlpha(config.DefaultShadowAlpha) // 使用配置的透明度

		// 绘制阴影
		screen.DrawImage(shadowImg, op)
	}
}

// drawZombieShadows 渲染僵尸阴影
// Story 10.7: 为僵尸添加阴影效果以增加场景深度感
//
// 阴影定位策略：
//   - 僵尸的 pos.Y 是格子中心 + ZombieVerticalOffset
//   - 僵尸脚底位置约在格子底部
//   - 阴影绘制在格子底部中心
//
// 参数:
//   - screen: 绘制目标屏幕
//   - zombieEntities: 僵尸实体的ID列表（已按Y坐标排序）
//   - cameraX: 摄像机X坐标
func (s *RenderSystem) drawZombieShadows(screen *ebiten.Image, zombieEntities []ecs.EntityID, cameraX float64) {
	if s.resourceManager == nil {
		return
	}

	// 加载阴影贴图
	shadowImg := s.resourceManager.GetShadowImage()
	if shadowImg == nil {
		return // 阴影贴图加载失败，不渲染阴影
	}

	// 获取阴影贴图的原始尺寸
	shadowImgBounds := shadowImg.Bounds()
	shadowImgWidth := float64(shadowImgBounds.Dx())
	shadowImgHeight := float64(shadowImgBounds.Dy())

	// 遍历所有僵尸实体，渲染阴影
	for _, id := range zombieEntities {
		// 只渲染有 BehaviorComponent 且是僵尸类型的实体
		behaviorComp, hasBehavior := ecs.GetComponent[*components.BehaviorComponent](s.entityManager, id)
		if !hasBehavior {
			continue
		}

		// 检查是否是僵尸类型
		isZombie := behaviorComp.Type == components.BehaviorZombieBasic ||
			behaviorComp.Type == components.BehaviorZombieEating ||
			behaviorComp.Type == components.BehaviorZombieDying ||
			behaviorComp.Type == components.BehaviorZombieSquashing ||
			behaviorComp.Type == components.BehaviorZombieConehead ||
			behaviorComp.Type == components.BehaviorZombieBuckethead

		if !isZombie {
			continue
		}

		// 获取位置组件
		pos, hasPos := ecs.GetComponent[*components.PositionComponent](s.entityManager, id)
		if !hasPos {
			continue
		}

		// 计算阴影位置：僵尸脚底中心
		// 僵尸 pos.Y = 格子中心 + ZombieVerticalOffset
		// 脚底位置 = pos.Y - ZombieVerticalOffset + CellHeight/2 + shadowOffsetY
		// 简化为：格子底部 + shadowOffsetY
		shadowOffsetX := config.ZombieShadowOffsetX // 可配置的 X 偏移量
		shadowOffsetY := config.ZombieShadowOffsetY // 可配置的 Y 偏移量
		footY := pos.Y - config.ZombieVerticalOffset + config.CellHeight/2 + shadowOffsetY
		screenX := pos.X - shadowImgWidth/2 - cameraX + shadowOffsetX
		screenY := footY - shadowImgHeight/2

		// 应用变换和透明度
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(screenX, screenY)
		op.ColorScale.ScaleAlpha(config.DefaultShadowAlpha) // 使用配置的透明度

		// 绘制阴影
		screen.DrawImage(shadowImg, op)
	}
}

