package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// ZombiePhysicsConfig 僵尸物理配置
//
// 包含僵尸出生点坐标和进家判定边界的配置。
// 所有坐标值都是相对于网格原点（GridWorldStartX, GridWorldStartY）的偏移量。
//
// 配置文件位置: data/zombie_physics.yaml
type ZombiePhysicsConfig struct {
	// SpawnX 出生点X坐标配置
	SpawnX SpawnXConfig `yaml:"spawnX"`

	// DefeatBoundary 进家判定X坐标映射表
	// key: 僵尸类型字符串 (如 "basic", "polevaulter", "gargantuar")
	// value: 相对于网格原点的X偏移量（负值表示在网格左侧）
	DefeatBoundary map[string]float64 `yaml:"defeatBoundary"`
}

// SpawnXConfig 出生点X坐标配置
//
// 定义不同波次类型和僵尸类型的出生点X坐标范围。
// 所有值都是相对于网格原点（GridWorldStartX）的偏移量。
type SpawnXConfig struct {
	// Normal 普通波出生点范围
	Normal SpawnRange `yaml:"normal"`

	// FlagWave 旗帜波出生点范围
	FlagWave SpawnRange `yaml:"flagWave"`

	// FlagZombie 旗帜僵尸固定出生点
	FlagZombie float64 `yaml:"flagZombie"`

	// Zomboni 冰车出生点范围
	Zomboni SpawnRange `yaml:"zomboni"`

	// Gargantuar 巨人出生点范围
	Gargantuar SpawnRange `yaml:"gargantuar"`
}

// SpawnRange 出生点范围
//
// 定义一个X坐标范围，僵尸在此范围内随机生成。
type SpawnRange struct {
	// Min 最小X偏移量
	Min float64 `yaml:"min"`

	// Max 最大X偏移量
	Max float64 `yaml:"max"`
}

// LoadZombiePhysicsConfig 加载僵尸物理配置
//
// 从指定路径加载 YAML 格式的僵尸物理配置文件。
//
// 参数:
//   - path: 配置文件路径（如 "data/zombie_physics.yaml"）
//
// 返回:
//   - *ZombiePhysicsConfig: 加载成功后的配置结构
//   - error: 加载失败时返回错误
func LoadZombiePhysicsConfig(path string) (*ZombiePhysicsConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read zombie physics config: %w", err)
	}

	var config ZombiePhysicsConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse zombie physics config: %w", err)
	}

	// 验证配置
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid zombie physics config: %w", err)
	}

	return &config, nil
}

// Validate 验证配置有效性
//
// 检查配置值是否在合理范围内：
//   - 出生点范围的 Min 应小于等于 Max
//   - 进家判定边界不能为正值（僵尸应该在网格左侧触发失败）
//
// 返回:
//   - error: 验证失败时返回错误，成功返回 nil
func (c *ZombiePhysicsConfig) Validate() error {
	// 验证普通波出生点范围
	if c.SpawnX.Normal.Min > c.SpawnX.Normal.Max {
		return fmt.Errorf("normal spawn range invalid: min(%.1f) > max(%.1f)",
			c.SpawnX.Normal.Min, c.SpawnX.Normal.Max)
	}

	// 验证旗帜波出生点范围
	if c.SpawnX.FlagWave.Min > c.SpawnX.FlagWave.Max {
		return fmt.Errorf("flagWave spawn range invalid: min(%.1f) > max(%.1f)",
			c.SpawnX.FlagWave.Min, c.SpawnX.FlagWave.Max)
	}

	// 验证冰车出生点范围
	if c.SpawnX.Zomboni.Min > c.SpawnX.Zomboni.Max {
		return fmt.Errorf("zomboni spawn range invalid: min(%.1f) > max(%.1f)",
			c.SpawnX.Zomboni.Min, c.SpawnX.Zomboni.Max)
	}

	// 验证巨人出生点范围
	if c.SpawnX.Gargantuar.Min > c.SpawnX.Gargantuar.Max {
		return fmt.Errorf("gargantuar spawn range invalid: min(%.1f) > max(%.1f)",
			c.SpawnX.Gargantuar.Min, c.SpawnX.Gargantuar.Max)
	}

	// 验证进家判定边界（应该是负值或零）
	for zombieType, boundary := range c.DefeatBoundary {
		if boundary > 0 {
			return fmt.Errorf("defeat boundary for '%s' should be <= 0, got %.1f",
				zombieType, boundary)
		}
	}

	return nil
}

// GetDefeatBoundary 获取指定僵尸类型的进家判定边界
//
// 如果僵尸类型未配置，返回默认边界。
// 返回值是相对于网格原点的X偏移量。
//
// 参数:
//   - zombieType: 僵尸类型字符串（如 "basic", "polevaulter"）
//
// 返回:
//   - float64: 进家判定边界（网格相对坐标）
func (c *ZombiePhysicsConfig) GetDefeatBoundary(zombieType string) float64 {
	if boundary, ok := c.DefeatBoundary[zombieType]; ok {
		return boundary
	}
	// 返回默认边界
	if defaultBoundary, ok := c.DefeatBoundary["default"]; ok {
		return defaultBoundary
	}
	// 兜底值：-100（与原版一致）
	return -100.0
}

// GetSpawnXRange 获取指定僵尸类型在指定波次类型的出生点范围
//
// 根据波次类型和僵尸类型返回对应的出生点X坐标范围。
// 返回值是相对于网格原点的X偏移量。
//
// 参数:
//   - zombieType: 僵尸类型字符串
//   - isFlagWave: 是否为旗帜波
//   - isFlagZombie: 是否为旗帜僵尸
//
// 返回:
//   - min: 最小X偏移量
//   - max: 最大X偏移量
func (c *ZombiePhysicsConfig) GetSpawnXRange(zombieType string, isFlagWave bool, isFlagZombie bool) (min, max float64) {
	// 旗帜僵尸：固定位置
	if isFlagZombie {
		return c.SpawnX.FlagZombie, c.SpawnX.FlagZombie
	}

	// 特殊僵尸类型：使用专属范围
	switch zombieType {
	case "zomboni":
		return c.SpawnX.Zomboni.Min, c.SpawnX.Zomboni.Max
	case "gargantuar", "gargantuar_redeye":
		return c.SpawnX.Gargantuar.Min, c.SpawnX.Gargantuar.Max
	}

	// 普通僵尸：根据波次类型选择范围
	if isFlagWave {
		return c.SpawnX.FlagWave.Min, c.SpawnX.FlagWave.Max
	}
	return c.SpawnX.Normal.Min, c.SpawnX.Normal.Max
}

// ========================================
// 坐标系转换工具函数
// ========================================

// GridToWorldX 将网格相对X坐标转换为世界坐标
//
// 网格坐标系以草坪网格左上角为原点 (0, 0)。
// 世界坐标系以背景图片左上角为原点。
//
// 参数:
//   - gridRelativeX: 相对于网格左上角的X坐标
//
// 返回:
//   - float64: 世界坐标X
//
// 示例:
//
//	GridToWorldX(780) = 255 + 780 = 1035 (普通波僵尸出生点)
//	GridToWorldX(-100) = 255 - 100 = 155 (默认进家边界)
func GridToWorldX(gridRelativeX float64) float64 {
	return GridWorldStartX + gridRelativeX
}

// WorldToGridX 将世界坐标X转换为网格相对X坐标
//
// 参数:
//   - worldX: 世界坐标X
//
// 返回:
//   - float64: 相对于网格左上角的X坐标
//
// 示例:
//
//	WorldToGridX(1035) = 1035 - 255 = 780
//	WorldToGridX(155) = 155 - 255 = -100
func WorldToGridX(worldX float64) float64 {
	return worldX - GridWorldStartX
}

// GridToWorldY 将网格相对Y坐标转换为世界坐标
//
// 参数:
//   - gridRelativeY: 相对于网格左上角的Y坐标
//
// 返回:
//   - float64: 世界坐标Y
func GridToWorldY(gridRelativeY float64) float64 {
	return GridWorldStartY + gridRelativeY
}

// WorldToGridY 将世界坐标Y转换为网格相对Y坐标
//
// 参数:
//   - worldY: 世界坐标Y
//
// 返回:
//   - float64: 相对于网格左上角的Y坐标
func WorldToGridY(worldY float64) float64 {
	return worldY - GridWorldStartY
}

// GetRowCenterY 获取指定行的中心Y坐标（世界坐标）
//
// 行号从 0 开始（第一行 = 0）。
//
// 参数:
//   - row: 行索引（0-4 对应前院5行）
//
// 返回:
//   - float64: 该行中心的世界Y坐标
//
// 示例:
//
//	GetRowCenterY(0) = 78 + 0*100 + 50 = 128 (第1行中心)
//	GetRowCenterY(2) = 78 + 2*100 + 50 = 328 (第3行中心)
func GetRowCenterY(row int) float64 {
	return GridWorldStartY + float64(row)*CellHeight + CellHeight/2.0
}

// GetDefeatBoundaryWorldX 获取指定僵尸类型的进家判定边界（世界坐标）
//
// 这是一个便捷函数，将配置中的网格相对坐标转换为世界坐标。
//
// 参数:
//   - c: 僵尸物理配置（可为 nil，使用默认值）
//   - zombieType: 僵尸类型字符串
//
// 返回:
//   - float64: 进家判定边界的世界X坐标
func GetDefeatBoundaryWorldX(c *ZombiePhysicsConfig, zombieType string) float64 {
	var gridBoundary float64
	if c != nil {
		gridBoundary = c.GetDefeatBoundary(zombieType)
	} else {
		// 无配置时使用默认值 -100
		gridBoundary = -100.0
	}
	return GridToWorldX(gridBoundary)
}
