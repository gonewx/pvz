package systems

import (
	"log"
	"math"
	"strings"

	"github.com/decker502/pvz/internal/reanim"
	"github.com/decker502/pvz/pkg/components"
	"github.com/decker502/pvz/pkg/config"
	"github.com/decker502/pvz/pkg/ecs"
	"github.com/decker502/pvz/pkg/types"
	"github.com/hajimehoshi/ebiten/v2"
)

// ==================================================================
// 渲染缓存 (Render Cache)
// ==================================================================

// prepareRenderCache 准备渲染缓存
// 新逻辑：外层循环动画，内层循环轨道，后面的动画自然覆盖前面的动画
func (s *ReanimSystem) prepareRenderCache(comp *components.ReanimComponent) {
	// Debug: 无条件打印向日葵和 SodRoll 的缓存准备
	if comp.ReanimName == "sunflower" && comp.CurrentFrame >= 4 && comp.CurrentFrame <= 10 {
		log.Printf("[ReanimSystem] 🌻 prepareRenderCache 被调用: frame=%d", comp.CurrentFrame)
	}
	if comp.ReanimName == "sodroll" && comp.CurrentFrame >= 4 && comp.CurrentFrame <= 10 {
		log.Printf("[ReanimSystem] 🟫 SodRoll prepareRenderCache 被调用: frame=%d, VisualTracks=%d",
			comp.CurrentFrame, len(comp.VisualTracks))
	}
	if comp.ReanimName == "SelectorScreen" && comp.CurrentFrame >= 4 && comp.CurrentFrame <= 100 {
		log.Printf("[ReanimSystem] 🎬 SelectorScreen prepareRenderCache 被调用: frame=%d, animations=%v",
			comp.CurrentFrame, comp.CurrentAnimations)
	}
	// 重用切片避免分配
	comp.CachedRenderData = comp.CachedRenderData[:0]

	// 先渲染叠加动画（如旗帜僵尸的旗杆），使其在主动画后面
	s.renderOverlayAnimation(comp)

	visibleCount := 0
	skippedHidden := 0
	skippedPaused := 0
	skippedNoFrames := 0
	skippedNoImage := 0

	trackRenderSource := make(map[string]string)

	// 这样可以确保云朵轨道（Track 16-21）在按钮轨道（Track 27+）之前添加到 CachedRenderData
	// 从而在渲染时云朵在下面，按钮在上面
	for _, trackName := range comp.VisualTracks {
		// Debug: SelectorScreen 打印轨道处理情况（前10帧）
		if comp.ReanimName == "SelectorScreen" && comp.CurrentFrame < 10 {
			log.Printf("[ReanimSystem] 🎨 处理轨道: %s", trackName)
		}
		// 检查隐藏轨道（黑名单）
		if comp.HiddenTracks != nil && comp.HiddenTracks[trackName] {
			skippedHidden++
			continue
		}

		// 获取该轨道的合并帧数据
		mergedFrames, ok := comp.MergedTracks[trackName]
		if !ok {
			skippedNoFrames++
			continue
		}

		// 用于存储该轨道的最终选中数据（后面的动画会覆盖前面的）
		var selectedFrame reanim.Frame
		var selectedImg *ebiten.Image
		var selectedOffsetX, selectedOffsetY float64
		var hasValidSelection bool

		// 内层循环：遍历所有动画，找到最后一个有效的数据
		for _, animName := range comp.CurrentAnimations {
			// 注意：暂停的动画仍然需要渲染当前帧，只是不推进帧索引
			// 所以这里不跳过暂停的动画（与 Update 函数不同）

			// 获取该动画的当前逻辑帧（支持独立帧索引）
			var logicalFrame float64
			if comp.AnimationFrameIndices != nil {
				if frame, exists := comp.AnimationFrameIndices[animName]; exists {
					logicalFrame = frame
				} else {
					logicalFrame = float64(comp.CurrentFrame)
				}
			} else {
				logicalFrame = float64(comp.CurrentFrame)
			}

			// 检查轨道是否被冻结（FrozenTracks）
			// 冻结的轨道始终使用第一帧，不随动画更新
			if comp.FrozenTracks != nil && comp.FrozenTracks[trackName] {
				logicalFrame = 0
			}

			// 获取动画的可见性数组
			animVisibles, ok := comp.AnimVisiblesMap[animName]
			if !ok {
				if comp.ReanimName == "simple_pea" {
					log.Printf("[ReanimSystem] simple_pea: AnimVisiblesMap[%s] 不存在, VisualTracks=%v, CurrentAnimations=%v",
						animName, comp.VisualTracks, comp.CurrentAnimations)
				}
				continue
			}

			// 映射逻辑帧到物理帧
			// 特殊处理：对于单动画文件（使用合成动画名 "_root"），逻辑帧=物理帧
			var physicalFrame int
			isSyntheticAnim := animName == "_root" || strings.HasPrefix(animName, "_")
			if isSyntheticAnim {
				// 单动画文件：直接使用逻辑帧作为物理帧
				physicalFrame = int(logicalFrame)
				// Story 8.8: 修复 ZombiesWon 动画越界问题
				// 如果是非循环动画且已播放到最后，钳制到最后一帧
				if !comp.IsLooping && physicalFrame >= len(mergedFrames) {
					physicalFrame = len(mergedFrames) - 1
				}
			} else {
				// 命名动画：使用 AnimVisibles 映射
				physicalFrame = MapLogicalToPhysical(int(logicalFrame), animVisibles)
			}

			// Debug: SodRoll 帧映射（前 15 帧）
			if comp.ReanimName == "sodroll" && comp.CurrentFrame < 15 {
				log.Printf("[ReanimSystem] 🟫 SodRoll Frame %d: trackName=%s, animName=%s, logicalFrame=%.2f, physicalFrame=%d, isSynthetic=%v",
					comp.CurrentFrame, trackName, animName, logicalFrame, physicalFrame, isSyntheticAnim)
			}

			if physicalFrame < 0 || physicalFrame >= len(mergedFrames) {
				if comp.ReanimName == "ZombiesWon" {
					log.Printf("[ReanimSystem] 🧟 ZombiesWon: ❌ physicalFrame 越界 (physicalFrame=%d, mergedFrames=%d)",
						physicalFrame, len(mergedFrames))
				}
				continue
			}

			// 检查动画定义轨道是否可见（f != -1）
			// 对于单动画文件（使用合成动画名如 "_root"），跳过这个检查
			// 因为 MergedTracks 只包含轨道名称，不包含合成的动画名称
			// isSyntheticAnim 已在上面定义
			if !isSyntheticAnim {
				// 只对命名动画（named animations）进行动画定义轨道检查
				animDefTrack, ok := comp.MergedTracks[animName]
				if !ok || physicalFrame >= len(animDefTrack) {
					continue
				}

				defFrame := animDefTrack[physicalFrame]
				if defFrame.FrameNum != nil && *defFrame.FrameNum == -1 {
					// 动画隐藏，跳过整个动画
					continue
				}
			}

			// 检查视觉轨道在该帧是否被隐藏（f=-1）
			currentTrackFrame := mergedFrames[physicalFrame]

			if currentTrackFrame.FrameNum != nil && *currentTrackFrame.FrameNum == -1 {
				// 视觉轨道在该帧被隐藏，跳过
				skippedHidden++
				if comp.ReanimName == "SelectorScreen" && comp.CurrentFrame < 10 {
					log.Printf("[ReanimSystem]   - 动画 %s: 轨道被隐藏 (f=-1)", animName)
				}
				continue
			}

			// 使用帧插值获取平滑的帧数据
			frame := s.getInterpolatedFrame(animName, logicalFrame, animVisibles, mergedFrames)

			// 图片继承逻辑：如果插值后的帧没有图片，向前搜索最近的有图片的帧
			hasValidImage := false
			if frame.ImagePath == "" {
				// 向前搜索有图片的帧（只搜索当前动画的可见帧范围）
				for i := physicalFrame - 1; i >= 0; i-- {
					isFrameVisible := false
					for _, visibleFrame := range animVisibles {
						if visibleFrame == i {
							isFrameVisible = true
							break
						}
					}
					if !isFrameVisible {
						break
					}

					if i < len(mergedFrames) && mergedFrames[i].ImagePath != "" {
						frame.ImagePath = mergedFrames[i].ImagePath
						hasValidImage = true
						break
					}
				}
			} else {
				hasValidImage = true
			}

			// 如果当前动画在这个轨道没有有效图片，跳过
			if !hasValidImage {
				skippedNoImage++
				if comp.ReanimName == "SelectorScreen" && comp.CurrentFrame < 10 {
					log.Printf("[ReanimSystem]   - 动画 %s: 无有效图片", animName)
				}
				continue
			}

			// 获取图片（优先使用 ImageOverrides）
			var img *ebiten.Image
			var imgOk bool
			if comp.ImageOverrides != nil {
				if overrideImg, hasOverride := comp.ImageOverrides[frame.ImagePath]; hasOverride && overrideImg != nil {
					img = overrideImg
					imgOk = true
				}
			}
			if !imgOk {
				img, imgOk = comp.PartImages[frame.ImagePath]
			}
			if !imgOk || img == nil {
				if comp.ReanimName == "simple_pea" {
					partImagesKeys := make([]string, 0, len(comp.PartImages))
					for k := range comp.PartImages {
						partImagesKeys = append(partImagesKeys, k)
					}
					log.Printf("[ReanimSystem] simple_pea: PartImages[%s] 不存在或为 nil (ok=%v, img==nil=%v), PartImages keys=%v",
						frame.ImagePath, ok, (img == nil), partImagesKeys)
				}
				continue
			}

			// 计算父轨道偏移
			offsetX, offsetY := 0.0, 0.0
			if parentTrackName, hasParent := comp.ParentTracks[trackName]; hasParent {
				offsetX, offsetY = s.getParentOffsetForAnimation(comp, parentTrackName, animName)
				// Debug: 豌豆射手的父偏移（前10帧）
				if (comp.ReanimName == "peashooter" || comp.ReanimName == "peashootersingle") && comp.CurrentFrame < 10 {
					log.Printf("[ReanimSystem] ParentOffset: track=%s, parent=%s, anim=%s, offset=(%.2f, %.2f)",
						trackName, parentTrackName, animName, offsetX, offsetY)
				}
			}

			// 更新选中数据（后面的动画会覆盖前面的）
			selectedFrame = frame
			selectedImg = img
			selectedOffsetX = offsetX
			selectedOffsetY = offsetY
			hasValidSelection = true
			trackRenderSource[trackName] = animName

			// Debug: SelectorScreen 记录选中的动画
			if comp.ReanimName == "SelectorScreen" && comp.CurrentFrame < 10 {
				log.Printf("[ReanimSystem]   - 动画 %s: 有效数据，选中", animName)
			}
		}

		// 如果该轨道有有效选中数据，添加到缓存
		if hasValidSelection {
			// 应用轨道偏移（用于抖动效果）
			if comp.TrackOffsets != nil {
				if offset, ok := comp.TrackOffsets[trackName]; ok {
					selectedOffsetX += offset[0]
					selectedOffsetY += offset[1]
				}
			}

			comp.CachedRenderData = append(comp.CachedRenderData, components.RenderPartData{
				Img:     selectedImg,
				Frame:   selectedFrame,
				OffsetX: selectedOffsetX,
				OffsetY: selectedOffsetY,
			})
			visibleCount++
		}
	}

	if comp.ReanimName == "SelectorScreen" && comp.CurrentFrame < 10 {
		log.Printf("[ReanimSystem] 📊 Frame %d 渲染统计 (总计: %d 个轨道):", comp.CurrentFrame, visibleCount)
		for _, trackName := range comp.VisualTracks {
			if source, ok := trackRenderSource[trackName]; ok {
				log.Printf("    - 轨道 %s: 来自动画 %s", trackName, source)
			}
		}
	}

	// Debug: 只在有变化时输出日志（避免刷屏）
	// 特殊调试：向日葵每帧都打印（前 10 帧）
	if comp.ReanimName == "sunflower" && comp.CurrentFrame < 10 {
		log.Printf("[ReanimSystem] 🔍 SunFlower frame %d → %d visible parts (skipped: hidden=%d, paused=%d, noFrames=%d, noImage=%d)",
			comp.CurrentFrame, visibleCount, skippedHidden, skippedPaused, skippedNoFrames, skippedNoImage)
	} else if len(comp.CachedRenderData) > 0 && comp.CurrentFrame%30 == 0 {
		log.Printf("[ReanimSystem] prepareRenderCache: %s frame %d → %d visible parts (skipped: hidden=%d, paused=%d, noFrames=%d, noImage=%d)",
			comp.ReanimName, comp.CurrentFrame, visibleCount, skippedHidden, skippedPaused, skippedNoFrames, skippedNoImage)
	}
}

// GetRenderData 获取渲染数据（供 RenderSystem 使用）
// 如果缓存失效，会自动重建缓存
// Story 10.6: 如果实体有 SquashAnimationComponent，跳过缓存重建
// 因为压扁动画的变换是在 LawnmowerSystem.ApplySquashTransforms() 中手动应用的
func (s *ReanimSystem) GetRenderData(entityID ecs.EntityID) []components.RenderPartData {
	comp, ok := ecs.GetComponent[*components.ReanimComponent](s.entityManager, entityID)
	if !ok {
		return nil
	}

	// 问题：使用整数 CurrentFrame 判断缓存失效，导致慢速动画（如 speed=0.05）
	//       的插值帧被忽略（帧 0.05、0.10...0.95 都被当作帧 0）
	// 解决：检查任意动画的浮点帧索引是否改变，确保插值生效
	needRebuild := false

	// 方法 1: 检查 AnimationFrameIndices 中是否有任何帧索引发生变化
	if comp.AnimationFrameIndices != nil && len(comp.AnimationFrameIndices) > 0 {
		// 计算所有动画的帧索引之和（作为缓存键）
		currentFrameSum := 0.0
		for _, frameIdx := range comp.AnimationFrameIndices {
			currentFrameSum += frameIdx
		}

		// Debug: SodRoll 缓存更新检查（前 15 帧）
		if comp.ReanimName == "sodroll" && comp.CurrentFrame < 15 {
			log.Printf("[ReanimSystem] 🟫 SodRoll GetRenderData Frame %d: currentFrameSum=%.3f, LastRenderFrame=%d, needRebuild=%v",
				comp.CurrentFrame, currentFrameSum, comp.LastRenderFrame, comp.LastRenderFrame == -1 || float64(comp.LastRenderFrame) != currentFrameSum)
		}

		// 如果帧索引和发生变化，或者是首次渲染
		if comp.LastRenderFrame == -1 || float64(comp.LastRenderFrame) != currentFrameSum {
			needRebuild = true
			comp.LastRenderFrame = int(currentFrameSum * 1000) // 使用千分之一精度作为缓存键
		}
	} else {
		// 后备逻辑：使用整数 CurrentFrame（兼容旧代码）
		// Debug: SodRoll 后备逻辑（前 15 帧）
		if comp.ReanimName == "sodroll" && comp.CurrentFrame < 15 {
			log.Printf("[ReanimSystem] 🟫 SodRoll GetRenderData（后备逻辑） Frame %d: LastRenderFrame=%d, CurrentFrame=%d, needRebuild=%v",
				comp.CurrentFrame, comp.LastRenderFrame, comp.CurrentFrame, comp.LastRenderFrame != comp.CurrentFrame)
		}
		if comp.LastRenderFrame != comp.CurrentFrame {
			needRebuild = true
			comp.LastRenderFrame = comp.CurrentFrame
		}
	}

	// Debug: SelectorScreen 前30帧打印
	if comp.ReanimName == "SelectorScreen" && comp.CurrentFrame >= 4 && comp.CurrentFrame <= 100 {
		log.Printf("[ReanimSystem] 🎨 GetRenderData: frame=%d, lastRenderFrame=%d, needRebuild=%v",
			comp.CurrentFrame, comp.LastRenderFrame, needRebuild)
	}

	// 重建缓存
	if needRebuild {
		// Debug: SodRoll 缓存重建（前 15 帧）
		if comp.ReanimName == "sodroll" && comp.CurrentFrame < 15 {
			log.Printf("[ReanimSystem] 🟫 SodRoll 重建缓存: Frame %d, needRebuild=true", comp.CurrentFrame)
		}
		s.prepareRenderCache(comp)
	}

	return comp.CachedRenderData
}

// ==================================================================
// 兼容性方法（临时保留，用于过渡）
// ==================================================================

// InitReanimComponent 初始化 Reanim 组件的基础数据
// 用于实体工厂创建实体时的初始化
func (s *ReanimSystem) InitReanimComponent(
	entityID ecs.EntityID,
	reanimName string,
	reanimXML *reanim.ReanimXML,
	partImages map[string]*ebiten.Image,
	mergedTracks map[string][]reanim.Frame,
	visualTracks []string,
	logicalTracks []string,
) error {
	comp, ok := ecs.GetComponent[*components.ReanimComponent](s.entityManager, entityID)
	if !ok {
		return errEntityNoReanimComponent(entityID)
	}

	comp.ReanimName = reanimName
	comp.ReanimXML = reanimXML
	comp.PartImages = partImages
	comp.MergedTracks = mergedTracks
	comp.VisualTracks = visualTracks
	comp.LogicalTracks = logicalTracks
	comp.AnimationFPS = float64(reanimXML.FPS)
	comp.IsLooping = true
	comp.LastRenderFrame = -1

	return nil
}

// PrepareStaticPreview prepares a Reanim entity for static preview (e.g., plant card icons).
// 使用配置驱动的方式选择最佳预览帧和隐藏轨道
//
// 策略（按优先级）：
// 1. 从 config.PlantConfigs 获取植物配置
// 2. 如果配置了 PreviewFrame >= 0，使用配置的帧
// 3. 如果 PreviewFrame == -1，使用动画的中间帧（自动选择）
// 4. 应用 HiddenTracks 配置（黑名单模式）隐藏不需要的轨道
// 5. 暂停动画播放（IsPaused = true）
//
// Parameters:
//   - entityID: the ID of the entity to prepare for static preview
//   - plantType: 植物类型（types.PlantType）
//
// Returns:
//   - An error if preparation fails
func (s *ReanimSystem) PrepareStaticPreview(entityID ecs.EntityID, plantType types.PlantType) error {
	// 从配置获取植物信息
	cfg := config.GetPlantConfig(plantType)
	if cfg == nil {
		return errNoPlantConfig(plantType)
	}

	// 使用 PlayCombo 播放默认动画
	if err := s.PlayCombo(entityID, cfg.ConfigID, ""); err != nil {
		return errPlayDefaultAnimation(err)
	}

	// 获取组件
	comp, ok := ecs.GetComponent[*components.ReanimComponent](s.entityManager, entityID)
	if !ok {
		return errEntityNoReanimComponent(entityID)
	}

	// 应用 HiddenTracks 配置（黑名单模式）
	if len(cfg.HiddenTracks) > 0 {
		if comp.HiddenTracks == nil {
			comp.HiddenTracks = make(map[string]bool)
		}
		for _, trackName := range cfg.HiddenTracks {
			comp.HiddenTracks[trackName] = true
		}
		log.Printf("[ReanimSystem] PrepareStaticPreview: %s hiding %d tracks: %v",
			cfg.ConfigID, len(cfg.HiddenTracks), cfg.HiddenTracks)
	}

	// 查找最佳预览帧
	var targetFrame int
	if cfg.PreviewFrame >= 0 {
		// 策略 1：使用配置的帧
		targetFrame = cfg.PreviewFrame
		log.Printf("[ReanimSystem] PrepareStaticPreview: %s using configured frame %d",
			cfg.ConfigID, cfg.PreviewFrame)
	} else if len(comp.CurrentAnimations) > 0 {
		// 策略 2：自动选择中间帧
		animName := comp.CurrentAnimations[0]
		if visibles, ok := comp.AnimVisiblesMap[animName]; ok && len(visibles) > 0 {
			targetFrame = len(visibles) / 2
			log.Printf("[ReanimSystem] PrepareStaticPreview: %s auto-selected frame %d/%d",
				cfg.ConfigID, targetFrame, len(visibles))
		}
	}

	// 同步设置 CurrentFrame 和 AnimationFrameIndices
	// 修复：渲染时优先使用 AnimationFrameIndices，所以必须同步更新
	comp.CurrentFrame = targetFrame
	if comp.AnimationFrameIndices != nil {
		for _, animName := range comp.CurrentAnimations {
			comp.AnimationFrameIndices[animName] = float64(targetFrame)
		}
	}

	// 暂停动画播放（静态预览）
	comp.IsPaused = true
	comp.IsLooping = false

	// 强制更新渲染缓存
	s.prepareRenderCache(comp)

	return nil
}

// RenderToTexture 将指定实体的 Reanim 渲染到目标纹理（离屏渲染）
// 用于生成植物卡片图标等静态纹理
//
// 参数：
//   - entityID: 实体 ID
//   - target: 目标纹理（调用者创建）
//
// 返回：
//   - error: 如果实体不存在或没有必要组件，返回错误
func (s *ReanimSystem) RenderToTexture(entityID ecs.EntityID, target *ebiten.Image) error {
	// 验证实体拥有必要的组件
	pos, hasPos := ecs.GetComponent[*components.PositionComponent](s.entityManager, entityID)
	_, hasReanim := ecs.GetComponent[*components.ReanimComponent](s.entityManager, entityID)

	if !hasPos || !hasReanim {
		return errMissingComponentsForRendering(entityID)
	}

	// 获取渲染数据（自动更新缓存）
	renderData := s.GetRenderData(entityID)
	if len(renderData) == 0 {
		return errNoRenderData(entityID)
	}

	// Step 1: 计算所有可见部件的 bounding box（用于居中）
	// 替代旧的 CenterOffset 机制
	minX, maxX := 9999.0, -9999.0
	minY, maxY := 9999.0, -9999.0
	hasVisibleParts := false

	for _, partData := range renderData {
		if partData.Img == nil {
			continue
		}

		frame := partData.Frame
		if frame.FrameNum != nil && *frame.FrameNum == -1 {
			continue
		}

		// 计算部件位置
		partX := getFloat(frame.X) + partData.OffsetX
		partY := getFloat(frame.Y) + partData.OffsetY

		// 获取图片尺寸
		bounds := partData.Img.Bounds()
		w := float64(bounds.Dx())
		h := float64(bounds.Dy())

		// 考虑缩放
		scaleX := getFloat(frame.ScaleX)
		scaleY := getFloat(frame.ScaleY)
		if scaleX == 0 {
			scaleX = 1.0
		}
		if scaleY == 0 {
			scaleY = 1.0
		}

		// 计算部件的 bounding box（考虑图片尺寸）
		partMinX := partX
		partMaxX := partX + w*scaleX
		partMinY := partY
		partMaxY := partY + h*scaleY

		if partMinX < minX {
			minX = partMinX
		}
		if partMaxX > maxX {
			maxX = partMaxX
		}
		if partMinY < minY {
			minY = partMinY
		}
		if partMaxY > maxY {
			maxY = partMaxY
		}

		hasVisibleParts = true
	}

	// Step 2: 计算居中偏移
	// 目标：将 bounding box 的中心对齐到实体的 Position
	centerOffsetX := 0.0
	centerOffsetY := 0.0
	if hasVisibleParts {
		boundingWidth := maxX - minX
		boundingHeight := maxY - minY
		centerOffsetX = -(minX + boundingWidth/2)
		centerOffsetY = -(minY + boundingHeight/2)
	}

	// Step 3: 渲染所有部件（应用居中偏移）
	for _, partData := range renderData {
		if partData.Img == nil {
			continue
		}

		frame := partData.Frame

		// 跳过隐藏帧（FrameNum == -1）
		if frame.FrameNum != nil && *frame.FrameNum == -1 {
			continue
		}

		// 计算部件位置（相对于实体原点）
		partX := getFloat(frame.X) + partData.OffsetX
		partY := getFloat(frame.Y) + partData.OffsetY

		// 应用变换
		opts := &ebiten.DrawImageOptions{}

		// 1. 缩放（先应用缩放，再应用旋转和平移）
		scaleX := getFloat(frame.ScaleX)
		scaleY := getFloat(frame.ScaleY)
		if scaleX == 0 {
			scaleX = 1.0
		}
		if scaleY == 0 {
			scaleY = 1.0
		}
		opts.GeoM.Scale(scaleX, scaleY)

		// 2. 旋转（如果需要）
		// 注意：Reanim 使用弧度制
		// 这里暂不处理旋转，因为大部分植物图标不需要

		// 3. 平移到最终位置（应用居中偏移）
		// 使用 Position 作为基准点（离屏渲染，不减去摄像机偏移）
		finalX := pos.X + partX + centerOffsetX
		finalY := pos.Y + partY + centerOffsetY
		opts.GeoM.Translate(finalX, finalY)

		// 绘制部件
		target.DrawImage(partData.Img, opts)
	}

	return nil
}

// renderOverlayAnimation 渲染叠加动画（如旗帜僵尸的旗杆）
// 叠加动画会在主动画轨道之上渲染，用于实现复合角色效果
//
// 旗帜僵尸实现说明：
// - Zombie_FlagPole.reanim 包含旗杆和旗帜的动画
// - 通过 OverlayBindTrack 绑定到 Zombie_flaghand 轨道
// - 叠加动画应用 Zombie_flaghand 轨道的增量变换（相对于初始帧的变化）
// - 旋转以手部位置为中心，使旗杆顶端的位移大于底端
func (s *ReanimSystem) renderOverlayAnimation(comp *components.ReanimComponent) {
	// 检查是否有叠加动画
	if comp.OverlayReanimXML == nil {
		return
	}

	// 检查绑定轨道是否被隐藏（如旗帜僵尸死亡时旗帜掉落，隐藏 Zombie_flaghand）
	// 如果绑定轨道被隐藏，则不渲染叠加动画
	if comp.OverlayBindTrack != "" && comp.HiddenTracks != nil {
		if comp.HiddenTracks[comp.OverlayBindTrack] {
			return
		}
	}

	// 初始化叠加动画的合并轨道（如果还没有）
	if comp.OverlayMergedTracks == nil {
		comp.OverlayMergedTracks = reanim.BuildMergedTracks(comp.OverlayReanimXML)
	}

	// 获取父轨道的增量变换数据（如果设置了绑定轨道）
	var deltaX, deltaY, deltaKx, deltaKy, pivotX, pivotY float64
	var hasParentTransform bool
	if comp.OverlayBindTrack != "" {
		deltaX, deltaY, deltaKx, deltaKy, pivotX, pivotY, hasParentTransform = s.getBindTrackDeltaTransform(comp)
	}

	// 计算叠加动画的当前帧
	overlayFrame := comp.OverlayCurrentFrame

	// 渲染叠加动画的所有轨道
	for _, track := range comp.OverlayReanimXML.Tracks {
		trackName := track.Name
		mergedFrames, ok := comp.OverlayMergedTracks[trackName]
		if !ok || len(mergedFrames) == 0 {
			continue
		}

		// 获取当前帧数据
		frameIdx := overlayFrame
		if frameIdx < 0 {
			frameIdx = 0
		}
		if frameIdx >= len(mergedFrames) {
			frameIdx = len(mergedFrames) - 1
		}
		frame := mergedFrames[frameIdx]

		// 获取图片（优先使用 ImageOverrides，用于旗帜损坏等效果）
		imgName := frame.ImagePath
		if imgName == "" {
			continue
		}
		var img *ebiten.Image
		var imgOk bool
		if comp.ImageOverrides != nil {
			if overrideImg, hasOverride := comp.ImageOverrides[imgName]; hasOverride && overrideImg != nil {
				img = overrideImg
				imgOk = true
			}
		}
		if !imgOk {
			img, imgOk = comp.PartImages[imgName]
		}
		if !imgOk || img == nil {
			continue
		}

		// 复制 frame，避免修改原始数据
		renderFrame := frame

		// 如果有父轨道变换，应用增量变换到叠加动画
		if hasParentTransform && (deltaX != 0 || deltaY != 0 || deltaKx != 0 || deltaKy != 0) {
			// 获取叠加动画部件的原始坐标
			childX := getFloat(renderFrame.X)
			childY := getFloat(renderFrame.Y)
			childKx := getFloat(renderFrame.SkewX)
			childKy := getFloat(renderFrame.SkewY)

			// 应用增量旋转（绕手部位置 pivotX, pivotY 旋转）
			if deltaKx != 0 {
				deltaRotRad := deltaKx * math.Pi / 180.0
				cosD := math.Cos(deltaRotRad)
				sinD := math.Sin(deltaRotRad)

				// 将位置相对于旋转中心（手部位置）
				relX := childX - pivotX
				relY := childY - pivotY

				// 绕旋转中心旋转
				rotX := relX*cosD - relY*sinD
				rotY := relX*sinD + relY*cosD

				// 转换回绝对坐标
				childX = rotX + pivotX
				childY = rotY + pivotY
			}

			// 应用增量位置偏移
			finalX := childX + deltaX
			finalY := childY + deltaY

			// 应用增量旋转角度
			finalKx := childKx + deltaKx
			finalKy := childKy + deltaKy

			// 更新 frame 的变换数据
			renderFrame.X = &finalX
			renderFrame.Y = &finalY
			renderFrame.SkewX = &finalKx
			renderFrame.SkewY = &finalKy
		}

		comp.CachedRenderData = append(comp.CachedRenderData, components.RenderPartData{
			Img:     img,
			Frame:   renderFrame,
			OffsetX: 0,
			OffsetY: 0,
		})
	}
}

// getBindTrackDeltaTransform 获取绑定轨道相对于初始帧的增量变换
// 返回：deltaX, deltaY, deltaKx, deltaKy, pivotX, pivotY, 是否成功
// pivotX, pivotY 是旋转中心点（手部初始位置）
func (s *ReanimSystem) getBindTrackDeltaTransform(comp *components.ReanimComponent) (float64, float64, float64, float64, float64, float64, bool) {
	bindTrack := comp.OverlayBindTrack
	if bindTrack == "" {
		return 0, 0, 0, 0, 0, 0, false
	}

	// 获取绑定轨道的帧数据
	bindFrames, ok := comp.MergedTracks[bindTrack]
	if !ok || len(bindFrames) == 0 {
		return 0, 0, 0, 0, 0, 0, false
	}

	// 获取当前动画名称和逻辑帧
	var animName string
	var logicalFrame float64
	if len(comp.CurrentAnimations) > 0 {
		animName = comp.CurrentAnimations[0]
		if comp.AnimationFrameIndices != nil {
			if frame, exists := comp.AnimationFrameIndices[animName]; exists {
				logicalFrame = frame
			} else {
				logicalFrame = float64(comp.CurrentFrame)
			}
		} else {
			logicalFrame = float64(comp.CurrentFrame)
		}
	} else {
		logicalFrame = float64(comp.CurrentFrame)
	}

	// 使用当前动画的可见性数组来映射帧（而不是绑定轨道的）
	// 这确保了绑定轨道的帧与主动画同步
	animVisibles, ok := comp.AnimVisiblesMap[animName]
	if !ok || len(animVisibles) == 0 {
		// 后备：直接使用物理帧
		frameIdx := int(logicalFrame)
		if frameIdx < 0 {
			frameIdx = 0
		}
		if frameIdx >= len(bindFrames) {
			frameIdx = len(bindFrames) - 1
		}

		// 获取初始帧和当前帧
		initFrame := bindFrames[0]
		currFrame := bindFrames[frameIdx]

		initX := getFloat(initFrame.X)
		initY := getFloat(initFrame.Y)

		return getFloat(currFrame.X) - initX,
			getFloat(currFrame.Y) - initY,
			getFloat(currFrame.SkewX) - getFloat(initFrame.SkewX),
			getFloat(currFrame.SkewY) - getFloat(initFrame.SkewY),
			initX, initY,
			true
	}

	// 获取初始帧（逻辑帧 0 对应的物理帧）
	initPhysicalFrame := MapLogicalToPhysical(0, animVisibles)
	if initPhysicalFrame < 0 || initPhysicalFrame >= len(bindFrames) {
		initPhysicalFrame = 0
	}
	initFrame := bindFrames[initPhysicalFrame]

	// 使用插值获取更平滑的当前帧数据
	currFrame := s.getInterpolatedFrame(bindTrack, logicalFrame, animVisibles, bindFrames)

	// 计算增量和旋转中心
	initX := getFloat(initFrame.X)
	initY := getFloat(initFrame.Y)
	initKx := getFloat(initFrame.SkewX)
	initKy := getFloat(initFrame.SkewY)

	currX := getFloat(currFrame.X)
	currY := getFloat(currFrame.Y)
	currKx := getFloat(currFrame.SkewX)
	currKy := getFloat(currFrame.SkewY)

	return currX - initX, currY - initY, currKx - initKx, currKy - initKy, initX, initY, true
}

// ==================================================================
// 辅助方法 (Helper Methods)
// ==================================================================

// rebuildAnimationData 重建动画数据（AnimVisiblesMap）
// 基于 AnimationCell.rebuildAnimationData()
func (s *ReanimSystem) rebuildAnimationData(comp *components.ReanimComponent) {
	if comp.ReanimName == "simple_pea" {
		log.Printf("[ReanimSystem] 🔍 rebuildAnimationData 被调用: ReanimName=%s, CurrentAnimations=%v, VisualTracks=%v",
			comp.ReanimName, comp.CurrentAnimations, comp.VisualTracks)
	}

	comp.AnimVisiblesMap = make(map[string][]int)

	// 1. 为当前播放的动画创建可见性数组
	for _, animName := range comp.CurrentAnimations {
		animVisibles := buildVisiblesArray(comp.ReanimXML, comp.MergedTracks, animName)
		comp.AnimVisiblesMap[animName] = animVisibles

		if comp.ReanimName == "simple_pea" {
			log.Printf("[ReanimSystem] 🔍 buildVisiblesArray(%s) = %v (len=%d)", animName, animVisibles, len(animVisibles))
		}
	}

	// 为 ParentTracks 中的父轨道创建可见性数组
	// 父轨道不在 CurrentAnimations 中，但计算父偏移时需要它们的可见性数组
	if comp.ParentTracks != nil {
		for _, parentTrackName := range comp.ParentTracks {
			// 如果该父轨道还没有可见性数组，创建一个
			if _, exists := comp.AnimVisiblesMap[parentTrackName]; !exists {
				animVisibles := buildVisiblesArray(comp.ReanimXML, comp.MergedTracks, parentTrackName)
				comp.AnimVisiblesMap[parentTrackName] = animVisibles
			}
		}
	}
}

// getInterpolatedFrame 获取插值后的帧数据
// 参数：
//   - animName: 动画名称
//   - logicalFrame: 浮点逻辑帧索引（如 2.7 表示第 2 帧和第 3 帧之间，插值因子 0.7）
//   - animVisibles: 动画可见性数组
//   - mergedFrames: 轨道的累加帧数组
//
// 返回：插值后的帧数据
func (s *ReanimSystem) getInterpolatedFrame(
	animName string,
	logicalFrame float64,
	animVisibles []int,
	mergedFrames []reanim.Frame,
) reanim.Frame {
	// 1. 获取整数部分和小数部分
	frame1Index := int(logicalFrame)         // 当前帧索引
	frame2Index := frame1Index + 1           // 下一帧索引
	t := logicalFrame - float64(frame1Index) // 插值因子 (0.0 - 1.0)

	// 2. 映射逻辑帧到物理帧
	physicalFrame1 := MapLogicalToPhysical(frame1Index, animVisibles)
	physicalFrame2 := MapLogicalToPhysical(frame2Index, animVisibles)

	// 3. 边界检查
	if physicalFrame1 < 0 || physicalFrame1 >= len(mergedFrames) {
		return reanim.Frame{} // 返回空帧
	}
	if physicalFrame2 < 0 || physicalFrame2 >= len(mergedFrames) {
		// 如果下一帧越界，直接返回当前帧（不插值）
		return mergedFrames[physicalFrame1]
	}

	// 4. 获取两个帧
	f1 := mergedFrames[physicalFrame1]
	f2 := mergedFrames[physicalFrame2]

	// 5. 线性插值
	result := reanim.Frame{
		ImagePath: f1.ImagePath, // 图片引用不插值，使用第一帧的
	}

	// 插值位置 (X, Y)
	if f1.X != nil && f2.X != nil {
		interpolatedX := *f1.X + (*f2.X-*f1.X)*t
		result.X = &interpolatedX
	} else if f1.X != nil {
		result.X = f1.X
	}

	if f1.Y != nil && f2.Y != nil {
		interpolatedY := *f1.Y + (*f2.Y-*f1.Y)*t
		result.Y = &interpolatedY
	} else if f1.Y != nil {
		result.Y = f1.Y
	}

	// 插值缩放 (ScaleX, ScaleY)
	if f1.ScaleX != nil && f2.ScaleX != nil {
		interpolatedScaleX := *f1.ScaleX + (*f2.ScaleX-*f1.ScaleX)*t
		result.ScaleX = &interpolatedScaleX
	} else if f1.ScaleX != nil {
		result.ScaleX = f1.ScaleX
	}

	if f1.ScaleY != nil && f2.ScaleY != nil {
		interpolatedScaleY := *f1.ScaleY + (*f2.ScaleY-*f1.ScaleY)*t
		result.ScaleY = &interpolatedScaleY
	} else if f1.ScaleY != nil {
		result.ScaleY = f1.ScaleY
	}

	// 插值倾斜角度 (SkewX, SkewY)
	if f1.SkewX != nil && f2.SkewX != nil {
		interpolatedSkewX := *f1.SkewX + (*f2.SkewX-*f1.SkewX)*t
		result.SkewX = &interpolatedSkewX
	} else if f1.SkewX != nil {
		result.SkewX = f1.SkewX
	}

	if f1.SkewY != nil && f2.SkewY != nil {
		interpolatedSkewY := *f1.SkewY + (*f2.SkewY-*f1.SkewY)*t
		result.SkewY = &interpolatedSkewY
	} else if f1.SkewY != nil {
		result.SkewY = f1.SkewY
	}

	// 插值透明度 (Alpha)
	if f1.Alpha != nil && f2.Alpha != nil {
		interpolatedAlpha := *f1.Alpha + (*f2.Alpha-*f1.Alpha)*t
		result.Alpha = &interpolatedAlpha
	} else if f1.Alpha != nil {
		result.Alpha = f1.Alpha
	}

	// FrameNum 不插值（可见性标志），使用第一帧的
	result.FrameNum = f1.FrameNum

	return result
}
