// render_shadow.go - 阴影渲染相关方法
//
// 本文件包含 RenderSystem 的阴影渲染功能：
//   - drawPlantShadows: 渲染植物阴影
//   - drawZombieShadows: 渲染僵尸阴影（无剪裁版本）
//   - drawZombieShadowsWithClipping: 渲染僵尸阴影（带剪裁版本）
//
// 所有方法都是 RenderSystem 的成员方法（接收者：*RenderSystem）。
// 使用相同的 package systems，可以直接访问 RenderSystem 的私有字段。

package systems

import (
	"image"

	"github.com/gonewx/pvz/pkg/components"
	"github.com/gonewx/pvz/pkg/config"
	"github.com/gonewx/pvz/pkg/ecs"
	"github.com/hajimehoshi/ebiten/v2"
)

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

		// 计算阴影位置：跟随植物本体
		// 植物 pos.Y 是本体位置，阴影应该在本体脚下
		// PlantShadowOffsetY 用于微调阴影相对于本体的垂直偏移
		shadowOffsetY := config.PlantShadowOffsetY
		footY := pos.Y + shadowOffsetY
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

// drawZombieShadowsWithClipping 渲染僵尸阴影
// Story 10.7: 为僵尸添加阴影效果以增加场景深度感
//
// 阴影定位策略：
//   - 僵尸的 pos.Y 是格子中心 + ZombieVerticalOffset
//   - 僵尸脚底位置约在格子底部
//   - 阴影绘制在格子底部中心
//
// 剪裁策略（僵尸进入房子时）：
//   - 当 clipLeftWorldX > 0 时，阴影超过门板左边界的部分会被剪裁
//   - 确保阴影与僵尸本身保持一致的渲染层级
//
// 参数:
//   - screen: 绘制目标屏幕
//   - zombieEntities: 僵尸实体的ID列表（已按Y坐标排序）
//   - cameraX: 摄像机X坐标
//   - clipLeftWorldX: 剪裁左边界的世界坐标（0 表示不剪裁）
func (s *RenderSystem) drawZombieShadowsWithClipping(screen *ebiten.Image, zombieEntities []ecs.EntityID, cameraX float64, clipLeftWorldX float64) {
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
			behaviorComp.Type == components.BehaviorZombieBuckethead ||
			behaviorComp.Type == components.BehaviorZombieFlag ||
			behaviorComp.Type == components.BehaviorZombiePolevaulter ||
			behaviorComp.Type == components.BehaviorZombiePreview

		if !isZombie {
			continue
		}

		// 撑杆僵尸跳跃期间隐藏阴影（包括等待位移补偿的那一帧）
		if poleVault, hasPoleVault := ecs.GetComponent[*components.PoleVaultComponent](s.entityManager, id); hasPoleVault {
			if poleVault.IsJumping || poleVault.NeedPositionCompensation {
				continue
			}
		}

		// 获取位置组件
		pos, hasPos := ecs.GetComponent[*components.PositionComponent](s.entityManager, id)
		if !hasPos {
			continue
		}

		// 计算阴影位置：跟随僵尸本体
		// 僵尸 pos.Y 是本体位置，阴影应该在本体脚下
		// ZombieShadowOffsetX/Y 用于微调阴影相对于本体的偏移
		shadowOffsetX := config.ZombieShadowOffsetX
		shadowOffsetY := config.ZombieShadowOffsetY
		footY := pos.Y + shadowOffsetY
		shadowWorldX := pos.X - shadowImgWidth/2 + shadowOffsetX
		shadowWorldRightX := shadowWorldX + shadowImgWidth
		screenX := shadowWorldX - cameraX
		screenY := footY - shadowImgHeight/2

		// 如果需要剪裁（僵尸进入房子）
		if clipLeftWorldX > 0 {
			// 1. 阴影完全在门板左侧（完全被遮挡）：不渲染
			if shadowWorldRightX <= clipLeftWorldX {
				continue
			}

			// 2. 阴影完全在门板右侧（无需剪裁）：正常渲染
			if shadowWorldX >= clipLeftWorldX {
				op := &ebiten.DrawImageOptions{}
				op.GeoM.Translate(screenX, screenY)
				op.ColorScale.ScaleAlpha(config.DefaultShadowAlpha)
				screen.DrawImage(shadowImg, op)
				continue
			}

			// 3. 阴影部分重叠（需要剪裁）
			// 计算剪裁起始位置（相对于阴影图片左边缘）
			clipStartX := int(clipLeftWorldX - shadowWorldX)
			if clipStartX < 0 {
				clipStartX = 0
			}

			// 获取剪裁后的子图像
			if clipStartX < shadowImgBounds.Dx() {
				clippedShadow := shadowImg.SubImage(image.Rect(
					clipStartX, 0,
					shadowImgBounds.Dx(), shadowImgBounds.Dy(),
				)).(*ebiten.Image)

				// 绘制剪裁后的阴影
				op := &ebiten.DrawImageOptions{}
				clippedScreenX := clipLeftWorldX - cameraX
				op.GeoM.Translate(clippedScreenX, screenY)
				op.ColorScale.ScaleAlpha(config.DefaultShadowAlpha)
				screen.DrawImage(clippedShadow, op)
			}
			continue
		}

		// 无需剪裁：正常渲染
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(screenX, screenY)
		op.ColorScale.ScaleAlpha(config.DefaultShadowAlpha) // 使用配置的透明度

		// 绘制阴影
		screen.DrawImage(shadowImg, op)
	}
}

// drawZombieShadows 渲染僵尸阴影（无剪裁版本，向后兼容）
func (s *RenderSystem) drawZombieShadows(screen *ebiten.Image, zombieEntities []ecs.EntityID, cameraX float64) {
	s.drawZombieShadowsWithClipping(screen, zombieEntities, cameraX, 0)
}

// drawProjectileShadows 渲染子弹阴影
// 为飞行中的子弹（豌豆、冰豌豆）渲染阴影以增加深度感
//
// 参数:
//   - screen: 绘制目标屏幕
//   - entities: 所有实体的ID列表
//   - cameraX: 摄像机X坐标
func (s *RenderSystem) drawProjectileShadows(screen *ebiten.Image, entities []ecs.EntityID, cameraX float64) {
	if s.resourceManager == nil {
		return
	}

	// 加载阴影贴图
	shadowImg := s.resourceManager.GetShadowImage()
	if shadowImg == nil {
		return
	}

	// 获取阴影贴图的原始尺寸
	shadowImgBounds := shadowImg.Bounds()
	shadowImgWidth := float64(shadowImgBounds.Dx())
	shadowImgHeight := float64(shadowImgBounds.Dy())

	// 遍历所有子弹实体，渲染阴影
	for _, id := range entities {
		// 只渲染有 BehaviorComponent 且是子弹类型的实体
		behaviorComp, hasBehavior := ecs.GetComponent[*components.BehaviorComponent](s.entityManager, id)
		if !hasBehavior {
			continue
		}

		// 检查是否是子弹类型
		isProjectile := behaviorComp.Type == components.BehaviorPeaProjectile ||
			behaviorComp.Type == components.BehaviorSnowPeaProjectile

		if !isProjectile {
			continue
		}

		// 获取位置组件
		pos, hasPos := ecs.GetComponent[*components.PositionComponent](s.entityManager, id)
		if !hasPos {
			continue
		}

		// 获取阴影组件
		shadowComp, hasShadow := ecs.GetComponent[*components.ShadowComponent](s.entityManager, id)
		if !hasShadow {
			continue
		}

		// 获取 ReanimComponent 来计算子弹图片宽度
		// 简单实体的 CenterOffsetX = 0，图片从 pos.X 开始向右绘制
		// 需要找到图片中心来对齐阴影
		bulletCenterX := pos.X // 默认使用 pos.X
		if reanim, hasReanim := ecs.GetComponent[*components.ReanimComponent](s.entityManager, id); hasReanim {
			// 获取图片实际宽度
			for _, img := range reanim.PartImages {
				if img != nil {
					imgWidth := float64(img.Bounds().Dx())
					// 子弹图片左边缘在 pos.X - CenterOffsetX，中心在 pos.X - CenterOffsetX + imgWidth/2
					bulletCenterX = pos.X - reanim.CenterOffsetX + imgWidth/2
					break
				}
			}
		}

		// 计算阴影位置：阴影中心对齐子弹中心
		shadowOffsetY := shadowComp.OffsetY
		footY := pos.Y + shadowOffsetY
		screenX := bulletCenterX - shadowComp.Width/2 - cameraX
		screenY := footY - shadowComp.Height/2

		// 计算缩放比例（子弹阴影比默认阴影小）
		scaleX := shadowComp.Width / shadowImgWidth
		scaleY := shadowComp.Height / shadowImgHeight

		// 应用变换和透明度
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Scale(scaleX, scaleY)
		op.GeoM.Translate(screenX, screenY)
		op.ColorScale.ScaleAlpha(shadowComp.Alpha)

		// 绘制阴影
		screen.DrawImage(shadowImg, op)
	}
}
