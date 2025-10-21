package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// LevelConfig 关卡配置数据结构
// 定义了关卡的基本信息和僵尸波次配置
type LevelConfig struct {
	ID          string       `yaml:"id"`          // 关卡ID，如 "1-1"
	Name        string       `yaml:"name"`        // 关卡名称，如 "前院白天 1-1"
	Description string       `yaml:"description"` // 关卡描述（可选）
	Waves       []WaveConfig `yaml:"waves"`       // 僵尸波次配置列表

	// Story 8.1 新增字段
	OpeningType     string         `yaml:"openingType"`     // 开场类型：\"tutorial\", \"standard\", \"special\"，默认\"standard\"
	EnabledLanes    []int          `yaml:"enabledLanes"`    // 启用的行列表，如 [1,2,3] 或 [3]，默认 [1,2,3,4,5]
	AvailablePlants []string       `yaml:"availablePlants"` // 可用植物ID列表，如 [\"peashooter\", \"sunflower\"]，默认为空（所有已解锁植物）
	SkipOpening     bool           `yaml:"skipOpening"`     // 是否跳过开场动画（调试用），默认 false
	TutorialSteps   []TutorialStep `yaml:"tutorialSteps"`   // 教学步骤（可选，Story 8.2 使用）
	SpecialRules    string         `yaml:"specialRules"`    // 特殊规则类型：\"bowling\", \"conveyor\"，默认为空
	InitialSun      int            `yaml:"initialSun"`      // 初始阳光值，默认50（Story 8.2 QA改进）

	// Story 8.3 新增字段
	RewardPlant string `yaml:"rewardPlant"` // 完成本关后奖励的植物ID，如 "sunflower"，默认为空（无奖励）

	// Story 8.2 QA改进：背景和草皮配置
	BackgroundImage   string `yaml:"backgroundImage"`   // 背景图片ID，如 \"IMAGE_BACKGROUND1_UNSODDED\"，默认 \"IMAGE_BACKGROUND1\"
	SodRowImage       string `yaml:"sodRowImage"`       // 草皮叠加图片ID，如 \"IMAGE_SOD1ROW\"，空表示无草皮
	ShowSoddingAnim   bool   `yaml:"showSoddingAnim"`   // 是否播放铺草皮动画，默认 false
	SoddingAnimDelay  float64 `yaml:"soddingAnimDelay"` // 铺草皮动画延迟（秒），默认 0
}

// TutorialStep 教学步骤配置（Story 8.2）
// 定义教学引导的触发条件、文本键和触发动作
type TutorialStep struct {
	Trigger      string        `yaml:"trigger"`      // 触发条件："gameStart", "sunClicked", "enoughSun", "seedClicked", "plantPlaced", "zombieSpawned"
	TextKey      string        `yaml:"textKey"`      // LawnStrings.txt 中的文本键（如 "ADVICE_CLICK_ON_SUN"）
	Action       string        `yaml:"action"`       // 触发动作："waitForSunClick", "waitForEnoughSun", "waitForSeedClick", "waitForPlantPlaced", "waitForZombieSpawn", "waitForLevelEnd"
	ZombieSpawns []ZombieSpawn `yaml:"zombieSpawns"` // 可选：该步骤触发时生成的僵尸（教学关卡专用）
}

// WaveConfig 单个僵尸波次配置
// 定义了僵尸波次的触发条件和生成的僵尸列表
type WaveConfig struct {
	MinDelay float64       `yaml:"minDelay"` // 可选：距离上一波结束的最小延迟（秒），默认 0（立即触发）
	Zombies  []ZombieSpawn `yaml:"zombies"`  // 本波次要生成的僵尸列表
}

// ZombieSpawn 单个僵尸生成配置
// 定义了僵尸的类型、出现行数和生成数量
type ZombieSpawn struct {
	Type  string `yaml:"type"`  // 僵尸类型："basic", "conehead", "buckethead"
	Lane  int    `yaml:"lane"`  // 僵尸出现的行（1-5，对应游戏界面的5行）
	Count int    `yaml:"count"` // 生成数量
}

// LoadLevelConfig 从YAML文件加载关卡配置
// 参数：
//
//	filepath - 关卡配置文件的路径（相对或绝对路径）
//
// 返回：
//
//	*LevelConfig - 解析后的关卡配置对象
//	error - 如果文件读取或解析失败，返回错误信息
func LoadLevelConfig(filepath string) (*LevelConfig, error) {
	// 读取文件内容
	data, err := os.ReadFile(filepath)
	if err != nil {
		return nil, fmt.Errorf("failed to read level config file %s: %w", filepath, err)
	}

	// 解析YAML数据
	var levelConfig LevelConfig
	if err := yaml.Unmarshal(data, &levelConfig); err != nil {
		return nil, fmt.Errorf("failed to parse level config YAML from %s: %w", filepath, err)
	}

	// 应用默认值（向后兼容性）
	applyDefaults(&levelConfig)

	// 验证必填字段
	if err := validateLevelConfig(&levelConfig); err != nil {
		return nil, fmt.Errorf("invalid level config in %s: %w", filepath, err)
	}

	return &levelConfig, nil
}

// applyDefaults 为 LevelConfig 中缺失的可选字段设置默认值
// 确保向后兼容性（旧配置文件可正常加载）
func applyDefaults(config *LevelConfig) {
	// 如果 EnabledLanes 为空，设置为所有5行
	if len(config.EnabledLanes) == 0 {
		config.EnabledLanes = []int{1, 2, 3, 4, 5}
	}

	// 如果 OpeningType 为空，设置为标准开场
	if config.OpeningType == "" {
		config.OpeningType = "standard"
	}

	// 如果 InitialSun 为0（未配置），设置为50（原版默认值）
	if config.InitialSun == 0 {
		config.InitialSun = 50
	}

	// Story 8.2 QA改进：背景和草皮默认值
	// 如果 BackgroundImage 为空，设置为标准背景
	if config.BackgroundImage == "" {
		config.BackgroundImage = "IMAGE_BACKGROUND1"
	}

	// AvailablePlants、TutorialSteps、SpecialRules、SodRowImage 默认为空值（nil/空字符串），无需处理
	// SkipOpening 默认为 false（bool 零值），无需处理
}

// validateLevelConfig 验证关卡配置的完整性和合法性
func validateLevelConfig(config *LevelConfig) error {
	// 验证关卡ID
	if config.ID == "" {
		return fmt.Errorf("level ID is required")
	}

	// 验证关卡名称
	if config.Name == "" {
		return fmt.Errorf("level name is required")
	}

	// 验证波次配置
	if len(config.Waves) == 0 {
		return fmt.Errorf("at least one wave is required")
	}

	// 验证每个波次的配置
	for i, wave := range config.Waves {
		if wave.MinDelay < 0 {
			return fmt.Errorf("wave %d: minDelay cannot be negative", i)
		}

		if len(wave.Zombies) == 0 {
			return fmt.Errorf("wave %d: at least one zombie spawn is required", i)
		}

		// 验证每个僵尸生成配置
		for j, zombie := range wave.Zombies {
			if zombie.Type == "" {
				return fmt.Errorf("wave %d, zombie %d: type is required", i, j)
			}

			if zombie.Lane < 1 || zombie.Lane > 5 {
				return fmt.Errorf("wave %d, zombie %d: lane must be between 1 and 5, got %d", i, j, zombie.Lane)
			}

			if zombie.Count < 1 {
				return fmt.Errorf("wave %d, zombie %d: count must be at least 1, got %d", i, j, zombie.Count)
			}
		}
	}

	// 验证 EnabledLanes（所有值必须在 1-5 范围内）
	for i, lane := range config.EnabledLanes {
		if lane < 1 || lane > 5 {
			return fmt.Errorf("enabledLanes[%d]: lane must be between 1 and 5, got %d", i, lane)
		}
	}

	// 验证 OpeningType（必须是合法值或空）
	validOpeningTypes := map[string]bool{
		"tutorial": true,
		"standard": true,
		"special":  true,
	}
	if config.OpeningType != "" && !validOpeningTypes[config.OpeningType] {
		return fmt.Errorf("openingType must be one of: tutorial, standard, special, got %q", config.OpeningType)
	}

	// 验证 SpecialRules（必须是合法值或空）
	validSpecialRules := map[string]bool{
		"bowling":  true,
		"conveyor": true,
	}
	if config.SpecialRules != "" && !validSpecialRules[config.SpecialRules] {
		return fmt.Errorf("specialRules must be one of: bowling, conveyor, got %q", config.SpecialRules)
	}

	return nil
}
