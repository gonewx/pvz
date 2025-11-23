package systems

import (
	"fmt"
	"log"

	"github.com/decker502/pvz/internal/reanim"
	"github.com/decker502/pvz/pkg/components"
)

// ==================================================================
// ReanimSystem 辅助方法 (Helper Methods)
// ==================================================================

// getParentOffsetForAnimation 获取指定动画中父轨道的偏移量（用于父子关系计算）
func (s *ReanimSystem) getParentOffsetForAnimation(comp *components.ReanimComponent, parentTrackName string, animName string) (float64, float64) {
	parentFrames, ok := comp.MergedTracks[parentTrackName]
	if !ok || len(parentFrames) == 0 {
		// Debug: 父轨道不存在
		if comp.ReanimName == "peashooter" && comp.CurrentFrame < 3 {
			log.Printf("[ReanimSystem] 父轨道不存在: parent=%s", parentTrackName)
		}
		return 0, 0
	}

	// 获取动画的逻辑帧索引
	var logicalFrame float64
	if comp.AnimationFrameIndices != nil {
		if frame, exists := comp.AnimationFrameIndices[animName]; exists {
			logicalFrame = frame
		} else {
			logicalFrame = float64(comp.CurrentFrame) // 后备：使用共享帧
		}
	} else {
		logicalFrame = float64(comp.CurrentFrame) // 后备：使用共享帧
	}

	// 获取父轨道的可见性数组
	parentAnimVisibles, ok := comp.AnimVisiblesMap[parentTrackName]
	if !ok || len(parentAnimVisibles) == 0 {
		// Debug: 父轨道的可见性数组不存在
		if comp.ReanimName == "peashooter" && comp.CurrentFrame < 3 {
			log.Printf("[ReanimSystem] 父轨道可见性数组不存在: parent=%s, AnimVisiblesMap keys=%v",
				parentTrackName, getMapKeys(comp.AnimVisiblesMap))
		}
		return 0, 0
	}

	// 获取第一个可见帧的物理索引
	// 不需要遍历查找，直接使用逻辑帧号 0 映射到物理帧
	firstPhysicalFrame := MapLogicalToPhysical(0, parentAnimVisibles)
	if firstPhysicalFrame < 0 || firstPhysicalFrame >= len(parentFrames) {
		return 0, 0
	}

	// 先初始化为 0，然后逐步设置有效值
	initX, initY := 0.0, 0.0
	if parentFrames[firstPhysicalFrame].X != nil {
		initX = *parentFrames[firstPhysicalFrame].X
	}
	if parentFrames[firstPhysicalFrame].Y != nil {
		initY = *parentFrames[firstPhysicalFrame].Y
	}

	// 使用父轨道自己的可见性数组
	currentFrame := s.getInterpolatedFrame(parentTrackName, logicalFrame, parentAnimVisibles, parentFrames)

	currentX, currentY := initX, initY
	if currentFrame.X != nil {
		currentX = *currentFrame.X
	}
	if currentFrame.Y != nil {
		currentY = *currentFrame.Y
	}

	// Debug: 父偏移计算详情（前3帧）
	if comp.ReanimName == "peashooter" && comp.CurrentFrame < 3 {
		log.Printf("[ReanimSystem] GetParentOffset[%s]: parent=%s, anim=%s, logicalFrame=%.2f, firstPhysical=%d",
			comp.ReanimName, parentTrackName, animName, logicalFrame, firstPhysicalFrame)
		log.Printf("[ReanimSystem]   init=(%.2f, %.2f), current=(%.2f, %.2f), offset=(%.2f, %.2f)",
			initX, initY, currentX, currentY, currentX-initX, currentY-initY)
	}

	return currentX - initX, currentY - initY
}

// analyzeTrackTypes 分析 Reanim 文件中的轨道类型，区分视觉轨道和逻辑轨道
//
// 同名轨道处理：当多个轨道具有相同名称时，会自动为它们添加唯一后缀
// （如 "rock", "rock#1", "rock#2"），以匹配 BuildMergedTracks 的行为
func (s *ReanimSystem) analyzeTrackTypes(reanimXML *reanim.ReanimXML) (visualTracks []string, logicalTracks []string) {
	// 原因：向日葵的 anim_idle 轨道包含头部图像，不应该被跳过
	// animation_showcase 的逻辑可能不适用于所有植物
	animationDefinitionTracks := map[string]bool{
		"anim_idle":      true,
		"anim_shooting":  true,
		"anim_head_idle": true,
		"anim_full_idle": true,
	}

	// 用于追踪同名轨道的出现次数（与 BuildMergedTracks 保持一致）
	trackNameCount := make(map[string]int)

	for _, track := range reanimXML.Tracks {
		// 生成唯一的轨道键名（与 BuildMergedTracks 保持一致）
		trackKey := track.Name
		if count, exists := trackNameCount[track.Name]; exists {
			// 同名轨道，添加序号后缀
			trackKey = fmt.Sprintf("%s#%d", track.Name, count)
		}
		trackNameCount[track.Name]++

		// 先检查轨道是否包含图片
		hasImage := false
		for _, frame := range track.Frames {
			if frame.ImagePath != "" {
				hasImage = true
				break
			}
		}

		// 也应该作为视觉轨道处理（例如向日葵的 anim_idle 轨道）
		if hasImage {
			visualTracks = append(visualTracks, trackKey)
		} else if animationDefinitionTracks[track.Name] {
			// 只有在没有图片的情况下，才跳过动画定义轨道
			logicalTracks = append(logicalTracks, trackKey)
		} else {
			// 其他无图片轨道也作为逻辑轨道
			logicalTracks = append(logicalTracks, trackKey)
		}
	}

	return visualTracks, logicalTracks
}

// calculateCenterOffset 计算并缓存 CenterOffset
// 在第一帧计算所有可见部件的 bounding box 中心,避免每帧重新计算导致位置抖动
func (s *ReanimSystem) calculateCenterOffset(comp *components.ReanimComponent) {
	// 确保已初始化
	if comp.MergedTracks == nil || len(comp.VisualTracks) == 0 {
		log.Printf("[ReanimSystem] calculateCenterOffset: %s → 提前返回（MergedTracks=%v, VisualTracks=%d）",
			comp.ReanimName, comp.MergedTracks != nil, len(comp.VisualTracks))
		comp.CenterOffsetX = 0
		comp.CenterOffsetY = 0
		return
	}

	// 强制帧索引为 0,计算第一帧的 bounding box
	comp.CurrentFrame = 0

	// 准备第一帧的渲染数据
	s.prepareRenderCache(comp)

	if len(comp.CachedRenderData) == 0 {
		log.Printf("[ReanimSystem] calculateCenterOffset: %s → 提前返回（CachedRenderData为空）", comp.ReanimName)
		comp.CenterOffsetX = 0
		comp.CenterOffsetY = 0
		return
	}

	// 计算 bounding box
	minX, maxX := 9999.0, -9999.0
	minY, maxY := 9999.0, -9999.0

	for _, partData := range comp.CachedRenderData {
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
	}

	// 计算中心点坐标
	comp.CenterOffsetX = (minX + maxX) / 2
	comp.CenterOffsetY = (minY + maxY) / 2

	// DEBUG: 输出 CenterOffset 计算结果
	log.Printf("[ReanimSystem] calculateCenterOffset: %s → CenterOffset=(%.1f, %.1f), BBox=(%.1f,%.1f)-(%.1f,%.1f)",
		comp.ReanimName, comp.CenterOffsetX, comp.CenterOffsetY, minX, minY, maxX, maxY)
}

// ==================================================================
// 全局辅助函数 (Global Helper Functions)
// ==================================================================

// getMapKeys 获取 map 的所有 key（辅助函数，用于调试）
func getMapKeys(m map[string][]int) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

// buildVisiblesArray 构建动画的可见性数组
func buildVisiblesArray(reanimXML *reanim.ReanimXML, mergedTracks map[string][]reanim.Frame, animName string) []int {
	var animTrack *reanim.Track
	for i := range reanimXML.Tracks {
		if reanimXML.Tracks[i].Name == animName {
			animTrack = &reanimXML.Tracks[i]
			break
		}
	}

	if animTrack == nil {
		return []int{}
	}

	standardFrameCount := 0
	for _, track := range reanimXML.Tracks {
		if len(track.Frames) > standardFrameCount {
			standardFrameCount = len(track.Frames)
		}
	}

	if standardFrameCount == 0 {
		return []int{}
	}

	visibles := make([]int, standardFrameCount)
	currentValue := 0

	for i := 0; i < standardFrameCount; i++ {
		if i < len(animTrack.Frames) {
			frame := animTrack.Frames[i]
			if frame.FrameNum != nil {
				currentValue = *frame.FrameNum
			}
		}
		visibles[i] = currentValue
	}

	return visibles
}

// countVisibleFrames 计算可见帧数（非隐藏帧的数量）
// animVisibles 中：-1 表示隐藏，>= 0 表示可见
func countVisibleFrames(animVisibles []int) int {
	count := 0
	for _, visible := range animVisibles {
		if visible >= 0 {
			count++
		}
	}
	return count
}

// MapLogicalToPhysical 将逻辑帧号映射到物理帧号
// 公共 API,供需要手动处理帧映射的业务代码使用
func MapLogicalToPhysical(logicalFrameNum int, animVisibles []int) int {
	if len(animVisibles) == 0 {
		return logicalFrameNum
	}

	logicalIndex := 0
	lastVisiblePhysicalFrame := -1
	for i := 0; i < len(animVisibles); i++ {
		if animVisibles[i] == 0 {
			lastVisiblePhysicalFrame = i // 记录最后一个可见帧的物理索引
			if logicalIndex == logicalFrameNum {
				return i
			}
			logicalIndex++
		}
	}

	// 这样 anim_open 完成后会保持显示，不会消失
	if lastVisiblePhysicalFrame >= 0 {
		return lastVisiblePhysicalFrame
	}

	return -1
}

// findVisibleWindow 查找动画的可见时间窗口
func findVisibleWindow(animVisibles []int) (int, int) {
	firstVisible, lastVisible := -1, -1
	for i, v := range animVisibles {
		if v == 0 {
			if firstVisible == -1 {
				firstVisible = i
			}
			lastVisible = i
		}
	}
	return firstVisible, lastVisible
}

// calculatePositionVariance 计算位置方差
func calculatePositionVariance(frames []reanim.Frame, startIdx, endIdx int) float64 {
	if startIdx < 0 || endIdx >= len(frames) || startIdx > endIdx {
		return 0.0
	}

	sumX, sumY := 0.0, 0.0
	count := 0
	for i := startIdx; i <= endIdx; i++ {
		if frames[i].X != nil && frames[i].Y != nil {
			sumX += *frames[i].X
			sumY += *frames[i].Y
			count++
		}
	}

	if count == 0 {
		return 0.0
	}

	meanX := sumX / float64(count)
	meanY := sumY / float64(count)

	variance := 0.0
	for i := startIdx; i <= endIdx; i++ {
		if frames[i].X != nil && frames[i].Y != nil {
			dx := *frames[i].X - meanX
			dy := *frames[i].Y - meanY
			variance += dx*dx + dy*dy
		}
	}

	return variance / float64(count)
}
