package components

import (
	"testing"
)

// TestBowlingNutComponent_Initialization 测试组件初始化
func TestBowlingNutComponent_Initialization(t *testing.T) {
	comp := &BowlingNutComponent{
		VelocityX:    250.0,
		Row:          2,
		IsRolling:    true,
		IsExplosive:  false,
		BounceCount:  0,
		SoundPlaying: false,
	}

	if comp.VelocityX != 250.0 {
		t.Errorf("VelocityX = %f, want 250.0", comp.VelocityX)
	}
	if comp.Row != 2 {
		t.Errorf("Row = %d, want 2", comp.Row)
	}
	if !comp.IsRolling {
		t.Error("IsRolling should be true")
	}
	if comp.IsExplosive {
		t.Error("IsExplosive should be false")
	}
	if comp.BounceCount != 0 {
		t.Errorf("BounceCount = %d, want 0", comp.BounceCount)
	}
	if comp.SoundPlaying {
		t.Error("SoundPlaying should be false")
	}
}

// TestBowlingNutComponent_DefaultValues 测试组件默认值
func TestBowlingNutComponent_DefaultValues(t *testing.T) {
	// 零值初始化
	comp := &BowlingNutComponent{}

	if comp.VelocityX != 0 {
		t.Errorf("Default VelocityX = %f, want 0", comp.VelocityX)
	}
	if comp.Row != 0 {
		t.Errorf("Default Row = %d, want 0", comp.Row)
	}
	if comp.IsRolling {
		t.Error("Default IsRolling should be false")
	}
	if comp.IsExplosive {
		t.Error("Default IsExplosive should be false")
	}
	if comp.BounceCount != 0 {
		t.Errorf("Default BounceCount = %d, want 0", comp.BounceCount)
	}
	if comp.SoundPlaying {
		t.Error("Default SoundPlaying should be false")
	}
}

// TestBowlingNutComponent_ExplosiveType 测试爆炸坚果类型
func TestBowlingNutComponent_ExplosiveType(t *testing.T) {
	comp := &BowlingNutComponent{
		VelocityX:   250.0,
		Row:         1,
		IsRolling:   true,
		IsExplosive: true,
		BounceCount: 0,
	}

	if !comp.IsExplosive {
		t.Error("IsExplosive should be true for explosive nut")
	}
}

// TestBowlingNutTypeConstants 测试类型常量
func TestBowlingNutTypeConstants(t *testing.T) {
	if BowlingNutTypeNormal != "normal" {
		t.Errorf("BowlingNutTypeNormal = %s, want 'normal'", BowlingNutTypeNormal)
	}
	if BowlingNutTypeExplosive != "explosive" {
		t.Errorf("BowlingNutTypeExplosive = %s, want 'explosive'", BowlingNutTypeExplosive)
	}
}

// TestBowlingNutComponent_BounceCountIncrement 测试弹射计数
func TestBowlingNutComponent_BounceCountIncrement(t *testing.T) {
	comp := &BowlingNutComponent{
		BounceCount: 0,
	}

	// 模拟弹射增加
	comp.BounceCount++
	if comp.BounceCount != 1 {
		t.Errorf("BounceCount after increment = %d, want 1", comp.BounceCount)
	}

	comp.BounceCount++
	if comp.BounceCount != 2 {
		t.Errorf("BounceCount after second increment = %d, want 2", comp.BounceCount)
	}
}

// TestBowlingNutComponent_SoundState 测试音效状态
func TestBowlingNutComponent_SoundState(t *testing.T) {
	comp := &BowlingNutComponent{
		SoundPlaying: false,
	}

	// 开始播放音效
	comp.SoundPlaying = true
	if !comp.SoundPlaying {
		t.Error("SoundPlaying should be true after starting")
	}

	// 停止播放
	comp.SoundPlaying = false
	if comp.SoundPlaying {
		t.Error("SoundPlaying should be false after stopping")
	}
}

// TestBowlingNutComponent_RollingState 测试滚动状态
func TestBowlingNutComponent_RollingState(t *testing.T) {
	comp := &BowlingNutComponent{
		IsRolling: false,
	}

	// 开始滚动
	comp.IsRolling = true
	if !comp.IsRolling {
		t.Error("IsRolling should be true after starting to roll")
	}

	// 停止滚动（碰撞后）
	comp.IsRolling = false
	if comp.IsRolling {
		t.Error("IsRolling should be false after stopping")
	}
}

