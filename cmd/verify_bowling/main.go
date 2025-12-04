package main

import (
	"flag"
	"fmt"
	"image/color"
	"log"
	"os"

	"github.com/decker502/pvz/pkg/components"
	"github.com/decker502/pvz/pkg/config"
	"github.com/decker502/pvz/pkg/ecs"
	"github.com/decker502/pvz/pkg/entities"
	"github.com/decker502/pvz/pkg/game"
	"github.com/decker502/pvz/pkg/systems"
	"github.com/decker502/pvz/pkg/systems/behavior"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

const (
	screenWidth  = 800
	screenHeight = 600
)

var (
	verbose = flag.Bool("verbose", false, "显示详细调试信息")
)

// VerifyBowlingGame 保龄球坚果验证程序
type VerifyBowlingGame struct {
	entityManager   *ecs.EntityManager
	gameState       *game.GameState
	resourceManager *game.ResourceManager

	// 核心系统
	reanimSystem      *systems.ReanimSystem
	renderSystem      *systems.RenderSystem
	behaviorSystem    *behavior.BehaviorSystem
	physicsSystem     *systems.PhysicsSystem
	bowlingNutSystem  *systems.BowlingNutSystem
	particleSystem    *systems.ParticleSystem
	lawnGridSystem    *systems.LawnGridSystem
	flashEffectSystem *systems.FlashEffectSystem

	// 调试字体
	debugFont *text.GoTextFace

	// 背景图片
	background *ebiten.Image

	// 草坪网格实体
	lawnGridEntityID ecs.EntityID

	// 僵尸生成开关
	zombieSpawnEnabled bool

	// 僵尸生成计时器（每行独立）
	zombieSpawnTimers [5]float64

	// 统计信息
	bowlingNutCount   int
	zombieKillCount   int
	totalBounceCount  int
}

// NewVerifyBowlingGame 创建验证游戏实例
func NewVerifyBowlingGame() (*VerifyBowlingGame, error) {
	// 创建 ECS 管理器
	em := ecs.NewEntityManager()

	// 创建音频上下文
	audioContext := audio.NewContext(48000)

	// 创建资源管理器
	rm := game.NewResourceManager(audioContext)

	// 加载资源配置
	if err := rm.LoadResourceConfig("assets/config/resources.yaml"); err != nil {
		return nil, fmt.Errorf("failed to load resource config: %w", err)
	}

	// 加载所有资源组
	log.Println("Loading all resources...")
	if err := rm.LoadAllResources(); err != nil {
		return nil, fmt.Errorf("failed to load resources: %w", err)
	}

	// 加载 Reanim 资源
	log.Println("Loading Reanim resources...")
	if err := rm.LoadReanimResources(); err != nil {
		return nil, fmt.Errorf("failed to load Reanim resources: %w", err)
	}

	// 加载 Reanim 配置
	log.Println("Loading Reanim config...")
	reanimConfigManager, err := config.NewReanimConfigManager("data/reanim_config")
	if err != nil {
		return nil, fmt.Errorf("failed to load Reanim config: %w", err)
	}

	// 获取游戏状态单例
	gs := game.GetGameState()
	gs.CameraX = config.GameCameraX

	// 创建系统
	reanimSystem := systems.NewReanimSystem(em)
	reanimSystem.SetConfigManager(reanimConfigManager)
	reanimSystem.SetResourceLoader(rm)

	renderSystem := systems.NewRenderSystem(em)
	renderSystem.SetReanimSystem(reanimSystem)
	renderSystem.SetResourceManager(rm)

	particleSystem := systems.NewParticleSystem(em, rm)

	// 创建草坪网格系统（启用所有行）
	enabledLanes := []int{1, 2, 3, 4, 5}
	lawnGridSystem := systems.NewLawnGridSystem(em, enabledLanes)

	// 创建行为系统
	behaviorSystem := behavior.NewBehaviorSystem(em, rm, gs, lawnGridSystem, 0)

	// 创建物理系统
	physicsSystem := systems.NewPhysicsSystem(em, rm)

	// 创建保龄球坚果系统
	bowlingNutSystem := systems.NewBowlingNutSystem(em, rm)

	// 创建闪烁效果系统
	flashEffectSystem := systems.NewFlashEffectSystem(em)

	// 加载字体
	debugFont, err := rm.LoadFont("assets/fonts/SimHei.ttf", 14)
	if err != nil {
		log.Printf("Warning: Failed to load debug font: %v", err)
	}

	// 加载背景图片
	background, err := rm.LoadImageByID("IMAGE_BACKGROUND1")
	if err != nil {
		log.Printf("Warning: Failed to load background: %v", err)
	}

	log.Println("╔════════════════════════════════════════════════════════╗")
	log.Println("║         保龄球坚果验证程序 (Level 1-5)                 ║")
	log.Println("╚════════════════════════════════════════════════════════╝")
	log.Println()
	log.Println("【功能说明】")
	log.Println("  - 验证保龄球坚果的滚动动画和位移")
	log.Println("  - 验证坚果与僵尸的碰撞检测")
	log.Println("  - 验证碰撞伤害和弹射效果")
	log.Println("  - 验证爆炸坚果的范围伤害")
	log.Println()
	log.Println("【快捷键】")
	log.Println("  1-5       - 在对应行放置普通坚果")
	log.Println("  Shift+1-5 - 在对应行放置爆炸坚果")
	log.Println("  Ctrl+1-5  - 在对应行生成僵尸")
	log.Println("  Z         - 开启/关闭自动生成僵尸")
	log.Println("  C         - 清除所有僵尸")
	log.Println("  N         - 清除所有坚果")
	log.Println("  R         - 重置统计信息")
	log.Println("  Q         - 退出程序")
	log.Println("════════════════════════════════════════════════════════")

	vg := &VerifyBowlingGame{
		entityManager:     em,
		gameState:         gs,
		resourceManager:   rm,
		reanimSystem:      reanimSystem,
		renderSystem:      renderSystem,
		behaviorSystem:    behaviorSystem,
		physicsSystem:     physicsSystem,
		bowlingNutSystem:  bowlingNutSystem,
		particleSystem:    particleSystem,
		lawnGridSystem:    lawnGridSystem,
		flashEffectSystem: flashEffectSystem,
		debugFont:         debugFont,
		background:        background,
	}

	// 初始化场景
	vg.setupScene()

	return vg, nil
}

// setupScene 设置测试场景
func (vg *VerifyBowlingGame) setupScene() {
	// 创建草坪网格实体
	vg.lawnGridEntityID = vg.entityManager.CreateEntity()
	ecs.AddComponent(vg.entityManager, vg.lawnGridEntityID, &components.LawnGridComponent{})

	// 自动生成几个僵尸进行测试
	log.Println("[VerifyBowling] 自动生成僵尸进行测试...")
	vg.spawnZombie(1) // 第2行
	vg.spawnZombie(2) // 第3行
	vg.spawnZombie(3) // 第4行

	// 自动放置保龄球坚果进行测试
	log.Println("[VerifyBowling] 自动放置保龄球坚果进行测试...")
	vg.spawnBowlingNut(2, false) // 在第3行放置普通坚果
}

// spawnBowlingNut 放置保龄球坚果
func (vg *VerifyBowlingGame) spawnBowlingNut(row int, isExplosive bool) {
	if row < 0 || row >= 5 {
		return
	}

	// 放置在第0列（最左边）
	col := 0

	nutID, err := entities.NewBowlingNutEntity(
		vg.entityManager,
		vg.resourceManager,
		row,
		col,
		isExplosive,
	)
	if err != nil {
		log.Printf("Warning: Failed to spawn bowling nut: %v", err)
		return
	}

	vg.bowlingNutCount++

	nutType := "普通"
	if isExplosive {
		nutType = "爆炸"
	}
	if *verbose {
		log.Printf("[VerifyBowling] 放置%s坚果: entityID=%d, row=%d, col=%d", nutType, nutID, row, col)
	}
}

// spawnZombie 在指定行生成僵尸
func (vg *VerifyBowlingGame) spawnZombie(row int) {
	if row < 0 || row >= 5 {
		return
	}

	// 僵尸出生在屏幕可视范围最右端
	spawnX := config.GameCameraX + float64(screenWidth) - 30.0

	// 随机选择僵尸类型
	var zombieID ecs.EntityID
	var err error
	var zombieType string

	// 50% 普通僵尸, 50% 路障僵尸
	if vg.zombieKillCount%2 == 0 {
		zombieID, err = entities.NewZombieEntity(vg.entityManager, vg.resourceManager, row, spawnX)
		zombieType = "basic"
	} else {
		zombieID, err = entities.NewConeheadZombieEntity(vg.entityManager, vg.resourceManager, row, spawnX)
		zombieType = "conehead"
	}

	if err != nil {
		log.Printf("Warning: Failed to spawn zombie: %v", err)
		return
	}

	// 立即激活僵尸（设置速度）
	vel, ok := ecs.GetComponent[*components.VelocityComponent](vg.entityManager, zombieID)
	if ok {
		vel.VX = config.ZombieWalkSpeed
	}

	// 切换到行走动画
	behaviorComp, ok := ecs.GetComponent[*components.BehaviorComponent](vg.entityManager, zombieID)
	if ok {
		behaviorComp.ZombieAnimState = components.ZombieAnimWalking
	}

	// 播放行走动画
	unitID := "zombie"
	if zombieType == "conehead" {
		unitID = "zombie_conehead"
	}
	ecs.AddComponent(vg.entityManager, zombieID, &components.AnimationCommandComponent{
		UnitID:    unitID,
		ComboName: "walk",
		Processed: false,
	})

	if *verbose {
		log.Printf("[VerifyBowling] Spawned %s zombie on row %d (entity=%d)", zombieType, row, zombieID)
	}
}

// clearAllZombies 清除所有僵尸
func (vg *VerifyBowlingGame) clearAllZombies() {
	zombies := ecs.GetEntitiesWith1[*components.BehaviorComponent](vg.entityManager)

	count := 0
	for _, entityID := range zombies {
		behaviorComp, _ := ecs.GetComponent[*components.BehaviorComponent](vg.entityManager, entityID)
		if behaviorComp.Type == components.BehaviorZombieBasic ||
			behaviorComp.Type == components.BehaviorZombieConehead ||
			behaviorComp.Type == components.BehaviorZombieBuckethead ||
			behaviorComp.Type == components.BehaviorZombieFlag {
			vg.entityManager.DestroyEntity(entityID)
			count++
		}
	}
	vg.entityManager.RemoveMarkedEntities()
	log.Printf("[VerifyBowling] Cleared %d zombies", count)
}

// clearAllNuts 清除所有保龄球坚果
func (vg *VerifyBowlingGame) clearAllNuts() {
	nuts := ecs.GetEntitiesWith1[*components.BowlingNutComponent](vg.entityManager)

	count := 0
	for _, entityID := range nuts {
		vg.bowlingNutSystem.StopAllSounds()
		vg.entityManager.DestroyEntity(entityID)
		count++
	}
	vg.entityManager.RemoveMarkedEntities()
	log.Printf("[VerifyBowling] Cleared %d bowling nuts", count)
}

// countZombies 统计当前僵尸数量
func (vg *VerifyBowlingGame) countZombies() int {
	zombies := ecs.GetEntitiesWith1[*components.BehaviorComponent](vg.entityManager)

	count := 0
	for _, entityID := range zombies {
		behaviorComp, _ := ecs.GetComponent[*components.BehaviorComponent](vg.entityManager, entityID)
		if behaviorComp.Type == components.BehaviorZombieBasic ||
			behaviorComp.Type == components.BehaviorZombieConehead ||
			behaviorComp.Type == components.BehaviorZombieBuckethead ||
			behaviorComp.Type == components.BehaviorZombieFlag {
			count++
		}
	}
	return count
}

// countNuts 统计当前保龄球坚果数量
func (vg *VerifyBowlingGame) countNuts() int {
	nuts := ecs.GetEntitiesWith1[*components.BowlingNutComponent](vg.entityManager)
	return len(nuts)
}

// getTotalBounceCount 获取当前所有坚果的总弹射次数
func (vg *VerifyBowlingGame) getTotalBounceCount() int {
	nuts := ecs.GetEntitiesWith1[*components.BowlingNutComponent](vg.entityManager)

	total := 0
	for _, entityID := range nuts {
		nutComp, _ := ecs.GetComponent[*components.BowlingNutComponent](vg.entityManager, entityID)
		total += nutComp.BounceCount
	}
	return total
}

// Update 更新游戏逻辑
func (vg *VerifyBowlingGame) Update() error {
	dt := 1.0 / 60.0

	// 快捷键处理
	// 1-5 放置普通坚果（不按修饰键时）
	if !ebiten.IsKeyPressed(ebiten.KeyShift) && !ebiten.IsKeyPressed(ebiten.KeyControl) {
		if inpututil.IsKeyJustPressed(ebiten.Key1) {
			vg.spawnBowlingNut(0, false)
		}
		if inpututil.IsKeyJustPressed(ebiten.Key2) {
			vg.spawnBowlingNut(1, false)
		}
		if inpututil.IsKeyJustPressed(ebiten.Key3) {
			vg.spawnBowlingNut(2, false)
		}
		if inpututil.IsKeyJustPressed(ebiten.Key4) {
			vg.spawnBowlingNut(3, false)
		}
		if inpututil.IsKeyJustPressed(ebiten.Key5) {
			vg.spawnBowlingNut(4, false)
		}
	}

	// Shift+1-5 放置爆炸坚果
	if ebiten.IsKeyPressed(ebiten.KeyShift) && !ebiten.IsKeyPressed(ebiten.KeyControl) {
		if inpututil.IsKeyJustPressed(ebiten.Key1) {
			vg.spawnBowlingNut(0, true)
		}
		if inpututil.IsKeyJustPressed(ebiten.Key2) {
			vg.spawnBowlingNut(1, true)
		}
		if inpututil.IsKeyJustPressed(ebiten.Key3) {
			vg.spawnBowlingNut(2, true)
		}
		if inpututil.IsKeyJustPressed(ebiten.Key4) {
			vg.spawnBowlingNut(3, true)
		}
		if inpututil.IsKeyJustPressed(ebiten.Key5) {
			vg.spawnBowlingNut(4, true)
		}
	}

	// Ctrl+1-5 生成僵尸
	if ebiten.IsKeyPressed(ebiten.KeyControl) {
		if inpututil.IsKeyJustPressed(ebiten.Key1) {
			vg.spawnZombie(0)
		}
		if inpututil.IsKeyJustPressed(ebiten.Key2) {
			vg.spawnZombie(1)
		}
		if inpututil.IsKeyJustPressed(ebiten.Key3) {
			vg.spawnZombie(2)
		}
		if inpututil.IsKeyJustPressed(ebiten.Key4) {
			vg.spawnZombie(3)
		}
		if inpututil.IsKeyJustPressed(ebiten.Key5) {
			vg.spawnZombie(4)
		}
	}

	// Z 开启/关闭自动生成僵尸
	if inpututil.IsKeyJustPressed(ebiten.KeyZ) {
		vg.zombieSpawnEnabled = !vg.zombieSpawnEnabled
		if vg.zombieSpawnEnabled {
			log.Println("[VerifyBowling] Zombie spawn ENABLED")
		} else {
			log.Println("[VerifyBowling] Zombie spawn DISABLED")
		}
	}

	// C 清除所有僵尸
	if inpututil.IsKeyJustPressed(ebiten.KeyC) {
		vg.clearAllZombies()
	}

	// N 清除所有坚果
	if inpututil.IsKeyJustPressed(ebiten.KeyN) {
		vg.clearAllNuts()
	}

	// R 重置统计信息
	if inpututil.IsKeyJustPressed(ebiten.KeyR) {
		vg.bowlingNutCount = 0
		vg.zombieKillCount = 0
		vg.totalBounceCount = 0
		log.Println("[VerifyBowling] Statistics reset")
	}

	// Q 退出
	if inpututil.IsKeyJustPressed(ebiten.KeyQ) {
		log.Println("[VerifyBowling] Exiting...")
		return fmt.Errorf("quit")
	}

	// 僵尸自动生成（每行每3秒生成一个）
	if vg.zombieSpawnEnabled {
		for row := 0; row < 5; row++ {
			vg.zombieSpawnTimers[row] += dt
			if vg.zombieSpawnTimers[row] >= 3.0 {
				vg.spawnZombie(row)
				vg.zombieSpawnTimers[row] = 0
			}
		}
	}

	// 更新各个系统
	vg.reanimSystem.Update(dt)
	vg.behaviorSystem.Update(dt)
	vg.physicsSystem.Update(dt)
	vg.bowlingNutSystem.Update(dt)
	vg.particleSystem.Update(dt)
	vg.lawnGridSystem.Update(dt)
	vg.flashEffectSystem.Update(dt)

	// 清理已删除的实体
	vg.entityManager.RemoveMarkedEntities()

	return nil
}

// Draw 绘制游戏画面
func (vg *VerifyBowlingGame) Draw(screen *ebiten.Image) {
	// 清空屏幕
	screen.Fill(color.RGBA{0, 0, 0, 255})

	// 绘制背景
	if vg.background != nil {
		opts := &ebiten.DrawImageOptions{}
		opts.GeoM.Translate(-vg.gameState.CameraX, 0)
		screen.DrawImage(vg.background, opts)
	}

	// 绘制红线（保龄球关卡的区域分隔线）
	vg.drawRedLine(screen)

	// 绘制游戏世界元素
	vg.renderSystem.DrawGameWorld(screen, vg.gameState.CameraX)

	// 绘制粒子效果
	vg.renderSystem.DrawGameWorldParticles(screen, vg.gameState.CameraX)

	// 绘制调试信息
	vg.drawDebugInfo(screen)
}

// drawRedLine 绘制保龄球关卡的红色区域分隔线
func (vg *VerifyBowlingGame) drawRedLine(screen *ebiten.Image) {
	// 计算红线位置
	lineX := config.GridWorldStartX + float64(config.BowlingRedLineColumn)*config.CellWidth - vg.gameState.CameraX
	lineY1 := config.GridWorldStartY + config.BowlingRedLineOffsetY
	lineY2 := config.GridWorldStartY + float64(config.GridRows)*config.CellHeight

	// 绘制红线
	lineImg := ebiten.NewImage(2, int(lineY2-lineY1))
	lineImg.Fill(color.RGBA{R: 255, G: 0, B: 0, A: 200})

	opts := &ebiten.DrawImageOptions{}
	opts.GeoM.Translate(lineX, lineY1)
	screen.DrawImage(lineImg, opts)
}

// drawDebugInfo 绘制调试信息
func (vg *VerifyBowlingGame) drawDebugInfo(screen *ebiten.Image) {
	if vg.debugFont == nil {
		return
	}

	y := 20.0
	clr := color.RGBA{255, 255, 255, 220}

	// 统计信息
	stats := []string{
		fmt.Sprintf("保龄球坚果: %d (已放置: %d)", vg.countNuts(), vg.bowlingNutCount),
		fmt.Sprintf("僵尸数量: %d", vg.countZombies()),
		fmt.Sprintf("总弹射次数: %d", vg.getTotalBounceCount()),
	}

	for _, stat := range stats {
		op := &text.DrawOptions{}
		op.GeoM.Translate(10, y)
		op.ColorScale.ScaleWithColor(clr)
		text.Draw(screen, stat, vg.debugFont, op)
		y += 18
	}

	// 分隔线
	y += 10

	// 快捷键提示
	hints := []string{
		"1-5: 放置普通坚果 | Shift+1-5: 放置爆炸坚果",
		"Ctrl+1-5: 生成僵尸 | Z: 自动生成僵尸",
		"C: 清除僵尸 | N: 清除坚果 | R: 重置统计 | Q: 退出",
	}

	spawnStatus := "已启用"
	if !vg.zombieSpawnEnabled {
		spawnStatus = "已关闭"
	}
	hints = append(hints, fmt.Sprintf("自动生成僵尸: %s", spawnStatus))

	y = float64(screenHeight - 80)
	for _, hint := range hints {
		op := &text.DrawOptions{}
		op.GeoM.Translate(10, y)
		op.ColorScale.ScaleWithColor(clr)
		text.Draw(screen, hint, vg.debugFont, op)
		y += 18
	}
}

// Layout 设置屏幕布局
func (vg *VerifyBowlingGame) Layout(outsideWidth, outsideHeight int) (int, int) {
	return screenWidth, screenHeight
}

func main() {
	flag.Parse()

	if !*verbose {
		log.SetOutput(os.Stdout)
	}

	verifyGame, err := NewVerifyBowlingGame()
	if err != nil {
		log.Fatalf("Failed to create verify game: %v", err)
	}

	ebiten.SetWindowTitle("保龄球坚果验证程序 (Level 1-5)")
	ebiten.SetWindowSize(screenWidth, screenHeight)

	if err := ebiten.RunGame(verifyGame); err != nil {
		if err.Error() != "quit" {
			log.Fatal(err)
		}
	}
}
