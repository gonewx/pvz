package game

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/quasilyte/gdata/v2"
)

// createTestGdataManager 创建用于测试的 gdata Manager
// 每个测试使用唯一的 AppName 来隔离数据
// 返回 manager 和清理函数
func createTestGdataManager(t *testing.T, testName string) *gdata.Manager {
	appName := fmt.Sprintf("pvz_test_%s_%d", testName, time.Now().UnixNano())
	manager, err := gdata.Open(gdata.Config{
		AppName: appName,
	})
	if err != nil {
		return nil
	}

	// 注册清理函数，测试结束后删除测试目录
	t.Cleanup(func() {
		// gdata 在 Linux 上使用 ~/.local/share/<appName>
		// 在其他平台上可能不同，但我们主要在 Linux 上测试
		homeDir, err := os.UserHomeDir()
		if err == nil {
			testDir := filepath.Join(homeDir, ".local", "share", appName)
			os.RemoveAll(testDir)
		}
	})

	return manager
}

func TestSaveManager_NewGame(t *testing.T) {
	// 使用 gdata manager 测试
	gdataManager := createTestGdataManager(t, "new_game")
	if gdataManager == nil {
		t.Skip("Cannot create gdata manager for testing")
	}

	// 创建SaveManager
	sm, err := NewSaveManager(gdataManager)
	if err != nil {
		t.Fatalf("Failed to create SaveManager: %v", err)
	}

	// 验证初始状态（没有用户）
	if sm.GetCurrentUser() != "" {
		t.Errorf("Expected empty current user, got %q", sm.GetCurrentUser())
	}

	users, err := sm.LoadUserList()
	if err != nil {
		t.Fatalf("Failed to load user list: %v", err)
	}
	if len(users) != 0 {
		t.Errorf("Expected 0 users, got %d", len(users))
	}
}

func TestSaveManager_NewSaveManagerNilGdata(t *testing.T) {
	// 测试降级场景：gdataManager 为 nil
	sm, err := NewSaveManager(nil)
	if err != nil {
		t.Fatalf("NewSaveManager should not return error for nil gdata: %v", err)
	}

	// 验证可以正常创建用户（内存模式）
	err = sm.CreateUser("testuser")
	if err != nil {
		t.Fatalf("CreateUser should work in degraded mode: %v", err)
	}

	if sm.GetCurrentUser() != "testuser" {
		t.Errorf("Expected current user 'testuser', got %q", sm.GetCurrentUser())
	}

	// Save 应该不报错（降级模式下静默成功）
	err = sm.Save()
	if err != nil {
		t.Errorf("Save should not error in degraded mode: %v", err)
	}

	// Load 应该不报错（降级模式下使用默认数据）
	err = sm.Load()
	if err != nil {
		t.Errorf("Load should not error in degraded mode: %v", err)
	}
}

func TestSaveManager_CreateUser(t *testing.T) {
	gdataManager := createTestGdataManager(t, "create_user")
	if gdataManager == nil {
		t.Skip("Cannot create gdata manager for testing")
	}

	sm, err := NewSaveManager(gdataManager)
	if err != nil {
		t.Fatalf("Failed to create SaveManager: %v", err)
	}

	// 创建用户
	err = sm.CreateUser("player1")
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	// 验证用户创建成功
	if sm.GetCurrentUser() != "player1" {
		t.Errorf("Expected current user 'player1', got %q", sm.GetCurrentUser())
	}

	users, _ := sm.LoadUserList()
	if len(users) != 1 {
		t.Errorf("Expected 1 user, got %d", len(users))
	}
	if users[0].Username != "player1" {
		t.Errorf("Expected username 'player1', got %q", users[0].Username)
	}

	// 验证存档已保存到 gdata（通过检查数据存在性）
	if !gdataManager.ObjectPropExists(savesObject, "player1") {
		t.Error("Expected save data to exist in gdata")
	}

	// 验证用户列表已保存到 gdata
	if !gdataManager.ObjectPropExists(savesObject, usersProperty) {
		t.Error("Expected user list to exist in gdata")
	}
}

func TestSaveManager_CreateUser_DuplicateName(t *testing.T) {
	gdataManager := createTestGdataManager(t, "duplicate_name")
	if gdataManager == nil {
		t.Skip("Cannot create gdata manager for testing")
	}

	sm, _ := NewSaveManager(gdataManager)

	// 创建第一个用户
	err := sm.CreateUser("player1")
	if err != nil {
		t.Fatalf("Failed to create first user: %v", err)
	}

	// 尝试创建同名用户
	err = sm.CreateUser("player1")
	if err == nil {
		t.Error("Expected error when creating duplicate user")
	}
}

func TestSaveManager_ValidateUsername(t *testing.T) {
	sm, _ := NewSaveManager(nil)

	tests := []struct {
		name     string
		username string
		wantErr  bool
	}{
		{"Valid username", "player1", false},
		{"Valid with space", "Player One", false},
		{"Valid alphanumeric", "User123", false},
		{"Empty username", "", true},
		{"Too long", "abcdefghijklmnopqrstuvwxyz", true},
		{"Special characters", "user@#$", true},
		{"Chinese characters", "用户", true},
		{"Underscore", "user_name", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := sm.ValidateUsername(tt.username)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateUsername(%q) error = %v, wantErr %v", tt.username, err, tt.wantErr)
			}
		})
	}
}

func TestSaveManager_SaveAndLoad_WithUser(t *testing.T) {
	gdataManager := createTestGdataManager(t, "save_load")
	if gdataManager == nil {
		t.Skip("Cannot create gdata manager for testing")
	}

	// 创建SaveManager并创建用户
	sm1, err := NewSaveManager(gdataManager)
	if err != nil {
		t.Fatalf("Failed to create SaveManager: %v", err)
	}

	err = sm1.CreateUser("testuser")
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	sm1.SetHighestLevel("1-3")
	sm1.UnlockPlant("peashooter")
	sm1.UnlockPlant("sunflower")
	sm1.UnlockTool("shovel")

	// 保存
	if err := sm1.Save(); err != nil {
		t.Fatalf("Failed to save: %v", err)
	}

	// 创建新的SaveManager加载数据（使用同一个 gdata manager 模拟重启）
	sm2, err := NewSaveManager(gdataManager)
	if err != nil {
		t.Fatalf("Failed to create second SaveManager: %v", err)
	}

	// 验证自动加载当前用户
	if sm2.GetCurrentUser() != "testuser" {
		t.Errorf("Expected current user 'testuser', got %q", sm2.GetCurrentUser())
	}

	// 验证加载的数据
	if sm2.GetHighestLevel() != "1-3" {
		t.Errorf("Expected highest level '1-3', got %q", sm2.GetHighestLevel())
	}

	plants := sm2.GetUnlockedPlants()
	if len(plants) != 2 {
		t.Errorf("Expected 2 unlocked plants, got %d", len(plants))
	}

	tools := sm2.GetUnlockedTools()
	if len(tools) != 1 {
		t.Errorf("Expected 1 unlocked tool, got %d", len(tools))
	}

	if !sm2.IsToolUnlocked("shovel") {
		t.Error("Expected shovel to be unlocked")
	}
}

func TestSaveManager_RenameUser(t *testing.T) {
	gdataManager := createTestGdataManager(t, "rename_user")
	if gdataManager == nil {
		t.Skip("Cannot create gdata manager for testing")
	}

	sm, _ := NewSaveManager(gdataManager)

	// 创建用户
	sm.CreateUser("oldname")

	// 重命名
	err := sm.RenameUser("oldname", "newname")
	if err != nil {
		t.Fatalf("Failed to rename user: %v", err)
	}

	// 验证当前用户更新
	if sm.GetCurrentUser() != "newname" {
		t.Errorf("Expected current user 'newname', got %q", sm.GetCurrentUser())
	}

	// 验证用户列表更新
	users, _ := sm.LoadUserList()
	if len(users) != 1 {
		t.Errorf("Expected 1 user, got %d", len(users))
	}
	if users[0].Username != "newname" {
		t.Errorf("Expected username 'newname', got %q", users[0].Username)
	}

	// 验证 gdata 中旧数据已删除，新数据已创建
	if gdataManager.ObjectPropExists(savesObject, "oldname") {
		t.Error("Expected old save data to be removed from gdata")
	}

	if !gdataManager.ObjectPropExists(savesObject, "newname") {
		t.Error("Expected new save data to exist in gdata")
	}
}

func TestSaveManager_RenameUser_WithBattleSave(t *testing.T) {
	gdataManager := createTestGdataManager(t, "rename_with_battle")
	if gdataManager == nil {
		t.Skip("Cannot create gdata manager for testing")
	}

	sm, _ := NewSaveManager(gdataManager)
	sm.CreateUser("oldname")

	// 创建战斗存档
	oldBattleKey := "oldname" + BattleSaveKeySuffix
	if err := gdataManager.SaveObjectProp(savesObject, oldBattleKey, []byte("battle data")); err != nil {
		t.Fatalf("Failed to create battle save: %v", err)
	}

	// 重命名
	err := sm.RenameUser("oldname", "newname")
	if err != nil {
		t.Fatalf("Failed to rename user: %v", err)
	}

	// 验证战斗存档也被迁移
	if gdataManager.ObjectPropExists(savesObject, oldBattleKey) {
		t.Error("Expected old battle save to be removed")
	}

	newBattleKey := "newname" + BattleSaveKeySuffix
	if !gdataManager.ObjectPropExists(savesObject, newBattleKey) {
		t.Error("Expected new battle save to exist")
	}
}

func TestSaveManager_RenameUser_NotExists(t *testing.T) {
	sm, _ := NewSaveManager(nil)

	err := sm.RenameUser("notexist", "newname")
	if err == nil {
		t.Error("Expected error when renaming non-existent user")
	}
}

func TestSaveManager_RenameUser_TargetExists(t *testing.T) {
	gdataManager := createTestGdataManager(t, "rename_target_exists")
	if gdataManager == nil {
		t.Skip("Cannot create gdata manager for testing")
	}

	sm, _ := NewSaveManager(gdataManager)

	sm.CreateUser("user1")
	sm.CreateUser("user2")

	err := sm.RenameUser("user1", "user2")
	if err == nil {
		t.Error("Expected error when renaming to existing username")
	}
}

func TestSaveManager_DeleteUser(t *testing.T) {
	gdataManager := createTestGdataManager(t, "delete_user")
	if gdataManager == nil {
		t.Skip("Cannot create gdata manager for testing")
	}

	sm, _ := NewSaveManager(gdataManager)

	sm.CreateUser("todelete")

	// 删除用户
	err := sm.DeleteUser("todelete")
	if err != nil {
		t.Fatalf("Failed to delete user: %v", err)
	}

	// 验证用户列表为空
	users, _ := sm.LoadUserList()
	if len(users) != 0 {
		t.Errorf("Expected 0 users after deletion, got %d", len(users))
	}

	// 验证当前用户清空
	if sm.GetCurrentUser() != "" {
		t.Errorf("Expected empty current user after deletion, got %q", sm.GetCurrentUser())
	}

	// 验证 gdata 中存档已删除
	if gdataManager.ObjectPropExists(savesObject, "todelete") {
		t.Error("Expected save data to be removed from gdata")
	}
}

func TestSaveManager_DeleteUser_WithBattleSave(t *testing.T) {
	gdataManager := createTestGdataManager(t, "delete_with_battle")
	if gdataManager == nil {
		t.Skip("Cannot create gdata manager for testing")
	}

	sm, _ := NewSaveManager(gdataManager)
	sm.CreateUser("todelete")

	// 创建战斗存档
	battleKey := "todelete" + BattleSaveKeySuffix
	if err := gdataManager.SaveObjectProp(savesObject, battleKey, []byte("battle data")); err != nil {
		t.Fatalf("Failed to create battle save: %v", err)
	}

	// 删除用户
	err := sm.DeleteUser("todelete")
	if err != nil {
		t.Fatalf("Failed to delete user: %v", err)
	}

	// 验证战斗存档也被删除
	if gdataManager.ObjectPropExists(savesObject, battleKey) {
		t.Error("Expected battle save to be removed")
	}
}

func TestSaveManager_DeleteUser_NotCurrent(t *testing.T) {
	gdataManager := createTestGdataManager(t, "delete_not_current")
	if gdataManager == nil {
		t.Skip("Cannot create gdata manager for testing")
	}

	sm, _ := NewSaveManager(gdataManager)

	sm.CreateUser("user1")
	sm.CreateUser("user2") // current user is now user2

	// 删除非当前用户
	err := sm.DeleteUser("user1")
	if err != nil {
		t.Fatalf("Failed to delete user: %v", err)
	}

	// 当前用户应该不变
	if sm.GetCurrentUser() != "user2" {
		t.Errorf("Expected current user 'user2', got %q", sm.GetCurrentUser())
	}

	users, _ := sm.LoadUserList()
	if len(users) != 1 {
		t.Errorf("Expected 1 user, got %d", len(users))
	}
}

func TestSaveManager_DeleteUser_NotExists(t *testing.T) {
	sm, _ := NewSaveManager(nil)

	err := sm.DeleteUser("notexist")
	if err == nil {
		t.Error("Expected error when deleting non-existent user")
	}
}

func TestSaveManager_SwitchUser(t *testing.T) {
	gdataManager := createTestGdataManager(t, "switch_user")
	if gdataManager == nil {
		t.Skip("Cannot create gdata manager for testing")
	}

	sm, _ := NewSaveManager(gdataManager)

	// 创建两个用户
	sm.CreateUser("user1")
	sm.SetHighestLevel("1-1")
	sm.Save()

	sm.CreateUser("user2")
	sm.SetHighestLevel("2-3")
	sm.Save()

	// 当前应该是 user2
	if sm.GetCurrentUser() != "user2" {
		t.Errorf("Expected current user 'user2', got %q", sm.GetCurrentUser())
	}

	// 切换到 user1
	err := sm.SwitchUser("user1")
	if err != nil {
		t.Fatalf("Failed to switch user: %v", err)
	}

	if sm.GetCurrentUser() != "user1" {
		t.Errorf("Expected current user 'user1', got %q", sm.GetCurrentUser())
	}

	// 验证加载了 user1 的数据
	if sm.GetHighestLevel() != "1-1" {
		t.Errorf("Expected highest level '1-1', got %q", sm.GetHighestLevel())
	}
}

func TestSaveManager_SwitchUser_NotExists(t *testing.T) {
	gdataManager := createTestGdataManager(t, "switch_not_exists")
	if gdataManager == nil {
		t.Skip("Cannot create gdata manager for testing")
	}

	sm, _ := NewSaveManager(gdataManager)

	sm.CreateUser("user1")

	err := sm.SwitchUser("notexist")
	if err == nil {
		t.Error("Expected error when switching to non-existent user")
	}
}

func TestSaveManager_MultipleUsers_DataIsolation(t *testing.T) {
	gdataManager := createTestGdataManager(t, "data_isolation")
	if gdataManager == nil {
		t.Skip("Cannot create gdata manager for testing")
	}

	sm, _ := NewSaveManager(gdataManager)

	// 创建用户1并设置数据
	sm.CreateUser("alice")
	sm.SetHighestLevel("1-5")
	sm.UnlockPlant("peashooter")
	sm.Save()

	// 创建用户2并设置不同数据
	sm.CreateUser("bob")
	sm.SetHighestLevel("2-1")
	sm.UnlockPlant("sunflower")
	sm.Save()

	// 切换回用户1
	sm.SwitchUser("alice")

	// 验证用户1的数据
	if sm.GetHighestLevel() != "1-5" {
		t.Errorf("User alice: Expected highest level '1-5', got %q", sm.GetHighestLevel())
	}

	plants := sm.GetUnlockedPlants()
	if len(plants) != 1 || plants[0] != "peashooter" {
		t.Errorf("User alice: Expected [peashooter], got %v", plants)
	}

	// 切换到用户2
	sm.SwitchUser("bob")

	// 验证用户2的数据
	if sm.GetHighestLevel() != "2-1" {
		t.Errorf("User bob: Expected highest level '2-1', got %q", sm.GetHighestLevel())
	}

	plants = sm.GetUnlockedPlants()
	if len(plants) != 1 || plants[0] != "sunflower" {
		t.Errorf("User bob: Expected [sunflower], got %v", plants)
	}
}

func TestSaveManager_UnlockPlant_Duplicate(t *testing.T) {
	sm, _ := NewSaveManager(nil)

	sm.CreateUser("testuser")

	// 多次解锁同一个植物
	sm.UnlockPlant("peashooter")
	sm.UnlockPlant("peashooter")
	sm.UnlockPlant("peashooter")

	plants := sm.GetUnlockedPlants()
	if len(plants) != 1 {
		t.Errorf("Expected 1 plant after duplicate unlocks, got %d", len(plants))
	}
}

func TestSaveManager_UnlockTool_Duplicate(t *testing.T) {
	sm, _ := NewSaveManager(nil)

	sm.CreateUser("testuser")

	// 多次解锁同一个工具
	sm.UnlockTool("shovel")
	sm.UnlockTool("shovel")

	tools := sm.GetUnlockedTools()
	if len(tools) != 1 {
		t.Errorf("Expected 1 tool after duplicate unlocks, got %d", len(tools))
	}
}

func TestSaveManager_UserListPersistence(t *testing.T) {
	gdataManager := createTestGdataManager(t, "user_persistence")
	if gdataManager == nil {
		t.Skip("Cannot create gdata manager for testing")
	}

	// 创建用户
	sm1, _ := NewSaveManager(gdataManager)
	sm1.CreateUser("persistent")

	// 新建 SaveManager 实例（模拟重启）
	sm2, err := NewSaveManager(gdataManager)
	if err != nil {
		t.Fatalf("Failed to create second SaveManager: %v", err)
	}

	// 验证用户列表被正确加载
	users, _ := sm2.LoadUserList()
	if len(users) != 1 {
		t.Errorf("Expected 1 user, got %d", len(users))
	}

	// 验证当前用户被正确恢复
	if sm2.GetCurrentUser() != "persistent" {
		t.Errorf("Expected current user 'persistent', got %q", sm2.GetCurrentUser())
	}
}

// --- 战斗存档管理方法测试 (Story 18.1, 重构于 Story 20.3) ---

func TestSaveManager_HasBattleSave_NotExists(t *testing.T) {
	gdataManager := createTestGdataManager(t, "has_battle_not_exists")
	if gdataManager == nil {
		t.Skip("Cannot create gdata manager for testing")
	}

	sm, _ := NewSaveManager(gdataManager)

	if sm.HasBattleSave("nonexistent") {
		t.Error("Expected HasBattleSave to return false for non-existent save")
	}
}

func TestSaveManager_HasBattleSave_Exists(t *testing.T) {
	gdataManager := createTestGdataManager(t, "has_battle_exists")
	if gdataManager == nil {
		t.Skip("Cannot create gdata manager for testing")
	}

	sm, _ := NewSaveManager(gdataManager)

	// 创建一个战斗存档
	battleKey := "testuser" + BattleSaveKeySuffix
	if err := gdataManager.SaveObjectProp(savesObject, battleKey, []byte("dummy data")); err != nil {
		t.Fatalf("Failed to create dummy save: %v", err)
	}

	if !sm.HasBattleSave("testuser") {
		t.Error("Expected HasBattleSave to return true for existing save")
	}
}

func TestSaveManager_HasBattleSave_NilGdata(t *testing.T) {
	sm, _ := NewSaveManager(nil)

	// 降级模式下应该返回 false
	if sm.HasBattleSave("testuser") {
		t.Error("Expected HasBattleSave to return false in degraded mode")
	}
}

func TestSaveManager_DeleteBattleSave(t *testing.T) {
	gdataManager := createTestGdataManager(t, "delete_battle")
	if gdataManager == nil {
		t.Skip("Cannot create gdata manager for testing")
	}

	sm, _ := NewSaveManager(gdataManager)

	// 创建战斗存档
	battleKey := "testuser" + BattleSaveKeySuffix
	if err := gdataManager.SaveObjectProp(savesObject, battleKey, []byte("dummy data")); err != nil {
		t.Fatalf("Failed to create save: %v", err)
	}

	// 验证存档存在
	if !sm.HasBattleSave("testuser") {
		t.Fatal("Save should exist before deletion")
	}

	// 删除
	err := sm.DeleteBattleSave("testuser")
	if err != nil {
		t.Fatalf("DeleteBattleSave failed: %v", err)
	}

	// 验证存档已删除
	if sm.HasBattleSave("testuser") {
		t.Error("Save should not exist after deletion")
	}
}

func TestSaveManager_DeleteBattleSave_NotExists(t *testing.T) {
	gdataManager := createTestGdataManager(t, "delete_battle_not_exists")
	if gdataManager == nil {
		t.Skip("Cannot create gdata manager for testing")
	}

	sm, _ := NewSaveManager(gdataManager)

	// 删除不存在的存档应该不返回错误
	err := sm.DeleteBattleSave("nonexistent")
	if err != nil {
		t.Errorf("DeleteBattleSave for non-existent save should not return error: %v", err)
	}
}

func TestSaveManager_GetBattleSaveInfo(t *testing.T) {
	gdataManager := createTestGdataManager(t, "get_battle_info")
	if gdataManager == nil {
		t.Skip("Cannot create gdata manager for testing")
	}

	sm, _ := NewSaveManager(gdataManager)

	// 使用 BattleSerializer 创建有效的存档
	saveData := NewBattleSaveData()
	saveData.LevelID = "1-3"
	saveData.Sun = 200
	saveData.CurrentWaveIndex = 4

	// 手动保存（使用 gob 编码）
	battleKey := "testuser" + BattleSaveKeySuffix
	var buffer bytes.Buffer
	encoder := gob.NewEncoder(&buffer)
	if err := encoder.Encode(saveData); err != nil {
		t.Fatalf("Failed to encode save data: %v", err)
	}
	if err := gdataManager.SaveObjectProp(savesObject, battleKey, buffer.Bytes()); err != nil {
		t.Fatalf("Failed to save battle data: %v", err)
	}

	// 获取存档信息
	info, err := sm.GetBattleSaveInfo("testuser")
	if err != nil {
		t.Fatalf("GetBattleSaveInfo failed: %v", err)
	}

	if info.LevelID != "1-3" {
		t.Errorf("Expected LevelID '1-3', got %q", info.LevelID)
	}
	if info.Sun != 200 {
		t.Errorf("Expected Sun 200, got %d", info.Sun)
	}
	if info.WaveIndex != 4 {
		t.Errorf("Expected WaveIndex 4, got %d", info.WaveIndex)
	}
}

func TestSaveManager_GetBattleSaveInfo_NotExists(t *testing.T) {
	gdataManager := createTestGdataManager(t, "get_info_not_exists")
	if gdataManager == nil {
		t.Skip("Cannot create gdata manager for testing")
	}

	sm, _ := NewSaveManager(gdataManager)

	_, err := sm.GetBattleSaveInfo("nonexistent")
	if err == nil {
		t.Error("Expected error when getting info for non-existent save")
	}
}

func TestSaveManager_GetBattleSaveInfo_NilGdata(t *testing.T) {
	sm, _ := NewSaveManager(nil)

	_, err := sm.GetBattleSaveInfo("testuser")
	if err == nil {
		t.Error("Expected error when getting info with nil gdata manager")
	}
}

func TestSaveManager_GetBattleSaveInfo_Corrupted(t *testing.T) {
	gdataManager := createTestGdataManager(t, "get_info_corrupted")
	if gdataManager == nil {
		t.Skip("Cannot create gdata manager for testing")
	}

	sm, _ := NewSaveManager(gdataManager)

	// 创建损坏的存档
	battleKey := "testuser" + BattleSaveKeySuffix
	if err := gdataManager.SaveObjectProp(savesObject, battleKey, []byte("corrupted data")); err != nil {
		t.Fatalf("Failed to create corrupted save: %v", err)
	}

	_, err := sm.GetBattleSaveInfo("testuser")
	if err == nil {
		t.Error("Expected error when getting info for corrupted save")
	}
}

// --- 游戏开始标记测试 (Story 20.3 QA Fix) ---

func TestSaveManager_GetHasStartedGame_Default(t *testing.T) {
	sm, _ := NewSaveManager(nil)
	sm.CreateUser("testuser")

	// 新用户默认未开始游戏
	if sm.GetHasStartedGame() {
		t.Error("Expected HasStartedGame to be false for new user")
	}
}

func TestSaveManager_SetHasStartedGame(t *testing.T) {
	sm, _ := NewSaveManager(nil)
	sm.CreateUser("testuser")

	// 设置已开始游戏
	sm.SetHasStartedGame()

	if !sm.GetHasStartedGame() {
		t.Error("Expected HasStartedGame to be true after SetHasStartedGame")
	}
}

func TestSaveManager_HasStartedGame_Persistence(t *testing.T) {
	gdataManager := createTestGdataManager(t, "has_started_persistence")
	if gdataManager == nil {
		t.Skip("Cannot create gdata manager for testing")
	}

	// 创建用户并设置已开始游戏
	sm1, _ := NewSaveManager(gdataManager)
	sm1.CreateUser("testuser")
	sm1.SetHasStartedGame()
	sm1.Save()

	// 重新加载
	sm2, _ := NewSaveManager(gdataManager)
	if !sm2.GetHasStartedGame() {
		t.Error("Expected HasStartedGame to persist after reload")
	}
}

func TestSaveManager_HasStartedGame_UserIsolation(t *testing.T) {
	gdataManager := createTestGdataManager(t, "has_started_isolation")
	if gdataManager == nil {
		t.Skip("Cannot create gdata manager for testing")
	}

	sm, _ := NewSaveManager(gdataManager)

	// 创建用户1并设置已开始游戏
	sm.CreateUser("user1")
	sm.SetHasStartedGame()
	sm.Save()

	// 创建用户2（不设置）
	sm.CreateUser("user2")
	sm.Save()

	// 验证用户2未开始游戏
	if sm.GetHasStartedGame() {
		t.Error("User2 should not have started game")
	}

	// 切换回用户1，验证已开始游戏
	sm.SwitchUser("user1")
	if !sm.GetHasStartedGame() {
		t.Error("User1 should have started game")
	}
}

func TestSaveManager_GetNextLevelToPlay(t *testing.T) {
	tests := []struct {
		name          string
		highestLevel  string
		expectedLevel string
	}{
		{
			name:          "新用户，未完成任何关卡",
			highestLevel:  "",
			expectedLevel: "1-1",
		},
		{
			name:          "完成1-1，应返回1-2",
			highestLevel:  "1-1",
			expectedLevel: "1-2",
		},
		{
			name:          "完成1-2，应返回1-3",
			highestLevel:  "1-2",
			expectedLevel: "1-3",
		},
		{
			name:          "完成1-9，应返回1-10",
			highestLevel:  "1-9",
			expectedLevel: "1-10",
		},
		{
			name:          "完成1-10（最后一关），应返回1-10",
			highestLevel:  "1-10",
			expectedLevel: "1-10",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 创建一个新的 SaveManager（使用 nil gdata，纯内存模式）
			sm, _ := NewSaveManager(nil)

			// 设置最高完成关卡
			if tt.highestLevel != "" {
				sm.SetHighestLevel(tt.highestLevel)
			}

			// 获取下一关
			nextLevel := sm.GetNextLevelToPlay()

			if nextLevel != tt.expectedLevel {
				t.Errorf("GetNextLevelToPlay() = %q, want %q (highestLevel: %q)",
					nextLevel, tt.expectedLevel, tt.highestLevel)
			}
		})
	}
}
