package components

// SpawnConstraintComponent 僵尸生成限制检查组件
// 存储关卡级别的生成限制状态，供 SpawnConstraintSystem 使用
// 注意：遵循 ECS 原则，组件仅存储数据，不包含方法
type SpawnConstraintComponent struct {
	// RedEyeCount 已生成的红眼巨人数量（本关累计）
	RedEyeCount int

	// CurrentWaveNum 当前波次编号（从 1 开始）
	CurrentWaveNum int

	// AllowedZombieTypes 当前关卡允许的僵尸类型列表
	// 从关卡配置中加载，用于检查僵尸类型是否合法
	AllowedZombieTypes []string

	// SceneType 场景类型（day/night/pool/fog/roof/moon）
	// 用于检查场景特定限制（如水路僵尸、舞王限制等）
	SceneType string
}
