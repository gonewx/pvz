package game

import "testing"

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

