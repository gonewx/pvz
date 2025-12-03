package systems

import (
	"testing"

	"github.com/decker502/pvz/internal/reanim"
)

// TestWallnutAnimFaceMergedTracksRotation 测试 Wallnut anim_face 轨道的旋转数据
// 验证滚动动画的帧 43-55 有正确的 kx/ky 值
func TestWallnutAnimFaceMergedTracksRotation(t *testing.T) {
	// 加载 Wallnut.reanim 文件
	reanimXML, err := reanim.ParseReanimFile("data/reanim/Wallnut.reanim")
	if err != nil {
		t.Fatalf("Failed to load Wallnut.reanim: %v", err)
	}

	// 构建 MergedTracks
	mergedTracks := reanim.BuildMergedTracks(reanimXML)

	// 获取 anim_face 轨道
	animFaceFrames, ok := mergedTracks["anim_face"]
	if !ok {
		t.Fatal("anim_face track not found in MergedTracks")
	}

	t.Logf("anim_face track has %d frames", len(animFaceFrames))

	// 验证帧 43-55（滚动动画）的 kx/ky 值
	expectedRotations := map[int]float64{
		43: 0,      // 继承自帧 16
		44: 27.6,
		45: 57.7,
		46: 90,
		47: 122.3,
		48: 152.4,
		49: 180,
		50: 207.6,
		51: 237.7,
		52: 270,
		53: 302.3,
		54: 332.4,
		55: 360,
	}

	for frameIdx, expectedKx := range expectedRotations {
		if frameIdx >= len(animFaceFrames) {
			t.Errorf("Frame %d out of range (total frames: %d)", frameIdx, len(animFaceFrames))
			continue
		}

		frame := animFaceFrames[frameIdx]

		if frame.SkewX == nil {
			t.Errorf("Frame %d: SkewX is nil", frameIdx)
			continue
		}

		actualKx := *frame.SkewX

		// 允许小误差
		if actualKx < expectedKx-0.1 || actualKx > expectedKx+0.1 {
			t.Errorf("Frame %d: SkewX = %.1f, expected %.1f", frameIdx, actualKx, expectedKx)
		} else {
			t.Logf("Frame %d: SkewX = %.1f (expected %.1f) ✓", frameIdx, actualKx, expectedKx)
		}
	}
}

// TestWallnutAnimVisiblesMapping 测试 AnimVisibles 的逻辑帧到物理帧映射
func TestWallnutAnimVisiblesMapping(t *testing.T) {
	// 加载 Wallnut.reanim 文件
	reanimXML, err := reanim.ParseReanimFile("data/reanim/Wallnut.reanim")
	if err != nil {
		t.Fatalf("Failed to load Wallnut.reanim: %v", err)
	}

	// 构建 MergedTracks
	mergedTracks := reanim.BuildMergedTracks(reanimXML)

	// 构建 AnimVisibles 数组
	animVisibles := buildVisiblesArray(reanimXML, mergedTracks, "anim_face")

	t.Logf("AnimVisibles length: %d", len(animVisibles))

	// 打印所有可见帧
	visibleCount := 0
	for i, v := range animVisibles {
		if v >= 0 {
			visibleCount++
			if visibleCount <= 17 || visibleCount >= 18 {
				t.Logf("  Logical %d → Physical %d (value=%d)", visibleCount-1, i, v)
			}
		}
	}
	t.Logf("Total visible frames: %d", visibleCount)

	// 验证逻辑帧 17 映射到物理帧 43
	logicalFrame := 17
	physicalFrame := MapLogicalToPhysical(logicalFrame, animVisibles)
	if physicalFrame != 43 {
		t.Errorf("Logical frame %d should map to physical frame 43, got %d", logicalFrame, physicalFrame)
	} else {
		t.Logf("Logical frame %d → Physical frame %d ✓", logicalFrame, physicalFrame)
	}

	// 验证逻辑帧 29 映射到物理帧 55
	logicalFrame = 29
	physicalFrame = MapLogicalToPhysical(logicalFrame, animVisibles)
	if physicalFrame != 55 {
		t.Errorf("Logical frame %d should map to physical frame 55, got %d", logicalFrame, physicalFrame)
	} else {
		t.Logf("Logical frame %d → Physical frame %d ✓", logicalFrame, physicalFrame)
	}
}
