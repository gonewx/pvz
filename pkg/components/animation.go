package components

import "github.com/hajimehoshi/ebiten/v2"

// AnimationComponent 管理基于spritesheet的帧动画
// 它存储了动画的所有帧、播放速度以及当前状态
type AnimationComponent struct {
	Frames       []*ebiten.Image // 动画的所有帧图片
	FrameSpeed   float64         // 每帧之间的延迟时间(秒)
	FrameCounter float64         // 当前帧计时器(秒)
	CurrentFrame int             // 当前显示的帧索引(0-based)
	IsLooping    bool            // 是否循环播放
	IsFinished   bool            // 动画是否已完成(仅对非循环动画有效)
}
