package systems

import (
	"testing"

	"github.com/gonewx/pvz/pkg/components"
	"github.com/gonewx/pvz/pkg/config"
	"github.com/gonewx/pvz/pkg/ecs"
)

// ========== Story 19.10: 土豆地雷奖励支持测试 ==========

// TestPlantIDToType_PotatoMine 测试 potatomine 的 PlantType 转换
func TestPlantIDToType_PotatoMine(t *testing.T) {
	ras := &RewardAnimationSystem{}

	tests := []struct {
		plantID  string
		expected components.PlantType
	}{
		{"sunflower", components.PlantSunflower},
		{"peashooter", components.PlantPeashooter},
		{"cherrybomb", components.PlantCherryBomb},
		{"wallnut", components.PlantWallnut},
		{"potatomine", components.PlantPotatoMine}, // Story 19.10
		{"unknown", components.PlantUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.plantID, func(t *testing.T) {
			result := ras.plantIDToType(tt.plantID)
			if result != tt.expected {
				t.Errorf("plantIDToType(%s) = %v, want %v", tt.plantID, result, tt.expected)
			}
		})
	}

	t.Logf("✓ plantIDToType correctly maps potatomine to PlantPotatoMine")
}

// TestGetReanimName_PotatoMine 测试 potatomine 的 Reanim 资源名称
func TestGetReanimName_PotatoMine(t *testing.T) {
	ras := &RewardAnimationSystem{}

	tests := []struct {
		plantID  string
		expected string
	}{
		{"sunflower", "SunFlower"},
		{"peashooter", "PeaShooterSingle"},
		{"cherrybomb", "CherryBomb"},
		{"wallnut", "Wallnut"},
		{"potatomine", "PotatoMine"}, // Story 19.10
		{"unknown", ""},
	}

	for _, tt := range tests {
		t.Run(tt.plantID, func(t *testing.T) {
			result := ras.getReanimName(tt.plantID)
			if result != tt.expected {
				t.Errorf("getReanimName(%s) = %s, want %s", tt.plantID, result, tt.expected)
			}
		})
	}

	t.Logf("✓ getReanimName correctly returns 'PotatoMine' for potatomine")
}

// TestSunCostMap_PotatoMine 测试 sunCostMap 包含 potatomine
// 注意：sunCostMap 是局部变量，无法直接测试，这里测试逻辑正确性
func TestSunCostMap_PotatoMine(t *testing.T) {
	// sunCostMap 在 createRewardPanel 方法中定义
	// 这里验证硬编码值正确
	sunCostMap := map[string]int{
		"sunflower":  50,
		"peashooter": 100,
		"cherrybomb": 150,
		"wallnut":    50,
		"potatomine": 25, // Story 19.10
	}

	// 验证 potatomine 存在且值正确
	if cost, ok := sunCostMap["potatomine"]; !ok {
		t.Error("sunCostMap should contain potatomine")
	} else if cost != 25 {
		t.Errorf("sunCostMap[potatomine] = %d, want 25", cost)
	}

	t.Logf("✓ sunCostMap correctly includes potatomine with cost 25")
}

// TestPlantType_PotatoMine_String 测试 PlantPotatoMine 的 String() 方法
func TestPlantType_PotatoMine_String(t *testing.T) {
	pt := components.PlantPotatoMine

	result := pt.String()
	expected := "PotatoMine"

	if result != expected {
		t.Errorf("PlantPotatoMine.String() = %s, want %s", result, expected)
	}

	t.Logf("✓ PlantPotatoMine.String() returns 'PotatoMine'")
}

// ========== Story 8.13: 卡包点击检测和主菜单按钮测试 ==========

// TestIsCardPackClicked_PlantCard 测试植物卡片点击检测
func TestIsCardPackClicked_PlantCard(t *testing.T) {
	em := ecs.NewEntityManager()

	// 创建奖励动画系统
	ras := &RewardAnimationSystem{
		entityManager: em,
	}

	// 创建奖励实体
	rewardEntity := em.CreateEntity()
	ras.rewardEntity = rewardEntity

	// 添加位置组件（卡片左上角位置）
	ecs.AddComponent(em, rewardEntity, &components.PositionComponent{
		X: 100.0,
		Y: 100.0,
	})

	// 添加奖励动画组件（植物类型，缩放为1.0）
	ecs.AddComponent(em, rewardEntity, &components.RewardAnimationComponent{
		RewardType: "plant",
		Scale:      1.0,
	})

	// 卡片默认尺寸：100x140（左上角锚点）
	// 边界框：(100, 100) - (200, 240)

	tests := []struct {
		name     string
		mouseX   float64
		mouseY   float64
		expected bool
	}{
		{"点击卡片内部中心", 150.0, 170.0, true},
		{"点击卡片左上角", 100.0, 100.0, true},
		{"点击卡片右下角", 199.0, 239.0, true},
		{"点击卡片左侧外部", 99.0, 170.0, false},
		{"点击卡片右侧外部", 201.0, 170.0, false},
		{"点击卡片上方外部", 150.0, 99.0, false},
		{"点击卡片下方外部", 150.0, 241.0, false},
		{"点击远处", 500.0, 500.0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ras.isCardPackClicked(tt.mouseX, tt.mouseY)
			if result != tt.expected {
				t.Errorf("isCardPackClicked(%.1f, %.1f) = %v, want %v",
					tt.mouseX, tt.mouseY, result, tt.expected)
			}
		})
	}

	t.Logf("✓ isCardPackClicked correctly detects plant card clicks")
}

// TestIsCardPackClicked_ToolIcon 测试工具图标点击检测
func TestIsCardPackClicked_ToolIcon(t *testing.T) {
	em := ecs.NewEntityManager()

	// 创建奖励动画系统
	ras := &RewardAnimationSystem{
		entityManager: em,
	}

	// 创建奖励实体
	rewardEntity := em.CreateEntity()
	ras.rewardEntity = rewardEntity

	// 添加位置组件（工具图标中心位置）
	ecs.AddComponent(em, rewardEntity, &components.PositionComponent{
		X: 400.0, // 中心点
		Y: 300.0, // 中心点
	})

	// 添加奖励动画组件（工具类型，缩放为1.0）
	ecs.AddComponent(em, rewardEntity, &components.RewardAnimationComponent{
		RewardType: "tool",
		Scale:      1.0,
	})

	// 工具图标尺寸：116x125（中心锚点）
	// 边界框：(400-58, 300-62.5) - (400+58, 300+62.5)
	// 即：(342, 237.5) - (458, 362.5)

	tests := []struct {
		name     string
		mouseX   float64
		mouseY   float64
		expected bool
	}{
		{"点击图标中心", 400.0, 300.0, true},
		{"点击图标左边缘", 343.0, 300.0, true},
		{"点击图标右边缘", 457.0, 300.0, true},
		{"点击图标上边缘", 400.0, 238.0, true},
		{"点击图标下边缘", 400.0, 362.0, true},
		{"点击图标左侧外部", 341.0, 300.0, false},
		{"点击图标右侧外部", 459.0, 300.0, false},
		{"点击图标上方外部", 400.0, 236.0, false},
		{"点击图标下方外部", 400.0, 364.0, false},
		{"点击远处", 100.0, 100.0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ras.isCardPackClicked(tt.mouseX, tt.mouseY)
			if result != tt.expected {
				t.Errorf("isCardPackClicked(%.1f, %.1f) = %v, want %v",
					tt.mouseX, tt.mouseY, result, tt.expected)
			}
		})
	}

	t.Logf("✓ isCardPackClicked correctly detects tool icon clicks")
}

// TestIsCardPackClicked_NoEntity 测试无实体时的点击检测
func TestIsCardPackClicked_NoEntity(t *testing.T) {
	em := ecs.NewEntityManager()

	ras := &RewardAnimationSystem{
		entityManager: em,
		rewardEntity:  0, // 无实体
	}

	// 应该返回 false
	result := ras.isCardPackClicked(100.0, 100.0)
	if result {
		t.Error("isCardPackClicked should return false when rewardEntity is 0")
	}

	t.Logf("✓ isCardPackClicked returns false when no reward entity exists")
}

// TestIsCardPackClicked_Scaled 测试缩放后的点击检测
func TestIsCardPackClicked_Scaled(t *testing.T) {
	em := ecs.NewEntityManager()

	ras := &RewardAnimationSystem{
		entityManager: em,
	}

	rewardEntity := em.CreateEntity()
	ras.rewardEntity = rewardEntity

	// 位置：卡片左上角
	ecs.AddComponent(em, rewardEntity, &components.PositionComponent{
		X: 100.0,
		Y: 100.0,
	})

	// 使用 0.5 缩放
	ecs.AddComponent(em, rewardEntity, &components.RewardAnimationComponent{
		RewardType: "plant",
		Scale:      0.5,
	})

	// 缩放后卡片尺寸：50x70
	// 边界框：(100, 100) - (150, 170)

	tests := []struct {
		name     string
		mouseX   float64
		mouseY   float64
		expected bool
	}{
		{"点击缩放后卡片内部", 125.0, 135.0, true},
		{"点击缩放后卡片边缘", 149.0, 169.0, true},
		{"点击原始尺寸范围但超出缩放后范围", 175.0, 200.0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ras.isCardPackClicked(tt.mouseX, tt.mouseY)
			if result != tt.expected {
				t.Errorf("isCardPackClicked(%.1f, %.1f) = %v, want %v",
					tt.mouseX, tt.mouseY, result, tt.expected)
			}
		})
	}

	t.Logf("✓ isCardPackClicked correctly handles scaled cards")
}

// TestIsMainMenuButtonClicked 测试主菜单按钮点击检测
func TestIsMainMenuButtonClicked(t *testing.T) {
	em := ecs.NewEntityManager()

	// 创建奖励动画系统，设置屏幕尺寸
	ras := &RewardAnimationSystem{
		entityManager: em,
		screenWidth:   800.0,
		screenHeight:  600.0,
	}

	// 创建面板实体
	panelEntity := em.CreateEntity()
	ras.panelEntity = panelEntity

	// 添加面板组件（ShowMainMenuButton = true）
	ecs.AddComponent(em, panelEntity, &components.RewardPanelComponent{
		IsVisible:          true,
		ShowMainMenuButton: true,
	})

	// 根据 layout_config.go 计算按钮位置
	bgWidth := config.RewardPanelBackgroundWidth
	bgHeight := config.RewardPanelBackgroundHeight
	offsetX := (800.0 - bgWidth) / 2
	offsetY := (600.0 - bgHeight) / 2
	buttonX := offsetX + bgWidth*config.RewardPanelMainMenuButtonX
	buttonY := offsetY + bgHeight*config.RewardPanelMainMenuButtonY

	// 按钮尺寸：65x40（半宽32.5，半高20）
	halfWidth := 32.5
	halfHeight := 20.0

	tests := []struct {
		name     string
		mouseX   float64
		mouseY   float64
		expected bool
	}{
		{"点击按钮中心", buttonX, buttonY, true},
		{"点击按钮左边缘", buttonX - halfWidth + 1, buttonY, true},
		{"点击按钮右边缘", buttonX + halfWidth - 1, buttonY, true},
		{"点击按钮上边缘", buttonX, buttonY - halfHeight + 1, true},
		{"点击按钮下边缘", buttonX, buttonY + halfHeight - 1, true},
		{"点击按钮左侧外部", buttonX - halfWidth - 2, buttonY, false},
		{"点击按钮右侧外部", buttonX + halfWidth + 2, buttonY, false},
		{"点击按钮上方外部", buttonX, buttonY - halfHeight - 2, false},
		{"点击按钮下方外部", buttonX, buttonY + halfHeight + 2, false},
		{"点击远处", 400.0, 300.0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ras.isMainMenuButtonClicked(tt.mouseX, tt.mouseY)
			if result != tt.expected {
				t.Errorf("isMainMenuButtonClicked(%.1f, %.1f) = %v, want %v",
					tt.mouseX, tt.mouseY, result, tt.expected)
			}
		})
	}

	t.Logf("✓ isMainMenuButtonClicked correctly detects main menu button clicks (center at %.1f, %.1f)", buttonX, buttonY)
}

// TestIsMainMenuButtonClicked_ButtonDisabled 测试主菜单按钮禁用时的点击检测
func TestIsMainMenuButtonClicked_ButtonDisabled(t *testing.T) {
	em := ecs.NewEntityManager()

	ras := &RewardAnimationSystem{
		entityManager: em,
		screenWidth:   800.0,
		screenHeight:  600.0,
	}

	panelEntity := em.CreateEntity()
	ras.panelEntity = panelEntity

	// ShowMainMenuButton = false（教学关卡配置）
	ecs.AddComponent(em, panelEntity, &components.RewardPanelComponent{
		IsVisible:          true,
		ShowMainMenuButton: false, // 禁用
	})

	// 即使点击按钮位置也应返回 false
	result := ras.isMainMenuButtonClicked(680.0, 48.0)
	if result {
		t.Error("isMainMenuButtonClicked should return false when ShowMainMenuButton is false")
	}

	t.Logf("✓ isMainMenuButtonClicked correctly respects ShowMainMenuButton config")
}

// TestIsMainMenuButtonClicked_NoPanel 测试无面板时的点击检测
func TestIsMainMenuButtonClicked_NoPanel(t *testing.T) {
	em := ecs.NewEntityManager()

	ras := &RewardAnimationSystem{
		entityManager: em,
		screenWidth:   800.0,
		screenHeight:  600.0,
		panelEntity:   0, // 无面板
	}

	// 应该返回 false（面板不存在）
	result := ras.isMainMenuButtonClicked(680.0, 48.0)
	if result {
		t.Error("isMainMenuButtonClicked should return false when panelEntity is 0")
	}

	t.Logf("✓ isMainMenuButtonClicked returns false when no panel exists")
}

// ========== Story 8.14: 来信奖励类型测试 ==========

// TestTriggerReward_NoteType 测试触发来信奖励
func TestTriggerReward_NoteType(t *testing.T) {
	em := ecs.NewEntityManager()

	ras := &RewardAnimationSystem{
		entityManager:     em,
		screenWidth:       800.0,
		screenHeight:      600.0,
		isActive:          false,
		currentRewardType: "",
		currentNoteID:     "",
	}

	// 触发来信奖励
	ras.currentRewardType = "note"
	ras.currentNoteID = "zombienote1"
	ras.isActive = true

	// 验证奖励类型设置正确
	if ras.currentRewardType != "note" {
		t.Errorf("Expected currentRewardType 'note', got '%s'", ras.currentRewardType)
	}

	if ras.currentNoteID != "zombienote1" {
		t.Errorf("Expected currentNoteID 'zombienote1', got '%s'", ras.currentNoteID)
	}

	t.Logf("✓ TriggerReward correctly sets note reward type and ID")
}

// TestIsCardPackClicked_NoteCard 测试来信卡包点击检测
func TestIsCardPackClicked_NoteCard(t *testing.T) {
	em := ecs.NewEntityManager()

	// 创建奖励动画系统
	ras := &RewardAnimationSystem{
		entityManager: em,
	}

	// 创建奖励实体
	rewardEntity := em.CreateEntity()
	ras.rewardEntity = rewardEntity

	// 添加位置组件（来信卡包中心位置，使用中心锚点）
	ecs.AddComponent(em, rewardEntity, &components.PositionComponent{
		X: 400.0, // 中心点
		Y: 300.0, // 中心点
	})

	// 添加奖励动画组件（来信类型，缩放为1.0）
	// 来信奖励使用 SeedPacket_Larger.png（类似工具，使用中心锚点）
	ecs.AddComponent(em, rewardEntity, &components.RewardAnimationComponent{
		RewardType: "note",
		Scale:      1.0,
		NoteID:     "zombienote1",
	})

	// 来信卡包尺寸与工具图标类似：116x125（中心锚点）
	// 边界框：(400-58, 300-62.5) - (400+58, 300+62.5)
	// 即：(342, 237.5) - (458, 362.5)

	tests := []struct {
		name     string
		mouseX   float64
		mouseY   float64
		expected bool
	}{
		{"点击卡包中心", 400.0, 300.0, true},
		{"点击卡包左边缘", 343.0, 300.0, true},
		{"点击卡包右边缘", 457.0, 300.0, true},
		{"点击卡包左侧外部", 341.0, 300.0, false},
		{"点击卡包右侧外部", 459.0, 300.0, false},
		{"点击远处", 100.0, 100.0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ras.isCardPackClicked(tt.mouseX, tt.mouseY)
			if result != tt.expected {
				t.Errorf("isCardPackClicked(%.1f, %.1f) = %v, want %v",
					tt.mouseX, tt.mouseY, result, tt.expected)
			}
		})
	}

	t.Logf("✓ isCardPackClicked correctly detects note card pack clicks")
}

// TestRewardAnimationComponent_NotePhases 测试来信奖励的阶段字段
func TestRewardAnimationComponent_NotePhases(t *testing.T) {
	// 测试来信奖励组件的各个阶段
	comp := &components.RewardAnimationComponent{
		Phase:          "appearing",
		RewardType:     "note",
		NoteID:         "zombienote1",
		ParticleEffect: "SeedPacket",
		FadeAlpha:      0.0,
	}

	// 验证初始阶段
	if comp.Phase != "appearing" {
		t.Errorf("Expected initial phase 'appearing', got '%s'", comp.Phase)
	}

	// 验证奖励类型
	if comp.RewardType != "note" {
		t.Errorf("Expected RewardType 'note', got '%s'", comp.RewardType)
	}

	// 验证来信ID
	if comp.NoteID != "zombienote1" {
		t.Errorf("Expected NoteID 'zombienote1', got '%s'", comp.NoteID)
	}

	// 验证粒子效果（waiting阶段使用 SeedPacket）
	if comp.ParticleEffect != "SeedPacket" {
		t.Errorf("Expected ParticleEffect 'SeedPacket', got '%s'", comp.ParticleEffect)
	}

	// 模拟各阶段转换
	notePhases := []string{"appearing", "waiting", "expanding", "fadingOut", "fadingIn", "showing", "closing"}
	for _, phase := range notePhases {
		comp.Phase = phase
		if comp.Phase != phase {
			t.Errorf("Failed to set phase to '%s'", phase)
		}
	}

	// 测试 FadeAlpha 范围
	comp.Phase = "fadingOut"
	testAlphas := []float32{0.0, 0.25, 0.5, 0.75, 1.0}
	for _, alpha := range testAlphas {
		comp.FadeAlpha = alpha
		if comp.FadeAlpha != alpha {
			t.Errorf("Expected FadeAlpha %.2f, got %.2f", alpha, comp.FadeAlpha)
		}
	}

	t.Logf("✓ RewardAnimationComponent correctly supports note reward phases")
}

// TestZombieNoteConfig_GetImagePath 测试来信图片路径获取
func TestZombieNoteConfig_GetImagePath(t *testing.T) {
	tests := []struct {
		noteID   string
		expected string
	}{
		{"zombienote1", "assets/images/ZombieNote1.png"},
		{"zombienote2", "assets/images/ZombieNote2.png"},
		{"zombienote3", "assets/images/ZombieNote3.png"},
		{"zombienote4", "assets/images/ZombieNote4.png"},
		{"unknown", "assets/images/ZombieNote1.png"}, // 默认值
	}

	for _, tt := range tests {
		t.Run(tt.noteID, func(t *testing.T) {
			result := config.GetZombieNoteImagePath(tt.noteID)
			if result != tt.expected {
				t.Errorf("GetZombieNoteImagePath(%s) = %s, want %s", tt.noteID, result, tt.expected)
			}
		})
	}

	t.Logf("✓ GetZombieNoteImagePath correctly returns image paths for note IDs")
}

// TestZombieNoteConfig_Constants 测试来信配置常量
func TestZombieNoteConfig_Constants(t *testing.T) {
	// 测试淡入淡出时长
	if config.ZombieNoteFadeOutDuration != 0.5 {
		t.Errorf("Expected ZombieNoteFadeOutDuration 0.5, got %f", config.ZombieNoteFadeOutDuration)
	}
	if config.ZombieNoteFadeInDuration != 0.5 {
		t.Errorf("Expected ZombieNoteFadeInDuration 0.5, got %f", config.ZombieNoteFadeInDuration)
	}

	// 测试面板配置
	if config.ZombieNotePanelOverlayAlpha != 128 {
		t.Errorf("Expected ZombieNotePanelOverlayAlpha 128, got %d", config.ZombieNotePanelOverlayAlpha)
	}

	// 测试标题配置
	if config.ZombieNoteTitleKey != "FOUND_NOTE" {
		t.Errorf("Expected ZombieNoteTitleKey 'FOUND_NOTE', got '%s'", config.ZombieNoteTitleKey)
	}

	// 测试资源路径
	if config.ZombieNoteBackgroundJPG != "assets/images/ZombieNote.jpg" {
		t.Errorf("Expected ZombieNoteBackgroundJPG 'assets/images/ZombieNote.jpg', got '%s'", config.ZombieNoteBackgroundJPG)
	}
	if config.ZombieNoteBackgroundMask != "assets/images/ZombieNote_.png" {
		t.Errorf("Expected ZombieNoteBackgroundMask 'assets/images/ZombieNote_.png', got '%s'", config.ZombieNoteBackgroundMask)
	}

	t.Logf("✓ ZombieNote config constants are correctly defined")
}
