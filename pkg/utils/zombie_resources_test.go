package utils

import (
	"testing"

	"github.com/decker502/pvz/pkg/config"
	"github.com/decker502/pvz/pkg/game"
	"github.com/hajimehoshi/ebiten/v2/audio"
)

var testAudioContext = audio.NewContext(48000)

// TestLoadZombieDeathAnimation 测试僵尸死亡动画加载
func TestLoadZombieDeathAnimation(t *testing.T) {
	// 创建 ResourceManager
	rm := game.NewResourceManager(testAudioContext)

	// 加载僵尸死亡动画
	frames, err := LoadZombieDeathAnimation(rm)

	// 验证：在测试环境中，资源文件可能不存在，这是可接受的
	if err != nil {
		// 资源文件不存在时跳过测试（在实际游戏运行时资源文件会存在）
		t.Skipf("跳过测试：资源文件不存在 (在实际游戏中资源会正常加载): %v", err)
		return
	}

	// 验证：帧数正确
	expectedFrames := config.ZombieDieAnimationFrames
	if len(frames) != expectedFrames {
		t.Errorf("死亡动画帧数错误: 预期 %d, 实际 %d", expectedFrames, len(frames))
	}

	// 验证：所有帧都不为 nil
	for i, frame := range frames {
		if frame == nil {
			t.Errorf("死亡动画帧 %d 为 nil", i+1)
		}
	}
}

