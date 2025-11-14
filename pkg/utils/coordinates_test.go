package utils

import (
	"errors"
	"testing"

	"github.com/decker502/pvz/internal/reanim"
	"github.com/decker502/pvz/pkg/components"
	"github.com/decker502/pvz/pkg/ecs"
)

// TestGetRenderScreenOrigin 测试 GetRenderScreenOrigin 函数
func TestGetRenderScreenOrigin(t *testing.T) {
	tests := []struct {
		name        string
		posX, posY  float64
		centerOffX  float64
		centerOffY  float64
		cameraX     float64
		isUI        bool
		hasReanim   bool
		wantScreenX float64
		wantScreenY float64
		wantErr     error
	}{
		{
			name:        "游戏实体-有动画组件",
			posX:        100,
			posY:        200,
			centerOffX:  30,
			centerOffY:  40,
			cameraX:     50,
			isUI:        false,
			hasReanim:   true,
			wantScreenX: 20,  // 100 - 50 - 30
			wantScreenY: 160, // 200 - 40
			wantErr:     nil,
		},
		{
			name:        "UI元素-不受摄像机影响",
			posX:        100,
			posY:        200,
			centerOffX:  30,
			centerOffY:  40,
			cameraX:     50,
			isUI:        true,
			hasReanim:   true,
			wantScreenX: 70,  // 100 - 0 - 30
			wantScreenY: 160, // 200 - 40
			wantErr:     nil,
		},
		{
			name:        "无ReanimComponent-返回错误",
			posX:        100,
			posY:        200,
			cameraX:     50,
			isUI:        false,
			hasReanim:   false,
			wantScreenX: 0,
			wantScreenY: 0,
			wantErr:     ErrNoReanimComponent,
		},
		{
			name:        "零值坐标",
			posX:        0,
			posY:        0,
			centerOffX:  0,
			centerOffY:  0,
			cameraX:     0,
			isUI:        false,
			hasReanim:   true,
			wantScreenX: 0,
			wantScreenY: 0,
			wantErr:     nil,
		},
		{
			name:        "负值坐标",
			posX:        -100,
			posY:        -200,
			centerOffX:  -30,
			centerOffY:  -40,
			cameraX:     -50,
			isUI:        false,
			hasReanim:   true,
			wantScreenX: -20,  // -100 - (-50) - (-30) = -100 + 50 + 30
			wantScreenY: -160, // -200 - (-40) = -200 + 40
			wantErr:     nil,
		},
		{
			name:        "大数值坐标",
			posX:        10000.5,
			posY:        20000.75,
			centerOffX:  123.45,
			centerOffY:  234.56,
			cameraX:     5000.25,
			isUI:        false,
			hasReanim:   true,
			wantScreenX: 4876.8,   // 10000.5 - 5000.25 - 123.45
			wantScreenY: 19766.19, // 20000.75 - 234.56
			wantErr:     nil,
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
					CenterOffsetX: tt.centerOffX,
					CenterOffsetY: tt.centerOffY,
				}
				ecs.AddComponent(em, entityID, reanimComp)
			}

			// 如果是 UI 元素，添加 UIComponent
			if tt.isUI {
				ecs.AddComponent(em, entityID, &components.UIComponent{})
			}

			// 调用函数
			gotScreenX, gotScreenY, gotErr := GetRenderScreenOrigin(em, entityID, pos, tt.cameraX)

			// 验证错误
			if !errors.Is(gotErr, tt.wantErr) {
				t.Errorf("GetRenderScreenOrigin() error = %v, wantErr %v", gotErr, tt.wantErr)
				return
			}

			// 如果预期有错误，跳过坐标验证
			if tt.wantErr != nil {
				return
			}

			// 验证屏幕坐标（使用容差比较浮点数）
			const epsilon = 0.01
			if diff := gotScreenX - tt.wantScreenX; diff < -epsilon || diff > epsilon {
				t.Errorf("GetRenderScreenOrigin() screenX = %v, want %v", gotScreenX, tt.wantScreenX)
			}
			if diff := gotScreenY - tt.wantScreenY; diff < -epsilon || diff > epsilon {
				t.Errorf("GetRenderScreenOrigin() screenY = %v, want %v", gotScreenY, tt.wantScreenY)
			}
		})
	}
}

// TestGetClickableCenter 测试 GetClickableCenter 函数
func TestGetClickableCenter(t *testing.T) {
	tests := []struct {
		name        string
		posX, posY  float64
		centerOffX  float64
		centerOffY  float64
		hasReanim   bool
		wantCenterX float64
		wantCenterY float64
		wantErr     error
	}{
		{
			name:        "阳光实体-有动画组件",
			posX:        150,
			posY:        250,
			centerOffX:  25,
			centerOffY:  30,
			hasReanim:   true,
			wantCenterX: 125, // 150 - 25
			wantCenterY: 220, // 250 - 30
			wantErr:     nil,
		},
		{
			name:        "无ReanimComponent-返回错误",
			posX:        150,
			posY:        250,
			hasReanim:   false,
			wantCenterX: 0,
			wantCenterY: 0,
			wantErr:     ErrNoReanimComponent,
		},
		{
			name:        "零值CenterOffset",
			posX:        100,
			posY:        200,
			centerOffX:  0,
			centerOffY:  0,
			hasReanim:   true,
			wantCenterX: 100,
			wantCenterY: 200,
			wantErr:     nil,
		},
		{
			name:        "负值CenterOffset",
			posX:        100,
			posY:        200,
			centerOffX:  -20,
			centerOffY:  -30,
			hasReanim:   true,
			wantCenterX: 120, // 100 - (-20)
			wantCenterY: 230, // 200 - (-30)
			wantErr:     nil,
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
					ReanimName:    "Sun",
					ReanimXML:     &reanim.ReanimXML{},
					CenterOffsetX: tt.centerOffX,
					CenterOffsetY: tt.centerOffY,
				}
				ecs.AddComponent(em, entityID, reanimComp)
			}

			// 调用函数
			gotCenterX, gotCenterY, gotErr := GetClickableCenter(em, entityID, pos)

			// 验证错误
			if !errors.Is(gotErr, tt.wantErr) {
				t.Errorf("GetClickableCenter() error = %v, wantErr %v", gotErr, tt.wantErr)
				return
			}

			// 如果预期有错误，跳过坐标验证
			if tt.wantErr != nil {
				return
			}

			// 验证中心坐标
			const epsilon = 0.01
			if diff := gotCenterX - tt.wantCenterX; diff < -epsilon || diff > epsilon {
				t.Errorf("GetClickableCenter() centerX = %v, want %v", gotCenterX, tt.wantCenterX)
			}
			if diff := gotCenterY - tt.wantCenterY; diff < -epsilon || diff > epsilon {
				t.Errorf("GetClickableCenter() centerY = %v, want %v", gotCenterY, tt.wantCenterY)
			}
		})
	}
}

// TestGetRenderOrigin 测试 GetRenderOrigin 函数
func TestGetRenderOrigin(t *testing.T) {
	tests := []struct {
		name        string
		posX, posY  float64
		centerOffX  float64
		centerOffY  float64
		hasReanim   bool
		wantOriginX float64
		wantOriginY float64
		wantErr     error
	}{
		{
			name:        "植物实体-有动画组件",
			posX:        300,
			posY:        400,
			centerOffX:  50,
			centerOffY:  60,
			hasReanim:   true,
			wantOriginX: 250, // 300 - 50
			wantOriginY: 340, // 400 - 60
			wantErr:     nil,
		},
		{
			name:        "无ReanimComponent-返回错误",
			posX:        300,
			posY:        400,
			hasReanim:   false,
			wantOriginX: 0,
			wantOriginY: 0,
			wantErr:     ErrNoReanimComponent,
		},
		{
			name:        "零值坐标和偏移",
			posX:        0,
			posY:        0,
			centerOffX:  0,
			centerOffY:  0,
			hasReanim:   true,
			wantOriginX: 0,
			wantOriginY: 0,
			wantErr:     nil,
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
					ReanimName:    "Peashooter",
					ReanimXML:     &reanim.ReanimXML{},
					CenterOffsetX: tt.centerOffX,
					CenterOffsetY: tt.centerOffY,
				}
				ecs.AddComponent(em, entityID, reanimComp)
			}

			// 调用函数
			gotOriginX, gotOriginY, gotErr := GetRenderOrigin(em, entityID, pos)

			// 验证错误
			if !errors.Is(gotErr, tt.wantErr) {
				t.Errorf("GetRenderOrigin() error = %v, wantErr %v", gotErr, tt.wantErr)
				return
			}

			// 如果预期有错误，跳过坐标验证
			if tt.wantErr != nil {
				return
			}

			// 验证原点坐标
			const epsilon = 0.01
			if diff := gotOriginX - tt.wantOriginX; diff < -epsilon || diff > epsilon {
				t.Errorf("GetRenderOrigin() originX = %v, want %v", gotOriginX, tt.wantOriginX)
			}
			if diff := gotOriginY - tt.wantOriginY; diff < -epsilon || diff > epsilon {
				t.Errorf("GetRenderOrigin() originY = %v, want %v", gotOriginY, tt.wantOriginY)
			}
		})
	}
}

// TestReanimLocalToWorld 测试 ReanimLocalToWorld 函数
func TestReanimLocalToWorld(t *testing.T) {
	tests := []struct {
		name       string
		posX, posY float64
		centerOffX float64
		centerOffY float64
		localX     float64
		localY     float64
		hasReanim  bool
		wantWorldX float64
		wantWorldY float64
		wantErr    error
	}{
		{
			name:       "草皮卷坐标转换",
			posX:       500,
			posY:       300,
			centerOffX: 100,
			centerOffY: 80,
			localX:     150,
			localY:     50,
			hasReanim:  true,
			wantWorldX: 550, // 500 - 100 + 150
			wantWorldY: 270, // 300 - 80 + 50
			wantErr:    nil,
		},
		{
			name:       "无ReanimComponent-返回错误",
			posX:       500,
			posY:       300,
			localX:     150,
			localY:     50,
			hasReanim:  false,
			wantWorldX: 0,
			wantWorldY: 0,
			wantErr:    ErrNoReanimComponent,
		},
		{
			name:       "零值局部坐标",
			posX:       200,
			posY:       100,
			centerOffX: 50,
			centerOffY: 30,
			localX:     0,
			localY:     0,
			hasReanim:  true,
			wantWorldX: 150, // 200 - 50 + 0
			wantWorldY: 70,  // 100 - 30 + 0
			wantErr:    nil,
		},
		{
			name:       "负值局部坐标",
			posX:       300,
			posY:       200,
			centerOffX: 40,
			centerOffY: 50,
			localX:     -20,
			localY:     -30,
			hasReanim:  true,
			wantWorldX: 240, // 300 - 40 + (-20)
			wantWorldY: 120, // 200 - 50 + (-30)
			wantErr:    nil,
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
					ReanimName:    "SodRoll",
					ReanimXML:     &reanim.ReanimXML{},
					CenterOffsetX: tt.centerOffX,
					CenterOffsetY: tt.centerOffY,
				}
				ecs.AddComponent(em, entityID, reanimComp)
			}

			// 调用函数
			gotWorldX, gotWorldY, gotErr := ReanimLocalToWorld(em, entityID, pos, tt.localX, tt.localY)

			// 验证错误
			if !errors.Is(gotErr, tt.wantErr) {
				t.Errorf("ReanimLocalToWorld() error = %v, wantErr %v", gotErr, tt.wantErr)
				return
			}

			// 如果预期有错误，跳过坐标验证
			if tt.wantErr != nil {
				return
			}

			// 验证世界坐标
			const epsilon = 0.01
			if diff := gotWorldX - tt.wantWorldX; diff < -epsilon || diff > epsilon {
				t.Errorf("ReanimLocalToWorld() worldX = %v, want %v", gotWorldX, tt.wantWorldX)
			}
			if diff := gotWorldY - tt.wantWorldY; diff < -epsilon || diff > epsilon {
				t.Errorf("ReanimLocalToWorld() worldY = %v, want %v", gotWorldY, tt.wantWorldY)
			}
		})
	}
}

// TestWorldToScreen 测试 WorldToScreen 函数
func TestWorldToScreen(t *testing.T) {
	tests := []struct {
		name        string
		worldX      float64
		worldY      float64
		cameraX     float64
		isUI        bool
		wantScreenX float64
		wantScreenY float64
	}{
		{
			name:        "游戏实体-应用摄像机偏移",
			worldX:      600,
			worldY:      400,
			cameraX:     200,
			isUI:        false,
			wantScreenX: 400, // 600 - 200
			wantScreenY: 400,
		},
		{
			name:        "UI元素-不应用摄像机偏移",
			worldX:      600,
			worldY:      400,
			cameraX:     200,
			isUI:        true,
			wantScreenX: 600, // 600 - 0
			wantScreenY: 400,
		},
		{
			name:        "零值摄像机偏移",
			worldX:      500,
			worldY:      300,
			cameraX:     0,
			isUI:        false,
			wantScreenX: 500,
			wantScreenY: 300,
		},
		{
			name:        "负值摄像机偏移",
			worldX:      500,
			worldY:      300,
			cameraX:     -100,
			isUI:        false,
			wantScreenX: 600, // 500 - (-100)
			wantScreenY: 300,
		},
		{
			name:        "负值世界坐标",
			worldX:      -200,
			worldY:      -150,
			cameraX:     100,
			isUI:        false,
			wantScreenX: -300, // -200 - 100
			wantScreenY: -150,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 调用函数
			gotScreenX, gotScreenY := WorldToScreen(tt.worldX, tt.worldY, tt.cameraX, tt.isUI)

			// 验证屏幕坐标
			const epsilon = 0.01
			if diff := gotScreenX - tt.wantScreenX; diff < -epsilon || diff > epsilon {
				t.Errorf("WorldToScreen() screenX = %v, want %v", gotScreenX, tt.wantScreenX)
			}
			if diff := gotScreenY - tt.wantScreenY; diff < -epsilon || diff > epsilon {
				t.Errorf("WorldToScreen() screenY = %v, want %v", gotScreenY, tt.wantScreenY)
			}
		})
	}
}

// ============================================================================
// 性能基准测试 (Benchmark Tests)
// ============================================================================

// BenchmarkGetRenderScreenOrigin 测试 GetRenderScreenOrigin 函数的性能
func BenchmarkGetRenderScreenOrigin(b *testing.B) {
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

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _ = GetRenderScreenOrigin(em, entityID, pos, cameraX)
	}
}

// BenchmarkGetClickableCenter 测试 GetClickableCenter 函数的性能
func BenchmarkGetClickableCenter(b *testing.B) {
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
		_, _, _ = GetClickableCenter(em, entityID, pos)
	}
}

// BenchmarkGetRenderOrigin 测试 GetRenderOrigin 函数的性能
func BenchmarkGetRenderOrigin(b *testing.B) {
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
		_, _, _ = GetRenderOrigin(em, entityID, pos)
	}
}

// BenchmarkReanimLocalToWorld 测试 ReanimLocalToWorld 函数的性能
func BenchmarkReanimLocalToWorld(b *testing.B) {
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

	localX := 100.0
	localY := 80.0

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _ = ReanimLocalToWorld(em, entityID, pos, localX, localY)
	}
}

// BenchmarkWorldToScreen 测试 WorldToScreen 函数的性能
func BenchmarkWorldToScreen(b *testing.B) {
	worldX := 500.0
	worldY := 300.0
	cameraX := 215.0
	isUI := false

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = WorldToScreen(worldX, worldY, cameraX, isUI)
	}
}

// BenchmarkManualCalculation 测试手工计算坐标的性能（对比基准）
func BenchmarkManualCalculation(b *testing.B) {
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

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// 手工计算（模拟当前系统方式）
		rc, ok := ecs.GetComponent[*components.ReanimComponent](em, entityID)
		if ok {
			_, isUI := ecs.GetComponent[*components.UIComponent](em, entityID)
			effectiveCameraX := cameraX
			if isUI {
				effectiveCameraX = 0
			}
			_ = pos.X - effectiveCameraX - rc.CenterOffsetX
			_ = pos.Y - rc.CenterOffsetY
		}
	}
}
