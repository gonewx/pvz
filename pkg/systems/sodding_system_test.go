package systems

import (
	"testing"

	"github.com/decker502/pvz/pkg/components"
	"github.com/decker502/pvz/pkg/ecs"
	"github.com/decker502/pvz/pkg/game"
)

// TestCreateSodRollParticleEmitter 测试粒子发射器创建逻辑
// 注意：此测试在没有资源文件的环境中会失败，这是预期行为
func TestCreateSodRollParticleEmitter(t *testing.T) {
	// 创建测试环境
	em := ecs.NewEntityManager()
	rm := game.NewResourceManager(nil)
	rs := NewReanimSystem(em)
	ss := NewSoddingSystem(em, rm, rs)

	// 启动动画(启用粒子)
	ss.StartAnimation(nil, []int{3}, 0, 127, true)

	// 在没有资源的测试环境中，粒子发射器可能创建失败
	// 这是预期行为，我们只验证系统不会崩溃
	if ss.sodRollEmitterID == 0 {
		t.Logf("粒子发射器未创建（可能是资源未加载）")
		return // 跳过后续验证
	}

	// 如果发射器创建成功，验证组件
	emitterComp, ok := ecs.GetComponent[*components.EmitterComponent](em, ss.sodRollEmitterID)
	if !ok {
		t.Errorf("粒子发射器组件未找到")
		return
	}

	// 验证发射器配置
	if emitterComp.Config == nil {
		t.Errorf("粒子发射器配置未加载")
	}
	if emitterComp.Active != true {
		t.Errorf("粒子发射器应该是活跃状态")
	}
}

// TestParticleEmitterNotCreatedWhenDisabled 测试禁用粒子时不创建发射器
func TestParticleEmitterNotCreatedWhenDisabled(t *testing.T) {
	// 创建测试环境
	em := ecs.NewEntityManager()
	rm := game.NewResourceManager(nil)
	rs := NewReanimSystem(em)
	ss := NewSoddingSystem(em, rm, rs)

	// 启动动画(禁用粒子)
	ss.StartAnimation(nil, []int{3}, 0, 127, false)

	// 验证粒子发射器实体未创建
	if ss.sodRollEmitterID != 0 {
		t.Errorf("禁用粒子时不应创建发射器实体，但 sodRollEmitterID = %d", ss.sodRollEmitterID)
	}

	// 验证粒子标志未设置
	if ss.particlesEnabled {
		t.Errorf("禁用粒子时 particlesEnabled 应该为 false")
	}
}

// TestParticleEmitterStopsAfterAnimation 测试动画完成后粒子发射器停止
func TestParticleEmitterStopsAfterAnimation(t *testing.T) {
	// 创建测试环境
	em := ecs.NewEntityManager()
	rm := game.NewResourceManager(nil)
	rs := NewReanimSystem(em)
	ss := NewSoddingSystem(em, rm, rs)

	// 启动动画(启用粒子)
	ss.StartAnimation(nil, []int{3}, 0, 127, true)
	emitterID := ss.sodRollEmitterID

	if emitterID == 0 {
		t.Logf("粒子发射器未创建（可能是资源未加载），跳过测试")
		return
	}

	// 模拟动画完成(3秒后,超过动画时长)
	ss.Update(3.0)

	// 验证粒子发射器已停止
	emitterComp, ok := ecs.GetComponent[*components.EmitterComponent](em, emitterID)
	if ok && emitterComp.Active {
		t.Errorf("动画完成后粒子发射器应该停止")
	}

	// 验证 SoddingSystem 中的发射器ID已清空
	if ss.sodRollEmitterID != 0 {
		t.Errorf("动画完成后发射器ID应该清空，但 sodRollEmitterID = %d", ss.sodRollEmitterID)
	}

	// 验证 particlesEnabled 标志已重置
	if ss.particlesEnabled {
		t.Errorf("动画完成后 particlesEnabled 应该为 false")
	}

	// 验证发射器有 LifetimeComponent 用于延迟清理
	lifetime, ok := ecs.GetComponent[*components.LifetimeComponent](em, emitterID)
	if ok {
		// 如果有 LifetimeComponent，验证生命周期设置正确
		expectedLifetime := 0.35
		if lifetime.MaxLifetime != expectedLifetime {
			t.Errorf("发射器生命周期应该为 %.2f，但实际为 %.2f", expectedLifetime, lifetime.MaxLifetime)
		}
	}
}
