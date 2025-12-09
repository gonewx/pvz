package game

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/gonewx/pvz/pkg/config"
	"github.com/quasilyte/gdata/v2"
)

// createTestGdataManagerForTool 创建用于测试的 gdata Manager
func createTestGdataManagerForTool(t *testing.T, testName string) *gdata.Manager {
	appName := fmt.Sprintf("pvz_tool_test_%s_%d", testName, time.Now().UnixNano())
	manager, err := gdata.Open(gdata.Config{
		AppName: appName,
	})
	if err != nil {
		return nil
	}

	// 注册清理函数，测试结束后删除测试目录
	t.Cleanup(func() {
		homeDir, err := os.UserHomeDir()
		if err == nil {
			testDir := filepath.Join(homeDir, ".local", "share", appName)
			os.RemoveAll(testDir)
		}
	})

	return manager
}

// TestToolUnlock 测试工具解锁功能
func TestToolUnlock(t *testing.T) {
	gdataManager := createTestGdataManagerForTool(t, "unlock")
	if gdataManager == nil {
		t.Skip("Cannot create gdata manager for testing")
	}

	saveManager, err := NewSaveManager(gdataManager)
	if err != nil {
		t.Fatalf("Failed to create SaveManager: %v", err)
	}

	// 创建测试用户（多用户架构要求）
	if err := saveManager.CreateUser("testuser"); err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	// 初始状态：铲子未解锁
	if saveManager.IsToolUnlocked("shovel") {
		t.Error("Shovel should not be unlocked initially")
	}

	// 解锁铲子
	saveManager.UnlockTool("shovel")
	if !saveManager.IsToolUnlocked("shovel") {
		t.Error("Shovel should be unlocked after UnlockTool()")
	}

	// 保存并重新加载
	if err := saveManager.Save(); err != nil {
		t.Fatalf("Failed to save: %v", err)
	}

	// 创建新的 SaveManager 实例加载存档（使用同一个 gdata manager 模拟重启）
	saveManager2, err := NewSaveManager(gdataManager)
	if err != nil {
		t.Fatalf("Failed to create SaveManager2: %v", err)
	}

	// 验证铲子仍然解锁
	if !saveManager2.IsToolUnlocked("shovel") {
		t.Error("Shovel should still be unlocked after reload")
	}
}

// TestCompleteLevelWithToolUnlock 测试关卡完成时解锁工具
func TestCompleteLevelWithToolUnlock(t *testing.T) {
	gdataManager := createTestGdataManagerForTool(t, "complete_level")
	if gdataManager == nil {
		t.Skip("Cannot create gdata manager for testing")
	}

	saveManager, err := NewSaveManager(gdataManager)
	if err != nil {
		t.Fatalf("Failed to create SaveManager: %v", err)
	}

	// 创建测试用户（多用户架构要求）
	if err := saveManager.CreateUser("testuser"); err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	// 创建 GameState（使用临时 SaveManager）
	gs := &GameState{
		saveManager:        saveManager,
		plantUnlockManager: NewPlantUnlockManager(),
	}

	// 模拟 1-4 关卡配置
	level14 := &config.LevelConfig{
		ID:          "1-4",
		Name:        "前院白天 1-4",
		UnlockTools: []string{"shovel"},
	}
	gs.CurrentLevel = level14

	// 初始状态：铲子未解锁
	if gs.IsToolUnlocked("shovel") {
		t.Error("Shovel should not be unlocked initially")
	}

	// 完成关卡（应该解锁铲子）
	err = gs.CompleteLevel("1-4", "", []string{"shovel"})
	if err != nil {
		t.Fatalf("CompleteLevel failed: %v", err)
	}

	// 验证铲子已解锁
	if !gs.IsToolUnlocked("shovel") {
		t.Error("Shovel should be unlocked after completing 1-4")
	}

	// 验证存档已保存到 gdata
	if !gdataManager.ObjectPropExists(savesObject, "testuser") {
		t.Error("Save data should exist in gdata after CompleteLevel()")
	}
}

// TestShovelDisplayLogic 测试铲子显示逻辑
func TestShovelDisplayLogic(t *testing.T) {
	gdataManager := createTestGdataManagerForTool(t, "display_logic")
	if gdataManager == nil {
		t.Skip("Cannot create gdata manager for testing")
	}

	saveManager, err := NewSaveManager(gdataManager)
	if err != nil {
		t.Fatalf("Failed to create SaveManager: %v", err)
	}

	// 创建测试用户
	if err := saveManager.CreateUser("testuser"); err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	// 创建 GameState
	gs := &GameState{
		saveManager:        saveManager,
		plantUnlockManager: NewPlantUnlockManager(),
	}

	// 测试场景1：教学关卡（1-1）- 即使铲子解锁了也不显示
	level11 := &config.LevelConfig{
		ID:          "1-1",
		Name:        "前院白天 1-1",
		OpeningType: "tutorial",
	}
	gs.CurrentLevel = level11

	// 即使解锁铲子
	saveManager.UnlockTool("shovel")

	// 教学关卡应该隐藏铲子（通过 OpeningType 判断）
	if level11.OpeningType == "tutorial" {
		t.Log("Tutorial level: shovel should be hidden regardless of unlock status")
	}

	// 测试场景2：标准关卡（1-2）- 铲子未解锁时不显示
	level12 := &config.LevelConfig{
		ID:          "1-2",
		Name:        "前院白天 1-2",
		OpeningType: "standard",
	}
	gs.CurrentLevel = level12

	// 使用新的 gdata Manager 重置状态
	gdataManager2 := createTestGdataManagerForTool(t, "display_logic_2")
	if gdataManager2 == nil {
		t.Skip("Cannot create gdata manager for testing")
	}
	saveManager2, _ := NewSaveManager(gdataManager2)
	saveManager2.CreateUser("testuser2")
	gs.saveManager = saveManager2

	// 铲子未解锁时不应该显示
	shouldDisplay := gs.CurrentLevel.OpeningType != "tutorial" && gs.IsToolUnlocked("shovel")
	if shouldDisplay {
		t.Error("Shovel should NOT be displayed in 1-2 when not unlocked")
	}

	// 测试场景3：标准关卡（1-5）- 铲子解锁后显示
	level15 := &config.LevelConfig{
		ID:          "1-5",
		Name:        "前院白天 1-5",
		OpeningType: "standard",
	}
	gs.CurrentLevel = level15

	// 解锁铲子
	gs.saveManager.UnlockTool("shovel")

	// 铲子解锁后应该显示
	shouldDisplay = gs.CurrentLevel.OpeningType != "tutorial" && gs.IsToolUnlocked("shovel")
	if !shouldDisplay {
		t.Error("Shovel SHOULD be displayed in 1-5 when unlocked")
	}
}
