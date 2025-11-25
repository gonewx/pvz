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
	BackgroundImage  string  `yaml:"backgroundImage"`  // 背景图片ID，如 \"IMAGE_BACKGROUND1_UNSODDED\"，默认 \"IMAGE_BACKGROUND1\"
	SodRowImage      string  `yaml:"sodRowImage"`      // 草皮叠加图片ID，如 \"IMAGE_SOD1ROW\"，空表示无草皮
	SodRowImageAnim  string  `yaml:"sodRowImageAnim"`  // 动画阶段草皮图片ID（如\"IMAGE_SOD3ROW\"），空表示使用SodRowImage
	ShowSoddingAnim  bool    `yaml:"showSoddingAnim"`  // 是否播放铺草皮动画，默认 false
	SoddingAnimDelay float64 `yaml:"soddingAnimDelay"` // 铺草皮动画延迟（秒），默认 0

	// Story 11.2：关卡进度条配置
	FlagWaves []int `yaml:"flagWaves"` // 旗帜波次索引列表（从0开始），如 [9, 19] 表示第10波和第20波有旗帜，默认为空

	// Story 11.4：铺草皮粒子特效配置
	SodRollAnimation bool  `yaml:"sodRollAnimation"` // 是否启用铺草皮动画，默认 false
	SodRollParticles bool  `yaml:"sodRollParticles"` // 是否启用土粒飞溅特效，默认 false
	SoddingAnimLanes []int `yaml:"soddingAnimLanes"` // 指定播放动画的行列表（如 [2,4]），空表示所有启用的行
	PreSoddedLanes   []int `yaml:"preSoddedLanes"`   // 预先渲染草皮的行列表（如 [3]），初始化时直接显示草皮

	// Story 8.6 新增字段
	UnlockTools []string `yaml:"unlockTools"` // 完成本关解锁的工具列表，如 ["shovel"]，默认为空

	// Story 8.7 新增字段：僵尸行转换模式
	// 取值："instant" (瞬间) 或 "gradual" (渐变)，默认 "instant"
	//
	// 控制僵尸从非有效行转移到目标有效行时的转换方式：
	//   - "instant": 瞬间模式 - 僵尸立即调整Y坐标到目标行（无动画）
	//   - "gradual": 渐变模式 - 僵尸通过Y轴速度平滑移动到目标行（约3秒）
	//
	// 适用场景：
	//   - "instant": 标准关卡（推荐）
	//   - "gradual": 需要视觉过渡效果的特殊关卡
	LaneTransitionMode string `yaml:"laneTransitionMode"`

	// Story 8.3.1 新增字段：预览僵尸数量配置
	// 开场动画中展示的预览僵尸数量，可选配置
	// 如果为 0（未配置），则根据关卡难度自动计算：
	//   - 简单关卡（≤2波）：3 只预览僵尸
	//   - 中等关卡（3-5波）：5 只预览僵尸
	//   - 困难关卡（>5波）：8 只预览僵尸
	PreviewZombieCount int `yaml:"previewZombieCount"`
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
// Story 8.6 扩展：支持旗帜波次和混合僵尸生成
type WaveConfig struct {
	Delay      float64       `yaml:"delay"`      // 游戏开始后延迟（第1波使用），单位：秒
	MinDelay   float64       `yaml:"minDelay"`   // 上一波消灭后最小延迟（秒），默认 0（立即触发）
	IsFlag     bool          `yaml:"isFlag"`     // 是否为旗帜波次（Story 8.6）
	FlagIndex  int           `yaml:"flagIndex"`  // 旗帜索引（第几面旗帜），从1开始（Story 8.6）
	Zombies    []ZombieGroup `yaml:"zombies"`    // 本波次要生成的僵尸组列表（Story 8.6 使用 ZombieGroup）
	OldZombies []ZombieSpawn `yaml:"oldZombies"` // 兼容旧格式：单个僵尸生成配置（已废弃，向后兼容）
}

// ZombieGroup 僵尸组配置（Story 8.6 新增）
// 支持随机行选择和逐个生成
type ZombieGroup struct {
	Type          string  `yaml:"type"`          // 僵尸类型："basic", "conehead", "buckethead"
	Lanes         []int   `yaml:"lanes"`         // 可出现的行列表（随机选择），如 [2,3,4]
	Count         int     `yaml:"count"`         // 数量
	SpawnInterval float64 `yaml:"spawnInterval"` // 生成间隔（秒），逐个生成
}

// ZombieSpawn 单个僵尸生成配置（旧格式，向后兼容）
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
		if wave.Delay < 0 {
			return fmt.Errorf("wave %d: delay cannot be negative", i)
		}

		if wave.MinDelay < 0 {
			return fmt.Errorf("wave %d: minDelay cannot be negative", i)
		}

		// Story 8.6: 支持 ZombieGroup 和旧格式 ZombieSpawn
		if len(wave.Zombies) == 0 && len(wave.OldZombies) == 0 {
			return fmt.Errorf("wave %d: at least one zombie group or spawn is required", i)
		}

		// 验证新格式 ZombieGroup
		for j, zombieGroup := range wave.Zombies {
			if zombieGroup.Type == "" {
				return fmt.Errorf("wave %d, zombie group %d: type is required", i, j)
			}

			if len(zombieGroup.Lanes) == 0 {
				return fmt.Errorf("wave %d, zombie group %d: at least one lane is required", i, j)
			}

			// 验证所有 lanes 必须在 1-5 范围内
			for k, lane := range zombieGroup.Lanes {
				if lane < 1 || lane > 5 {
					return fmt.Errorf("wave %d, zombie group %d, lane %d: lane must be between 1 and 5, got %d", i, j, k, lane)
				}
			}

			if zombieGroup.Count < 1 {
				return fmt.Errorf("wave %d, zombie group %d: count must be at least 1, got %d", i, j, zombieGroup.Count)
			}

			if zombieGroup.SpawnInterval < 0 {
				return fmt.Errorf("wave %d, zombie group %d: spawnInterval cannot be negative", i, j)
			}
		}

		// 验证旧格式 ZombieSpawn（向后兼容）
		for j, zombie := range wave.OldZombies {
			if zombie.Type == "" {
				return fmt.Errorf("wave %d, old zombie %d: type is required", i, j)
			}

			if zombie.Lane < 1 || zombie.Lane > 5 {
				return fmt.Errorf("wave %d, old zombie %d: lane must be between 1 and 5, got %d", i, j, zombie.Lane)
			}

			if zombie.Count < 1 {
				return fmt.Errorf("wave %d, old zombie %d: count must be at least 1, got %d", i, j, zombie.Count)
			}
		}

		// 验证旗帜波次配置
		if wave.IsFlag && wave.FlagIndex < 1 {
			return fmt.Errorf("wave %d: flagIndex must be at least 1 for flag waves", i)
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

	// Story 8.6 QA修正: 验证 SoddingAnimLanes（如果配置了）
	if len(config.SoddingAnimLanes) > 0 {
		for _, lane := range config.SoddingAnimLanes {
			if lane < 1 || lane > 5 {
				return fmt.Errorf("invalid sodding animation lane %d (must be 1-5)", lane)
			}
		}
	}

	// Story 8.6 QA修正: 验证 PreSoddedLanes（如果配置了）
	if len(config.PreSoddedLanes) > 0 {
		for _, lane := range config.PreSoddedLanes {
			if lane < 1 || lane > 5 {
				return fmt.Errorf("invalid pre-sodded lane %d (must be 1-5)", lane)
			}
		}
	}

	return nil
}
