package systems

import (
	"testing"

	"github.com/gonewx/pvz/pkg/components"
	"github.com/gonewx/pvz/pkg/ecs"
)

// TestReanimSystem_ProcessAnimationCommands_ComboMode 测试组合模式命令处理
func TestReanimSystem_ProcessAnimationCommands_ComboMode(t *testing.T) {
	// 1. 创建测试环境
	em := ecs.NewEntityManager()
	rs := NewReanimSystem(em)

	// 2. 创建实体并添加 AnimationCommand
	entityID := em.CreateEntity()
	ecs.AddComponent(em, entityID, &components.AnimationCommandComponent{
		UnitID:    "zombie",
		ComboName: "death",
		Processed: false,
	})

	// 3. 执行命令处理
	rs.processAnimationCommands()

	// 4. 验证命令已标记为已处理
	cmd, ok := ecs.GetComponent[*components.AnimationCommandComponent](em, entityID)
	if !ok {
		t.Fatal("AnimationCommand component not found")
	}
	if !cmd.Processed {
		t.Error("Expected Processed to be true after processing")
	}
}

// TestReanimSystem_ProcessAnimationCommands_SingleMode 测试单动画模式
func TestReanimSystem_ProcessAnimationCommands_SingleMode(t *testing.T) {
	// 1. 创建测试环境
	em := ecs.NewEntityManager()
	rs := NewReanimSystem(em)

	// 2. 创建实体并添加 AnimationCommand（单动画模式）
	entityID := em.CreateEntity()
	ecs.AddComponent(em, entityID, &components.AnimationCommandComponent{
		AnimationName: "anim_idle",
		Processed:     false,
	})

	// 3. 执行命令处理
	rs.processAnimationCommands()

	// 4. 验证命令已标记为已处理
	cmd, ok := ecs.GetComponent[*components.AnimationCommandComponent](em, entityID)
	if !ok {
		t.Fatal("AnimationCommand component not found")
	}
	if !cmd.Processed {
		t.Error("Expected Processed to be true after processing")
	}
}

// TestReanimSystem_ProcessAnimationCommands_InvalidCommand 测试无效命令
func TestReanimSystem_ProcessAnimationCommands_InvalidCommand(t *testing.T) {
	em := ecs.NewEntityManager()
	rs := NewReanimSystem(em)

	entityID := em.CreateEntity()
	ecs.AddComponent(em, entityID, &components.AnimationCommandComponent{
		UnitID:        "", // 无效：两个字段都为空
		AnimationName: "",
		Processed:     false,
	})

	// 执行命令处理（应该处理错误但不崩溃）
	rs.processAnimationCommands()

	// 验证：即使失败也应标记 Processed
	cmd, ok := ecs.GetComponent[*components.AnimationCommandComponent](em, entityID)
	if !ok {
		t.Fatal("AnimationCommand component not found")
	}
	if !cmd.Processed {
		t.Error("Expected Processed to be true even on error")
	}
}

// TestReanimSystem_ProcessAnimationCommands_SkipProcessed 测试跳过已处理命令
func TestReanimSystem_ProcessAnimationCommands_SkipProcessed(t *testing.T) {
	em := ecs.NewEntityManager()
	rs := NewReanimSystem(em)

	entityID := em.CreateEntity()
	cmd := &components.AnimationCommandComponent{
		UnitID:    "zombie",
		ComboName: "death",
		Processed: true, // 已处理
	}
	ecs.AddComponent(em, entityID, cmd)

	// 执行命令处理
	rs.processAnimationCommands()

	// 验证：已处理的命令不应再次执行
	// 由于命令已标记为 Processed，processAnimationCommands 应该跳过它
	// 这里我们验证组件仍然存在且 Processed 仍为 true
	cmdAfter, ok := ecs.GetComponent[*components.AnimationCommandComponent](em, entityID)
	if !ok {
		t.Fatal("AnimationCommand component not found")
	}
	if !cmdAfter.Processed {
		t.Error("Expected Processed to remain true")
	}
}

// TestReanimSystem_ProcessAnimationCommands_MultipleCommands 测试批量命令处理
func TestReanimSystem_ProcessAnimationCommands_MultipleCommands(t *testing.T) {
	em := ecs.NewEntityManager()
	rs := NewReanimSystem(em)

	// 创建多个实体，每个都有命令
	entity1 := em.CreateEntity()
	ecs.AddComponent(em, entity1, &components.AnimationCommandComponent{
		UnitID:    "zombie",
		ComboName: "death",
		Processed: false,
	})

	entity2 := em.CreateEntity()
	ecs.AddComponent(em, entity2, &components.AnimationCommandComponent{
		AnimationName: "anim_idle",
		Processed:     false,
	})

	entity3 := em.CreateEntity()
	ecs.AddComponent(em, entity3, &components.AnimationCommandComponent{
		UnitID:    "peashooter",
		ComboName: "attack",
		Processed: false,
	})

	// 执行命令处理
	rs.processAnimationCommands()

	// 验证所有命令都已标记为已处理
	cmd1, ok := ecs.GetComponent[*components.AnimationCommandComponent](em, entity1)
	if !ok || !cmd1.Processed {
		t.Error("Expected entity1 command to be processed")
	}

	cmd2, ok := ecs.GetComponent[*components.AnimationCommandComponent](em, entity2)
	if !ok || !cmd2.Processed {
		t.Error("Expected entity2 command to be processed")
	}

	cmd3, ok := ecs.GetComponent[*components.AnimationCommandComponent](em, entity3)
	if !ok || !cmd3.Processed {
		t.Error("Expected entity3 command to be processed")
	}
}

// TestReanimSystem_CleanupProcessedCommands 测试清理机制
func TestReanimSystem_CleanupProcessedCommands(t *testing.T) {
	em := ecs.NewEntityManager()
	rs := NewReanimSystem(em)
	rs.SetCommandCleanup(true, 0.5) // 启用清理，0.5 秒间隔

	// 创建已处理的命令
	entityID := em.CreateEntity()
	ecs.AddComponent(em, entityID, &components.AnimationCommandComponent{
		UnitID:    "zombie",
		ComboName: "death",
		Processed: true,
	})

	// 模拟时间推进（超过清理间隔）
	rs.cleanupProcessedCommands(0.6)

	// 验证：组件已被移除
	_, exists := ecs.GetComponent[*components.AnimationCommandComponent](em, entityID)
	if exists {
		t.Error("Expected processed command to be removed")
	}
}

// TestReanimSystem_CleanupProcessedCommands_Disabled 测试禁用清理
func TestReanimSystem_CleanupProcessedCommands_Disabled(t *testing.T) {
	em := ecs.NewEntityManager()
	rs := NewReanimSystem(em)
	// 默认清理是禁用的

	// 创建已处理的命令
	entityID := em.CreateEntity()
	ecs.AddComponent(em, entityID, &components.AnimationCommandComponent{
		UnitID:    "zombie",
		ComboName: "death",
		Processed: true,
	})

	// 模拟时间推进
	rs.cleanupProcessedCommands(1.0)

	// 验证：组件仍然存在（因为清理被禁用）
	_, exists := ecs.GetComponent[*components.AnimationCommandComponent](em, entityID)
	if !exists {
		t.Error("Expected command to remain when cleanup is disabled")
	}
}

// TestReanimSystem_CleanupProcessedCommands_IntervalNotReached 测试清理间隔未到
func TestReanimSystem_CleanupProcessedCommands_IntervalNotReached(t *testing.T) {
	em := ecs.NewEntityManager()
	rs := NewReanimSystem(em)
	rs.SetCommandCleanup(true, 1.0) // 启用清理，1 秒间隔

	// 创建已处理的命令
	entityID := em.CreateEntity()
	ecs.AddComponent(em, entityID, &components.AnimationCommandComponent{
		UnitID:    "zombie",
		ComboName: "death",
		Processed: true,
	})

	// 模拟时间推进（未超过清理间隔）
	rs.cleanupProcessedCommands(0.5)

	// 验证：组件仍然存在（因为间隔未到）
	_, exists := ecs.GetComponent[*components.AnimationCommandComponent](em, entityID)
	if !exists {
		t.Error("Expected command to remain when cleanup interval not reached")
	}
}

// TestReanimSystem_CleanupProcessedCommands_OnlyRemovesProcessed 测试只移除已处理的命令
func TestReanimSystem_CleanupProcessedCommands_OnlyRemovesProcessed(t *testing.T) {
	em := ecs.NewEntityManager()
	rs := NewReanimSystem(em)
	rs.SetCommandCleanup(true, 0.5)

	// 创建已处理的命令
	processedEntity := em.CreateEntity()
	ecs.AddComponent(em, processedEntity, &components.AnimationCommandComponent{
		UnitID:    "zombie",
		ComboName: "death",
		Processed: true,
	})

	// 创建未处理的命令
	unprocessedEntity := em.CreateEntity()
	ecs.AddComponent(em, unprocessedEntity, &components.AnimationCommandComponent{
		UnitID:    "peashooter",
		ComboName: "attack",
		Processed: false,
	})

	// 执行清理
	rs.cleanupProcessedCommands(0.6)

	// 验证：已处理的命令被移除
	_, existsProcessed := ecs.GetComponent[*components.AnimationCommandComponent](em, processedEntity)
	if existsProcessed {
		t.Error("Expected processed command to be removed")
	}

	// 验证：未处理的命令仍然存在
	_, existsUnprocessed := ecs.GetComponent[*components.AnimationCommandComponent](em, unprocessedEntity)
	if !existsUnprocessed {
		t.Error("Expected unprocessed command to remain")
	}
}
