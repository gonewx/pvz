package game

import (
	"fmt"
	"log"

	"github.com/quasilyte/gdata/v2"
	"gopkg.in/yaml.v3"
)

// GameSettings 全局游戏设置
// 注意：这些设置是全局的，不绑定到特定用户（与原版 PvZ 一致）
type GameSettings struct {
	// 音频设置
	MusicVolume  float64 `yaml:"musicVolume"`  // 音乐音量 0.0 ~ 1.0
	SoundVolume  float64 `yaml:"soundVolume"`  // 音效音量 0.0 ~ 1.0
	MusicEnabled bool    `yaml:"musicEnabled"` // 音乐开关
	SoundEnabled bool    `yaml:"soundEnabled"` // 音效开关

	// 显示设置
	Fullscreen bool `yaml:"fullscreen"` // 启动时是否全屏
}

// DefaultSettings 返回默认设置
func DefaultSettings() *GameSettings {
	return &GameSettings{
		MusicVolume:  0.7,
		SoundVolume:  0.8,
		MusicEnabled: true,
		SoundEnabled: true,
		Fullscreen:   false,
	}
}

// SettingsManager 设置管理器
// 负责游戏设置的加载、保存和内存管理
type SettingsManager struct {
	gdataManager *gdata.Manager // gdata 跨平台存储管理器，可为 nil（降级模式）
	settings     *GameSettings  // 当前设置
}

// 存储路径常量
const (
	settingsObject   = "settings"
	settingsProperty = "global"
)

// NewSettingsManager 创建新的设置管理器实例
//
// 参数：
//   - gdataManager: gdata 跨平台存储管理器，可为 nil（降级模式，仅内存设置）
//
// 返回：
//   - *SettingsManager: 设置管理器实例
//   - error: 如果加载设置失败返回错误（不影响创建）
func NewSettingsManager(gdataManager *gdata.Manager) (*SettingsManager, error) {
	sm := &SettingsManager{
		gdataManager: gdataManager,
		settings:     DefaultSettings(),
	}

	// 尝试加载已保存的设置
	if err := sm.Load(); err != nil {
		// 加载失败不是致命错误，使用默认设置
		log.Printf("[SettingsManager] Warning: Failed to load settings: %v (using defaults)", err)
	}

	return sm, nil
}

// Load 从 gdata 加载设置
//
// 如果 gdataManager 为 nil 或文件不存在，使用默认设置
//
// 返回：
//   - error: 如果反序列化失败返回错误
func (sm *SettingsManager) Load() error {
	// 降级模式：无法持久化，使用默认设置
	if sm.gdataManager == nil {
		sm.settings = DefaultSettings()
		return nil
	}

	// 检查设置文件是否存在
	if !sm.gdataManager.ObjectPropExists(settingsObject, settingsProperty) {
		// 文件不存在，使用默认设置
		sm.settings = DefaultSettings()
		return nil
	}

	// 从 gdata 加载数据
	data, err := sm.gdataManager.LoadObjectProp(settingsObject, settingsProperty)
	if err != nil {
		// 文件存在但加载失败，使用默认设置
		sm.settings = DefaultSettings()
		return fmt.Errorf("failed to load settings: %w", err)
	}

	// 反序列化 YAML 数据
	var loadedSettings GameSettings
	if err := yaml.Unmarshal(data, &loadedSettings); err != nil {
		sm.settings = DefaultSettings()
		return fmt.Errorf("failed to unmarshal settings: %w", err)
	}

	sm.settings = &loadedSettings
	log.Printf("[SettingsManager] Settings loaded successfully")
	return nil
}

// Save 保存设置到 gdata
//
// 如果 gdataManager 为 nil，返回 nil（降级模式，不报错）
//
// 返回：
//   - error: 如果序列化或保存失败返回错误
func (sm *SettingsManager) Save() error {
	// 降级模式：无法持久化，但不报错
	if sm.gdataManager == nil {
		return nil
	}

	// 序列化设置为 YAML
	data, err := yaml.Marshal(sm.settings)
	if err != nil {
		return fmt.Errorf("failed to marshal settings: %w", err)
	}

	// 保存到 gdata
	if err := sm.gdataManager.SaveObjectProp(settingsObject, settingsProperty, data); err != nil {
		return fmt.Errorf("failed to save settings: %w", err)
	}

	log.Printf("[SettingsManager] Settings saved successfully")
	return nil
}

// GetSettings 获取当前设置
//
// 返回：
//   - *GameSettings: 当前设置实例
func (sm *SettingsManager) GetSettings() *GameSettings {
	return sm.settings
}

// SetMusicVolume 设置音乐音量
//
// 音量值会被限制在 0.0 ~ 1.0 范围内
// 注意：仅修改内存中的设置，需调用 Save() 方法持久化
//
// 参数：
//   - volume: 音乐音量 (0.0 ~ 1.0)
func (sm *SettingsManager) SetMusicVolume(volume float64) {
	sm.settings.MusicVolume = clampVolume(volume)
}

// SetSoundVolume 设置音效音量
//
// 音量值会被限制在 0.0 ~ 1.0 范围内
// 注意：仅修改内存中的设置，需调用 Save() 方法持久化
//
// 参数：
//   - volume: 音效音量 (0.0 ~ 1.0)
func (sm *SettingsManager) SetSoundVolume(volume float64) {
	sm.settings.SoundVolume = clampVolume(volume)
}

// SetMusicEnabled 设置音乐开关
//
// 注意：仅修改内存中的设置，需调用 Save() 方法持久化
//
// 参数：
//   - enabled: 是否启用音乐
func (sm *SettingsManager) SetMusicEnabled(enabled bool) {
	sm.settings.MusicEnabled = enabled
}

// SetSoundEnabled 设置音效开关
//
// 注意：仅修改内存中的设置，需调用 Save() 方法持久化
//
// 参数：
//   - enabled: 是否启用音效
func (sm *SettingsManager) SetSoundEnabled(enabled bool) {
	sm.settings.SoundEnabled = enabled
}

// SetFullscreen 设置全屏模式
//
// 注意：仅修改内存中的设置，需调用 Save() 方法持久化
//
// 参数：
//   - enabled: 是否启用全屏
func (sm *SettingsManager) SetFullscreen(enabled bool) {
	sm.settings.Fullscreen = enabled
}

// clampVolume 将音量值限制在 0.0 ~ 1.0 范围内
func clampVolume(volume float64) float64 {
	if volume < 0.0 {
		return 0.0
	}
	if volume > 1.0 {
		return 1.0
	}
	return volume
}

