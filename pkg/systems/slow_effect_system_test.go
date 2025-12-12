package systems

import (
	"testing"

	"github.com/gonewx/pvz/pkg/components"
	"github.com/gonewx/pvz/pkg/ecs"
)

func TestSlowEffectSystem_ApplyAndRecover(t *testing.T) {
	em := ecs.NewEntityManager()
	system := NewSlowEffectSystem(em)

	// 创建测试实体
	entityID := em.CreateEntity()
	em.AddComponent(entityID, &components.VelocityComponent{
		VX: -4.7, // 普通僵尸速度
		VY: 0,
	})

	// 应用减速效果
	ApplySlowEffect(em, entityID, 0.5, 2.0) // 减速 50%，持续 2 秒

	// 验证减速组件已添加
	slowed, ok := ecs.GetComponent[*components.SlowedComponent](em, entityID)
	if !ok {
		t.Fatal("SlowedComponent 应该被添加")
	}
	if slowed.SpeedMultiplier != 0.5 {
		t.Errorf("SpeedMultiplier 应为 0.5，实际为 %f", slowed.SpeedMultiplier)
	}
	if slowed.Applied {
		t.Error("减速效果不应该在添加时立即应用")
	}

	// 第一次更新，减速效果应该被应用
	system.Update(0.1)

	velocity, _ := ecs.GetComponent[*components.VelocityComponent](em, entityID)
	if velocity.VX != -2.35 { // -4.7 * 0.5
		t.Errorf("减速后速度应为 -2.35，实际为 %f", velocity.VX)
	}

	slowed, _ = ecs.GetComponent[*components.SlowedComponent](em, entityID)
	if !slowed.Applied {
		t.Error("减速效果应该已被应用")
	}
	if slowed.OriginalVX != -4.7 {
		t.Errorf("原始速度应为 -4.7，实际为 %f", slowed.OriginalVX)
	}

	// 模拟时间流逝，减速效果结束
	system.Update(2.0)

	// 验证减速组件已移除
	_, ok = ecs.GetComponent[*components.SlowedComponent](em, entityID)
	if ok {
		t.Error("减速效果结束后 SlowedComponent 应被移除")
	}

	// 验证速度恢复
	velocity, _ = ecs.GetComponent[*components.VelocityComponent](em, entityID)
	if velocity.VX != -4.7 {
		t.Errorf("减速效果结束后速度应恢复至 -4.7，实际为 %f", velocity.VX)
	}
}

func TestSlowEffectSystem_RefreshDuration(t *testing.T) {
	em := ecs.NewEntityManager()
	system := NewSlowEffectSystem(em)

	// 创建测试实体
	entityID := em.CreateEntity()
	em.AddComponent(entityID, &components.VelocityComponent{
		VX: -4.7,
		VY: 0,
	})

	// 首次应用减速效果
	ApplySlowEffect(em, entityID, 0.5, 2.0)
	system.Update(0.1) // 应用减速

	// 模拟时间流逝 1.5 秒
	system.Update(1.5)

	// 再次应用减速效果（应刷新持续时间）
	ApplySlowEffect(em, entityID, 0.5, 2.0)

	slowed, ok := ecs.GetComponent[*components.SlowedComponent](em, entityID)
	if !ok {
		t.Fatal("SlowedComponent 应该存在")
	}
	if slowed.Duration != 2.0 {
		t.Errorf("持续时间应被刷新为 2.0，实际为 %f", slowed.Duration)
	}

	// 速度不应该被再次降低
	velocity, _ := ecs.GetComponent[*components.VelocityComponent](em, entityID)
	if velocity.VX != -2.35 {
		t.Errorf("速度应保持在 -2.35（不应重复应用减速），实际为 %f", velocity.VX)
	}
}

func TestSlowEffectSystem_NoVelocityComponent(t *testing.T) {
	em := ecs.NewEntityManager()
	system := NewSlowEffectSystem(em)

	// 创建没有 VelocityComponent 的实体
	entityID := em.CreateEntity()
	em.AddComponent(entityID, &components.SlowedComponent{
		SpeedMultiplier: 0.5,
		Duration:        2.0,
	})

	// 更新系统不应崩溃
	system.Update(0.1)
}

func TestSlowedComponentConstants(t *testing.T) {
	if components.DefaultSlowSpeedMultiplier != 0.5 {
		t.Errorf("DefaultSlowSpeedMultiplier 应为 0.5，实际为 %f", components.DefaultSlowSpeedMultiplier)
	}
	if components.DefaultSlowDuration != 10.0 {
		t.Errorf("DefaultSlowDuration 应为 10.0，实际为 %f", components.DefaultSlowDuration)
	}
}
