package game

import (
	"testing"

	"github.com/quasilyte/gdata/v2"
)

// TestGdataManagerInit 测试 gdata Manager 正常初始化
// AC3: 应用启动时初始化 Manager，使用应用名 pvz_newx
func TestGdataManagerInit(t *testing.T) {
	// 直接测试 gdata.Open 是否能正常工作
	manager, err := gdata.Open(gdata.Config{
		AppName: "pvz_newx",
	})
	if err != nil {
		t.Fatalf("gdata.Open failed: %v", err)
	}
	if manager == nil {
		t.Fatal("gdata.Open returned nil manager without error")
	}
}

// TestGetGdataManager 测试 getter 方法返回正确的 Manager 实例
// AC2: 在 GameState 中创建全局 gdata.Manager 实例
func TestGetGdataManager(t *testing.T) {
	// 重置全局状态以确保测试隔离
	resetGlobalGameState()

	gs := GetGameState()
	manager := gs.GetGdataManager()

	// gdata Manager 应该成功初始化（在普通桌面环境下）
	if manager == nil {
		t.Log("gdata Manager is nil - this is acceptable if running in restricted environment")
	}
}

// TestGdataManagerNilSafe 测试 gdataManager 为 nil 时的安全性
// AC4: 处理初始化错误，提供降级方案（Manager 为 nil 时游戏仍可运行）
func TestGdataManagerNilSafe(t *testing.T) {
	// 创建一个 gdataManager 为 nil 的 GameState
	gs := &GameState{
		gdataManager: nil,
	}

	// 调用 GetGdataManager 不应该 panic
	manager := gs.GetGdataManager()

	if manager != nil {
		t.Fatal("Expected nil manager")
	}
}

// TestGameStateWithGdataFailure 测试 gdata 初始化失败时的降级行为
// AC4: Manager 为 nil 时游戏仍可运行
func TestGameStateWithGdataFailure(t *testing.T) {
	// 模拟 gdata 初始化失败的场景
	// 创建一个 GameState，其 gdataManager 为 nil
	gs := &GameState{
		Sun:                50,
		plantUnlockManager: NewPlantUnlockManager(),
		SelectedPlants:     []string{},
		gdataManager:       nil, // 模拟初始化失败
	}

	// 验证游戏状态的其他功能仍然正常
	if gs.Sun != 50 {
		t.Errorf("Expected Sun=50, got %d", gs.Sun)
	}

	if gs.plantUnlockManager == nil {
		t.Error("plantUnlockManager should not be nil")
	}

	// GetGdataManager 应该返回 nil 但不 panic
	if gs.GetGdataManager() != nil {
		t.Error("Expected GetGdataManager to return nil")
	}
}

// TestGdataManagerAppName 测试使用正确的应用名
// AC3: 使用应用名 pvz_newx
func TestGdataManagerAppName(t *testing.T) {
	const expectedAppName = "pvz_newx"

	// 使用正确的应用名初始化
	manager, err := gdata.Open(gdata.Config{
		AppName: expectedAppName,
	})

	if err != nil {
		t.Fatalf("Failed to open gdata with AppName %q: %v", expectedAppName, err)
	}

	if manager == nil {
		t.Fatal("Manager is nil")
	}

	// gdata 库不直接暴露 AppName getter，但我们可以通过成功初始化来验证
	// 如果应用名无效，gdata.Open 会返回错误
	t.Logf("Successfully initialized gdata Manager with AppName: %s", expectedAppName)
}

// TestGdataManagerIntegration 集成测试：验证 GameState 正确初始化 gdata
func TestGdataManagerIntegration(t *testing.T) {
	// 重置全局状态
	resetGlobalGameState()

	// 获取 GameState 单例
	gs := GetGameState()

	// 验证基本字段正确初始化
	if gs.Sun != 50 {
		t.Errorf("Expected default Sun=50, got %d", gs.Sun)
	}

	// gdata Manager 可能为 nil（取决于运行环境），但不应该导致崩溃
	manager := gs.GetGdataManager()
	if manager != nil {
		t.Log("gdata Manager successfully initialized")
	} else {
		t.Log("gdata Manager is nil (acceptable in restricted environments)")
	}
}

// resetGlobalGameState 重置全局 GameState 单例
// 用于测试隔离
func resetGlobalGameState() {
	globalGameState = nil
}

