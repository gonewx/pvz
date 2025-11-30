package components

// AnimationCommandComponent 动画播放命令组件(纯数据)
//
// 设计目的:
//
//	解除系统间的直接耦合,使动画播放请求通过 ECS 组件机制传递
//
// 使用场景:
//  1. 播放配置化的动画组合 (UnitID + ComboName)
//  2. 播放单个动画 (AnimationName)
//
// 生命周期:
//  1. 其他系统(如 BehaviorSystem)添加此组件到实体
//  2. ReanimSystem 在 Update() 中查询并执行命令
//  3. 执行后标记 Processed = true
//  4. 可选:定期清理已处理的命令组件
//
// 示例:
//
//	// 场景1: 僵尸死亡动画 (使用 types.UnitIDZombie 常量)
//	ecs.AddComponent(em, zombieID, &AnimationCommandComponent{
//	    UnitID:    types.UnitIDZombie,
//	    ComboName: "death",
//	})
//
//	// 场景2: 最终波次警告动画
//	ecs.AddComponent(em, warningID, &AnimationCommandComponent{
//	    AnimationName: "FinalWave",
//	})
//
// 注意事项:
//   - 组件只包含数据,不包含方法(符合 ECS 数据纯净性原则)
//   - ReanimSystem 负责读取和执行命令
//   - 一个实体同时只应有一个 AnimationCommand(后续命令会覆盖前一个)
type AnimationCommandComponent struct {
	// ==========================================================================
	// 配置组合模式 (Config Combo Mode)
	// ==========================================================================

	// UnitID 单位 ID(如 "peashooter", "zombie", "sun")
	// 对应 data/reanim_config/<unitID>.yaml 中的单位配置
	// 使用场景:需要播放配置化的动画组合
	UnitID string

	// ComboName 组合名称(如 "attack", "death", "idle")
	// 对应 animation_combos 配置中的 combo.name
	// 如果为空,ReanimSystem 会使用单位的第一个 combo(default combo)
	// 使用场景:指定播放哪个动画组合
	ComboName string

	// ==========================================================================
	// 单动画模式 (Single Animation Mode)
	// ==========================================================================

	// AnimationName 单个动画名称(可选)
	// 如果指定,忽略 UnitID 和 ComboName,直接播放此动画
	// 使用场景:播放不在配置中的动画(如 "FinalWave" 警告动画)
	// 示例:"FinalWave", "anim_idle", "anim_death"
	AnimationName string

	// ==========================================================================
	// 执行状态 (Execution State)
	// ==========================================================================

	// Processed 是否已被 ReanimSystem 处理
	// false: 待处理(刚添加)
	// true: 已处理(ReanimSystem 已执行)
	// 用途:
	//   - 避免重复执行
	//   - 调试(可以查看命令是否被处理)
	//   - 命令历史记录(如果不删除组件)
	Processed bool

	// Timestamp 命令创建时间(游戏时间,单位:秒)
	// 用途:
	//   - 调试(分析命令执行延迟)
	//   - 可选的命令超时机制
	// 由添加组件的系统设置(可选)
	Timestamp float64

	// ==========================================================================
	// 动画过渡选项 (Animation Transition Options)
	// ==========================================================================

	// PreserveProgress 是否保留动画进度
	// true: 新动画从当前动画的相对进度位置开始播放(平滑过渡)
	// false: 新动画从头开始播放(默认行为)
	// 使用场景:植物从空闲状态切换到攻击状态时，保持身体摇摆的连续性
	PreserveProgress bool
}
