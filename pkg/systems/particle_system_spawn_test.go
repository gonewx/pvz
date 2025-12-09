package systems

import (
	"testing"

	"github.com/gonewx/pvz/internal/particle"
	"github.com/gonewx/pvz/pkg/components"
	"github.com/gonewx/pvz/pkg/ecs"
	"github.com/gonewx/pvz/pkg/game"
)

// TestSpawnRate0_OneShotMode 测试 SpawnRate=0 时的一次性发射模式
// 场景：Planting.xml (种植土粒效果)
// 配置：SpawnRate=0, SpawnMinActive=8, SpawnMaxLaunched=0 (未配置)
// 预期：一次性发射 8 个粒子，然后停止（总粒子数不超过 8）
func TestSpawnRate0_OneShotMode(t *testing.T) {
	// 创建实体管理器和粒子系统
	em := ecs.NewEntityManager()
	rm := game.NewResourceManager(nil)
	ps := NewParticleSystem(em, rm)

	// 创建发射器实体
	emitterID := em.CreateEntity()

	// 创建 PositionComponent
	posComp := &components.PositionComponent{
		X: 100,
		Y: 200,
	}
	em.AddComponent(emitterID, posComp)

	// 创建 EmitterComponent（模拟 Planting.xml 配置）
	emitterConfig := particle.EmitterConfig{
		Name:             "Planting",
		SpawnMinActive:   "8",  // 活跃粒子数 = 8
		SpawnMaxLaunched: "",   // 未配置（默认为0）
		SpawnRate:        "",   // 未配置（默认为0）
		SystemDuration:   "30", // 30 厘秒 = 0.3 秒
		ParticleDuration: "30", // 30 厘秒 = 0.3 秒
		LaunchSpeed:      "400",
		LaunchAngle:      "180",
		Image:            "IMAGE_DIRTSMALL",
	}

	emitterComp := &components.EmitterComponent{
		Config:           &emitterConfig,
		Active:           true,
		Age:              0,
		SystemDuration:   0.3, // 0.3 秒
		NextSpawnTime:    0,
		ActiveParticles:  make([]ecs.EntityID, 0),
		TotalLaunched:    0,
		SpawnRate:        0, // SpawnRate=0
		SpawnMinActive:   8, // 保持 8 个粒子活跃
		SpawnMaxActive:   0, // 未配置
		SpawnMaxLaunched: 0, // 未配置（关键：这个字段应该默认等于 SpawnMinActive）
	}
	em.AddComponent(emitterID, emitterComp)

	// 第一帧更新：应该生成 8 个粒子
	dt := 1.0 / 60.0 // 假设 60 FPS
	ps.updateEmitters(dt)

	// 验证：应该发射了 8 个粒子
	if emitterComp.TotalLaunched != 8 {
		t.Errorf("第一帧应该发射 8 个粒子，实际: %d", emitterComp.TotalLaunched)
	}

	if len(emitterComp.ActiveParticles) != 8 {
		t.Errorf("第一帧应该有 8 个活跃粒子，实际: %d", len(emitterComp.ActiveParticles))
	}

	// 模拟粒子消失（删除一些粒子）
	deletedCount := 0
	for i := 0; i < 3; i++ {
		if i < len(emitterComp.ActiveParticles) {
			particleID := emitterComp.ActiveParticles[i]
			t.Logf("删除粒子 ID=%d", particleID)
			em.DestroyEntity(particleID)
			deletedCount++
		}
	}
	// 真正删除标记的实体（延迟删除机制）
	em.RemoveMarkedEntities()
	t.Logf("已删除 %d 个粒子，ActiveParticles（删除前）=%d", deletedCount, len(emitterComp.ActiveParticles))

	// 第二帧更新：即使有粒子消失，也不应该补充新粒子（因为已经达到 SpawnMaxLaunched=8）
	ps.updateEmitters(dt)

	t.Logf("第二帧更新后：TotalLaunched=%d, ActiveParticles（清理后）=%d", emitterComp.TotalLaunched, len(emitterComp.ActiveParticles))

	// 验证：总发射数应该仍然是 8（不补充新粒子）
	if emitterComp.TotalLaunched != 8 {
		t.Errorf("第二帧不应该补充新粒子，总发射数应该仍然是 8，实际: %d", emitterComp.TotalLaunched)
	}

	// 验证：活跃粒子数应该减少（因为有粒子被删除了）
	if len(emitterComp.ActiveParticles) != 5 {
		t.Errorf("第二帧活跃粒子数应该是 5（8-3），实际: %d", len(emitterComp.ActiveParticles))
	}

	t.Logf("✅ 一次性发射模式测试通过：TotalLaunched=%d（预期8），ActiveParticles=%d（预期<8）",
		emitterComp.TotalLaunched, len(emitterComp.ActiveParticles))
}

// TestSpawnRate0_ContinuousMode 测试 SpawnRate=0 时的持续补充模式
// 场景：Award.xml (奖励粒子效果)
// 配置：SpawnRate=0, SpawnMinActive=8, SpawnMaxLaunched=20
// 预期：持续补充粒子到 8 个活跃，直到总发射数达到 20
func TestSpawnRate0_ContinuousMode(t *testing.T) {
	// 创建实体管理器和粒子系统
	em := ecs.NewEntityManager()
	rm := game.NewResourceManager(nil)
	ps := NewParticleSystem(em, rm)

	// 创建发射器实体
	emitterID := em.CreateEntity()

	// 创建 PositionComponent
	posComp := &components.PositionComponent{
		X: 100,
		Y: 200,
	}
	em.AddComponent(emitterID, posComp)

	// 创建 EmitterComponent（模拟 Award.xml 配置）
	emitterConfig := particle.EmitterConfig{
		Name:             "Award",
		SpawnMinActive:   "8",  // 活跃粒子数 = 8
		SpawnMaxLaunched: "20", // 总共最多发射 20 个粒子
		SpawnRate:        "",   // 未配置（默认为0）
		SystemDuration:   "200",
		ParticleDuration: "100",
	}

	emitterComp := &components.EmitterComponent{
		Config:           &emitterConfig,
		Active:           true,
		Age:              0,
		SystemDuration:   2.0,
		NextSpawnTime:    0,
		ActiveParticles:  make([]ecs.EntityID, 0),
		TotalLaunched:    0,
		SpawnRate:        0,  // SpawnRate=0
		SpawnMinActive:   8,  // 保持 8 个粒子活跃
		SpawnMaxActive:   0,  // 未配置
		SpawnMaxLaunched: 20, // 总共最多发射 20 个
	}
	em.AddComponent(emitterID, emitterComp)

	// 第一帧更新：应该生成 8 个粒子
	dt := 1.0 / 60.0
	ps.updateEmitters(dt)

	if emitterComp.TotalLaunched != 8 {
		t.Errorf("第一帧应该发射 8 个粒子，实际: %d", emitterComp.TotalLaunched)
	}

	// 模拟粒子消失（删除 5 个粒子）
	for i := 0; i < 5; i++ {
		if i < len(emitterComp.ActiveParticles) {
			particleID := emitterComp.ActiveParticles[i]
			em.DestroyEntity(particleID)
		}
	}
	em.RemoveMarkedEntities() // 真正删除标记的实体

	// 第二帧更新：应该补充 5 个粒子（因为有 SpawnMaxLaunched=20 的限制）
	ps.updateEmitters(dt)

	if emitterComp.TotalLaunched != 13 {
		t.Errorf("第二帧应该补充 5 个粒子（总数 8+5=13），实际: %d", emitterComp.TotalLaunched)
	}

	// 继续模拟粒子消失，直到达到 SpawnMaxLaunched=20
	for emitterComp.TotalLaunched < 20 {
		// 删除一些粒子
		for i := 0; i < 3 && len(emitterComp.ActiveParticles) > 0; i++ {
			particleID := emitterComp.ActiveParticles[0]
			em.DestroyEntity(particleID)
			emitterComp.ActiveParticles = emitterComp.ActiveParticles[1:]
		}
		em.RemoveMarkedEntities() // 真正删除标记的实体

		// 更新发射器
		ps.updateEmitters(dt)

		// 防止无限循环
		if emitterComp.TotalLaunched > 25 {
			t.Fatalf("总发射数超过预期（应该停在20），实际: %d", emitterComp.TotalLaunched)
		}
	}

	// 验证：总发射数应该等于 20
	if emitterComp.TotalLaunched != 20 {
		t.Errorf("最终总发射数应该等于 SpawnMaxLaunched=20，实际: %d", emitterComp.TotalLaunched)
	}

	// 再删除一些粒子，验证不会补充新粒子（因为已达到上限）
	for i := 0; i < 5 && len(emitterComp.ActiveParticles) > 0; i++ {
		particleID := emitterComp.ActiveParticles[0]
		em.DestroyEntity(particleID)
		emitterComp.ActiveParticles = emitterComp.ActiveParticles[1:]
	}
	em.RemoveMarkedEntities() // 真正删除标记的实体

	ps.updateEmitters(dt)

	if emitterComp.TotalLaunched != 20 {
		t.Errorf("达到上限后不应该补充新粒子，总发射数应该仍然是 20，实际: %d", emitterComp.TotalLaunched)
	}

	t.Logf("✅ 持续补充模式测试通过：TotalLaunched=%d（预期20）", emitterComp.TotalLaunched)
}
