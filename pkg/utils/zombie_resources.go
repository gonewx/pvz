package utils

import (
	"fmt"

	"github.com/decker502/pvz/pkg/config"
	"github.com/decker502/pvz/pkg/game"
	"github.com/hajimehoshi/ebiten/v2"
)

// LoadZombieWalkAnimation 加载僵尸走路动画帧序列
// 从 assets/images/Zombies/Zombie/Zombie_*.png 加载所有走路动画帧
// 返回:
//   - []*ebiten.Image: 动画帧数组
func LoadZombieWalkAnimation(rm *game.ResourceManager) []*ebiten.Image {
	frames := make([]*ebiten.Image, config.ZombieWalkAnimationFrames)

	for i := 0; i < config.ZombieWalkAnimationFrames; i++ {
		// 资源路径：assets/images/Zombies/Zombie/Zombie_1.png, Zombie_2.png, ..., Zombie_22.png
		framePath := fmt.Sprintf("assets/images/Zombies/Zombie/Zombie_%d.png", i+1)
		frameImage, err := rm.LoadImage(framePath)
		if err != nil {
			// 如果加载失败，使用第一帧作为占位
			// 这确保动画系统不会崩溃
			if i > 0 && frames[0] != nil {
				frames[i] = frames[0]
			}
			continue
		}
		frames[i] = frameImage
	}

	return frames
}

// LoadZombieEatAnimation 加载僵尸啃食动画帧序列
// 从 assets/images/Zombies/Zombie/ZombieAttack_*.png 加载所有啃食动画帧
// 返回:
//   - []*ebiten.Image: 动画帧数组
func LoadZombieEatAnimation(rm *game.ResourceManager) []*ebiten.Image {
	frames := make([]*ebiten.Image, config.ZombieEatAnimationFrames)

	for i := 0; i < config.ZombieEatAnimationFrames; i++ {
		// 资源路径：assets/images/Zombies/Zombie/ZombieAttack_1.png, ..., ZombieAttack_21.png
		framePath := fmt.Sprintf("assets/images/Zombies/Zombie/ZombieAttack_%d.png", i+1)
		frameImage, err := rm.LoadImage(framePath)
		if err != nil {
			// 如果加载失败，使用走路动画作为降级
			// 这确保即使啃食动画资源缺失，游戏也能继续运行
			walkFrames := LoadZombieWalkAnimation(rm)
			if len(walkFrames) > 0 && walkFrames[0] != nil {
				return walkFrames
			}
			continue
		}
		frames[i] = frameImage
	}

	return frames
}

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
