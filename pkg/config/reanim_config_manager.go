package config

import (
	"fmt"
	"path/filepath"
	"sync"

	"github.com/decker502/pvz/pkg/embedded"
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
	ID                  string                 `yaml:"id"`
	Name                string                 `yaml:"name"`
	ReanimFile          string                 `yaml:"reanim_file"`
	DefaultAnimation    string                 `yaml:"default_animation"`
	Scale               float64                `yaml:"scale"` // 整体缩放比例，默认 1.0
	CenterOffset        []float64              `yaml:"center_offset,omitempty"` // 可选：手动指定 CenterOffset [x, y]，如果不指定则自动计算
	Images              map[string]string      `yaml:"images"`
	AvailableAnimations []AnimationInfo        `yaml:"available_animations"`
	AnimationCombos     []AnimationComboConfig `yaml:"animation_combos"`
}

// AnimationInfo 动画信息
type AnimationInfo struct {
	Name        string  `yaml:"name"`
	DisplayName string  `yaml:"display_name"`
	Loop        *bool   `yaml:"loop,omitempty"`  // 可选：是否循环播放，nil=默认true，显式false=不循环
	FPS         float64 `yaml:"fps,omitempty"`   // 可选：该动画的独立 FPS，若未指定则使用全局/Reanim 文件的 FPS
	Speed       float64 `yaml:"speed,omitempty"` // 可选：动画速度倍率（0.0-1.0），1.0=正常速度，0.5=50%速度，默认 1.0
}

// 注意：AnimationComboConfig 已在 reanim_config.go 中定义，这里直接使用

// ReanimConfigManager Reanim 配置管理器
// 负责加载和管理全量 Reanim 配置
type ReanimConfigManager struct {
	config  *GameReanimConfig               // 全量配置
	unitMap map[string]*AnimationUnitConfig // 按 id 索引的配置映射
	mu      sync.RWMutex                    // 读写锁（并发安全）
}

// NewReanimConfigManager 创建配置管理器
//
// 参数：
//   - configPath: 配置文件路径或目录路径
//   - 如果是文件路径（如 "data/reanim_config.yaml"），则使用单文件模式（向后兼容）
//   - 如果是目录路径（如 "data/reanim_config"），则从目录加载所有 YAML 文件
//
// 返回：
//   - *ReanimConfigManager: 配置管理器实例
//   - error: 加载或解析错误
func NewReanimConfigManager(configPath string) (*ReanimConfigManager, error) {
	// 1. 判断路径类型（检查是否为目录）
	// 使用 embedded.ReadDir 尝试读取目录来判断
	if _, err := embedded.ReadDir(configPath); err == nil {
		// 是目录
		return loadFromDirectory(configPath)
	}

	// 尝试作为文件加载
	if embedded.Exists(configPath) {
		return loadFromFile(configPath)
	}

	return nil, fmt.Errorf("无法访问路径 %s", configPath)
}

// loadFromFile 从单个文件加载配置（向后兼容）
func loadFromFile(configPath string) (*ReanimConfigManager, error) {
	// 1. 从 embedded FS 读取配置文件
	data, err := embedded.ReadFile(configPath)
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
		// 设置 Scale 默认值
		if unit.Scale == 0 {
			unit.Scale = 1.0
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

// loadFromDirectory 从目录加载所有配置文件
func loadFromDirectory(dirPath string) (*ReanimConfigManager, error) {
	// 1. 加载全局配置
	globalConfig, err := loadGlobalConfig(dirPath)
	if err != nil {
		return nil, fmt.Errorf("加载全局配置失败: %w", err)
	}

	// 2. 扫描目录中的所有 YAML 文件（使用 embedded.Glob）
	pattern := dirPath + "/*.yaml"
	files, err := embedded.Glob(pattern)
	if err != nil {
		return nil, fmt.Errorf("扫描目录 %s 失败: %w", dirPath, err)
	}

	// 3. 加载所有动画单元配置
	var animations []AnimationUnitConfig
	unitMap := make(map[string]*AnimationUnitConfig)

	for _, file := range files {
		unit, err := loadAnimationUnit(file)
		if err != nil {
			return nil, fmt.Errorf("加载文件 %s 失败: %w", file, err)
		}

		// 跳过全局配置文件（没有 id 字段）
		if unit.ID == "" {
			continue
		}

		animations = append(animations, unit)
		// 注意：这里先添加到数组，后面再设置指针
	}

	// 4. 构建索引（指向数组中的元素）
	for i := range animations {
		unit := &animations[i]
		if _, exists := unitMap[unit.ID]; exists {
			return nil, fmt.Errorf("重复的动画单元 ID: %s", unit.ID)
		}
		// 设置 Scale 默认值
		if unit.Scale == 0 {
			unit.Scale = 1.0
		}
		unitMap[unit.ID] = unit
	}

	// 5. 创建管理器
	config := &GameReanimConfig{
		Global:     globalConfig,
		Animations: animations,
	}

	manager := &ReanimConfigManager{
		config:  config,
		unitMap: unitMap,
	}

	return manager, nil
}

// loadAnimationUnit 加载单个动画单元配置文件
func loadAnimationUnit(filePath string) (AnimationUnitConfig, error) {
	data, err := embedded.ReadFile(filePath)
	if err != nil {
		return AnimationUnitConfig{}, fmt.Errorf("无法读取文件: %w", err)
	}

	var unit AnimationUnitConfig
	if err := yaml.Unmarshal(data, &unit); err != nil {
		return AnimationUnitConfig{}, fmt.Errorf("无法解析 YAML: %w", err)
	}

	return unit, nil
}

// loadGlobalConfig 加载全局配置
// 优先查找 data/reanim_config.yaml (上级目录)
// 如果不存在，使用默认配置
func loadGlobalConfig(dirPath string) (GlobalConfig, error) {
	// 1. 尝试从上级目录加载全局配置文件
	globalPath := filepath.Dir(dirPath) + "/reanim_config.yaml"
	if embedded.Exists(globalPath) {
		// 文件存在，从 embedded FS 读取全局配置
		data, err := embedded.ReadFile(globalPath)
		if err != nil {
			return GlobalConfig{}, fmt.Errorf("无法读取全局配置文件 %s: %w", globalPath, err)
		}

		var config struct {
			Global GlobalConfig `yaml:"global"`
		}
		if err := yaml.Unmarshal(data, &config); err != nil {
			return GlobalConfig{}, fmt.Errorf("无法解析全局配置文件 %s: %w", globalPath, err)
		}

		return config.Global, nil
	}

	// 2. 如果全局配置文件不存在，使用默认配置
	return GlobalConfig{
		Playback: PlaybackConfig{
			TPS: 60,
			FPS: 12,
		},
	}, nil
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
