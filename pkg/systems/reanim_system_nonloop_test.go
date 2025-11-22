package systems

import (
	"testing"

	"github.com/decker502/pvz/internal/reanim"
	"github.com/decker502/pvz/pkg/components"
	"github.com/decker502/pvz/pkg/ecs"
)

// TestNonLoopingAnimation_Completion 测试非循环动画完成后设置 IsFinished 标志
func TestNonLoopingAnimation_Completion(t *testing.T) {
	// 创建实体管理器
	em := ecs.NewEntityManager()

	// 创建 ReanimSystem
	rs := NewReanimSystem(em)

	// 创建测试实体
	entity := em.CreateEntity()

	// 添加 ReanimComponent（模拟一个简单的非循环动画）
	// 注意：必须设置 ReanimXML（哪怕是空结构）来避免被 Update 跳过
	comp := &components.ReanimComponent{
		ReanimXML: &reanim.ReanimXML{
			FPS: 12, // 设置 FPS
		},
		CurrentAnimations: []string{"test_anim"},
		AnimVisiblesMap: map[string][]int{
			"test_anim": {0, 0, 0}, // 3 个可见帧 (visible=0 表示可见)
		},
		AnimationFPS:     12.0,
		IsLooping:        false, // 非循环
		IsFinished:       false,
		CurrentFrame:     0,
		FrameAccumulator: 0.0,
	}
	ecs.AddComponent(em, entity, comp)

	// 模拟帧更新，直到动画完成
	// 计算需要的更新次数：每帧需要 60/12 = 5 次更新
	// 3 帧 * 5 次/帧 = 15 次更新
	maxUpdates := 20 // 留一些余量

	for i := 0; i < maxUpdates; i++ {
		rs.Update(1.0 / 60.0) // deltaTime = 1/60 秒
		comp, _ = ecs.GetComponent[*components.ReanimComponent](em, entity)

		// 检查是否完成
		if comp.IsFinished {
			t.Logf("动画在第 %d 次更新后完成，CurrentFrame=%d", i+1, comp.CurrentFrame)
			// 验证帧数
			// 修复后：CurrentFrame 会达到 visibleCount (3)，表示动画已完成
			// 渲染系统会将 3 映射到最后一帧 (2)
			if comp.CurrentFrame < 2 {
				t.Errorf("期望 CurrentFrame >= 2（最后一帧），实际=%d", comp.CurrentFrame)
			}
			return
		}
	}

	// 如果循环结束还没完成，测试失败
	t.Errorf("动画应该在 %d 次更新后完成，但 IsFinished 仍为 false", maxUpdates)
}

// TestLoopingAnimation_NeverFinishes 测试循环动画永不设置 IsFinished 标志
func TestLoopingAnimation_NeverFinishes(t *testing.T) {
	// 创建实体管理器
	em := ecs.NewEntityManager()

	// 创建 ReanimSystem
	rs := NewReanimSystem(em)

	// 创建测试实体
	entity := em.CreateEntity()

	// 添加 ReanimComponent（模拟一个简单的循环动画）
	// 注意：必须设置 ReanimXML（哪怕是空结构）来避免被 Update 跳过
	comp := &components.ReanimComponent{
		ReanimXML: &reanim.ReanimXML{
			FPS: 12, // 设置 FPS
		},
		CurrentAnimations: []string{"test_anim"},
		AnimVisiblesMap: map[string][]int{
			"test_anim": {0, 0, 0}, // 3 个可见帧
		},
		AnimationFPS:     12.0,
		IsLooping:        true, // 循环
		IsFinished:       false,
		CurrentFrame:     0,
		FrameAccumulator: 0.0,
	}
	ecs.AddComponent(em, entity, comp)

	// 模拟大量帧更新（超过一个完整循环）
	for i := 0; i < 30; i++ {
		rs.Update(1.0 / 60.0)
		comp, _ = ecs.GetComponent[*components.ReanimComponent](em, entity)

		// 循环动画永远不应该设置 IsFinished
		if comp.IsFinished {
			t.Errorf("循环动画不应该设置 IsFinished=true，但在第 %d 次更新后设置了", i+1)
			return
		}
	}

	t.Logf("循环动画正确运行 30 次更新，IsFinished 保持为 false")
}
