package game

import (
	"os"
	"testing"

	"github.com/quasilyte/gdata/v2"
)

// TestDefaultSettings 测试 DefaultSettings() 返回正确的默认值
func TestDefaultSettings(t *testing.T) {
	settings := DefaultSettings()

	if settings == nil {
		t.Fatal("DefaultSettings() returned nil")
	}

	// 验证音乐音量默认值
	if settings.MusicVolume != 0.7 {
		t.Errorf("MusicVolume: got %v, want 0.7", settings.MusicVolume)
	}

	// 验证音效音量默认值
	if settings.SoundVolume != 0.8 {
		t.Errorf("SoundVolume: got %v, want 0.8", settings.SoundVolume)
	}

	// 验证音乐开关默认值
	if !settings.MusicEnabled {
		t.Error("MusicEnabled: got false, want true")
	}

	// 验证音效开关默认值
	if !settings.SoundEnabled {
		t.Error("SoundEnabled: got false, want true")
	}

	// 验证全屏模式默认值
	if settings.Fullscreen {
		t.Error("Fullscreen: got true, want false")
	}
}

// TestNewSettingsManager 测试正常初始化 SettingsManager
func TestNewSettingsManager(t *testing.T) {
	// 使用临时目录创建 gdata manager
	tempDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", originalHome)

	gdataManager, err := gdata.Open(gdata.Config{
		AppName: "test_settings",
	})
	if err != nil {
		t.Fatalf("Failed to create gdata manager: %v", err)
	}

	sm, err := NewSettingsManager(gdataManager)
	if err != nil {
		t.Fatalf("NewSettingsManager() error: %v", err)
	}

	if sm == nil {
		t.Fatal("NewSettingsManager() returned nil")
	}

	// 验证初始化后使用默认设置
	settings := sm.GetSettings()
	if settings == nil {
		t.Fatal("GetSettings() returned nil after initialization")
	}

	if settings.MusicVolume != 0.7 {
		t.Errorf("Initial MusicVolume: got %v, want 0.7", settings.MusicVolume)
	}
}

// TestNewSettingsManagerNilGdata 测试 gdataManager 为 nil 时的降级场景
func TestNewSettingsManagerNilGdata(t *testing.T) {
	sm, err := NewSettingsManager(nil)
	if err != nil {
		t.Fatalf("NewSettingsManager(nil) error: %v", err)
	}

	if sm == nil {
		t.Fatal("NewSettingsManager(nil) returned nil")
	}

	// 验证使用默认设置
	settings := sm.GetSettings()
	if settings == nil {
		t.Fatal("GetSettings() returned nil in degraded mode")
	}

	if settings.MusicVolume != 0.7 {
		t.Errorf("Degraded mode MusicVolume: got %v, want 0.7", settings.MusicVolume)
	}
}

// TestSettingsLoadSave 测试 Load() 和 Save() 功能
func TestSettingsLoadSave(t *testing.T) {
	// 使用临时目录创建 gdata manager
	tempDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", originalHome)

	gdataManager, err := gdata.Open(gdata.Config{
		AppName: "test_settings_load_save",
	})
	if err != nil {
		t.Fatalf("Failed to create gdata manager: %v", err)
	}

	// 创建设置管理器并修改设置
	sm1, err := NewSettingsManager(gdataManager)
	if err != nil {
		t.Fatalf("NewSettingsManager() error: %v", err)
	}

	sm1.SetMusicVolume(0.5)
	sm1.SetSoundVolume(0.6)
	sm1.SetMusicEnabled(false)
	sm1.SetSoundEnabled(false)
	sm1.SetFullscreen(true)

	// 保存设置
	if err := sm1.Save(); err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	// 创建新的设置管理器，验证加载
	sm2, err := NewSettingsManager(gdataManager)
	if err != nil {
		t.Fatalf("NewSettingsManager() error on reload: %v", err)
	}

	settings := sm2.GetSettings()

	if settings.MusicVolume != 0.5 {
		t.Errorf("Loaded MusicVolume: got %v, want 0.5", settings.MusicVolume)
	}

	if settings.SoundVolume != 0.6 {
		t.Errorf("Loaded SoundVolume: got %v, want 0.6", settings.SoundVolume)
	}

	if settings.MusicEnabled {
		t.Error("Loaded MusicEnabled: got true, want false")
	}

	if settings.SoundEnabled {
		t.Error("Loaded SoundEnabled: got true, want false")
	}

	if !settings.Fullscreen {
		t.Error("Loaded Fullscreen: got false, want true")
	}
}

// TestSetMusicVolumeClamp 测试 SetMusicVolume 范围校验
func TestSetMusicVolumeClamp(t *testing.T) {
	sm, _ := NewSettingsManager(nil)

	tests := []struct {
		input    float64
		expected float64
	}{
		{0.5, 0.5},   // 正常值
		{0.0, 0.0},   // 下限
		{1.0, 1.0},   // 上限
		{-0.5, 0.0},  // 低于下限，应 clamp 到 0.0
		{1.5, 1.0},   // 高于上限，应 clamp 到 1.0
		{-100, 0.0},  // 极小值
		{100, 1.0},   // 极大值
	}

	for _, tt := range tests {
		sm.SetMusicVolume(tt.input)
		if sm.GetSettings().MusicVolume != tt.expected {
			t.Errorf("SetMusicVolume(%v): got %v, want %v",
				tt.input, sm.GetSettings().MusicVolume, tt.expected)
		}
	}
}

// TestSetSoundVolumeClamp 测试 SetSoundVolume 范围校验
func TestSetSoundVolumeClamp(t *testing.T) {
	sm, _ := NewSettingsManager(nil)

	tests := []struct {
		input    float64
		expected float64
	}{
		{0.5, 0.5},   // 正常值
		{0.0, 0.0},   // 下限
		{1.0, 1.0},   // 上限
		{-0.5, 0.0},  // 低于下限，应 clamp 到 0.0
		{1.5, 1.0},   // 高于上限，应 clamp 到 1.0
		{-100, 0.0},  // 极小值
		{100, 1.0},   // 极大值
	}

	for _, tt := range tests {
		sm.SetSoundVolume(tt.input)
		if sm.GetSettings().SoundVolume != tt.expected {
			t.Errorf("SetSoundVolume(%v): got %v, want %v",
				tt.input, sm.GetSettings().SoundVolume, tt.expected)
		}
	}
}

// TestSetMusicEnabled 测试 SetMusicEnabled 功能
func TestSetMusicEnabled(t *testing.T) {
	sm, _ := NewSettingsManager(nil)

	// 默认为 true
	if !sm.GetSettings().MusicEnabled {
		t.Error("Initial MusicEnabled: got false, want true")
	}

	// 设置为 false
	sm.SetMusicEnabled(false)
	if sm.GetSettings().MusicEnabled {
		t.Error("After SetMusicEnabled(false): got true, want false")
	}

	// 设置为 true
	sm.SetMusicEnabled(true)
	if !sm.GetSettings().MusicEnabled {
		t.Error("After SetMusicEnabled(true): got false, want true")
	}
}

// TestSetSoundEnabled 测试 SetSoundEnabled 功能
func TestSetSoundEnabled(t *testing.T) {
	sm, _ := NewSettingsManager(nil)

	// 默认为 true
	if !sm.GetSettings().SoundEnabled {
		t.Error("Initial SoundEnabled: got false, want true")
	}

	// 设置为 false
	sm.SetSoundEnabled(false)
	if sm.GetSettings().SoundEnabled {
		t.Error("After SetSoundEnabled(false): got true, want false")
	}

	// 设置为 true
	sm.SetSoundEnabled(true)
	if !sm.GetSettings().SoundEnabled {
		t.Error("After SetSoundEnabled(true): got false, want true")
	}
}

// TestSetFullscreen 测试 SetFullscreen 功能
func TestSetFullscreen(t *testing.T) {
	sm, _ := NewSettingsManager(nil)

	// 默认为 false
	if sm.GetSettings().Fullscreen {
		t.Error("Initial Fullscreen: got true, want false")
	}

	// 设置为 true
	sm.SetFullscreen(true)
	if !sm.GetSettings().Fullscreen {
		t.Error("After SetFullscreen(true): got false, want true")
	}

	// 设置为 false
	sm.SetFullscreen(false)
	if sm.GetSettings().Fullscreen {
		t.Error("After SetFullscreen(false): got true, want false")
	}
}

// TestGetSettings 测试 GetSettings() 返回正确实例
func TestGetSettings(t *testing.T) {
	sm, _ := NewSettingsManager(nil)

	settings1 := sm.GetSettings()
	settings2 := sm.GetSettings()

	// 应该返回相同的实例
	if settings1 != settings2 {
		t.Error("GetSettings() should return the same instance")
	}

	// 修改 settings1，settings2 也应该改变（同一实例）
	settings1.MusicVolume = 0.1
	if settings2.MusicVolume != 0.1 {
		t.Error("Settings should be the same instance")
	}
}

// TestSaveNilGdataManager 测试降级模式下 Save() 不报错
func TestSaveNilGdataManager(t *testing.T) {
	sm, _ := NewSettingsManager(nil)

	// 降级模式下 Save() 应该返回 nil（不报错）
	err := sm.Save()
	if err != nil {
		t.Errorf("Save() in degraded mode should return nil, got: %v", err)
	}
}

// TestLoadNilGdataManager 测试降级模式下 Load() 使用默认设置
func TestLoadNilGdataManager(t *testing.T) {
	sm, _ := NewSettingsManager(nil)

	// 修改设置
	sm.SetMusicVolume(0.3)

	// 重新 Load()
	err := sm.Load()
	if err != nil {
		t.Errorf("Load() in degraded mode should return nil, got: %v", err)
	}

	// 应该恢复为默认值
	if sm.GetSettings().MusicVolume != 0.7 {
		t.Errorf("After Load() in degraded mode, MusicVolume: got %v, want 0.7",
			sm.GetSettings().MusicVolume)
	}
}

// TestClampVolume 测试 clampVolume 辅助函数
func TestClampVolume(t *testing.T) {
	tests := []struct {
		input    float64
		expected float64
	}{
		{0.5, 0.5},
		{0.0, 0.0},
		{1.0, 1.0},
		{-1.0, 0.0},
		{2.0, 1.0},
		{0.001, 0.001},
		{0.999, 0.999},
	}

	for _, tt := range tests {
		result := clampVolume(tt.input)
		if result != tt.expected {
			t.Errorf("clampVolume(%v): got %v, want %v", tt.input, result, tt.expected)
		}
	}
}

// TestGameStateSettingsManager 测试 GameState 集成 SettingsManager
// 注意：这个测试需要重置全局 GameState
func TestGameStateSettingsManager(t *testing.T) {
	// 保存原始的 globalGameState
	originalGameState := globalGameState
	defer func() {
		globalGameState = originalGameState
	}()

	// 重置全局状态以测试初始化
	globalGameState = nil

	// 使用临时目录
	tempDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", originalHome)

	// 注意：SaveManager 现在使用 gdata API，不再需要 data/saves 目录

	// 获取 GameState（触发初始化）
	gs := GetGameState()
	if gs == nil {
		t.Fatal("GetGameState() returned nil")
	}

	// 获取 SettingsManager
	sm := gs.GetSettingsManager()
	// SettingsManager 可能为 nil（如果 gdata 初始化失败），但不应该阻止游戏运行
	// 在正常情况下应该不为 nil
	if sm == nil {
		t.Log("Note: SettingsManager is nil (gdata may have failed to initialize)")
		return
	}

	// 验证 SettingsManager 功能正常
	settings := sm.GetSettings()
	if settings == nil {
		t.Fatal("GetSettings() returned nil")
	}

	// 测试修改设置
	sm.SetMusicVolume(0.5)
	if sm.GetSettings().MusicVolume != 0.5 {
		t.Errorf("SetMusicVolume(0.5): got %v, want 0.5", sm.GetSettings().MusicVolume)
	}
}

