package game

import (
	"testing"

	"github.com/decker502/pvz/pkg/components"
)

// TestGameStateSingleton 测试单例模式是否正确实现
// 验证多次调用 GetGameState() 返回同一个实例
func TestGameStateSingleton(t *testing.T) {
	gs1 := GetGameState()
	gs2 := GetGameState()

	if gs1 != gs2 {
		t.Error("GetGameState() should return the same instance")
	}
}

// TestGameStateInitialValue 测试初始阳光值是否为50
func TestGameStateInitialValue(t *testing.T) {
	// 重置全局状态以测试初始化
	globalGameState = nil
	gs := GetGameState()

	if gs.Sun != 50 {
		t.Errorf("Expected initial sun to be 50, got %d", gs.Sun)
	}
}

// TestGetSun 测试 GetSun 方法是否正确返回阳光值
func TestGetSun(t *testing.T) {
	gs := GetGameState()
	gs.Sun = 100

	if gs.GetSun() != 100 {
		t.Errorf("Expected GetSun() to return 100, got %d", gs.GetSun())
	}
}

// TestAddSun 测试 AddSun 方法是否正确增加阳光
func TestAddSun(t *testing.T) {
	gs := GetGameState()
	gs.Sun = 100 // 设置初始值

	gs.AddSun(50)
	if gs.Sun != 150 {
		t.Errorf("Expected 150, got %d", gs.Sun)
	}
}

// TestAddSunCap 测试 AddSun 是否正确限制阳光上限为9990
func TestAddSunCap(t *testing.T) {
	gs := GetGameState()
	gs.Sun = 9980

	gs.AddSun(50)
	if gs.Sun != 9990 {
		t.Errorf("Expected 9990 (capped), got %d", gs.Sun)
	}
}

// TestAddSunExceedsCap 测试超过上限的情况
func TestAddSunExceedsCap(t *testing.T) {
	gs := GetGameState()
	gs.Sun = 9990

	gs.AddSun(100) // 尝试超过上限
	if gs.Sun != 9990 {
		t.Errorf("Expected 9990 (capped), got %d", gs.Sun)
	}
}

// TestSpendSunSuccess 测试阳光充足时 SpendSun 成功扣除
func TestSpendSunSuccess(t *testing.T) {
	gs := GetGameState()
	gs.Sun = 100

	success := gs.SpendSun(50)
	if !success {
		t.Error("Expected SpendSun to succeed")
	}
	if gs.Sun != 50 {
		t.Errorf("Expected 50, got %d", gs.Sun)
	}
}

// TestSpendSunFailure 测试阳光不足时 SpendSun 失败且阳光不变
func TestSpendSunFailure(t *testing.T) {
	gs := GetGameState()
	gs.Sun = 30

	success := gs.SpendSun(50)
	if success {
		t.Error("Expected SpendSun to fail")
	}
	if gs.Sun != 30 {
		t.Errorf("Expected sun to remain 30, got %d", gs.Sun)
	}
}

// TestSpendSunExactAmount 测试恰好花费全部阳光
func TestSpendSunExactAmount(t *testing.T) {
	gs := GetGameState()
	gs.Sun = 50

	success := gs.SpendSun(50)
	if !success {
		t.Error("Expected SpendSun to succeed")
	}
	if gs.Sun != 0 {
		t.Errorf("Expected 0, got %d", gs.Sun)
	}
}

// TestSpendSunZeroSun 测试阳光为0时无法扣除
func TestSpendSunZeroSun(t *testing.T) {
	gs := GetGameState()
	gs.Sun = 0

	success := gs.SpendSun(25)
	if success {
		t.Error("Expected SpendSun to fail when sun is 0")
	}
	if gs.Sun != 0 {
		t.Errorf("Expected sun to remain 0, got %d", gs.Sun)
	}
}

// TestEnterPlantingMode 测试进入种植模式
// 验证 IsPlantingMode 设置为 true，SelectedPlantType 正确设置
func TestEnterPlantingMode(t *testing.T) {
	gs := GetGameState()
	gs.IsPlantingMode = false // 初始状态

	gs.EnterPlantingMode(components.PlantSunflower)

	if !gs.IsPlantingMode {
		t.Error("Expected IsPlantingMode to be true")
	}
	if gs.SelectedPlantType != components.PlantSunflower {
		t.Errorf("Expected SelectedPlantType to be PlantSunflower, got %v", gs.SelectedPlantType)
	}
}

// TestExitPlantingMode 测试退出种植模式
// 验证 IsPlantingMode 设置为 false
func TestExitPlantingMode(t *testing.T) {
	gs := GetGameState()
	gs.IsPlantingMode = true // 先进入种植模式
	gs.SelectedPlantType = components.PlantPeashooter

	gs.ExitPlantingMode()

	if gs.IsPlantingMode {
		t.Error("Expected IsPlantingMode to be false")
	}
	// SelectedPlantType 保持不变（可选行为）
	if gs.SelectedPlantType != components.PlantPeashooter {
		t.Errorf("Expected SelectedPlantType to remain PlantPeashooter, got %v", gs.SelectedPlantType)
	}
}

// TestGetPlantingMode 测试获取种植模式状态
// 验证正确返回当前状态和选择的植物类型
func TestGetPlantingMode(t *testing.T) {
	gs := GetGameState()
	gs.IsPlantingMode = true
	gs.SelectedPlantType = components.PlantSunflower

	isPlanting, plantType := gs.GetPlantingMode()

	if !isPlanting {
		t.Error("Expected isPlanting to be true")
	}
	if plantType != components.PlantSunflower {
		t.Errorf("Expected plantType to be PlantSunflower, got %v", plantType)
	}
}

// TestPlantingModeToggle 测试种植模式切换
// 验证可以正确进入和退出种植模式多次
func TestPlantingModeToggle(t *testing.T) {
	gs := GetGameState()

	// 第一次进入
	gs.EnterPlantingMode(components.PlantSunflower)
	if !gs.IsPlantingMode {
		t.Error("Expected IsPlantingMode to be true after first enter")
	}

	// 退出
	gs.ExitPlantingMode()
	if gs.IsPlantingMode {
		t.Error("Expected IsPlantingMode to be false after exit")
	}

	// 第二次进入（不同植物类型）
	gs.EnterPlantingMode(components.PlantPeashooter)
	if !gs.IsPlantingMode {
		t.Error("Expected IsPlantingMode to be true after second enter")
	}
	if gs.SelectedPlantType != components.PlantPeashooter {
		t.Errorf("Expected SelectedPlantType to be PlantPeashooter, got %v", gs.SelectedPlantType)
	}
}
