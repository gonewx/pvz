package game

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// SaveData 保存数据结构
//
// Story 8.6: 关卡进度保存系统
//
// 保存内容：
//   - 最高完成关卡（如 "1-3" 表示完成了 1-3，可以玩 1-4）
//   - 解锁的植物列表
//   - 解锁的工具列表
type SaveData struct {
	HighestLevel   string   `yaml:"highestLevel"`   // 最高完成关卡ID，如 "1-3"
	UnlockedPlants []string `yaml:"unlockedPlants"` // 已解锁植物ID列表
	UnlockedTools  []string `yaml:"unlockedTools"`  // 已解锁工具ID列表，如 ["shovel"]
}

// SaveManager 保存管理器
//
// 职责：
//   - 加载和保存游戏进度
//   - 管理关卡解锁状态
//   - 管理植物和工具解锁状态
//
// 架构说明：
//   - 单例模式，全局唯一实例
//   - 数据持久化到本地文件（YAML格式，与项目其他配置文件保持一致）
//   - 由 GameState 调用，不直接与系统交互
type SaveManager struct {
	saveFilePath string
	data         *SaveData
}

// NewSaveManager 创建保存管理器
//
// 参数：
//   - saveDir: 保存文件目录路径（如 "data/saves"）
//
// 返回：
//   - *SaveManager: 新创建的保存管理器实例
//   - error: 如果创建失败返回错误
func NewSaveManager(saveDir string) (*SaveManager, error) {
	// 确保保存目录存在
	if err := os.MkdirAll(saveDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create save directory: %w", err)
	}

	// 保存文件路径（使用 YAML 格式）
	saveFilePath := filepath.Join(saveDir, "progress.yaml")

	sm := &SaveManager{
		saveFilePath: saveFilePath,
		data: &SaveData{
			HighestLevel:   "", // 初始状态：未完成任何关卡
			UnlockedPlants: []string{},
			UnlockedTools:  []string{},
		},
	}

	// 尝试加载现有存档
	if err := sm.Load(); err != nil {
		// 如果文件不存在，使用默认数据
		if !os.IsNotExist(err) {
			return nil, fmt.Errorf("failed to load save data: %w", err)
		}
		// 文件不存在时，使用默认数据（新游戏）
	}

	return sm, nil
}

// Load 从文件加载保存数据
//
// 返回：
//   - error: 如果加载失败返回错误（文件不存在返回 os.ErrNotExist）
func (sm *SaveManager) Load() error {
	// 读取文件
	data, err := os.ReadFile(sm.saveFilePath)
	if err != nil {
		return err
	}

	// 解析 YAML
	var saveData SaveData
	if err := yaml.Unmarshal(data, &saveData); err != nil {
		return fmt.Errorf("failed to parse save data: %w", err)
	}

	sm.data = &saveData
	return nil
}

// Save 保存数据到文件
//
// 返回：
//   - error: 如果保存失败返回错误
func (sm *SaveManager) Save() error {
	// 序列化为 YAML（格式化输出，便于人工阅读和调试）
	data, err := yaml.Marshal(sm.data)
	if err != nil {
		return fmt.Errorf("failed to marshal save data: %w", err)
	}

	// 写入文件
	if err := os.WriteFile(sm.saveFilePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write save file: %w", err)
	}

	return nil
}

// GetHighestLevel 获取最高完成关卡
//
// 返回：
//   - string: 最高完成关卡ID，如 "1-3"，空字符串表示未完成任何关卡
func (sm *SaveManager) GetHighestLevel() string {
	return sm.data.HighestLevel
}

// SetHighestLevel 设置最高完成关卡
//
// 只有当新关卡比当前记录更高时才更新
//
// 参数：
//   - levelID: 关卡ID，如 "1-3"
func (sm *SaveManager) SetHighestLevel(levelID string) {
	// 简单比较：只要levelID不为空，就更新
	// TODO: 实现关卡ID的大小比较（如 "1-4" > "1-3"）
	if levelID != "" {
		sm.data.HighestLevel = levelID
	}
}

// GetUnlockedPlants 获取已解锁植物列表
//
// 返回：
//   - []string: 已解锁植物ID列表（副本，修改不影响原数据）
func (sm *SaveManager) GetUnlockedPlants() []string {
	// 返回副本
	plants := make([]string, len(sm.data.UnlockedPlants))
	copy(plants, sm.data.UnlockedPlants)
	return plants
}

// UnlockPlant 解锁植物
//
// 参数：
//   - plantID: 植物ID，如 "sunflower"
func (sm *SaveManager) UnlockPlant(plantID string) {
	// 检查是否已解锁
	for _, p := range sm.data.UnlockedPlants {
		if p == plantID {
			return // 已解锁，无需重复添加
		}
	}

	// 添加到列表
	sm.data.UnlockedPlants = append(sm.data.UnlockedPlants, plantID)
}

// GetUnlockedTools 获取已解锁工具列表
//
// 返回：
//   - []string: 已解锁工具ID列表（副本，修改不影响原数据）
func (sm *SaveManager) GetUnlockedTools() []string {
	// 返回副本
	tools := make([]string, len(sm.data.UnlockedTools))
	copy(tools, sm.data.UnlockedTools)
	return tools
}

// UnlockTool 解锁工具
//
// 参数：
//   - toolID: 工具ID，如 "shovel"
func (sm *SaveManager) UnlockTool(toolID string) {
	// 检查是否已解锁
	for _, t := range sm.data.UnlockedTools {
		if t == toolID {
			return // 已解锁，无需重复添加
		}
	}

	// 添加到列表
	sm.data.UnlockedTools = append(sm.data.UnlockedTools, toolID)
}

// IsToolUnlocked 检查工具是否已解锁
//
// 参数：
//   - toolID: 工具ID，如 "shovel"
//
// 返回：
//   - bool: true 表示已解锁，false 表示未解锁
func (sm *SaveManager) IsToolUnlocked(toolID string) bool {
	for _, t := range sm.data.UnlockedTools {
		if t == toolID {
			return true
		}
	}
	return false
}
