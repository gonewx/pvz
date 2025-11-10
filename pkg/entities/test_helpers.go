package entities

import (
	"reflect"

	"github.com/decker502/pvz/internal/reanim"
	"github.com/decker502/pvz/pkg/components"
	"github.com/decker502/pvz/pkg/config"
	"github.com/decker502/pvz/pkg/ecs"
	"github.com/decker502/pvz/pkg/game"
	"github.com/hajimehoshi/ebiten/v2"
)

// mockReanimSystem 是一个用于测试的 mock ReanimSystem
type mockReanimSystem struct {
	em *ecs.EntityManager
}

func (m *mockReanimSystem) PlayAnimation(entityID ecs.EntityID, animName string) error {
	// Mock implementation - 设置 CurrentAnim 字段以满足测试
	// Story 13.2: 移除 CurrentFrame 设置（已废弃）
	if m.em != nil {
		if reanimComp, ok := m.em.GetComponent(entityID, reflect.TypeOf(&components.ReanimComponent{})); ok {
			reanim := reanimComp.(*components.ReanimComponent)
			// reanim.CurrentAnim = animName
			reanim.CurrentAnimations = []string{animName}
			reanim.FrameAccumulator = 0.0
			reanim.IsLooping = true
			reanim.IsFinished = false
		}
	}
	return nil
}

func (m *mockReanimSystem) PlayAnimationNoLoop(entityID ecs.EntityID, animName string) error {
	// Mock implementation - 设置 CurrentAnim 字段并标记为不循环
	// Story 13.2: 移除 CurrentFrame 设置（已废弃）
	if m.em != nil {
		if reanimComp, ok := m.em.GetComponent(entityID, reflect.TypeOf(&components.ReanimComponent{})); ok {
			reanim := reanimComp.(*components.ReanimComponent)
			// reanim.CurrentAnim = animName
			reanim.CurrentAnimations = []string{animName}
			reanim.FrameAccumulator = 0.0
			reanim.IsLooping = false
			reanim.IsFinished = false
		}
	}
	return nil
}

func (m *mockReanimSystem) RenderToTexture(entityID ecs.EntityID, target *ebiten.Image) error {
	// Mock implementation - 用于植物卡片图标离屏渲染
	// 在测试中，我们不需要真正渲染，只需返回 nil 表示成功
	return nil
}

func (m *mockReanimSystem) PrepareStaticPreview(entityID ecs.EntityID, reanimName string) error {
	// Mock implementation - 用于静态预览准备（Story 11.1）
	// 在测试中，设置基本的静态预览状态
	// Story 13.2: 移除 CurrentFrame 设置（已废弃）
	if m.em != nil {
		if reanimComp, ok := m.em.GetComponent(entityID, reflect.TypeOf(&components.ReanimComponent{})); ok {
			reanim := reanimComp.(*components.ReanimComponent)
			// reanim.CurrentAnim = "static_preview"
			reanim.CurrentAnimations = []string{"static_preview"}
			reanim.IsLooping = false
			reanim.IsFinished = true
		}
	}
	return nil
}

// ApplyReanimConfig 应用 Reanim 配置到指定实体 (Story 13.5)
// Mock implementation - 在测试中不需要真正应用配置，只需返回成功
func (m *mockReanimSystem) ApplyReanimConfig(entityID ecs.EntityID, cfg *config.ReanimConfig) error {
	// Mock implementation - 测试中不需要真正的配置应用
	return nil
}

// PlayCombo 播放配置驱动的动画组合 (Story 13.6)
// Mock implementation - 在测试中设置基本的动画状态
func (m *mockReanimSystem) PlayCombo(entityID ecs.EntityID, unitID, comboName string) error {
	// Mock implementation - 简化版本，直接播放第一个动画
	if m.em != nil {
		if reanimComp, ok := m.em.GetComponent(entityID, reflect.TypeOf(&components.ReanimComponent{})); ok {
			reanim := reanimComp.(*components.ReanimComponent)
			// reanim.CurrentAnim = comboName
			reanim.CurrentAnimations = []string{comboName}
			reanim.IsLooping = true
			reanim.IsFinished = false
		}
	}
	return nil
}

// PlayDefaultAnimation 播放默认动画 (Story 13.6)
// Mock implementation - 在测试中设置基本的默认动画状态
func (m *mockReanimSystem) PlayDefaultAnimation(entityID ecs.EntityID, unitID string) error {
	// Mock implementation - 播放名为 "default" 的动画
	if m.em != nil {
		if reanimComp, ok := m.em.GetComponent(entityID, reflect.TypeOf(&components.ReanimComponent{})); ok {
			reanim := reanimComp.(*components.ReanimComponent)
			// reanim.CurrentAnim = "anim_idle"
			reanim.CurrentAnimations = []string{"anim_idle"}
			reanim.IsLooping = true
			reanim.IsFinished = false
		}
	}
	return nil
}

// ResourceLoader 定义测试中需要的资源加载接口
// 这允许我们在测试中使用 mock 实现，而在生产代码中使用真实的 ResourceManager
type ResourceLoader interface {
	LoadImage(path string) (*ebiten.Image, error)
	GetReanimXML(unitName string) *reanim.ReanimXML
	GetReanimPartImages(unitName string) map[string]*ebiten.Image
}

// mockResourceManager 实现 ResourceLoader 接口，避免文件 I/O
type mockResourceManager struct{}

// newMockResourceManager 创建一个不需要文件的 mock 资源管理器
func newMockResourceManager() ResourceLoader {
	return &mockResourceManager{}
}

// LoadImage 返回测试图像，无需文件 I/O
func (m *mockResourceManager) LoadImage(path string) (*ebiten.Image, error) {
	// 返回一个 10x10 的测试图像
	return ebiten.NewImage(10, 10), nil
}

// GetReanimXML 返回 mock Reanim 数据，无需文件加载
func (m *mockResourceManager) GetReanimXML(unitName string) *reanim.ReanimXML {
	// 返回最小的 mock Reanim 数据
	return &reanim.ReanimXML{
		FPS: 12,
		Tracks: []reanim.Track{
			{
				Name: "body",
				Frames: []reanim.Frame{
					{}, // 空帧足够用于测试
				},
			},
		},
	}
}

// GetReanimPartImages 返回 mock 部件图像，无需文件加载
func (m *mockResourceManager) GetReanimPartImages(unitName string) map[string]*ebiten.Image {
	// 返回一个包含单个测试图像的 map
	return map[string]*ebiten.Image{
		"test_part": ebiten.NewImage(32, 32),
	}
}

// Ensure mockResourceManager implements ResourceLoader
var _ ResourceLoader = (*mockResourceManager)(nil)

// Ensure game.ResourceManager also implements ResourceLoader (at compile time)
var _ ResourceLoader = (*game.ResourceManager)(nil)
