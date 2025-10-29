package game

import (
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

	// 验证初始状态
	if sm.GetHighestLevel() != "" {
		t.Errorf("Expected empty highest level, got %q", sm.GetHighestLevel())
	}

	if len(sm.GetUnlockedPlants()) != 0 {
		t.Errorf("Expected 0 unlocked plants, got %d", len(sm.GetUnlockedPlants()))
	}

	if len(sm.GetUnlockedTools()) != 0 {
		t.Errorf("Expected 0 unlocked tools, got %d", len(sm.GetUnlockedTools()))
	}
}

func TestSaveManager_SaveAndLoad(t *testing.T) {
	// 创建临时目录
	tempDir := t.TempDir()

	// 创建SaveManager并修改数据
	sm1, err := NewSaveManager(tempDir)
	if err != nil {
		t.Fatalf("Failed to create SaveManager: %v", err)
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

func TestSaveManager_UnlockPlant_Duplicate(t *testing.T) {
	tempDir := t.TempDir()
	sm, _ := NewSaveManager(tempDir)

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

	// 多次解锁同一个工具
	sm.UnlockTool("shovel")
	sm.UnlockTool("shovel")

	tools := sm.GetUnlockedTools()
	if len(tools) != 1 {
		t.Errorf("Expected 1 tool after duplicate unlocks, got %d", len(tools))
	}
}

func TestSaveManager_LoadCorruptedFile(t *testing.T) {
	tempDir := t.TempDir()

	// 创建损坏的 YAML 文件
	corruptedFile := filepath.Join(tempDir, "progress.yaml")
	if err := os.WriteFile(corruptedFile, []byte("invalid: yaml: content: ["), 0644); err != nil {
		t.Fatalf("Failed to write corrupted file: %v", err)
	}

	// 尝试加载
	_, err := NewSaveManager(tempDir)
	if err == nil {
		t.Error("Expected error when loading corrupted file, got nil")
	}
}
