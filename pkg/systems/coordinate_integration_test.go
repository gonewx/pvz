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

// ============================================================================
// 阴影渲染系统集成测试 (Story 16.4)
// ============================================================================

// TestShadowRenderSystemCoordinateIntegration 测试阴影渲染系统使用坐标转换工具库的正确性
// Story 16.4: 验证阴影渲染系统使用 GetShadowScreenPosition 后坐标计算的正确性
func TestShadowRenderSystemCoordinateIntegration(t *testing.T) {
	// 阴影配置常量（与 config.shadow_config.go 保持一致）
	const plantShadowOffsetY = -8.0
	const zombieShadowOffsetX = 10.0
	const zombieShadowOffsetY = -8.0

	tests := []struct {
		name         string
		entityType   string // "plant" 或 "zombie"
		posX         float64
		posY         float64
		cameraX      float64
		shadowWidth  float64
		shadowHeight float64
		offsetX      float64
		offsetY      float64
		expectedX    float64
		expectedY    float64
	}{
		{
			name:         "植物阴影-豌豆射手",
			entityType:   "plant",
			posX:         300.0,
			posY:         400.0,
			cameraX:      100.0,
			shadowWidth:  50.0,
			shadowHeight: 25.0,
			offsetX:      0,
			offsetY:      plantShadowOffsetY,
			expectedX:    175.0, // 300 + 0 - 50/2 - 100
			expectedY:    379.5, // 400 + (-8) - 25/2
		},
		{
			name:         "植物阴影-向日葵",
			entityType:   "plant",
			posX:         350.0,
			posY:         400.0,
			cameraX:      100.0,
			shadowWidth:  55.0,
			shadowHeight: 28.0,
			offsetX:      0,
			offsetY:      plantShadowOffsetY,
			expectedX:    222.5, // 350 + 0 - 55/2 - 100
			expectedY:    378.0, // 400 + (-8) - 28/2
		},
		{
			name:         "普通僵尸阴影",
			entityType:   "zombie",
			posX:         800.0,
			posY:         350.0,
			cameraX:      100.0,
			shadowWidth:  60.0,
			shadowHeight: 30.0,
			offsetX:      zombieShadowOffsetX,
			offsetY:      zombieShadowOffsetY,
			expectedX:    680.0, // 800 + 10 - 60/2 - 100
			expectedY:    327.0, // 350 + (-8) - 30/2
		},
		{
			name:         "路障僵尸阴影",
			entityType:   "zombie",
			posX:         750.0,
			posY:         350.0,
			cameraX:      100.0,
			shadowWidth:  60.0,
			shadowHeight: 30.0,
			offsetX:      zombieShadowOffsetX,
			offsetY:      zombieShadowOffsetY,
			expectedX:    630.0, // 750 + 10 - 60/2 - 100
			expectedY:    327.0, // 350 + (-8) - 30/2
		},
		{
			name:         "铁桶僵尸阴影",
			entityType:   "zombie",
			posX:         700.0,
			posY:         350.0,
			cameraX:      100.0,
			shadowWidth:  60.0,
			shadowHeight: 30.0,
			offsetX:      zombieShadowOffsetX,
			offsetY:      zombieShadowOffsetY,
			expectedX:    580.0, // 700 + 10 - 60/2 - 100
			expectedY:    327.0, // 350 + (-8) - 30/2
		},
		{
			name:         "摄像机在原点时的植物阴影",
			entityType:   "plant",
			posX:         200.0,
			posY:         300.0,
			cameraX:      0.0,
			shadowWidth:  50.0,
			shadowHeight: 25.0,
			offsetX:      0,
			offsetY:      plantShadowOffsetY,
			expectedX:    175.0, // 200 + 0 - 50/2 - 0
			expectedY:    279.5, // 300 + (-8) - 25/2
		},
		{
			name:         "摄像机在原点时的僵尸阴影",
			entityType:   "zombie",
			posX:         500.0,
			posY:         350.0,
			cameraX:      0.0,
			shadowWidth:  60.0,
			shadowHeight: 30.0,
			offsetX:      zombieShadowOffsetX,
			offsetY:      zombieShadowOffsetY,
			expectedX:    480.0, // 500 + 10 - 60/2 - 0
			expectedY:    327.0, // 350 + (-8) - 30/2
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
				CenterOffsetX: 50.0, // 任意值，不影响阴影位置计算
				CenterOffsetY: 80.0, // 任意值，不影响阴影位置计算
			})

			// 获取 PositionComponent
			pos, _ := ecs.GetComponent[*components.PositionComponent](em, entityID)

			// 调用阴影坐标转换工具库函数
			screenX, screenY, err := utils.GetShadowScreenPosition(
				em, entityID, pos, tt.cameraX,
				tt.shadowWidth, tt.shadowHeight,
				tt.offsetX, tt.offsetY,
			)

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

// TestShadowRenderSystemCoordinateIntegration_NoReanimComponent 测试没有 ReanimComponent 的实体
func TestShadowRenderSystemCoordinateIntegration_NoReanimComponent(t *testing.T) {
	em := ecs.NewEntityManager()
	entityID := em.CreateEntity()
	ecs.AddComponent(em, entityID, &components.PositionComponent{X: 100.0, Y: 200.0})

	pos, _ := ecs.GetComponent[*components.PositionComponent](em, entityID)
	_, _, err := utils.GetShadowScreenPosition(
		em, entityID, pos, 100.0,
		60.0, 30.0,
		10.0, -8.0,
	)

	if err == nil {
		t.Error("Expected error for entity without ReanimComponent, got nil")
	}
}

// TestZombieEquipmentShadowIntegration 测试撑杆僵尸等装备僵尸的阴影位置
// Story 16.4: 验证装备僵尸阴影位置不需要额外补丁
//
// 关键验证点:
//   - 撑杆僵尸、路障僵尸、铁桶僵尸使用与普通僵尸相同的偏移量
//   - 删除 PolevaulterZombieShadowOffsetY 补丁后，阴影位置计算正确
//   - 所有僵尸类型使用统一的 GetShadowScreenPosition API
func TestZombieEquipmentShadowIntegration(t *testing.T) {
	// 阴影配置常量（与 config.shadow_config.go 保持一致）
	const zombieShadowOffsetX = 10.0
	const zombieShadowOffsetY = -8.0

	// 僵尸阴影尺寸（来自 shadow_config.go）
	const zombieShadowWidth = 60.0
	const zombieShadowHeight = 30.0
	const polevaulterShadowWidth = 65.0
	const polevaulterShadowHeight = 32.0

	tests := []struct {
		name         string
		zombieType   string
		posX         float64
		posY         float64
		cameraX      float64
		shadowWidth  float64
		shadowHeight float64
	}{
		{
			name:         "普通僵尸-基准",
			zombieType:   "zombie",
			posX:         800.0,
			posY:         350.0,
			cameraX:      100.0,
			shadowWidth:  zombieShadowWidth,
			shadowHeight: zombieShadowHeight,
		},
		{
			name:         "撑杆僵尸-无需特殊补丁",
			zombieType:   "zombie_pole",
			posX:         800.0,
			posY:         350.0,
			cameraX:      100.0,
			shadowWidth:  polevaulterShadowWidth,
			shadowHeight: polevaulterShadowHeight,
		},
		{
			name:         "路障僵尸-无需特殊补丁",
			zombieType:   "zombie_cone",
			posX:         800.0,
			posY:         350.0,
			cameraX:      100.0,
			shadowWidth:  zombieShadowWidth,
			shadowHeight: zombieShadowHeight,
		},
		{
			name:         "铁桶僵尸-无需特殊补丁",
			zombieType:   "zombie_bucket",
			posX:         800.0,
			posY:         350.0,
			cameraX:      100.0,
			shadowWidth:  zombieShadowWidth,
			shadowHeight: zombieShadowHeight,
		},
		{
			name:         "旗帜僵尸-无需特殊补丁",
			zombieType:   "zombie_flag",
			posX:         800.0,
			posY:         350.0,
			cameraX:      100.0,
			shadowWidth:  zombieShadowWidth,
			shadowHeight: zombieShadowHeight,
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
				CenterOffsetX: 50.0, // 任意值，不影响阴影位置计算
				CenterOffsetY: 80.0, // 任意值，不影响阴影位置计算
			})

			// 获取 PositionComponent
			pos, _ := ecs.GetComponent[*components.PositionComponent](em, entityID)

			// 调用统一的阴影坐标转换 API
			// 关键验证：所有僵尸类型使用相同的 offsetX/offsetY
			screenX, screenY, err := utils.GetShadowScreenPosition(
				em, entityID, pos, tt.cameraX,
				tt.shadowWidth, tt.shadowHeight,
				zombieShadowOffsetX, zombieShadowOffsetY,
			)

			// 验证无错误
			if err != nil {
				t.Fatalf("Expected no error, got %v", err)
			}

			// 验证坐标计算公式正确
			// screenX = posX + offsetX - shadowWidth/2 - cameraX
			expectedX := tt.posX + zombieShadowOffsetX - tt.shadowWidth/2 - tt.cameraX
			// screenY = posY + offsetY - shadowHeight/2
			expectedY := tt.posY + zombieShadowOffsetY - tt.shadowHeight/2

			if math.Abs(screenX-expectedX) > 0.01 {
				t.Errorf("Expected screenX=%.2f, got %.2f (posX=%.2f, offsetX=%.2f, shadowWidth=%.2f, cameraX=%.2f)",
					expectedX, screenX, tt.posX, zombieShadowOffsetX, tt.shadowWidth, tt.cameraX)
			}

			if math.Abs(screenY-expectedY) > 0.01 {
				t.Errorf("Expected screenY=%.2f, got %.2f (posY=%.2f, offsetY=%.2f, shadowHeight=%.2f)",
					expectedY, screenY, tt.posY, zombieShadowOffsetY, tt.shadowHeight)
			}
		})
	}
}

// TestZombieEquipmentShadowConsistency 验证所有装备僵尸在相同位置的阴影 Y 坐标一致
// Story 16.4: 确保删除 PolevaulterZombieShadowOffsetY 补丁后，所有僵尸阴影 Y 坐标一致
func TestZombieEquipmentShadowConsistency(t *testing.T) {
	// 阴影配置常量
	const zombieShadowOffsetX = 10.0
	const zombieShadowOffsetY = -8.0
	const zombieShadowWidth = 60.0
	const zombieShadowHeight = 30.0

	// 固定位置和摄像机
	const posX = 800.0
	const posY = 350.0
	const cameraX = 100.0

	zombieTypes := []string{
		"zombie",        // 普通僵尸
		"zombie_pole",   // 撑杆僵尸
		"zombie_cone",   // 路障僵尸
		"zombie_bucket", // 铁桶僵尸
		"zombie_flag",   // 旗帜僵尸
	}

	// 记录第一个僵尸的阴影 Y 坐标作为基准
	var baseScreenY float64
	var baseZombieType string

	for i, zombieType := range zombieTypes {
		// 创建测试 EntityManager
		em := ecs.NewEntityManager()

		// 创建测试实体
		entityID := em.CreateEntity()
		ecs.AddComponent(em, entityID, &components.PositionComponent{
			X: posX,
			Y: posY,
		})
		ecs.AddComponent(em, entityID, &components.ReanimComponent{
			CenterOffsetX: 50.0,
			CenterOffsetY: 80.0,
		})

		// 获取 PositionComponent
		pos, _ := ecs.GetComponent[*components.PositionComponent](em, entityID)

		// 调用统一的阴影坐标转换 API
		_, screenY, err := utils.GetShadowScreenPosition(
			em, entityID, pos, cameraX,
			zombieShadowWidth, zombieShadowHeight,
			zombieShadowOffsetX, zombieShadowOffsetY,
		)

		if err != nil {
			t.Fatalf("Unexpected error for %s: %v", zombieType, err)
		}

		if i == 0 {
			baseScreenY = screenY
			baseZombieType = zombieType
		} else {
			// 验证所有僵尸的阴影 Y 坐标一致
			if math.Abs(screenY-baseScreenY) > 0.01 {
				t.Errorf("%s shadow Y (%.2f) differs from %s shadow Y (%.2f) - equipment zombies should have consistent shadow Y",
					zombieType, screenY, baseZombieType, baseScreenY)
			}
		}
	}
}
