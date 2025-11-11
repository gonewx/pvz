package reanim

import (
	"encoding/xml"
	"fmt"
	"os"
	"strings"
)

// ParseReanimFile parses a Reanim XML file and returns the animation data.
// Reanim files are XML files without a root element, so this function wraps
// the content with a <reanim> root element before parsing.
//
// Parameters:
//   - path: Path to the Reanim file, e.g., "assets/effect/reanim/PeaShooter.reanim"
//
// Returns:
//   - *ReanimXML: The parsed animation data
//   - error: Parsing error, or nil if successful
//
// Example:
//
//	reanim, err := ParseReanimFile("assets/effect/reanim/PeaShooter.reanim")
//	if err != nil {
//	    log.Fatalf("Failed to parse reanim: %v", err)
//	}
//	fmt.Printf("Animation FPS: %d\n", reanim.FPS)
func ParseReanimFile(path string) (*ReanimXML, error) {
	// Read the file
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read reanim file '%s': %w", path, err)
	}

	// Wrap the XML content with a root element
	// Original PVZ reanim files don't have a root element, so we add one
	wrappedXML := "<reanim>" + string(data) + "</reanim>"

	// Parse the XML
	var reanim ReanimXML
	if err := xml.Unmarshal([]byte(wrappedXML), &reanim); err != nil {
		return nil, fmt.Errorf("failed to parse XML from '%s': %w", path, err)
	}

	return &reanim, nil
}

// BuildAnimVisiblesMap 构建所有动画定义轨道的时间窗口映射
//
// 此函数用于 ComplexScene 模式的独立动画系统，在实体初始化时调用。
//
// 参数：
//   - reanimXML: Reanim 动画数据
//
// 返回：
//   - map[string][]int: 动画时间窗口映射
//   - Key: 动画名称（如 "anim_cloud1"）
//   - Value: 时间窗口数组，0 表示可见，-1 表示隐藏
func BuildAnimVisiblesMap(reanimXML *ReanimXML) map[string][]int {
	if reanimXML == nil {
		return nil
	}

	animVisiblesMap := make(map[string][]int)

	// 确定标准帧数（所有轨道中的最大帧数）
	standardFrameCount := 0
	for _, track := range reanimXML.Tracks {
		if len(track.Frames) > standardFrameCount {
			standardFrameCount = len(track.Frames)
		}
	}

	if standardFrameCount == 0 {
		return animVisiblesMap
	}

	// 遍历所有轨道，查找动画定义轨道
	for i := range reanimXML.Tracks {
		track := &reanimXML.Tracks[i]

		// 检查是否是动画定义轨道（以 "anim_" 开头）
		if !strings.HasPrefix(track.Name, "anim_") {
			continue
		}

		// 构建时间窗口数组
		visibles := make([]int, standardFrameCount)
		currentValue := 0 // 默认第一帧可见

		for frameIdx := 0; frameIdx < standardFrameCount; frameIdx++ {
			if frameIdx < len(track.Frames) {
				frame := track.Frames[frameIdx]
				// 如果指定了 FrameNum，使用它；否则继承上一帧的值
				if frame.FrameNum != nil {
					currentValue = *frame.FrameNum
				}
			}
			// 分配当前值（显式设置或继承）
			visibles[frameIdx] = currentValue
		}

		animVisiblesMap[track.Name] = visibles
	}

	return animVisiblesMap
}

// BuildMergedTracks 构建带帧继承的轨道数据
//
// 帧继承机制：Reanim 文件使用稀疏关键帧来节省空间。
// 许多帧是空的（nil 值），依赖前一帧的值。
// 此函数累积所有物理帧的帧值，确保每帧都有完整数据。
//
// 参数：
//   - reanimXML: Reanim 动画数据
//
// 返回：
//   - map[string][]Frame: 轨道名称 -> 合并后的帧数组
func BuildMergedTracks(reanimXML *ReanimXML) map[string][]Frame {
	if reanimXML == nil {
		return map[string][]Frame{}
	}

	// 确定标准帧数（所有轨道中的最大帧数）
	standardFrameCount := 0
	for _, track := range reanimXML.Tracks {
		if len(track.Frames) > standardFrameCount {
			standardFrameCount = len(track.Frames)
		}
	}

	if standardFrameCount == 0 {
		return map[string][]Frame{}
	}

	mergedTracks := make(map[string][]Frame)

	// 处理所有轨道（包括动画定义轨道）
	for _, track := range reanimXML.Tracks {
		// 初始化累积状态
		accX := 0.0
		accY := 0.0
		accSX := 1.0
		accSY := 1.0
		accKX := 0.0
		accKY := 0.0
		accF := 0 // 默认第一帧 f=0（与参考代码一致）
		accImg := ""

		// 为此轨道构建合并帧数组
		mergedFrames := make([]Frame, standardFrameCount)

		for i := 0; i < standardFrameCount; i++ {
			// 如果原始轨道在此索引有帧，更新累积状态
			if i < len(track.Frames) {
				frame := track.Frames[i]

				// 仅当字段非 nil 时更新累积值
				if frame.X != nil {
					accX = *frame.X
				}
				if frame.Y != nil {
					accY = *frame.Y
				}
				if frame.ScaleX != nil {
					accSX = *frame.ScaleX
				}
				if frame.ScaleY != nil {
					accSY = *frame.ScaleY
				}
				if frame.SkewX != nil {
					accKX = *frame.SkewX
				}
				if frame.SkewY != nil {
					accKY = *frame.SkewY
				}
				if frame.FrameNum != nil {
					accF = *frame.FrameNum
				}
				if frame.ImagePath != "" {
					accImg = frame.ImagePath
				}
			}

			// 创建具有独立指针的新帧（避免指针共享）
			x := accX
			y := accY
			sx := accSX
			sy := accSY
			kx := accKX
			ky := accKY
			f := accF

			mergedFrame := Frame{
				X:         &x,
				Y:         &y,
				ScaleX:    &sx,
				ScaleY:    &sy,
				SkewX:     &kx,
				SkewY:     &ky,
				FrameNum:  &f, // 所有轨道的所有帧都设置 FrameNum
				ImagePath: accImg,
			}

			mergedFrames[i] = mergedFrame
		}

		mergedTracks[track.Name] = mergedFrames
	}

	return mergedTracks
}
