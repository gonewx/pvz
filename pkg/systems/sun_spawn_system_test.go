package systems

import (
	"testing"

	"github.com/decker502/pvz/pkg/components"
	"github.com/decker502/pvz/pkg/ecs"
	"github.com/decker502/pvz/pkg/game"
)

// TestSunSpawnIntervalFormula_Initial 测试 count=0 时间隔范围 (4.25-7.00秒)
func TestSunSpawnIntervalFormula_Initial(t *testing.T) {
	em := ecs.NewEntityManager()
	rm := game.NewResourceManager(nil)
	system := NewSunSpawnSystem(em, rm, 250.0, 900.0, 100.0, 550.0)

	// count=0 时，间隔应在 4.25-7.00 秒之间
	// 公式: (0*10 + 425 + rand(0~275)) / 100 = (425 + 0~275) / 100 = 4.25 ~ 7.00
	for i := 0; i < 100; i++ {
		system.sunDroppedCount = 0
		interval := system.calculateNextInterval()
		if interval < 4.25 || interval > 7.00 {
			t.Errorf("Initial interval (count=0) out of range: got %.2f, want 4.25-7.00", interval)
		}
	}
	t.Logf("✓ Initial interval formula correct (count=0): 4.25-7.00s")
}

// TestSunSpawnIntervalFormula_Mid 测试 count=30 时间隔范围 (7.25-10.00秒)
func TestSunSpawnIntervalFormula_Mid(t *testing.T) {
	em := ecs.NewEntityManager()
	rm := game.NewResourceManager(nil)
	system := NewSunSpawnSystem(em, rm, 250.0, 900.0, 100.0, 550.0)

	// count=30 时，间隔应在 7.25-10.00 秒之间
	// 公式: (30*10 + 425 + rand(0~275)) / 100 = (725 + 0~275) / 100 = 7.25 ~ 10.00
	for i := 0; i < 100; i++ {
		system.sunDroppedCount = 30
		interval := system.calculateNextInterval()
		if interval < 7.25 || interval > 10.00 {
			t.Errorf("Mid interval (count=30) out of range: got %.2f, want 7.25-10.00", interval)
		}
	}
	t.Logf("✓ Mid interval formula correct (count=30): 7.25-10.00s")
}

// TestSunSpawnIntervalFormula_Stable 测试 count=53+ 时间隔范围 (9.50-12.25秒)
// 注：count=52时基础间隔为945cs(还未达到950上限)，count=53时才真正稳定
func TestSunSpawnIntervalFormula_Stable(t *testing.T) {
	em := ecs.NewEntityManager()
	rm := game.NewResourceManager(nil)
	system := NewSunSpawnSystem(em, rm, 250.0, 900.0, 100.0, 550.0)

	// count=53+ 时，间隔应在 9.50-12.25 秒之间
	// 公式: (min(53*10 + 425, 950) + rand(0~275)) / 100 = (950 + 0~275) / 100 = 9.50 ~ 12.25
	testCounts := []int{53, 60, 100, 1000}
	for _, count := range testCounts {
		for i := 0; i < 50; i++ {
			system.sunDroppedCount = count
			interval := system.calculateNextInterval()
			if interval < 9.50 || interval > 12.25 {
				t.Errorf("Stable interval (count=%d) out of range: got %.2f, want 9.50-12.25", count, interval)
			}
		}
	}
	t.Logf("✓ Stable interval formula correct (count=53+): 9.50-12.25s")
}

// TestSunDroppedCountIncrement 测试计数器递增逻辑
func TestSunDroppedCountIncrement(t *testing.T) {
	em := ecs.NewEntityManager()
	rm := game.NewResourceManager(nil)
	system := NewSunSpawnSystem(em, rm, 250.0, 900.0, 100.0, 550.0)

	// 初始计数应为 0
	if system.sunDroppedCount != 0 {
		t.Errorf("Initial sunDroppedCount should be 0, got %d", system.sunDroppedCount)
	}

	// 初始间隔应在 4.25-7.00 之间
	initialInterval := system.spawnInterval
	if initialInterval < 4.25 || initialInterval > 7.00 {
		t.Errorf("Initial interval out of range: got %.2f, want 4.25-7.00", initialInterval)
	}

	// 模拟第一次阳光生成（等待初始间隔）
	system.Update(initialInterval + 0.1)

	// 计数应增加到 1
	if system.sunDroppedCount != 1 {
		t.Errorf("sunDroppedCount after first spawn should be 1, got %d", system.sunDroppedCount)
	}

	// 验证有一个阳光实体
	sunEntities := ecs.GetEntitiesWith1[*components.SunComponent](em)
	if len(sunEntities) != 1 {
		t.Errorf("Expected 1 sun entity, got %d", len(sunEntities))
	}

	t.Logf("✓ Sun dropped count increments correctly")
}

// TestSunSpawnSystem_Reset 测试 Reset 方法
func TestSunSpawnSystem_Reset(t *testing.T) {
	em := ecs.NewEntityManager()
	rm := game.NewResourceManager(nil)
	system := NewSunSpawnSystem(em, rm, 250.0, 900.0, 100.0, 550.0)

	// 模拟生成多个阳光
	for i := 0; i < 5; i++ {
		system.Update(10.0) // 足够触发生成
	}

	// 确认计数器已增加
	if system.sunDroppedCount == 0 {
		t.Error("sunDroppedCount should be > 0 after spawning suns")
	}

	// 调用 Reset
	system.Reset()

	// 验证重置状态
	if system.sunDroppedCount != 0 {
		t.Errorf("sunDroppedCount after Reset should be 0, got %d", system.sunDroppedCount)
	}
	if system.spawnTimer != 0 {
		t.Errorf("spawnTimer after Reset should be 0, got %.2f", system.spawnTimer)
	}
	// 重置后间隔应在初始范围内
	if system.spawnInterval < 4.25 || system.spawnInterval > 7.00 {
		t.Errorf("spawnInterval after Reset out of range: got %.2f, want 4.25-7.00", system.spawnInterval)
	}

	t.Logf("✓ Reset method works correctly")
}

// TestSunSpawnSystem_SpawnWithOriginalFormula 测试使用原版公式生成阳光
func TestSunSpawnSystem_SpawnWithOriginalFormula(t *testing.T) {
	em := ecs.NewEntityManager()
	rm := game.NewResourceManager(nil)

	system := NewSunSpawnSystem(em, rm, 250.0, 900.0, 100.0, 550.0)

	// 初始间隔应在原版范围内 (4.25-7.00秒)
	if system.spawnInterval < 4.25 || system.spawnInterval > 7.00 {
		t.Errorf("Initial interval should be 4.25-7.00, got %.2f", system.spawnInterval)
	}

	// 更新足够时间触发第一次生成
	system.Update(7.5)

	// 检查是否生成了阳光实体
	sunEntities := ecs.GetEntitiesWith1[*components.SunComponent](em)
	if len(sunEntities) != 1 {
		t.Errorf("Expected 1 sun after first interval, got %d", len(sunEntities))
	}

	// 检查计时器是否重置
	if system.spawnTimer != 0 {
		t.Errorf("Expected spawnTimer reset to 0, got %.2f", system.spawnTimer)
	}

	// 检查计数器是否增加
	if system.sunDroppedCount != 1 {
		t.Errorf("Expected sunDroppedCount to be 1, got %d", system.sunDroppedCount)
	}

	t.Logf("✓ Sun spawn with original formula works correctly")
}

// TestSunSpawnSystem_MultipleCycles 测试多个生成周期
func TestSunSpawnSystem_MultipleCycles(t *testing.T) {
	em := ecs.NewEntityManager()
	rm := game.NewResourceManager(nil)

	system := NewSunSpawnSystem(em, rm, 250.0, 900.0, 100.0, 550.0)

	// 使用足够大的间隔来确保每次都能触发生成
	// 最大可能间隔是 12.25 秒
	for i := 0; i < 5; i++ {
		system.Update(13.0) // 每次更新13秒，确保触发生成
	}

	// 应该生成 5 个阳光
	sunEntities := ecs.GetEntitiesWith1[*components.SunComponent](em)
	if len(sunEntities) != 5 {
		t.Errorf("Expected 5 suns after 5 cycles, got %d", len(sunEntities))
	}

	// 计数器应为 5
	if system.sunDroppedCount != 5 {
		t.Errorf("Expected sunDroppedCount to be 5, got %d", system.sunDroppedCount)
	}

	t.Logf("✓ Multiple spawn cycles work correctly")
}
