package behavior

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/decker502/pvz/internal/particle"
	"github.com/decker502/pvz/pkg/components"
	"github.com/decker502/pvz/pkg/ecs"
	"github.com/decker502/pvz/pkg/game"
	"github.com/decker502/pvz/pkg/systems"
	"github.com/hajimehoshi/ebiten/v2"
)

// init 函数在测试开始前切换到项目根目录
// 确保所有相对路径（如 assets/）都能正确访问
func init() {
	// 查找项目根目录（包含 go.mod 文件的目录）
	dir, err := os.Getwd()
	if err != nil {
		return
	}

	// 向上查找直到找到 go.mod
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			// 找到项目根目录，切换到该目录
			os.Chdir(dir)
			return
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			// 已经到达文件系统根目录，停止查找
			return
		}
		dir = parent
	}
}

// createTestBehaviorSystem 创建测试用的 BehaviorSystem（包含必需的依赖）
// Bug Fix: 所有测试需要传入 LawnGridSystem 和 lawnGridEntityID
// Story 14.3: Epic 14 - Removed ReanimSystem dependency
func createTestBehaviorSystem(em *ecs.EntityManager, rm *game.ResourceManager, gs *game.GameState) *BehaviorSystem {
	// 创建测试用的 LawnGridSystem
	lgs := systems.NewLawnGridSystem(em, []int{1, 2, 3, 4, 5})

	// 创建草坪网格实体
	gridID := em.CreateEntity()
	ecs.AddComponent(em, gridID, &components.LawnGridComponent{})

	// 返回完整的 BehaviorSystem
	return NewBehaviorSystem(em, rm, gs, lgs, gridID)
}

// TestZombieDeathParticleEffect 测试僵尸死亡时是否正确触发粒子效果
// AC 9: 验证僵尸死亡时创建粒子发射器实体（MoweredZombieArm, MoweredZombieHead）
func TestZombieDeathParticleEffect(t *testing.T) {
	// 准备测试环境
	em := ecs.NewEntityManager()
	rm := game.NewResourceManager(getTestAudioContext())
	gs := game.GetGameState()

	// 加载粒子配置（模拟真实环境）
	if _, err := rm.LoadParticleConfig("PeaSplat"); err != nil {
		t.Skipf("跳过测试：无法加载粒子资源: %v", err)
	}

	bs := createTestBehaviorSystem(em, rm, gs)

	// 创建测试僵尸实体
	zombieID := em.CreateEntity()
	em.AddComponent(zombieID, &components.BehaviorComponent{
		Type: components.BehaviorZombieBasic,
	})
	em.AddComponent(zombieID, &components.PositionComponent{
		X: 500.0,
		Y: 300.0,
	})
	em.AddComponent(zombieID, &components.HealthComponent{
		CurrentHealth: 0, // 生命值为0，触发死亡
		MaxHealth:     100,
	})
	em.AddComponent(zombieID, &components.VelocityComponent{
		VX: -20.0,
	})
	// 添加 ReanimComponent 用于动画播放（避免 ReanimSystem 报错）
	em.AddComponent(zombieID, &components.ReanimComponent{
		ReanimXML:  nil, // 测试环境简化，不加载真实动画数据
		PartImages: make(map[string]*ebiten.Image),
	})

	// 记录触发前的发射器数量
	initialEmitterCount := len(em.GetEntitiesWith(
		reflect.TypeOf(&components.EmitterComponent{}),
		reflect.TypeOf(&components.PositionComponent{}),
	))

	// 触发僵尸死亡
	bs.triggerZombieDeath(zombieID)

	// 验证：至少创建了粒子发射器实体
	currentEmitterCount := len(em.GetEntitiesWith(
		reflect.TypeOf(&components.EmitterComponent{}),
		reflect.TypeOf(&components.PositionComponent{}),
	))
	if currentEmitterCount <= initialEmitterCount {
		t.Errorf("僵尸死亡后未创建粒子发射器实体。初始发射器数: %d, 当前发射器数: %d",
			initialEmitterCount, currentEmitterCount)
	}

	// 验证：查找粒子发射器实体
	emitterEntities := em.GetEntitiesWith(
		reflect.TypeOf(&components.EmitterComponent{}),
		reflect.TypeOf(&components.PositionComponent{}),
	)

	if len(emitterEntities) < 2 {
		t.Errorf("期望至少创建2个粒子发射器（手臂+头部），实际创建: %d", len(emitterEntities))
	}

	// 验证：粒子发射器位置与僵尸位置匹配
	foundCorrectPosition := false
	for _, emitterID := range emitterEntities {
		posComp, ok := em.GetComponent(emitterID, reflect.TypeOf(&components.PositionComponent{}))
		if !ok {
			continue
		}
		pos := posComp.(*components.PositionComponent)
		// 允许微小的浮点误差
		if pos.X >= 499.0 && pos.X <= 501.0 && pos.Y >= 299.0 && pos.Y <= 301.0 {
			foundCorrectPosition = true
			break
		}
	}

	if !foundCorrectPosition {
		t.Error("粒子发射器位置与僵尸位置不匹配（期望位置：500, 300）")
	}

	// 验证：僵尸行为切换为 BehaviorZombieDying
	behaviorComp, ok := em.GetComponent(zombieID, reflect.TypeOf(&components.BehaviorComponent{}))
	if !ok {
		t.Error("僵尸 BehaviorComponent 丢失")
	} else {
		behavior := behaviorComp.(*components.BehaviorComponent)
		if behavior.Type != components.BehaviorZombieDying {
			t.Errorf("僵尸行为类型未切换为 BehaviorZombieDying，当前类型: %v", behavior.Type)
		}
	}

	// 验证：僵尸 VelocityComponent 被移除（停止移动）
	if em.HasComponent(zombieID, reflect.TypeOf(&components.VelocityComponent{})) {
		t.Error("僵尸 VelocityComponent 应该被移除，但仍然存在")
	}
}

// TestCherryBombExplosionParticleEffect 测试樱桃炸弹爆炸时是否正确触发粒子效果
// AC 9: 验证樱桃炸弹爆炸时创建粒子发射器实体（BossExplosion）
func TestCherryBombExplosionParticleEffect(t *testing.T) {
	// 准备测试环境
	em := ecs.NewEntityManager()
	// 使用共享的 getTestAudioContext()
	rm := game.NewResourceManager(getTestAudioContext())
	gs := game.GetGameState()

	// 加载粒子配置（模拟真实环境）
	if _, err := rm.LoadParticleConfig("PeaSplat"); err != nil {
		t.Skipf("跳过测试：无法加载粒子资源: %v", err)
	}

	bs := createTestBehaviorSystem(em, rm, gs)

	// 创建测试樱桃炸弹实体
	cherryBombID := em.CreateEntity()
	em.AddComponent(cherryBombID, &components.BehaviorComponent{
		Type: components.BehaviorCherryBomb,
	})
	em.AddComponent(cherryBombID, &components.PositionComponent{
		X: 400.0,
		Y: 250.0,
	})
	em.AddComponent(cherryBombID, &components.PlantComponent{
		GridCol: 3,
		GridRow: 2,
	})

	// 记录触发前的发射器数量
	initialEmitterCount := len(em.GetEntitiesWith(
		reflect.TypeOf(&components.EmitterComponent{}),
		reflect.TypeOf(&components.PositionComponent{}),
	))

	// 触发樱桃炸弹爆炸
	bs.triggerCherryBombExplosion(cherryBombID)

	// 验证：至少创建了粒子发射器实体
	currentEmitterCount := len(em.GetEntitiesWith(
		reflect.TypeOf(&components.EmitterComponent{}),
		reflect.TypeOf(&components.PositionComponent{}),
	))
	if currentEmitterCount <= initialEmitterCount {
		t.Errorf("樱桃炸弹爆炸后未创建粒子发射器实体。初始发射器数: %d, 当前发射器数: %d",
			initialEmitterCount, currentEmitterCount)
	}

	// 验证：查找粒子发射器实体
	emitterEntities := em.GetEntitiesWith(
		reflect.TypeOf(&components.EmitterComponent{}),
		reflect.TypeOf(&components.PositionComponent{}),
	)

	if len(emitterEntities) < 1 {
		t.Errorf("期望至少创建1个粒子发射器（爆炸效果），实际创建: %d", len(emitterEntities))
	}

	// 验证：粒子发射器位置与樱桃炸弹位置匹配
	foundCorrectPosition := false
	for _, emitterID := range emitterEntities {
		posComp, ok := em.GetComponent(emitterID, reflect.TypeOf(&components.PositionComponent{}))
		if !ok {
			continue
		}
		pos := posComp.(*components.PositionComponent)
		// 允许微小的浮点误差
		if pos.X >= 399.0 && pos.X <= 401.0 && pos.Y >= 249.0 && pos.Y <= 251.0 {
			foundCorrectPosition = true
			break
		}
	}

	if !foundCorrectPosition {
		t.Error("粒子发射器位置与樱桃炸弹位置不匹配（期望位置：400, 250）")
	}

	// 验证：粒子发射器配置正确（应该使用 BossExplosion 配置）
	// 由于我们无法直接验证配置名称，这里验证发射器确实被创建且有效
	for _, emitterID := range emitterEntities {
		emitterComp, ok := em.GetComponent(emitterID, reflect.TypeOf(&components.EmitterComponent{}))
		if !ok {
			continue
		}
		emitter := emitterComp.(*components.EmitterComponent)
		if emitter.Config == nil {
			t.Error("粒子发射器配置为空")
		}
		if !emitter.Active {
			t.Error("粒子发射器应该处于激活状态")
		}
	}
}

// TestParticleEffectErrorHandling 测试粒子效果创建失败时的错误处理
// AC 9: 验证粒子配置加载失败时不阻塞游戏逻辑
func TestParticleEffectErrorHandling(t *testing.T) {
	// 准备测试环境（不加载粒子配置，模拟失败场景）
	em := ecs.NewEntityManager()
	// 使用共享的 getTestAudioContext()
	rm := game.NewResourceManager(getTestAudioContext())
	gs := game.GetGameState()

	bs := createTestBehaviorSystem(em, rm, gs)

	// 创建测试僵尸实体
	zombieID := em.CreateEntity()
	em.AddComponent(zombieID, &components.BehaviorComponent{
		Type: components.BehaviorZombieBasic,
	})
	em.AddComponent(zombieID, &components.PositionComponent{
		X: 500.0,
		Y: 300.0,
	})
	em.AddComponent(zombieID, &components.HealthComponent{
		CurrentHealth: 0,
		MaxHealth:     100,
	})
	em.AddComponent(zombieID, &components.VelocityComponent{
		VX: -20.0,
	})
	em.AddComponent(zombieID, &components.ReanimComponent{
		ReanimXML:  nil, //  "Zombie",
		PartImages: make(map[string]*ebiten.Image),
	})

	// 触发僵尸死亡（粒子配置未加载，应该失败但不阻塞）
	// 这里不应该 panic 或返回错误，游戏逻辑应该继续
	bs.triggerZombieDeath(zombieID)

	// 验证：僵尸行为仍然切换为 BehaviorZombieDying（游戏逻辑未被阻塞）
	behaviorComp, ok := em.GetComponent(zombieID, reflect.TypeOf(&components.BehaviorComponent{}))
	if !ok {
		t.Error("僵尸 BehaviorComponent 丢失")
	} else {
		behavior := behaviorComp.(*components.BehaviorComponent)
		if behavior.Type != components.BehaviorZombieDying {
			t.Errorf("粒子创建失败不应阻塞游戏逻辑。僵尸行为类型应为 BehaviorZombieDying，实际: %v", behavior.Type)
		}
	}

	// 验证：僵尸 VelocityComponent 仍然被移除
	if em.HasComponent(zombieID, reflect.TypeOf(&components.VelocityComponent{})) {
		t.Error("粒子创建失败不应阻塞游戏逻辑。VelocityComponent 应该被移除")
	}
}

// TestZombieDeathNoPosition 测试僵尸缺少 PositionComponent 时的错误处理
// 验证缺少位置信息时不会导致崩溃
func TestZombieDeathNoPosition(t *testing.T) {
	// 准备测试环境
	em := ecs.NewEntityManager()
	// 使用共享的 getTestAudioContext()
	rm := game.NewResourceManager(getTestAudioContext())
	gs := game.GetGameState()

	bs := createTestBehaviorSystem(em, rm, gs)

	// 创建测试僵尸实体（故意不添加 PositionComponent）
	zombieID := em.CreateEntity()
	em.AddComponent(zombieID, &components.BehaviorComponent{
		Type: components.BehaviorZombieBasic,
	})
	em.AddComponent(zombieID, &components.VelocityComponent{
		VX: -20.0,
	})
	em.AddComponent(zombieID, &components.ReanimComponent{
		ReanimXML:  nil, //  "Zombie",
		PartImages: make(map[string]*ebiten.Image),
	})

	// 触发僵尸死亡（应该记录警告但不崩溃）
	bs.triggerZombieDeath(zombieID)

	// 验证：僵尸行为仍然切换为 BehaviorZombieDying
	behaviorComp, ok := em.GetComponent(zombieID, reflect.TypeOf(&components.BehaviorComponent{}))
	if !ok {
		t.Error("僵尸 BehaviorComponent 丢失")
	} else {
		behavior := behaviorComp.(*components.BehaviorComponent)
		if behavior.Type != components.BehaviorZombieDying {
			t.Errorf("缺少 PositionComponent 不应阻塞游戏逻辑。僵尸行为类型应为 BehaviorZombieDying，实际: %v", behavior.Type)
		}
	}
}

// TestCherryBombExplosionNoPosition 测试樱桃炸弹缺少 PositionComponent 时的错误处理
func TestCherryBombExplosionNoPosition(t *testing.T) {
	// 准备测试环境
	em := ecs.NewEntityManager()
	// 使用共享的 getTestAudioContext()
	rm := game.NewResourceManager(getTestAudioContext())
	gs := game.GetGameState()

	if _, err := rm.LoadParticleConfig("PeaSplat"); err != nil {
		t.Skipf("跳过测试：无法加载粒子资源: %v", err)
	}

	bs := createTestBehaviorSystem(em, rm, gs)

	// 创建测试樱桃炸弹实体（有 PlantComponent 但无 PositionComponent）
	cherryBombID := em.CreateEntity()
	em.AddComponent(cherryBombID, &components.BehaviorComponent{
		Type: components.BehaviorCherryBomb,
	})
	em.AddComponent(cherryBombID, &components.PlantComponent{
		GridCol: 3,
		GridRow: 2,
	})
	// 故意不添加 PositionComponent

	// 记录触发前的实体数量
	initialEntityCount := countAllEntities(em)

	// 触发樱桃炸弹爆炸（应该记录警告但不崩溃）
	bs.triggerCherryBombExplosion(cherryBombID)

	// 验证：即使缺少 PositionComponent，也应该尝试处理爆炸逻辑
	// 粒子效果可能未创建，但游戏不应崩溃
	currentEntityCount := countAllEntities(em)
	// 这里我们不强制要求创建粒子发射器，因为缺少位置信息
	// 只验证游戏逻辑未崩溃（测试通过即可）
	t.Logf("樱桃炸弹爆炸处理完成。初始实体数: %d, 当前实体数: %d", initialEntityCount, currentEntityCount)
}

// mockEmitterConfig 创建一个模拟的粒子发射器配置（用于不需要真实资源的测试）
func mockEmitterConfig() *particle.EmitterConfig {
	return &particle.EmitterConfig{
		ParticleDuration: "1000", // 1秒
		LaunchSpeed:      "100",
		LaunchAngle:      "0",
	}
}

// countAllEntities 统计EntityManager中的所有实体数量（辅助函数）
func countAllEntities(em *ecs.EntityManager) int {
	// 通过查询所有可能的组件类型来统计实体数量
	// 这是一个简化的实现，真实环境中EntityManager应该提供GetAllEntities方法
	count := 0
	seen := make(map[ecs.EntityID]bool)

	// 查询所有拥有各种常见组件的实体
	componentTypes := []reflect.Type{
		reflect.TypeOf(&components.PositionComponent{}),
		reflect.TypeOf(&components.BehaviorComponent{}),
		reflect.TypeOf(&components.EmitterComponent{}),
		reflect.TypeOf(&components.ParticleComponent{}),
	}

	for _, compType := range componentTypes {
		entities := em.GetEntitiesWith(compType)
		for _, entityID := range entities {
			if !seen[entityID] {
				seen[entityID] = true
				count++
			}
		}
	}

	return count
}
