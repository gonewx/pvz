package utils

import (
	"errors"
	"testing"

	"github.com/gonewx/pvz/internal/reanim"
	"github.com/gonewx/pvz/pkg/components"
	"github.com/gonewx/pvz/pkg/ecs"
)

// TestGetFootPosition 测试 GetFootPosition 函数
func TestGetFootPosition(t *testing.T) {
	tests := []struct {
		name       string
		posX, posY float64
		hasReanim  bool
		wantFootX  float64
		wantFootY  float64
		wantErr    error
	}{
		{
			name:      "有ReanimComponent的植物实体",
			posX:      300,
			posY:      400,
			hasReanim: true,
			wantFootX: 300,
			wantFootY: 400,
			wantErr:   nil,
		},
		{
			name:      "有ReanimComponent的僵尸实体",
			posX:      800,
			posY:      350,
			hasReanim: true,
			wantFootX: 800,
			wantFootY: 350,
			wantErr:   nil,
		},
		{
			name:      "无ReanimComponent的实体-返回错误",
			posX:      500,
			posY:      250,
			hasReanim: false,
			wantFootX: 0,
			wantFootY: 0,
			wantErr:   ErrNoReanimComponent,
		},
		{
			name:      "零值坐标",
			posX:      0,
			posY:      0,
			hasReanim: true,
			wantFootX: 0,
			wantFootY: 0,
			wantErr:   nil,
		},
		{
			name:      "负值坐标",
			posX:      -100,
			posY:      -200,
			hasReanim: true,
			wantFootX: -100,
			wantFootY: -200,
			wantErr:   nil,
		},
		{
			name:      "大数值坐标",
			posX:      10000.5,
			posY:      20000.75,
			hasReanim: true,
			wantFootX: 10000.5,
			wantFootY: 20000.75,
			wantErr:   nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 创建 EntityManager 和实体
			em := ecs.NewEntityManager()
			entityID := em.CreateEntity()

			// 添加 PositionComponent
			pos := &components.PositionComponent{
				X: tt.posX,
				Y: tt.posY,
			}
			ecs.AddComponent(em, entityID, pos)

			// 如果需要 ReanimComponent，则添加
			if tt.hasReanim {
				reanimComp := &components.ReanimComponent{
					ReanimName:    "Test",
					ReanimXML:     &reanim.ReanimXML{},
					CenterOffsetX: 50, // 任意值，不影响脚底位置
					CenterOffsetY: 80, // 任意值，不影响脚底位置
				}
				ecs.AddComponent(em, entityID, reanimComp)
			}

			// 调用函数
			gotFootX, gotFootY, gotErr := GetFootPosition(em, entityID, pos)

			// 验证错误
			if !errors.Is(gotErr, tt.wantErr) {
				t.Errorf("GetFootPosition() error = %v, wantErr %v", gotErr, tt.wantErr)
				return
			}

			// 如果预期有错误，跳过坐标验证
			if tt.wantErr != nil {
				return
			}

			// 验证脚底坐标
			const epsilon = 0.01
			if diff := gotFootX - tt.wantFootX; diff < -epsilon || diff > epsilon {
				t.Errorf("GetFootPosition() footX = %v, want %v", gotFootX, tt.wantFootX)
			}
			if diff := gotFootY - tt.wantFootY; diff < -epsilon || diff > epsilon {
				t.Errorf("GetFootPosition() footY = %v, want %v", gotFootY, tt.wantFootY)
			}
		})
	}
}

// TestGetShadowScreenPosition 测试 GetShadowScreenPosition 函数
func TestGetShadowScreenPosition(t *testing.T) {
	tests := []struct {
		name                 string
		posX, posY           float64
		cameraX              float64
		shadowWidth          float64
		shadowHeight         float64
		offsetX, offsetY     float64
		hasReanim            bool
		wantScreenX          float64
		wantScreenY          float64
		wantErr              error
	}{
		{
			name:         "植物阴影-无装备",
			posX:         300,
			posY:         400,
			cameraX:      100,
			shadowWidth:  50,
			shadowHeight: 25,
			offsetX:      0,
			offsetY:      -8, // config.PlantShadowOffsetY
			hasReanim:    true,
			wantScreenX:  175, // 300 + 0 - 50/2 - 100
			wantScreenY:  379.5, // 400 + (-8) - 25/2
			wantErr:      nil,
		},
		{
			name:         "普通僵尸阴影",
			posX:         800,
			posY:         350,
			cameraX:      100,
			shadowWidth:  60,
			shadowHeight: 30,
			offsetX:      10,  // config.ZombieShadowOffsetX
			offsetY:      -8, // config.ZombieShadowOffsetY
			hasReanim:    true,
			wantScreenX:  680, // 800 + 10 - 60/2 - 100
			wantScreenY:  327, // 350 + (-8) - 30/2
			wantErr:      nil,
		},
		{
			name:         "撑杆僵尸阴影-无需特殊补丁",
			posX:         750,
			posY:         350,
			cameraX:      100,
			shadowWidth:  65,
			shadowHeight: 32,
			offsetX:      10,  // config.ZombieShadowOffsetX
			offsetY:      -8, // config.ZombieShadowOffsetY（不需要特殊补丁）
			hasReanim:    true,
			wantScreenX:  627.5, // 750 + 10 - 65/2 - 100
			wantScreenY:  326, // 350 + (-8) - 32/2
			wantErr:      nil,
		},
		{
			name:         "路障僵尸阴影",
			posX:         700,
			posY:         350,
			cameraX:      100,
			shadowWidth:  60,
			shadowHeight: 30,
			offsetX:      10,
			offsetY:      -8,
			hasReanim:    true,
			wantScreenX:  580, // 700 + 10 - 60/2 - 100
			wantScreenY:  327, // 350 + (-8) - 30/2
			wantErr:      nil,
		},
		{
			name:         "铁桶僵尸阴影",
			posX:         650,
			posY:         350,
			cameraX:      100,
			shadowWidth:  60,
			shadowHeight: 30,
			offsetX:      10,
			offsetY:      -8,
			hasReanim:    true,
			wantScreenX:  530, // 650 + 10 - 60/2 - 100
			wantScreenY:  327, // 350 + (-8) - 30/2
			wantErr:      nil,
		},
		{
			name:         "UI元素-摄像机偏移为0",
			posX:         100,
			posY:         200,
			cameraX:      0, // UI 元素传入 0
			shadowWidth:  40,
			shadowHeight: 20,
			offsetX:      0,
			offsetY:      0,
			hasReanim:    true,
			wantScreenX:  80, // 100 + 0 - 40/2 - 0
			wantScreenY:  190, // 200 + 0 - 20/2
			wantErr:      nil,
		},
		{
			name:         "零值偏移",
			posX:         500,
			posY:         300,
			cameraX:      200,
			shadowWidth:  50,
			shadowHeight: 25,
			offsetX:      0,
			offsetY:      0,
			hasReanim:    true,
			wantScreenX:  275, // 500 + 0 - 50/2 - 200
			wantScreenY:  287.5, // 300 + 0 - 25/2
			wantErr:      nil,
		},
		{
			name:         "无ReanimComponent-返回错误",
			posX:         500,
			posY:         300,
			cameraX:      200,
			shadowWidth:  50,
			shadowHeight: 25,
			offsetX:      0,
			offsetY:      0,
			hasReanim:    false,
			wantScreenX:  0,
			wantScreenY:  0,
			wantErr:      ErrNoReanimComponent,
		},
		{
			name:         "负值偏移",
			posX:         400,
			posY:         300,
			cameraX:      100,
			shadowWidth:  60,
			shadowHeight: 30,
			offsetX:      -10,
			offsetY:      -20,
			hasReanim:    true,
			wantScreenX:  260, // 400 + (-10) - 60/2 - 100
			wantScreenY:  265, // 300 + (-20) - 30/2
			wantErr:      nil,
		},
		{
			name:         "大数值坐标",
			posX:         10000.5,
			posY:         5000.25,
			cameraX:      3000.0,
			shadowWidth:  100.0,
			shadowHeight: 50.0,
			offsetX:      15.5,
			offsetY:      -12.25,
			hasReanim:    true,
			wantScreenX:  6966.0, // 10000.5 + 15.5 - 100/2 - 3000
			wantScreenY:  4963.0, // 5000.25 + (-12.25) - 50/2
			wantErr:      nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 创建 EntityManager 和实体
			em := ecs.NewEntityManager()
			entityID := em.CreateEntity()

			// 添加 PositionComponent
			pos := &components.PositionComponent{
				X: tt.posX,
				Y: tt.posY,
			}
			ecs.AddComponent(em, entityID, pos)

			// 如果需要 ReanimComponent，则添加
			if tt.hasReanim {
				reanimComp := &components.ReanimComponent{
					ReanimName:    "Test",
					ReanimXML:     &reanim.ReanimXML{},
					CenterOffsetX: 50, // 任意值，不影响阴影位置计算
					CenterOffsetY: 80, // 任意值，不影响阴影位置计算
				}
				ecs.AddComponent(em, entityID, reanimComp)
			}

			// 调用函数
			gotScreenX, gotScreenY, gotErr := GetShadowScreenPosition(
				em, entityID, pos, tt.cameraX,
				tt.shadowWidth, tt.shadowHeight,
				tt.offsetX, tt.offsetY,
			)

			// 验证错误
			if !errors.Is(gotErr, tt.wantErr) {
				t.Errorf("GetShadowScreenPosition() error = %v, wantErr %v", gotErr, tt.wantErr)
				return
			}

			// 如果预期有错误，跳过坐标验证
			if tt.wantErr != nil {
				return
			}

			// 验证屏幕坐标（使用容差比较浮点数）
			const epsilon = 0.01
			if diff := gotScreenX - tt.wantScreenX; diff < -epsilon || diff > epsilon {
				t.Errorf("GetShadowScreenPosition() screenX = %v, want %v", gotScreenX, tt.wantScreenX)
			}
			if diff := gotScreenY - tt.wantScreenY; diff < -epsilon || diff > epsilon {
				t.Errorf("GetShadowScreenPosition() screenY = %v, want %v", gotScreenY, tt.wantScreenY)
			}
		})
	}
}

// ============================================================================
// 性能基准测试 (Benchmark Tests)
// ============================================================================

// BenchmarkGetFootPosition 测试 GetFootPosition 函数的性能
func BenchmarkGetFootPosition(b *testing.B) {
	// 准备测试数据
	em := ecs.NewEntityManager()
	entityID := em.CreateEntity()

	pos := &components.PositionComponent{X: 500, Y: 300}
	ecs.AddComponent(em, entityID, pos)

	reanimComp := &components.ReanimComponent{
		ReanimName:    "Benchmark",
		ReanimXML:     &reanim.ReanimXML{},
		CenterOffsetX: 50,
		CenterOffsetY: 40,
	}
	ecs.AddComponent(em, entityID, reanimComp)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _ = GetFootPosition(em, entityID, pos)
	}
}

// BenchmarkGetShadowScreenPosition 测试 GetShadowScreenPosition 函数的性能
func BenchmarkGetShadowScreenPosition(b *testing.B) {
	// 准备测试数据
	em := ecs.NewEntityManager()
	entityID := em.CreateEntity()

	pos := &components.PositionComponent{X: 500, Y: 300}
	ecs.AddComponent(em, entityID, pos)

	reanimComp := &components.ReanimComponent{
		ReanimName:    "Benchmark",
		ReanimXML:     &reanim.ReanimXML{},
		CenterOffsetX: 50,
		CenterOffsetY: 40,
	}
	ecs.AddComponent(em, entityID, reanimComp)

	cameraX := 215.0
	shadowWidth := 60.0
	shadowHeight := 30.0
	offsetX := 10.0
	offsetY := -8.0

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _ = GetShadowScreenPosition(em, entityID, pos, cameraX, shadowWidth, shadowHeight, offsetX, offsetY)
	}
}

// BenchmarkManualShadowCalculation 测试手工计算阴影坐标的性能（对比基准）
func BenchmarkManualShadowCalculation(b *testing.B) {
	// 准备测试数据
	em := ecs.NewEntityManager()
	entityID := em.CreateEntity()

	pos := &components.PositionComponent{X: 500, Y: 300}
	ecs.AddComponent(em, entityID, pos)

	reanimComp := &components.ReanimComponent{
		ReanimName:    "Benchmark",
		ReanimXML:     &reanim.ReanimXML{},
		CenterOffsetX: 50,
		CenterOffsetY: 40,
	}
	ecs.AddComponent(em, entityID, reanimComp)

	cameraX := 215.0
	shadowWidth := 60.0
	shadowHeight := 30.0
	offsetX := 10.0
	offsetY := -8.0

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// 手工计算（模拟当前 render_system 方式）
		_, ok := ecs.GetComponent[*components.ReanimComponent](em, entityID)
		if ok {
			_ = pos.X + offsetX - shadowWidth/2 - cameraX
			_ = pos.Y + offsetY - shadowHeight/2
		}
	}
}
