package utils

import (
	"fmt"

	"github.com/decker502/pvz/pkg/config"
	"github.com/decker502/pvz/pkg/game"
	"github.com/hajimehoshi/ebiten/v2"
)

// LoadZombieDeathAnimation 加载僵尸死亡动画帧序列
// 从 assets/images/Zombies/Zombie/ZombieDie_*.png 加载所有死亡动画帧
// 返回:
//   - []*ebiten.Image: 动画帧数组
//   - error: 加载过程中的任何错误
func LoadZombieDeathAnimation(rm *game.ResourceManager) ([]*ebiten.Image, error) {
	frames := make([]*ebiten.Image, config.ZombieDieAnimationFrames)

	for i := 0; i < config.ZombieDieAnimationFrames; i++ {
		// 资源路径：assets/images/Zombies/Zombie/ZombieDie_1.png, ZombieDie_2.png, ..., ZombieDie_10.png
		framePath := fmt.Sprintf("assets/images/Zombies/Zombie/ZombieDie_%d.png", i+1)
		frameImage, err := rm.LoadImage(framePath)
		if err != nil {
			return nil, fmt.Errorf("加载僵尸死亡动画帧 %d 失败: %w", i+1, err)
		}
		frames[i] = frameImage
	}

	return frames, nil
}
