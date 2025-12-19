// render_gameover.go - 游戏结束/房门渲染相关方法
//
// 本文件包含 RenderSystem 的游戏结束（僵尸获胜）渲染功能：
//   - drawEntityWithClipping: 绘制实体并应用剪裁
//   - drawGameOverDoorUnderlay: 渲染房门下层图片（阴影）
//   - drawGameOverDoorOverlay: 渲染房门上层图片（门板）
//
// 所有方法都是 RenderSystem 的成员方法（接收者：*RenderSystem）。
// 使用相同的 package systems，可以直接访问 RenderSystem 的私有字段。

package systems

import (
	"image"
	"log"

	"github.com/gonewx/pvz/pkg/components"
	"github.com/gonewx/pvz/pkg/config"
	"github.com/gonewx/pvz/pkg/ecs"
	"github.com/hajimehoshi/ebiten/v2"
)

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

	// 只对僵尸应用剪裁（使用 ZombieTagComponent 统一判断）
	// Story 8.16: 使用标签组件而非硬编码类型列表，新增僵尸类型自动被识别
	_, isZombie := ecs.GetComponent[*components.ZombieTagComponent](s.entityManager, id)

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
