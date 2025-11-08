package config

import (
	"fmt"
	"os"
	"sync"

	"gopkg.in/yaml.v3"
)

// GameReanimConfig 游戏 Reanim 配置文件的顶层结构
// 与 animation_showcase 的 ShowcaseConfig 结构相同
type GameReanimConfig struct {
	Global     GlobalConfig          `yaml:"global"`
	Animations []AnimationUnitConfig `yaml:"animations"`
}

// GlobalConfig 全局配置
type GlobalConfig struct {
	Playback PlaybackConfig `yaml:"playback"`
}

// PlaybackConfig 播放配置
type PlaybackConfig struct {
	TPS int `yaml:"tps"` // 游戏目标 TPS
	FPS int `yaml:"fps"` // 默认动画帧率
}

// AnimationUnitConfig 动画单元配置
// 与 animation_showcase 的结构完全一致
type AnimationUnitConfig struct {
	ID                  string                   `yaml:"id"`
	Name                string                   `yaml:"name"`
	ReanimFile          string                   `yaml:"reanim_file"`
	DefaultAnimation    string                   `yaml:"default_animation"`
	Images              map[string]string        `yaml:"images"`
	AvailableAnimations []AnimationInfo          `yaml:"available_animations"`
	AnimationCombos     []AnimationComboConfig   `yaml:"animation_combos"`
}

// AnimationInfo 动画信息
type AnimationInfo struct {
	Name        string `yaml:"name"`
	DisplayName string `yaml:"display_name"`
}

// 注意：AnimationComboConfig 已在 reanim_config.go 中定义，这里直接使用

// ReanimConfigManager Reanim 配置管理器
// 负责加载和管理全量 Reanim 配置
type ReanimConfigManager struct {
	config  *GameReanimConfig              // 全量配置
	unitMap map[string]*AnimationUnitConfig // 按 id 索引的配置映射
	mu      sync.RWMutex                   // 读写锁（并发安全）
}

// NewReanimConfigManager 创建配置管理器
//
// 参数：
//   - configPath: 配置文件路径（如 "data/reanim_config.yaml"）
//
// 返回：
//   - *ReanimConfigManager: 配置管理器实例
//   - error: 加载或解析错误
func NewReanimConfigManager(configPath string) (*ReanimConfigManager, error) {
	// 1. 读取配置文件
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("无法读取配置文件 %s: %w", configPath, err)
	}

	// 2. 解析 YAML
	var config GameReanimConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("无法解析配置文件 %s: %w", configPath, err)
	}

	// 3. 构建索引
	unitMap := make(map[string]*AnimationUnitConfig)
	for i := range config.Animations {
		unit := &config.Animations[i]
		if unit.ID == "" {
			return nil, fmt.Errorf("动画单元 #%d 缺少 'id' 字段", i)
		}
		unitMap[unit.ID] = unit
	}

	// 4. 创建管理器
	manager := &ReanimConfigManager{
		config:  &config,
		unitMap: unitMap,
	}

	return manager, nil
}

// GetUnit 获取动画单元配置
//
// 参数：
//   - id: 动画单元 ID（如 "peashooter", "zombie"）
//
// 返回：
//   - *AnimationUnitConfig: 动画单元配置
//   - error: 单元不存在时返回错误
func (m *ReanimConfigManager) GetUnit(id string) (*AnimationUnitConfig, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	unit, exists := m.unitMap[id]
	if !exists {
		return nil, fmt.Errorf("动画单元 '%s' 不存在", id)
	}

	return unit, nil
}

// GetCombo 获取动画组合配置
//
// 参数：
//   - unitID: 动画单元 ID（如 "peashooter"）
//   - comboName: 组合名称（如 "attack", "idle"）
//
// 返回：
//   - *AnimationComboConfig: 动画组合配置
//   - error: 单元或组合不存在时返回错误
func (m *ReanimConfigManager) GetCombo(unitID, comboName string) (*AnimationComboConfig, error) {
	// 1. 获取动画单元
	unit, err := m.GetUnit(unitID)
	if err != nil {
		return nil, err
	}

	// 2. 查找组合
	for i := range unit.AnimationCombos {
		combo := &unit.AnimationCombos[i]
		if combo.Name == comboName {
			return combo, nil
		}
	}

	return nil, fmt.Errorf("动画组合 '%s/%s' 不存在", unitID, comboName)
}

// GetDefaultAnimation 获取默认动画名称
//
// 参数：
//   - unitID: 动画单元 ID
//
// 返回：
//   - string: 默认动画名称
//   - error: 单元不存在或未配置默认动画时返回错误
func (m *ReanimConfigManager) GetDefaultAnimation(unitID string) (string, error) {
	unit, err := m.GetUnit(unitID)
	if err != nil {
		return "", err
	}

	if unit.DefaultAnimation == "" {
		return "", fmt.Errorf("动画单元 '%s' 未配置默认动画", unitID)
	}

	return unit.DefaultAnimation, nil
}

// GetGlobalConfig 获取全局配置
func (m *ReanimConfigManager) GetGlobalConfig() *GlobalConfig {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return &m.config.Global
}

// ListUnits 列出所有动画单元 ID
func (m *ReanimConfigManager) ListUnits() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	ids := make([]string, 0, len(m.unitMap))
	for id := range m.unitMap {
		ids = append(ids, id)
	}

	return ids
}
