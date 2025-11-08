package systems

import (
	"testing"

	"github.com/decker502/pvz/internal/reanim"
	"github.com/decker502/pvz/pkg/components"
	"github.com/decker502/pvz/pkg/config"
	"github.com/decker502/pvz/pkg/ecs"
	"github.com/hajimehoshi/ebiten/v2"
)

// BenchmarkPlayCombo 测试 PlayCombo 的性能
func BenchmarkPlayCombo(b *testing.B) {
	// Setup
	em := ecs.NewEntityManager()
	rs := NewReanimSystem(em)

	// 加载配置管理器
	configManager, err := config.NewReanimConfigManager("data/reanim_config.yaml")
	if err != nil {
		b.Skipf("跳过基准测试：无法加载配置文件: %v", err)
	}
	rs.SetConfigManager(configManager)

	// 创建测试实体
	entity := em.CreateEntity()
	reanimXML := createBenchReanimData()

	ecs.AddComponent(em, entity, &components.ReanimComponent{
		Reanim:       reanimXML,
		PartImages:   make(map[string]*ebiten.Image),
		MergedTracks: reanim.BuildMergedTracks(reanimXML),
	})

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = rs.PlayCombo(entity, "peashooter", "attack")
	}
}

// BenchmarkPlayAnimations 测试旧 API PlayAnimations 的性能（用于对比）
func BenchmarkPlayAnimations(b *testing.B) {
	// Setup
	em := ecs.NewEntityManager()
	rs := NewReanimSystem(em)

	// 创建测试实体
	entity := em.CreateEntity()
	reanimXML := createBenchReanimData()

	ecs.AddComponent(em, entity, &components.ReanimComponent{
		Reanim:       reanimXML,
		PartImages:   make(map[string]*ebiten.Image),
		MergedTracks: reanim.BuildMergedTracks(reanimXML),
	})

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = rs.PlayAnimations(entity, []string{"anim_shooting", "anim_head_idle"})
	}
}

// BenchmarkPlayComboVsPlayAnimations 对比 PlayCombo 和 PlayAnimations 的性能
func BenchmarkPlayComboVsPlayAnimations(b *testing.B) {
	b.Run("PlayCombo", func(b *testing.B) {
		em := ecs.NewEntityManager()
		rs := NewReanimSystem(em)

		configManager, err := config.NewReanimConfigManager("data/reanim_config.yaml")
		if err != nil {
			b.Skipf("跳过基准测试：无法加载配置文件: %v", err)
		}
		rs.SetConfigManager(configManager)

		entity := em.CreateEntity()
		reanimXML := createBenchReanimData()

		ecs.AddComponent(em, entity, &components.ReanimComponent{
			Reanim:       reanimXML,
			PartImages:   make(map[string]*ebiten.Image),
			MergedTracks: reanim.BuildMergedTracks(reanimXML),
		})

		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			_ = rs.PlayCombo(entity, "peashooter", "attack")
		}
	})

	b.Run("PlayAnimations", func(b *testing.B) {
		em := ecs.NewEntityManager()
		rs := NewReanimSystem(em)

		entity := em.CreateEntity()
		reanimXML := createBenchReanimData()

		ecs.AddComponent(em, entity, &components.ReanimComponent{
			Reanim:       reanimXML,
			PartImages:   make(map[string]*ebiten.Image),
			MergedTracks: reanim.BuildMergedTracks(reanimXML),
		})

		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			_ = rs.PlayAnimations(entity, []string{"anim_shooting", "anim_head_idle"})
		}
	})
}

// BenchmarkPlayDefaultAnimation 测试 PlayDefaultAnimation 的性能
func BenchmarkPlayDefaultAnimation(b *testing.B) {
	// Setup
	em := ecs.NewEntityManager()
	rs := NewReanimSystem(em)

	// 加载配置管理器
	configManager, err := config.NewReanimConfigManager("data/reanim_config.yaml")
	if err != nil {
		b.Skipf("跳过基准测试：无法加载配置文件: %v", err)
	}
	rs.SetConfigManager(configManager)

	// 创建测试实体
	entity := em.CreateEntity()
	reanimXML := createBenchReanimData()

	ecs.AddComponent(em, entity, &components.ReanimComponent{
		Reanim:       reanimXML,
		PartImages:   make(map[string]*ebiten.Image),
		MergedTracks: reanim.BuildMergedTracks(reanimXML),
	})

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = rs.PlayDefaultAnimation(entity, "peashooter")
	}
}

// createBenchReanimData 创建用于基准测试的 Reanim 数据
func createBenchReanimData() *reanim.ReanimXML {
	return &reanim.ReanimXML{
		FPS: 12,
		Tracks: []reanim.Track{
			{
				Name: "anim_shooting",
				Frames: []reanim.Frame{
					{X: ptrFloat64Bench(0), Y: ptrFloat64Bench(0), FrameNum: ptrIntBench(0), ImagePath: "test_image"},
					{X: ptrFloat64Bench(10), Y: ptrFloat64Bench(0), FrameNum: ptrIntBench(1), ImagePath: "test_image"},
				},
			},
			{
				Name: "anim_head_idle",
				Frames: []reanim.Frame{
					{X: ptrFloat64Bench(0), Y: ptrFloat64Bench(0), FrameNum: ptrIntBench(0), ImagePath: "test_head"},
				},
			},
			{
				Name: "anim_full_idle",
				Frames: []reanim.Frame{
					{X: ptrFloat64Bench(0), Y: ptrFloat64Bench(0), FrameNum: ptrIntBench(0), ImagePath: "test_idle"},
				},
			},
			{
				Name: "anim_face",
				Frames: []reanim.Frame{
					{X: ptrFloat64Bench(0), Y: ptrFloat64Bench(0), FrameNum: ptrIntBench(0), ImagePath: "test_face"},
				},
			},
			{
				Name: "anim_stem",
				Frames: []reanim.Frame{
					{X: ptrFloat64Bench(37.6), Y: ptrFloat64Bench(48.7), FrameNum: ptrIntBench(0)},
				},
			},
		},
	}
}

func ptrFloat64Bench(v float64) *float64 {
	return &v
}

func ptrIntBench(v int) *int {
	return &v
}
