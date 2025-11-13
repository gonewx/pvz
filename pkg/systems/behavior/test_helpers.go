package behavior

import (
	"sync"

	"github.com/hajimehoshi/ebiten/v2/audio"
)

// testAudioContext 是测试用的共享音频上下文
var (
	testAudioContext     *audio.Context
	testAudioContextOnce sync.Once
)

// getTestAudioContext 获取测试音频上下文（延迟创建）
func getTestAudioContext() *audio.Context {
	testAudioContextOnce.Do(func() {
		testAudioContext = audio.NewContext(48000)
	})
	return testAudioContext
}
