package systems

import (
	"reflect"
	"testing"

	"github.com/decker502/pvz/pkg/components"
	"github.com/decker502/pvz/pkg/ecs"
	"github.com/hajimehoshi/ebiten/v2"
)

// TestAnimationFrameAdvance 测试动画帧推进逻辑
func TestAnimationFrameAdvance(t *testing.T) {
	em := ecs.NewEntityManager()
	system := NewAnimationSystem(em)

	// 创建测试实体
	id := em.CreateEntity()

	// 创建模拟的动画帧
	frame1 := ebiten.NewImage(10, 10)
	frame2 := ebiten.NewImage(10, 10)
	frame3 := ebiten.NewImage(10, 10)

	// 添加动画组件 (帧速度0.1秒)
	em.AddComponent(id, &components.AnimationComponent{
		Frames:       []*ebiten.Image{frame1, frame2, frame3},
		FrameSpeed:   0.1,
		FrameCounter: 0,
		CurrentFrame: 0,
		IsLooping:    true,
		IsFinished:   false,
	})

	// 添加精灵组件
	em.AddComponent(id, &components.SpriteComponent{
		Image: frame1,
	})

	// 第一次更新: deltaTime < FrameSpeed, 应该不切换帧
	system.Update(0.05)

	animComp, _ := em.GetComponent(id, reflect.TypeOf(&components.AnimationComponent{}))
	anim := animComp.(*components.AnimationComponent)

	if anim.CurrentFrame != 0 {
		t.Errorf("Expected CurrentFrame=0, got %d", anim.CurrentFrame)
	}
	if anim.FrameCounter != 0.05 {
		t.Errorf("Expected FrameCounter=0.05, got %f", anim.FrameCounter)
	}

	// 第二次更新: 累积时间 >= FrameSpeed, 应该切换到下一帧
	system.Update(0.06)

	if anim.CurrentFrame != 1 {
		t.Errorf("Expected CurrentFrame=1, got %d", anim.CurrentFrame)
	}
	if anim.FrameCounter != 0 { // 计数器重置为0
		t.Errorf("Expected FrameCounter=0, got %f", anim.FrameCounter)
	}

	// 验证SpriteComponent已更新
	spriteComp, _ := em.GetComponent(id, reflect.TypeOf(&components.SpriteComponent{}))
	sprite := spriteComp.(*components.SpriteComponent)

	if sprite.Image != frame2 {
		t.Error("SpriteComponent.Image should be updated to frame2")
	}
}

// TestLoopingAnimation 测试循环动画重置到第0帧
func TestLoopingAnimation(t *testing.T) {
	em := ecs.NewEntityManager()
	system := NewAnimationSystem(em)

	id := em.CreateEntity()

	frame1 := ebiten.NewImage(10, 10)
	frame2 := ebiten.NewImage(10, 10)

	// 添加循环动画组件，当前已在最后一帧
	em.AddComponent(id, &components.AnimationComponent{
		Frames:       []*ebiten.Image{frame1, frame2},
		FrameSpeed:   0.1,
		FrameCounter: 0,
		CurrentFrame: 1, // 最后一帧
		IsLooping:    true,
		IsFinished:   false,
	})

	em.AddComponent(id, &components.SpriteComponent{
		Image: frame2,
	})

	// 更新应该重置到第0帧
	system.Update(0.1)

	animComp, _ := em.GetComponent(id, reflect.TypeOf(&components.AnimationComponent{}))
	anim := animComp.(*components.AnimationComponent)

	if anim.CurrentFrame != 0 {
		t.Errorf("Looping animation should reset to frame 0, got %d", anim.CurrentFrame)
	}

	if anim.IsFinished {
		t.Error("Looping animation should never be marked as finished")
	}

	// 验证SpriteComponent已更新
	spriteComp, _ := em.GetComponent(id, reflect.TypeOf(&components.SpriteComponent{}))
	sprite := spriteComp.(*components.SpriteComponent)

	if sprite.Image != frame1 {
		t.Error("SpriteComponent.Image should be updated to frame1 after loop")
	}
}

// TestNonLoopingAnimationFinish 测试非循环动画完成后标记IsFinished=true
func TestNonLoopingAnimationFinish(t *testing.T) {
	em := ecs.NewEntityManager()
	system := NewAnimationSystem(em)

	id := em.CreateEntity()

	frame1 := ebiten.NewImage(10, 10)
	frame2 := ebiten.NewImage(10, 10)

	// 添加非循环动画组件，当前在最后一帧之前
	em.AddComponent(id, &components.AnimationComponent{
		Frames:       []*ebiten.Image{frame1, frame2},
		FrameSpeed:   0.1,
		FrameCounter: 0,
		CurrentFrame: 1, // 最后一帧
		IsLooping:    false,
		IsFinished:   false,
	})

	em.AddComponent(id, &components.SpriteComponent{
		Image: frame2,
	})

	// 更新应该标记动画完成
	system.Update(0.1)

	animComp, _ := em.GetComponent(id, reflect.TypeOf(&components.AnimationComponent{}))
	anim := animComp.(*components.AnimationComponent)

	if !anim.IsFinished {
		t.Error("Non-looping animation should be marked as finished")
	}

	if anim.CurrentFrame != 1 {
		t.Errorf("Non-looping animation should stay at last frame (1), got %d", anim.CurrentFrame)
	}

	// 验证SpriteComponent停留在最后一帧
	spriteComp, _ := em.GetComponent(id, reflect.TypeOf(&components.SpriteComponent{}))
	sprite := spriteComp.(*components.SpriteComponent)

	if sprite.Image != frame2 {
		t.Error("SpriteComponent.Image should stay at last frame")
	}
}

// TestFinishedAnimationNotUpdated 测试已完成的动画不再更新
func TestFinishedAnimationNotUpdated(t *testing.T) {
	em := ecs.NewEntityManager()
	system := NewAnimationSystem(em)

	id := em.CreateEntity()

	frame1 := ebiten.NewImage(10, 10)
	frame2 := ebiten.NewImage(10, 10)

	// 添加已完成的非循环动画
	em.AddComponent(id, &components.AnimationComponent{
		Frames:       []*ebiten.Image{frame1, frame2},
		FrameSpeed:   0.1,
		FrameCounter: 0,
		CurrentFrame: 1,
		IsLooping:    false,
		IsFinished:   true, // 已完成
	})

	em.AddComponent(id, &components.SpriteComponent{
		Image: frame2,
	})

	// 更新系统
	system.Update(0.2)

	animComp, _ := em.GetComponent(id, reflect.TypeOf(&components.AnimationComponent{}))
	anim := animComp.(*components.AnimationComponent)

	// FrameCounter不应该变化
	if anim.FrameCounter != 0 {
		t.Errorf("Finished animation FrameCounter should not update, got %f", anim.FrameCounter)
	}

	// CurrentFrame不应该变化
	if anim.CurrentFrame != 1 {
		t.Errorf("Finished animation CurrentFrame should not change, got %d", anim.CurrentFrame)
	}
}

// TestEmptyFramesAnimation 测试没有帧的动画不会崩溃
func TestEmptyFramesAnimation(t *testing.T) {
	em := ecs.NewEntityManager()
	system := NewAnimationSystem(em)

	id := em.CreateEntity()

	// 添加没有帧的动画组件
	em.AddComponent(id, &components.AnimationComponent{
		Frames:       []*ebiten.Image{}, // 空帧列表
		FrameSpeed:   0.1,
		FrameCounter: 0,
		CurrentFrame: 0,
		IsLooping:    true,
		IsFinished:   false,
	})

	em.AddComponent(id, &components.SpriteComponent{
		Image: nil,
	})

	// 更新不应该崩溃
	system.Update(0.1)

	// 验证系统没有崩溃即可
	animComp, _ := em.GetComponent(id, reflect.TypeOf(&components.AnimationComponent{}))
	if animComp == nil {
		t.Error("Animation component should still exist")
	}
}
