package game

import (
	"sort"
	"testing"
)

// TestNewPlantUnlockManager 测试植物解锁管理器的初始化
func TestNewPlantUnlockManager(t *testing.T) {
	manager := NewPlantUnlockManager()

	if manager == nil {
		t.Fatal("NewPlantUnlockManager() returned nil")
	}

	// 验证默认解锁植物数量
	unlockedPlants := manager.GetUnlockedPlants()
	if len(unlockedPlants) == 0 {
		t.Error("Expected default unlocked plants, got empty list")
	}

	// 验证第一章基础植物已解锁
	expectedPlants := []string{"peashooter", "sunflower", "cherrybomb", "wallnut"}
	for _, plantID := range expectedPlants {
		if !manager.IsUnlocked(plantID) {
			t.Errorf("Expected plant %s to be unlocked by default", plantID)
		}
	}
}

// TestIsUnlocked 测试植物解锁状态检查
func TestIsUnlocked(t *testing.T) {
	manager := NewPlantUnlockManager()

	tests := []struct {
		name     string
		plantID  string
		expected bool
	}{
		{
			name:     "已解锁植物：豌豆射手",
			plantID:  "peashooter",
			expected: true,
		},
		{
			name:     "已解锁植物：向日葵",
			plantID:  "sunflower",
			expected: true,
		},
		{
			name:     "未解锁植物：不存在的ID",
			plantID:  "nonexistent_plant",
			expected: false,
		},
		{
			name:     "空字符串",
			plantID:  "",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := manager.IsUnlocked(tt.plantID)
			if result != tt.expected {
				t.Errorf("IsUnlocked(%s) = %v, want %v", tt.plantID, result, tt.expected)
			}
		})
	}
}

// TestUnlockPlant 测试植物解锁功能
func TestUnlockPlant(t *testing.T) {
	manager := NewPlantUnlockManager()

	// 解锁一个新植物
	newPlantID := "threepeater"
	if manager.IsUnlocked(newPlantID) {
		t.Errorf("Plant %s should not be unlocked initially", newPlantID)
	}

	// 执行解锁
	manager.UnlockPlant(newPlantID)

	// 验证解锁成功
	if !manager.IsUnlocked(newPlantID) {
		t.Errorf("Plant %s should be unlocked after UnlockPlant()", newPlantID)
	}

	// 重复解锁应该不产生错误
	manager.UnlockPlant(newPlantID)
	if !manager.IsUnlocked(newPlantID) {
		t.Errorf("Plant %s should still be unlocked after duplicate UnlockPlant()", newPlantID)
	}
}

// TestGetUnlockedPlants 测试获取已解锁植物列表
func TestGetUnlockedPlants(t *testing.T) {
	manager := NewPlantUnlockManager()

	// 获取初始解锁列表
	initialPlants := manager.GetUnlockedPlants()
	if len(initialPlants) == 0 {
		t.Fatal("GetUnlockedPlants() returned empty list")
	}

	// 验证列表已排序（字母顺序）
	sortedPlants := make([]string, len(initialPlants))
	copy(sortedPlants, initialPlants)
	sort.Strings(sortedPlants)

	for i := range initialPlants {
		if initialPlants[i] != sortedPlants[i] {
			t.Errorf("GetUnlockedPlants() is not sorted. Got %v, want %v", initialPlants, sortedPlants)
			break
		}
	}

	// 解锁新植物，验证列表更新
	newPlantID := "squash"
	initialCount := len(initialPlants)
	manager.UnlockPlant(newPlantID)

	updatedPlants := manager.GetUnlockedPlants()
	if len(updatedPlants) != initialCount+1 {
		t.Errorf("Expected %d plants after unlock, got %d", initialCount+1, len(updatedPlants))
	}

	// 验证新植物在列表中
	found := false
	for _, plantID := range updatedPlants {
		if plantID == newPlantID {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Newly unlocked plant %s not found in GetUnlockedPlants()", newPlantID)
	}
}

// TestGetUnlockedPlants_ReturnsCopy 测试返回值是副本，不影响内部状态
func TestGetUnlockedPlants_ReturnsCopy(t *testing.T) {
	manager := NewPlantUnlockManager()

	plants1 := manager.GetUnlockedPlants()
	plants2 := manager.GetUnlockedPlants()

	// 验证是不同的切片实例
	if len(plants1) > 0 {
		plants1[0] = "modified"
	}

	// 验证修改不影响后续调用
	if len(plants2) > 0 && plants2[0] == "modified" {
		t.Error("GetUnlockedPlants() should return a copy, not a reference")
	}
}

// TestLoadFromSave_Placeholder 测试 LoadFromSave 预留方法
func TestLoadFromSave_Placeholder(t *testing.T) {
	manager := NewPlantUnlockManager()

	// 当前应该返回 nil（预留实现）
	err := manager.LoadFromSave()
	if err != nil {
		t.Errorf("LoadFromSave() should return nil (placeholder), got error: %v", err)
	}
}

// TestSaveToFile_Placeholder 测试 SaveToFile 预留方法
func TestSaveToFile_Placeholder(t *testing.T) {
	manager := NewPlantUnlockManager()

	// 当前应该返回 nil（预留实现）
	err := manager.SaveToFile()
	if err != nil {
		t.Errorf("SaveToFile() should return nil (placeholder), got error: %v", err)
	}
}

// TestUnlockMultiplePlants 测试批量解锁多个植物
func TestUnlockMultiplePlants(t *testing.T) {
	manager := NewPlantUnlockManager()

	newPlants := []string{"plant1", "plant2", "plant3"}

	// 解锁多个植物
	for _, plantID := range newPlants {
		manager.UnlockPlant(plantID)
	}

	// 验证所有植物都已解锁
	for _, plantID := range newPlants {
		if !manager.IsUnlocked(plantID) {
			t.Errorf("Plant %s should be unlocked", plantID)
		}
	}

	// 验证列表包含所有新植物
	allPlants := manager.GetUnlockedPlants()
	for _, plantID := range newPlants {
		found := false
		for _, unlocked := range allPlants {
			if unlocked == plantID {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Plant %s not found in unlocked plants list", plantID)
		}
	}
}

