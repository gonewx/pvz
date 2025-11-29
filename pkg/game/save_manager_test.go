package game

import (
	"encoding/gob"
	"os"
	"path/filepath"
	"testing"
)

func TestSaveManager_NewGame(t *testing.T) {
	// 创建临时目录
	tempDir := t.TempDir()

	// 创建SaveManager
	sm, err := NewSaveManager(tempDir)
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

func TestSaveManager_CreateUser(t *testing.T) {
	tempDir := t.TempDir()
	sm, err := NewSaveManager(tempDir)
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

	// 验证存档文件创建
	saveFile := filepath.Join(tempDir, "player1.yaml")
	if _, err := os.Stat(saveFile); os.IsNotExist(err) {
		t.Error("Expected save file to exist")
	}

	// 验证用户列表文件创建
	userListFile := filepath.Join(tempDir, "users.yaml")
	if _, err := os.Stat(userListFile); os.IsNotExist(err) {
		t.Error("Expected user list file to exist")
	}
}

func TestSaveManager_CreateUser_DuplicateName(t *testing.T) {
	tempDir := t.TempDir()
	sm, _ := NewSaveManager(tempDir)

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
	tempDir := t.TempDir()
	sm, _ := NewSaveManager(tempDir)

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
	// 创建临时目录
	tempDir := t.TempDir()

	// 创建SaveManager并创建用户
	sm1, err := NewSaveManager(tempDir)
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

	// 创建新的SaveManager加载数据
	sm2, err := NewSaveManager(tempDir)
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
	tempDir := t.TempDir()
	sm, _ := NewSaveManager(tempDir)

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

	// 验证旧存档文件删除，新文件创建
	oldFile := filepath.Join(tempDir, "oldname.yaml")
	if _, err := os.Stat(oldFile); !os.IsNotExist(err) {
		t.Error("Expected old save file to be removed")
	}

	newFile := filepath.Join(tempDir, "newname.yaml")
	if _, err := os.Stat(newFile); os.IsNotExist(err) {
		t.Error("Expected new save file to exist")
	}
}

func TestSaveManager_RenameUser_NotExists(t *testing.T) {
	tempDir := t.TempDir()
	sm, _ := NewSaveManager(tempDir)

	err := sm.RenameUser("notexist", "newname")
	if err == nil {
		t.Error("Expected error when renaming non-existent user")
	}
}

func TestSaveManager_RenameUser_TargetExists(t *testing.T) {
	tempDir := t.TempDir()
	sm, _ := NewSaveManager(tempDir)

	sm.CreateUser("user1")
	sm.CreateUser("user2")

	err := sm.RenameUser("user1", "user2")
	if err == nil {
		t.Error("Expected error when renaming to existing username")
	}
}

func TestSaveManager_DeleteUser(t *testing.T) {
	tempDir := t.TempDir()
	sm, _ := NewSaveManager(tempDir)

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

	// 验证存档文件删除
	saveFile := filepath.Join(tempDir, "todelete.yaml")
	if _, err := os.Stat(saveFile); !os.IsNotExist(err) {
		t.Error("Expected save file to be removed")
	}
}

func TestSaveManager_DeleteUser_NotCurrent(t *testing.T) {
	tempDir := t.TempDir()
	sm, _ := NewSaveManager(tempDir)

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
	tempDir := t.TempDir()
	sm, _ := NewSaveManager(tempDir)

	err := sm.DeleteUser("notexist")
	if err == nil {
		t.Error("Expected error when deleting non-existent user")
	}
}

func TestSaveManager_SwitchUser(t *testing.T) {
	tempDir := t.TempDir()
	sm, _ := NewSaveManager(tempDir)

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
	tempDir := t.TempDir()
	sm, _ := NewSaveManager(tempDir)

	sm.CreateUser("user1")

	err := sm.SwitchUser("notexist")
	if err == nil {
		t.Error("Expected error when switching to non-existent user")
	}
}

func TestSaveManager_MultipleUsers_DataIsolation(t *testing.T) {
	tempDir := t.TempDir()
	sm, _ := NewSaveManager(tempDir)

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
	tempDir := t.TempDir()
	sm, _ := NewSaveManager(tempDir)

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
	tempDir := t.TempDir()
	sm, _ := NewSaveManager(tempDir)

	sm.CreateUser("testuser")

	// 多次解锁同一个工具
	sm.UnlockTool("shovel")
	sm.UnlockTool("shovel")

	tools := sm.GetUnlockedTools()
	if len(tools) != 1 {
		t.Errorf("Expected 1 tool after duplicate unlocks, got %d", len(tools))
	}
}

func TestSaveManager_LoadCorruptedUserList(t *testing.T) {
	tempDir := t.TempDir()

	// 创建损坏的 users.yaml 文件
	corruptedFile := filepath.Join(tempDir, "users.yaml")
	if err := os.WriteFile(corruptedFile, []byte("invalid: yaml: content: ["), 0644); err != nil {
		t.Fatalf("Failed to write corrupted file: %v", err)
	}

	// 尝试加载
	_, err := NewSaveManager(tempDir)
	if err == nil {
		t.Error("Expected error when loading corrupted user list file, got nil")
	}
}

func TestSaveManager_UserListPersistence(t *testing.T) {
	tempDir := t.TempDir()

	// 创建用户
	sm1, _ := NewSaveManager(tempDir)
	sm1.CreateUser("persistent")

	// 新建 SaveManager 实例
	sm2, err := NewSaveManager(tempDir)
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

// --- 战斗存档管理方法测试 (Story 18.1) ---

func TestSaveManager_GetBattleSavePath(t *testing.T) {
	tempDir := t.TempDir()
	sm, _ := NewSaveManager(tempDir)

	path := sm.GetBattleSavePath("testuser")
	expected := filepath.Join(tempDir, "testuser"+BattleSaveFileSuffix)

	if path != expected {
		t.Errorf("Expected path %q, got %q", expected, path)
	}
}

func TestSaveManager_HasBattleSave_NotExists(t *testing.T) {
	tempDir := t.TempDir()
	sm, _ := NewSaveManager(tempDir)

	if sm.HasBattleSave("nonexistent") {
		t.Error("Expected HasBattleSave to return false for non-existent save")
	}
}

func TestSaveManager_HasBattleSave_Exists(t *testing.T) {
	tempDir := t.TempDir()
	sm, _ := NewSaveManager(tempDir)

	// 创建一个空的战斗存档文件
	battleSavePath := sm.GetBattleSavePath("testuser")
	if err := os.WriteFile(battleSavePath, []byte("dummy data"), 0644); err != nil {
		t.Fatalf("Failed to create dummy save file: %v", err)
	}

	if !sm.HasBattleSave("testuser") {
		t.Error("Expected HasBattleSave to return true for existing save")
	}
}

func TestSaveManager_DeleteBattleSave(t *testing.T) {
	tempDir := t.TempDir()
	sm, _ := NewSaveManager(tempDir)

	// 创建战斗存档文件
	battleSavePath := sm.GetBattleSavePath("testuser")
	if err := os.WriteFile(battleSavePath, []byte("dummy data"), 0644); err != nil {
		t.Fatalf("Failed to create save file: %v", err)
	}

	// 验证文件存在
	if !sm.HasBattleSave("testuser") {
		t.Fatal("Save file should exist before deletion")
	}

	// 删除
	err := sm.DeleteBattleSave("testuser")
	if err != nil {
		t.Fatalf("DeleteBattleSave failed: %v", err)
	}

	// 验证文件已删除
	if sm.HasBattleSave("testuser") {
		t.Error("Save file should not exist after deletion")
	}
}

func TestSaveManager_DeleteBattleSave_NotExists(t *testing.T) {
	tempDir := t.TempDir()
	sm, _ := NewSaveManager(tempDir)

	// 删除不存在的文件应该不返回错误
	err := sm.DeleteBattleSave("nonexistent")
	if err != nil {
		t.Errorf("DeleteBattleSave for non-existent file should not return error: %v", err)
	}
}

func TestSaveManager_GetBattleSaveInfo(t *testing.T) {
	tempDir := t.TempDir()
	sm, _ := NewSaveManager(tempDir)

	// 使用 BattleSerializer 创建有效的存档
	serializer := NewBattleSerializer()
	saveData := NewBattleSaveData()
	saveData.LevelID = "1-3"
	saveData.Sun = 200
	saveData.CurrentWaveIndex = 4

	// 手动保存（模拟完整保存流程）
	battleSavePath := sm.GetBattleSavePath("testuser")
	file, err := os.Create(battleSavePath)
	if err != nil {
		t.Fatalf("Failed to create save file: %v", err)
	}
	defer file.Close()

	// 用于测试的简单序列化（直接使用 gob）
	_ = serializer
	encoder := gob.NewEncoder(file)
	if err := encoder.Encode(saveData); err != nil {
		t.Fatalf("Failed to encode save data: %v", err)
	}
	file.Close() // 确保写入完成

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
	tempDir := t.TempDir()
	sm, _ := NewSaveManager(tempDir)

	_, err := sm.GetBattleSaveInfo("nonexistent")
	if err == nil {
		t.Error("Expected error when getting info for non-existent save")
	}
}

func TestSaveManager_GetBattleSaveInfo_Corrupted(t *testing.T) {
	tempDir := t.TempDir()
	sm, _ := NewSaveManager(tempDir)

	// 创建损坏的存档文件
	battleSavePath := sm.GetBattleSavePath("testuser")
	if err := os.WriteFile(battleSavePath, []byte("corrupted data"), 0644); err != nil {
		t.Fatalf("Failed to create corrupted file: %v", err)
	}

	_, err := sm.GetBattleSaveInfo("testuser")
	if err == nil {
		t.Error("Expected error when getting info for corrupted save")
	}
}
