package components

import "github.com/decker502/pvz/pkg/ecs"

// ZombiesWonPhaseComponent 僵尸获胜流程阶段组件
//
// 管理四阶段状态机（僵尸获胜完整流程）：
//
// Phase 1: 游戏冻结
//   - 所有植物停止攻击
//   - 子弹消失
//   - UI 元素隐藏
//   - 背景音乐淡出
//   - 持续 ~1.5 秒后进入 Phase 2
//
// Phase 2: 僵尸入侵动画
//   - 触发失败的僵尸继续行走至屏幕外
//   - 摄像机从当前位置平滑移动到世界坐标 0（显示房子完整场景）
//   - 其他僵尸保持冻结
//   - 当僵尸 X < -100 且摄像机到达目标位置时进入 Phase 3
//
// Phase 3: 惨叫与"吃脑子"动画
//   - 播放惨叫音效（scream.ogg）
//   - 延迟 0.5 秒播放咀嚼音效（chomp_soft.ogg）
//   - 显示 ZombiesWon.reanim 动画
//   - 屏幕抖动特效（振幅 ±5px，频率 10Hz）
//   - 持续 3-4 秒后进入 Phase 4
//
// Phase 4: 游戏结束对话框
//   - 监听鼠标点击或等待 3-5 秒超时
//   - 显示游戏结束对话框（"再次尝试"/"返回主菜单"）
type ZombiesWonPhaseComponent struct {
	CurrentPhase int     // 当前阶段 (1: 冻结, 2: 僵尸入侵, 3: 惨叫动画, 4: 对话框)
	PhaseTimer   float64 // 阶段计时器（秒）

	// Phase 2 专用字段
	TriggerZombieID     ecs.EntityID // 触发失败的僵尸ID
	CameraMovedToTarget bool          // 摄像机是否已移动到目标位置（世界坐标 0）
	InitialCameraX      float64       // 初始摄像机X位置（用于计算平滑移动）
	ZombieStartedWalking bool         // 僵尸是否已开始行走（摄像机到位后）
	ZombieReachedTarget  bool         // 僵尸是否已到达目标位置（X <= 100, Y >= 300）

	// Phase 3 专用字段
	ScreamPlayed    bool    // 是否已播放惨叫音效
	ChompPlayed     bool    // 是否已播放咀嚼音效
	AnimationReady  bool    // 动画是否已准备好显示对话框
	ScreenShakeTime float64 // 屏幕抖动计时器

	// Phase 4 专用字段
	DialogShown bool    // 是否已显示对话框
	WaitTimer   float64 // 等待玩家点击的超时计时器
}
