package systems

import (
	"math"
	"testing"

	"github.com/decker502/pvz/pkg/components"
	"github.com/decker502/pvz/pkg/ecs"
	"github.com/decker502/pvz/pkg/game"
)

// TestRewardAnimationSystem_NewRewardAnimationSystem 测试系统创建。
func TestRewardAnimationSystem_NewRewardAnimationSystem(t *testing.T) {
	em := ecs.NewEntityManager()
	gs := game.GetGameState()
	rm := game.NewResourceManager(nil)

	system := NewRewardAnimationSystem(em, gs, rm)

	if system == nil {
		t.Fatal("NewRewardAnimationSystem 返回 nil")
	}

	if system.entityManager != em {
		t.Error("EntityManager 未正确设置")
	}

	if system.gameState != gs {
		t.Error("GameState 未正确设置")
	}

	if system.resourceManager != rm {
		t.Error("ResourceManager 未正确设置")
	}

	if system.isActive {
		t.Error("初始状态应为未激活")
	}

	if system.rewardEntity != 0 {
		t.Error("初始奖励实体ID应为0")
	}

	if system.panelEntity != 0 {
		t.Error("初始面板实体ID应为0")
	}
}

// TestRewardAnimationSystem_TriggerReward 测试奖励触发。
func TestRewardAnimationSystem_TriggerReward(t *testing.T) {
	em := ecs.NewEntityManager()
	gs := game.GetGameState()
	rm := game.NewResourceManager(nil)

	system := NewRewardAnimationSystem(em, gs, rm)

	// 触发奖励
	plantID := "sunflower"
	system.TriggerReward(plantID)

	// 验证系统状态
	if !system.isActive {
		t.Error("TriggerReward 后系统应为激活状态")
	}

	if system.rewardEntity == 0 {
		t.Error("TriggerReward 后应创建奖励实体")
	}

	// 验证组件已添加
	rewardComp, ok := ecs.GetComponent[*components.RewardAnimationComponent](em, system.rewardEntity)
	if !ok {
		t.Fatal("未找到 RewardAnimationComponent")
	}

	// 验证初始状态
	if rewardComp.Phase != "dropping" {
		t.Errorf("初始阶段应为 dropping，实际为 %s", rewardComp.Phase)
	}

	if rewardComp.PlantID != plantID {
		t.Errorf("PlantID 应为 %s，实际为 %s", plantID, rewardComp.PlantID)
	}

	if rewardComp.VelocityX != RewardInitialVelocityX {
		t.Errorf("VelocityX 应为 %.0f，实际为 %.0f", RewardInitialVelocityX, rewardComp.VelocityX)
	}

	if rewardComp.VelocityY != RewardInitialVelocityY {
		t.Errorf("VelocityY 应为 %.0f，实际为 %.0f", RewardInitialVelocityY, rewardComp.VelocityY)
	}

	// 验证位置组件
	posComp, ok := ecs.GetComponent[*components.PositionComponent](em, system.rewardEntity)
	if !ok {
		t.Fatal("未找到 PositionComponent")
	}

	if posComp.X != rewardComp.StartX || posComp.Y != rewardComp.StartY {
		t.Errorf("初始位置不匹配")
	}
}

// TestRewardAnimationSystem_TriggerReward_IgnoreDuplicate 测试重复触发被忽略。
func TestRewardAnimationSystem_TriggerReward_IgnoreDuplicate(t *testing.T) {
	em := ecs.NewEntityManager()
	gs := game.GetGameState()
	rm := game.NewResourceManager(nil)

	system := NewRewardAnimationSystem(em, gs, rm)

	// 第一次触发
	system.TriggerReward("sunflower")
	firstEntity := system.rewardEntity

	// 第二次触发（应被忽略）
	system.TriggerReward("peashooter")
	secondEntity := system.rewardEntity

	// 验证实体ID未变化
	if firstEntity != secondEntity {
		t.Error("重复触发应被忽略，实体ID不应变化")
	}

	// 验证PlantID未变化
	rewardComp, _ := ecs.GetComponent[*components.RewardAnimationComponent](em, system.rewardEntity)
	if rewardComp.PlantID != "sunflower" {
		t.Errorf("PlantID应保持为 sunflower，实际为 %s", rewardComp.PlantID)
	}
}

// TestRewardAnimationSystem_DroppingPhase 测试掉落阶段。
func TestRewardAnimationSystem_DroppingPhase(t *testing.T) {
	em := ecs.NewEntityManager()
	gs := game.GetGameState()
	rm := game.NewResourceManager(nil)

	system := NewRewardAnimationSystem(em, gs, rm)
	system.TriggerReward("sunflower")

	// 获取初始状态
	posComp, _ := ecs.GetComponent[*components.PositionComponent](em, system.rewardEntity)
	rewardComp, _ := ecs.GetComponent[*components.RewardAnimationComponent](em, system.rewardEntity)

	initialX := posComp.X
	initialVelY := rewardComp.VelocityY

	// 模拟掉落（0.5秒）
	system.Update(0.5)

	// 验证位置变化
	posComp, _ = ecs.GetComponent[*components.PositionComponent](em, system.rewardEntity)

	// 初始VelocityY是负数(-400)，所以Y会先减小（向上），然后受重力影响开始增加
	// 经过0.5秒后，受重力影响，Y坐标应该开始向下移动
	// 但是由于初始速度向上很大，可能还在向上移动，我们只验证速度方向改变

	// X坐标应减少（向左移动）
	if posComp.X >= initialX {
		t.Errorf("掉落阶段X坐标应减少（向左移动），初始: %.0f，现在: %.0f", initialX, posComp.X)
	}

	// 验证速度变化（受重力影响）
	rewardComp, _ = ecs.GetComponent[*components.RewardAnimationComponent](em, system.rewardEntity)
	if rewardComp.VelocityY <= initialVelY {
		t.Errorf("受重力影响，VelocityY应增加，初始: %.0f，现在: %.0f", initialVelY, rewardComp.VelocityY)
	}

	// 验证仍在掉落阶段
	if rewardComp.Phase != "dropping" {
		t.Errorf("应仍处于 dropping 阶段，实际为 %s", rewardComp.Phase)
	}
}

// TestRewardAnimationSystem_DroppingToBouncingTransition 测试掉落到弹跳的转换。
func TestRewardAnimationSystem_DroppingToBouncingTransition(t *testing.T) {
	em := ecs.NewEntityManager()
	gs := game.GetGameState()
	rm := game.NewResourceManager(nil)

	system := NewRewardAnimationSystem(em, gs, rm)
	system.TriggerReward("sunflower")

	// 模拟足够长的时间让卡片到达目标位置
	for i := 0; i < 50; i++ {
		system.Update(0.1)
		rewardComp, _ := ecs.GetComponent[*components.RewardAnimationComponent](em, system.rewardEntity)
		if rewardComp.Phase != "dropping" {
			break
		}
	}

	// 验证已切换到弹跳阶段
	rewardComp, _ := ecs.GetComponent[*components.RewardAnimationComponent](em, system.rewardEntity)
	if rewardComp.Phase != "bouncing" {
		t.Errorf("应切换到 bouncing 阶段，实际为 %s", rewardComp.Phase)
	}

	// 验证位置已修正到目标点
	posComp, _ := ecs.GetComponent[*components.PositionComponent](em, system.rewardEntity)
	if posComp.X != rewardComp.TargetX {
		t.Errorf("X坐标应修正到目标点 %.0f，实际为 %.0f", rewardComp.TargetX, posComp.X)
	}

	if posComp.Y != rewardComp.TargetY {
		t.Errorf("Y坐标应修正到目标点 %.0f，实际为 %.0f", rewardComp.TargetY, posComp.Y)
	}

	// 验证弹跳计数器重置
	if rewardComp.BounceCount != 0 {
		t.Errorf("弹跳计数器应重置为0，实际为 %d", rewardComp.BounceCount)
	}

	if rewardComp.ElapsedTime != 0 {
		t.Errorf("ElapsedTime应重置为0，实际为 %.2f", rewardComp.ElapsedTime)
	}
}

// TestRewardAnimationSystem_BouncingPhase 测试弹跳阶段。
func TestRewardAnimationSystem_BouncingPhase(t *testing.T) {
	em := ecs.NewEntityManager()
	gs := game.GetGameState()
	rm := game.NewResourceManager(nil)

	system := NewRewardAnimationSystem(em, gs, rm)
	system.TriggerReward("sunflower")

	// 推进到弹跳阶段
	for i := 0; i < 50; i++ {
		system.Update(0.1)
		rewardComp, _ := ecs.GetComponent[*components.RewardAnimationComponent](em, system.rewardEntity)
		if rewardComp.Phase == "bouncing" {
			break
		}
	}

	rewardComp, _ := ecs.GetComponent[*components.RewardAnimationComponent](em, system.rewardEntity)
	if rewardComp.Phase != "bouncing" {
		t.Skip("未能到达弹跳阶段")
	}

	targetY := rewardComp.TargetY

	// 模拟弹跳（0.1秒）
	system.Update(0.1)

	// 验证Y坐标变化（应在目标点上方）
	posComp, _ := ecs.GetComponent[*components.PositionComponent](em, system.rewardEntity)
	if posComp.Y >= targetY {
		t.Error("弹跳阶段Y坐标应在目标点上方")
	}

	// 模拟完整的弹跳周期
	for i := 0; i < 20; i++ {
		system.Update(0.1)
		rewardComp, _ = ecs.GetComponent[*components.RewardAnimationComponent](em, system.rewardEntity)
		if rewardComp.Phase != "bouncing" {
			break
		}
	}

	// 验证已切换到展开阶段
	rewardComp, _ = ecs.GetComponent[*components.RewardAnimationComponent](em, system.rewardEntity)
	if rewardComp.Phase != "expanding" {
		t.Errorf("弹跳完成后应切换到 expanding，实际为 %s", rewardComp.Phase)
	}

	// 验证弹跳次数
	if rewardComp.BounceCount < RewardMaxBounces {
		t.Errorf("弹跳次数应达到 %d，实际为 %d", RewardMaxBounces, rewardComp.BounceCount)
	}

	// 验证位置已重置到目标点
	posComp, _ = ecs.GetComponent[*components.PositionComponent](em, system.rewardEntity)
	if posComp.Y != targetY {
		t.Errorf("弹跳结束后Y坐标应重置到目标点 %.0f，实际为 %.0f", targetY, posComp.Y)
	}
}

// TestRewardAnimationSystem_ParabolaCalculation 测试抛物线计算。
func TestRewardAnimationSystem_ParabolaCalculation(t *testing.T) {
	em := ecs.NewEntityManager()
	gs := game.GetGameState()
	rm := game.NewResourceManager(nil)

	system := NewRewardAnimationSystem(em, gs, rm)
	system.TriggerReward("sunflower")

	// 记录初始状态
	rewardComp, _ := ecs.GetComponent[*components.RewardAnimationComponent](em, system.rewardEntity)
	posComp, _ := ecs.GetComponent[*components.PositionComponent](em, system.rewardEntity)

	initialX := posComp.X
	initialVelY := rewardComp.VelocityY

	dt := 0.1

	// 执行一帧更新
	system.Update(dt)

	// 验证实际位置和速度变化
	posComp, _ = ecs.GetComponent[*components.PositionComponent](em, system.rewardEntity)
	rewardComp, _ = ecs.GetComponent[*components.RewardAnimationComponent](em, system.rewardEntity)

	// 验证X坐标向左移动（减少）
	if posComp.X >= initialX {
		t.Errorf("X坐标应向左移动，初始: %.2f，现在: %.2f", initialX, posComp.X)
	}

	// 验证VelocityY受重力影响增加
	if rewardComp.VelocityY <= initialVelY {
		t.Errorf("VelocityY应受重力影响增加，初始: %.2f，现在: %.2f", initialVelY, rewardComp.VelocityY)
	}

	// 验证VelocityY的增量接近重力*dt
	expectedDelta := RewardGravity * dt
	actualDelta := rewardComp.VelocityY - initialVelY

	tolerance := 0.1
	if math.Abs(actualDelta-expectedDelta) > tolerance {
		t.Errorf("VelocityY增量应接近 %.2f，实际为 %.2f", expectedDelta, actualDelta)
	}
}

// TestRewardAnimationSystem_EasingFunction 测试缓动函数。
func TestRewardAnimationSystem_EasingFunction(t *testing.T) {
	em := ecs.NewEntityManager()
	gs := game.GetGameState()
	rm := game.NewResourceManager(nil)

	system := NewRewardAnimationSystem(em, gs, rm)

	tests := []struct {
		input    float64
		expected float64
	}{
		{0.0, 0.0},   // 起点
		{0.5, 0.5},   // 中点（二次缓动的特性）
		{1.0, 1.0},   // 终点
		{0.25, 0.125}, // 第一象限
		{0.75, 0.875}, // 第三象限
	}

	for _, tt := range tests {
		result := system.easeInOutQuad(tt.input)
		// 允许浮点误差
		if math.Abs(result-tt.expected) > 0.001 {
			t.Errorf("easeInOutQuad(%.2f) = %.3f，期望 %.3f", tt.input, result, tt.expected)
		}
	}
}

// TestRewardAnimationSystem_IsActive 测试激活状态查询。
func TestRewardAnimationSystem_IsActive(t *testing.T) {
	em := ecs.NewEntityManager()
	gs := game.GetGameState()
	rm := game.NewResourceManager(nil)

	system := NewRewardAnimationSystem(em, gs, rm)

	// 初始状态应为未激活
	if system.IsActive() {
		t.Error("初始状态应为未激活")
	}

	// 触发奖励后应激活
	system.TriggerReward("sunflower")
	if !system.IsActive() {
		t.Error("触发奖励后应为激活状态")
	}
}

// TestRewardAnimationSystem_IsCompleted 测试完成状态查询。
func TestRewardAnimationSystem_IsCompleted(t *testing.T) {
	em := ecs.NewEntityManager()
	gs := game.GetGameState()
	rm := game.NewResourceManager(nil)

	system := NewRewardAnimationSystem(em, gs, rm)

	// 初始状态应为已完成（未激活）
	if !system.IsCompleted() {
		t.Error("初始状态应为已完成")
	}

	// 触发奖励后应为未完成
	system.TriggerReward("sunflower")
	if system.IsCompleted() {
		t.Error("触发奖励后应为未完成")
	}
}

// TestRewardAnimationSystem_GetPlantInfo 测试获取植物信息。
func TestRewardAnimationSystem_GetPlantInfo(t *testing.T) {
	em := ecs.NewEntityManager()
	gs := game.GetGameState()
	rm := game.NewResourceManager(nil)

	system := NewRewardAnimationSystem(em, gs, rm)

	// 测试获取植物信息
	plantID := "sunflower"
	name, desc := system.getPlantInfo(plantID)

	// 验证返回值不为空（具体值取决于 PlantUnlockManager 的实现）
	if name == "" {
		t.Error("植物名称不应为空")
	}

	if desc == "" {
		t.Error("植物描述不应为空")
	}

	t.Logf("植物信息 - 名称: %s, 描述: %s", name, desc)
}

// TestRewardAnimationSystem_CreateRewardPanel 测试创建奖励面板。
func TestRewardAnimationSystem_CreateRewardPanel(t *testing.T) {
	em := ecs.NewEntityManager()
	gs := game.GetGameState()
	rm := game.NewResourceManager(nil)

	system := NewRewardAnimationSystem(em, gs, rm)

	// 创建奖励面板
	plantID := "sunflower"
	system.createRewardPanel(plantID)

	// 验证面板实体已创建
	if system.panelEntity == 0 {
		t.Fatal("面板实体未创建")
	}

	// 验证面板组件
	panelComp, ok := ecs.GetComponent[*components.RewardPanelComponent](em, system.panelEntity)
	if !ok {
		t.Fatal("未找到 RewardPanelComponent")
	}

	// 验证初始状态
	if panelComp.PlantID != plantID {
		t.Errorf("PlantID 应为 %s，实际为 %s", plantID, panelComp.PlantID)
	}

	if !panelComp.IsVisible {
		t.Error("面板应为可见状态")
	}

	if panelComp.CardScale != RewardCardScaleStart {
		t.Errorf("初始缩放应为 %.2f，实际为 %.2f", RewardCardScaleStart, panelComp.CardScale)
	}

	if panelComp.FadeAlpha != 0 {
		t.Errorf("初始淡入透明度应为 0，实际为 %.2f", panelComp.FadeAlpha)
	}

	// 验证卡片位置
	expectedX := system.screenWidth / 2
	expectedY := system.screenHeight * 0.35

	if panelComp.CardX != expectedX {
		t.Errorf("卡片X坐标应为 %.0f，实际为 %.0f", expectedX, panelComp.CardX)
	}

	if panelComp.CardY != expectedY {
		t.Errorf("卡片Y坐标应为 %.0f，实际为 %.0f", expectedY, panelComp.CardY)
	}
}

// TestRewardAnimationSystem_ShowingPhaseAnimation 测试显示阶段动画。
func TestRewardAnimationSystem_ShowingPhaseAnimation(t *testing.T) {
	em := ecs.NewEntityManager()
	gs := game.GetGameState()
	rm := game.NewResourceManager(nil)

	system := NewRewardAnimationSystem(em, gs, rm)

	// 创建奖励面板
	system.createRewardPanel("sunflower")

	// 手动设置到 showing 阶段
	rewardEntity := em.CreateEntity()
	ecs.AddComponent(em, rewardEntity, &components.RewardAnimationComponent{
		Phase:       "showing",
		ElapsedTime: 0,
		PlantID:     "sunflower",
	})
	system.rewardEntity = rewardEntity
	system.isActive = true

	// 模拟动画（0.5秒）
	dt := 0.5
	system.Update(dt)

	// 验证面板动画状态
	panelComp, _ := ecs.GetComponent[*components.RewardPanelComponent](em, system.panelEntity)

	// 验证缩放动画进度（应在 0.5 ~ 1.0 之间）
	if panelComp.CardScale < RewardCardScaleStart || panelComp.CardScale > RewardCardScaleEnd {
		t.Errorf("卡片缩放应在 %.2f ~ %.2f 之间，实际为 %.2f",
			RewardCardScaleStart, RewardCardScaleEnd, panelComp.CardScale)
	}

	// 验证淡入动画（应接近 1.0）
	if panelComp.FadeAlpha < 0.9 {
		t.Errorf("淡入透明度应接近 1.0，实际为 %.2f", panelComp.FadeAlpha)
	}

	t.Logf("动画进度 - 缩放: %.2f, 透明度: %.2f", panelComp.CardScale, panelComp.FadeAlpha)
}

// TestRewardAnimationSystem_UpdateInactive 测试未激活状态的更新。
func TestRewardAnimationSystem_UpdateInactive(t *testing.T) {
	em := ecs.NewEntityManager()
	gs := game.GetGameState()
	rm := game.NewResourceManager(nil)

	system := NewRewardAnimationSystem(em, gs, rm)

	// 未触发奖励，直接更新（不应崩溃）
	system.Update(0.1)

	// 验证状态未变化
	if system.isActive {
		t.Error("未触发奖励时，系统应保持未激活")
	}
}
