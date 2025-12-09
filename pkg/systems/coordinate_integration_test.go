package systems

import (
	"math"
	"testing"

	"github.com/gonewx/pvz/pkg/components"
	"github.com/gonewx/pvz/pkg/ecs"
	"github.com/gonewx/pvz/pkg/utils"
)

// TestRenderSystemCoordinateIntegration 测试渲染系统使用坐标转换工具库的正确性
func TestRenderSystemCoordinateIntegration(t *testing.T) {
	tests := []struct {
		name          string
		posX          float64
		posY          float64
		centerOffsetX float64
		centerOffsetY float64
		cameraX       float64
		isUI          bool
		expectedX     float64
		expectedY     float64
	}{
		{
			name:          "游戏实体（应用摄像机偏移）",
			posX:          100.0,
			posY:          200.0,
			centerOffsetX: 30.0,
			centerOffsetY: 40.0,
			cameraX:       50.0,
			isUI:          false,
			expectedX:     20.0,  // 100 - 50 - 30
			expectedY:     160.0, // 200 - 40
		},
		{
			name:          "UI元素（不应用摄像机偏移）",
			posX:          100.0,
			posY:          200.0,
			centerOffsetX: 30.0,
			centerOffsetY: 40.0,
			cameraX:       50.0,
			isUI:          true,
			expectedX:     70.0,  // 100 - 0 - 30
			expectedY:     160.0, // 200 - 40
		},
		{
			name:          "摄像机在原点",
			posX:          150.0,
			posY:          250.0,
			centerOffsetX: 25.0,
			centerOffsetY: 35.0,
			cameraX:       0.0,
			isUI:          false,
			expectedX:     125.0, // 150 - 0 - 25
			expectedY:     215.0, // 250 - 35
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 创建测试 EntityManager
			em := ecs.NewEntityManager()

			// 创建测试实体
			entityID := em.CreateEntity()
			ecs.AddComponent(em, entityID, &components.PositionComponent{
				X: tt.posX,
				Y: tt.posY,
			})
			ecs.AddComponent(em, entityID, &components.ReanimComponent{
				CenterOffsetX: tt.centerOffsetX,
				CenterOffsetY: tt.centerOffsetY,
			})

			// 如果是 UI 元素，添加 UIComponent
			if tt.isUI {
				ecs.AddComponent(em, entityID, &components.UIComponent{})
			}

			// 获取 PositionComponent
			pos, _ := ecs.GetComponent[*components.PositionComponent](em, entityID)

			// 调用工具库函数
			screenX, screenY, err := utils.GetRenderScreenOrigin(em, entityID, pos, tt.cameraX)

			// 验证结果
			if err != nil {
				t.Fatalf("Expected no error, got %v", err)
			}

			if math.Abs(screenX-tt.expectedX) > 0.01 {
				t.Errorf("Expected screenX=%.2f, got %.2f", tt.expectedX, screenX)
			}

			if math.Abs(screenY-tt.expectedY) > 0.01 {
				t.Errorf("Expected screenY=%.2f, got %.2f", tt.expectedY, screenY)
			}
		})
	}
}

// TestRenderSystemCoordinateIntegration_NoReanimComponent 测试没有 ReanimComponent 的实体
func TestRenderSystemCoordinateIntegration_NoReanimComponent(t *testing.T) {
	em := ecs.NewEntityManager()
	entityID := em.CreateEntity()
	ecs.AddComponent(em, entityID, &components.PositionComponent{X: 100.0, Y: 200.0})

	pos, _ := ecs.GetComponent[*components.PositionComponent](em, entityID)
	_, _, err := utils.GetRenderScreenOrigin(em, entityID, pos, 50.0)

	if err == nil {
		t.Error("Expected error for entity without ReanimComponent, got nil")
	}
}

// TestInputSystemCoordinateIntegration 测试点击检测系统使用坐标转换工具库的正确性
func TestInputSystemCoordinateIntegration(t *testing.T) {
	tests := []struct {
		name          string
		posX          float64
		posY          float64
		centerOffsetX float64
		centerOffsetY float64
		expectedX     float64
		expectedY     float64
	}{
		{
			name:          "阳光实体（有ReanimComponent）",
			posX:          100.0,
			posY:          200.0,
			centerOffsetX: 44.2,
			centerOffsetY: 19.4,
			expectedX:     55.8,  // 100 - 44.2
			expectedY:     180.6, // 200 - 19.4
		},
		{
			name:          "零偏移",
			posX:          150.0,
			posY:          250.0,
			centerOffsetX: 0.0,
			centerOffsetY: 0.0,
			expectedX:     150.0, // 150 - 0
			expectedY:     250.0, // 250 - 0
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 创建测试 EntityManager
			em := ecs.NewEntityManager()

			// 创建测试实体
			entityID := em.CreateEntity()
			ecs.AddComponent(em, entityID, &components.PositionComponent{
				X: tt.posX,
				Y: tt.posY,
			})
			ecs.AddComponent(em, entityID, &components.ReanimComponent{
				CenterOffsetX: tt.centerOffsetX,
				CenterOffsetY: tt.centerOffsetY,
			})

			// 获取 PositionComponent
			pos, _ := ecs.GetComponent[*components.PositionComponent](em, entityID)

			// 调用工具库函数
			clickCenterX, clickCenterY, err := utils.GetClickableCenter(em, entityID, pos)

			// 验证结果
			if err != nil {
				t.Fatalf("Expected no error, got %v", err)
			}

			if math.Abs(clickCenterX-tt.expectedX) > 0.01 {
				t.Errorf("Expected clickCenterX=%.2f, got %.2f", tt.expectedX, clickCenterX)
			}

			if math.Abs(clickCenterY-tt.expectedY) > 0.01 {
				t.Errorf("Expected clickCenterY=%.2f, got %.2f", tt.expectedY, clickCenterY)
			}
		})
	}
}

// TestInputSystemCoordinateIntegration_NoReanimComponent 测试没有 ReanimComponent 的实体
func TestInputSystemCoordinateIntegration_NoReanimComponent(t *testing.T) {
	em := ecs.NewEntityManager()
	entityID := em.CreateEntity()
	ecs.AddComponent(em, entityID, &components.PositionComponent{X: 100.0, Y: 200.0})

	pos, _ := ecs.GetComponent[*components.PositionComponent](em, entityID)
	_, _, err := utils.GetClickableCenter(em, entityID, pos)

	if err == nil {
		t.Error("Expected error for entity without ReanimComponent, got nil")
	}
}

// TestSoddingSystemCoordinateIntegration 测试草皮系统使用坐标转换工具库的正确性
func TestSoddingSystemCoordinateIntegration(t *testing.T) {
	tests := []struct {
		name          string
		posX          float64
		posY          float64
		centerOffsetX float64
		centerOffsetY float64
		localX        float64
		localY        float64
		expectedX     float64
		expectedY     float64
	}{
		{
			name:          "草皮卷左边缘",
			posX:          200.0,
			posY:          300.0,
			centerOffsetX: 50.0,
			centerOffsetY: 60.0,
			localX:        -34.0, // 左边缘
			localY:        0.0,
			expectedX:     116.0, // 200 - 50 + (-34)
			expectedY:     240.0, // 300 - 60 + 0
		},
		{
			name:          "草皮卷中心",
			posX:          200.0,
			posY:          300.0,
			centerOffsetX: 50.0,
			centerOffsetY: 60.0,
			localX:        0.0, // 中心
			localY:        0.0,
			expectedX:     150.0, // 200 - 50 + 0
			expectedY:     240.0, // 300 - 60 + 0
		},
		{
			name:          "草皮卷右边缘",
			posX:          200.0,
			posY:          300.0,
			centerOffsetX: 50.0,
			centerOffsetY: 60.0,
			localX:        34.0, // 右边缘
			localY:        0.0,
			expectedX:     184.0, // 200 - 50 + 34
			expectedY:     240.0, // 300 - 60 + 0
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 创建测试 EntityManager
			em := ecs.NewEntityManager()

			// 创建测试实体
			entityID := em.CreateEntity()
			ecs.AddComponent(em, entityID, &components.PositionComponent{
				X: tt.posX,
				Y: tt.posY,
			})
			ecs.AddComponent(em, entityID, &components.ReanimComponent{
				CenterOffsetX: tt.centerOffsetX,
				CenterOffsetY: tt.centerOffsetY,
			})

			// 获取 PositionComponent
			pos, _ := ecs.GetComponent[*components.PositionComponent](em, entityID)

			// 调用工具库函数
			worldX, worldY, err := utils.ReanimLocalToWorld(em, entityID, pos, tt.localX, tt.localY)

			// 验证结果
			if err != nil {
				t.Fatalf("Expected no error, got %v", err)
			}

			if math.Abs(worldX-tt.expectedX) > 0.01 {
				t.Errorf("Expected worldX=%.2f, got %.2f", tt.expectedX, worldX)
			}

			if math.Abs(worldY-tt.expectedY) > 0.01 {
				t.Errorf("Expected worldY=%.2f, got %.2f", tt.expectedY, worldY)
			}
		})
	}
}

// TestSoddingSystemCoordinateIntegration_NoReanimComponent 测试没有 ReanimComponent 的实体
func TestSoddingSystemCoordinateIntegration_NoReanimComponent(t *testing.T) {
	em := ecs.NewEntityManager()
	entityID := em.CreateEntity()
	ecs.AddComponent(em, entityID, &components.PositionComponent{X: 100.0, Y: 200.0})

	pos, _ := ecs.GetComponent[*components.PositionComponent](em, entityID)
	_, _, err := utils.ReanimLocalToWorld(em, entityID, pos, 10.0, 20.0)

	if err == nil {
		t.Error("Expected error for entity without ReanimComponent, got nil")
	}
}
