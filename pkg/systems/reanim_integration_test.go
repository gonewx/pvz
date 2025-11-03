package systems

import (
	"testing"

	"github.com/decker502/pvz/internal/reanim"
	"github.com/decker502/pvz/pkg/components"
	"github.com/decker502/pvz/pkg/ecs"
)

// ==================================================================
// Story 6.6: Integration Tests (集成测试)
// ==================================================================

// TestPlaybackModeIntegration_PeaShooter 测试豌豆射手的模式检测
func TestPlaybackModeIntegration_PeaShooter(t *testing.T) {
	// 模拟豌豆射手的 Reanim 数据
	frameNum0 := 0
	frameNumMinus1 := -1

	reanimData := &reanim.ReanimXML{
		FPS: 12,
		Tracks: []reanim.Track{
			// 动画定义轨道
			{Name: "anim_idle", Frames: []reanim.Frame{{FrameNum: &frameNum0}}},
			{Name: "anim_shooting", Frames: []reanim.Frame{{FrameNum: &frameNum0}}},
			{Name: "anim_head_idle", Frames: []reanim.Frame{{FrameNum: &frameNum0}}},
			// 部件轨道
			{Name: "head", Frames: []reanim.Frame{{ImagePath: "head.png", FrameNum: &frameNumMinus1}}},
			{Name: "body", Frames: []reanim.Frame{{ImagePath: "body.png", FrameNum: &frameNumMinus1}}},
			{Name: "stalk", Frames: []reanim.Frame{{ImagePath: "stalk.png", FrameNum: &frameNumMinus1}}},
		},
	}

	// 检测模式
	mode := detectPlaybackMode(reanimData)

	// 豌豆射手应该被检测为混合模式
	if mode != ModeBlended {
		t.Errorf("Expected ModeBlended for PeaShooter, got %v", mode)
	}
}

// TestPlaybackModeIntegration_SunFlower 测试向日葵的模式检测
func TestPlaybackModeIntegration_SunFlower(t *testing.T) {
	// 模拟向日葵的 Reanim 数据（多轨道骨骼动画）
	reanimData := &reanim.ReanimXML{
		FPS: 12,
		Tracks: []reanim.Track{},
	}

	// 向日葵有约 30 个部件轨道
	for i := 0; i < 30; i++ {
		track := reanim.Track{
			Name:   "part_" + string(rune('A'+i%26)),
			Frames: []reanim.Frame{{ImagePath: "part.png"}},
		}
		reanimData.Tracks = append(reanimData.Tracks, track)
	}

	// 检测模式
	mode := detectPlaybackMode(reanimData)

	// 向日葵应该被检测为骨骼动画模式
	if mode != ModeSkeleton {
		t.Errorf("Expected ModeSkeleton for SunFlower, got %v", mode)
	}
}

// TestPlaybackModeIntegration_SelectorScreen 测试主菜单的模式检测
func TestPlaybackModeIntegration_SelectorScreen(t *testing.T) {
	// 模拟 SelectorScreen 的 Reanim 数据（复杂场景动画）
	reanimData := &reanim.ReanimXML{
		FPS:    12,
		Tracks: []reanim.Track{},
	}

	// SelectorScreen 有 500+ 个轨道
	for i := 0; i < 500; i++ {
		track := reanim.Track{
			Name:   "element_" + string(rune('0'+i%10)),
			Frames: []reanim.Frame{{ImagePath: "element.png"}},
		}
		reanimData.Tracks = append(reanimData.Tracks, track)
	}

	// 检测模式
	mode := detectPlaybackMode(reanimData)

	// SelectorScreen 应该被检测为复杂场景模式
	if mode != ModeComplexScene {
		t.Errorf("Expected ModeComplexScene for SelectorScreen, got %v", mode)
	}
}

// TestReanimSystemIntegration_PlayAnimation 测试 ReanimSystem 的 PlayAnimation 集成
func TestReanimSystemIntegration_PlayAnimation(t *testing.T) {
	// 创建 ECS 环境
	em := ecs.NewEntityManager()
	reanimSys := NewReanimSystem(em)

	// 创建测试实体
	entity := em.CreateEntity()

	// 创建混合模式的 Reanim 数据
	frameNum0 := 0
	frameNumMinus1 := -1

	reanimData := &reanim.ReanimXML{
		FPS: 12,
		Tracks: []reanim.Track{
			{Name: "anim_idle", Frames: []reanim.Frame{{FrameNum: &frameNum0}}},
			{Name: "anim_shooting", Frames: []reanim.Frame{{FrameNum: &frameNum0}}},
			{Name: "head", Frames: []reanim.Frame{{ImagePath: "head.png", FrameNum: &frameNumMinus1}}},
		},
	}

	// 添加 ReanimComponent
	ecs.AddComponent(em, entity, &components.ReanimComponent{
		Reanim: reanimData,
	})

	// 播放 anim_idle
	err := reanimSys.PlayAnimation(entity, "anim_idle")
	if err != nil {
		t.Fatalf("PlayAnimation failed: %v", err)
	}

	// 验证模式被设置
	comp, _ := ecs.GetComponent[*components.ReanimComponent](em, entity)
	if comp.PlaybackMode != int(ModeBlended) {
		t.Errorf("Expected PlaybackMode to be ModeBlended, got %v", PlaybackMode(comp.PlaybackMode))
	}

	// 验证 CurrentAnim 被设置
	if comp.CurrentAnim != "anim_idle" {
		t.Errorf("Expected CurrentAnim to be 'anim_idle', got '%s'", comp.CurrentAnim)
	}

	// 验证 MergedTracks 被构建
	if len(comp.MergedTracks) == 0 {
		t.Error("Expected MergedTracks to be built")
	}
}

// TestReanimSystemIntegration_StrategySelection 测试策略选择
func TestReanimSystemIntegration_StrategySelection(t *testing.T) {
	em := ecs.NewEntityManager()
	reanimSys := NewReanimSystem(em)

	// 验证所有策略都已初始化
	if len(reanimSys.strategies) != 5 {
		t.Errorf("Expected 5 strategies, got %d", len(reanimSys.strategies))
	}

	// 验证每种模式都有对应的策略
	modes := []PlaybackMode{
		ModeSimple,
		ModeSkeleton,
		ModeSequence,
		ModeComplexScene,
		ModeBlended,
	}

	for _, mode := range modes {
		if reanimSys.strategies[mode] == nil {
			t.Errorf("Strategy for mode %v is nil", mode)
		}
	}
}

// TestReanimSystemIntegration_BackwardCompatibility 测试向后兼容性
func TestReanimSystemIntegration_BackwardCompatibility(t *testing.T) {
	em := ecs.NewEntityManager()
	reanimSys := NewReanimSystem(em)
	entity := em.CreateEntity()

	// 创建豌豆射手的 Reanim 数据
	frameNum0 := 0
	frameNumMinus1 := -1

	reanimData := &reanim.ReanimXML{
		FPS: 12,
		Tracks: []reanim.Track{
			{Name: "anim_idle", Frames: []reanim.Frame{{FrameNum: &frameNum0}}},
			{Name: "anim_shooting", Frames: []reanim.Frame{{FrameNum: &frameNum0}}},
			{Name: "head", Frames: []reanim.Frame{{ImagePath: "head.png", FrameNum: &frameNumMinus1}}},
			{Name: "body", Frames: []reanim.Frame{{ImagePath: "body.png", FrameNum: &frameNumMinus1}}},
		},
	}

	ecs.AddComponent(em, entity, &components.ReanimComponent{
		Reanim: reanimData,
	})

	// 测试播放 anim_shooting（应该启用双动画叠加）
	err := reanimSys.PlayAnimation(entity, "anim_shooting")
	if err != nil {
		t.Fatalf("PlayAnimation failed: %v", err)
	}

	comp, _ := ecs.GetComponent[*components.ReanimComponent](em, entity)

	// 验证向后兼容性：anim_shooting 应该启用 IsBlending
	if !comp.IsBlending {
		t.Error("Expected IsBlending to be true for anim_shooting")
	}

	if comp.PrimaryAnimation != "anim_idle" {
		t.Errorf("Expected PrimaryAnimation to be 'anim_idle', got '%s'", comp.PrimaryAnimation)
	}

	if comp.SecondaryAnimation != "anim_shooting" {
		t.Errorf("Expected SecondaryAnimation to be 'anim_shooting', got '%s'", comp.SecondaryAnimation)
	}

	// 测试播放 anim_idle（不应该启用双动画叠加）
	err = reanimSys.PlayAnimation(entity, "anim_idle")
	if err != nil {
		t.Fatalf("PlayAnimation failed: %v", err)
	}

	comp, _ = ecs.GetComponent[*components.ReanimComponent](em, entity)

	if comp.IsBlending {
		t.Error("Expected IsBlending to be false for anim_idle")
	}
}
