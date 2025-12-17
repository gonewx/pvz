package systems

import (
	"log"

	"github.com/gonewx/pvz/pkg/components"
	"github.com/gonewx/pvz/pkg/config"
	"github.com/gonewx/pvz/pkg/ecs"
)

// ========================================
// 关卡进度条支持
// ========================================

// SetProgressBarEntity 设置关卡进度条实体ID（由 GameScene 初始化时调用）
func (s *LevelSystem) SetProgressBarEntity(entityID ecs.EntityID) {
	s.progressBarEntityID = entityID
	s.initializeProgressBar()
}

// initializeProgressBar 初始化进度条（计算总僵尸数、旗帜位置）
//
// Story 11.5: 增加原版进度条机制的初始化
func (s *LevelSystem) initializeProgressBar() {
	if s.progressBarEntityID == 0 {
		return
	}

	progressBar, ok := ecs.GetComponent[*components.LevelProgressBarComponent](s.entityManager, s.progressBarEntityID)
	if !ok {
		log.Printf("[LevelSystem] WARNING: LevelProgressBarComponent not found for entity %d", s.progressBarEntityID)
		return
	}

	// 1. 计算总僵尸数（废弃字段，保留向后兼容）
	totalZombies := s.calculateTotalZombies()
	progressBar.TotalZombies = totalZombies
	progressBar.KilledZombies = 0
	progressBar.ProgressPercent = 0.0

	// 2. 计算旗帜位置百分比
	if s.gameState.CurrentLevel != nil && len(s.gameState.CurrentLevel.FlagWaves) > 0 {
		progressBar.FlagPositions = s.calculateFlagPositions()
	} else {
		progressBar.FlagPositions = []float64{} // 无旗帜
	}

	// 3. 设置关卡文本
	if s.gameState.CurrentLevel != nil {
		progressBar.LevelText = "关卡 " + s.gameState.CurrentLevel.ID
	}

	// 4. 默认只显示文本（第一波生成前）
	progressBar.ShowLevelTextOnly = true

	// === Story 11.5: 初始化原版进度条机制 ===
	s.initProgressBarStructure(progressBar)

	log.Printf("[LevelSystem] Progress bar initialized: total zombies=%d, flags=%v, totalWaves=%d",
		totalZombies, progressBar.FlagPositions, progressBar.TotalWaves)
}

// calculateTotalZombies 计算关卡总僵尸数
func (s *LevelSystem) calculateTotalZombies() int {
	if s.gameState.CurrentLevel == nil {
		return 0
	}

	total := 0
	for _, wave := range s.gameState.CurrentLevel.Waves {
		// 新格式：Zombies 字段
		for _, zombie := range wave.Zombies {
			total += zombie.Count
		}
		// 旧格式：OldZombies 字段（向后兼容）
		for _, zombie := range wave.OldZombies {
			total += zombie.Count
		}
	}

	return total
}

// calculateFlagPositions 计算旗帜在进度条上的位置百分比
//
// Story 11.5 修复：使用双段式结构计算旗帜位置
// 原版机制：
//   - 进度条总长 = 150 单位
//   - 红字波段 = 旗帜数 × 12（每个旗帜波刷新时立即 +12）
//   - 普通波段 = 150 - 红字波段（平均分配给普通波）
//   - 旗帜位置 = 该旗帜波刷新前的进度位置
//
// 计算公式：
//
//	旗帜位置 = (已完成红字波数 × 12 + 已完成普通波数 × 每波普通进度) / 150
func (s *LevelSystem) calculateFlagPositions() []float64 {
	if s.gameState.CurrentLevel == nil {
		return []float64{}
	}

	levelConfig := s.gameState.CurrentLevel
	flagWaves := levelConfig.FlagWaves
	totalWaves := len(levelConfig.Waves)
	flagCount := len(flagWaves)

	if flagCount == 0 || totalWaves == 0 {
		return []float64{}
	}

	// 双段式结构计算
	totalLength := float64(config.ProgressBarTotalLength)       // 150
	flagSegment := float64(config.ProgressBarFlagSegmentLength) // 12
	normalSegment := totalLength - float64(flagCount)*flagSegment

	// 普通波数 = 总波数 - 旗帜波数
	normalWaveCount := totalWaves - flagCount

	// 每波普通进度
	progressPerNormalWave := 0.0
	if normalWaveCount > 0 {
		progressPerNormalWave = normalSegment / float64(normalWaveCount)
	}

	positions := make([]float64, 0, flagCount)

	// 构建旗帜波索引集合（用于快速查找）
	flagWaveSet := make(map[int]bool)
	for _, idx := range flagWaves {
		flagWaveSet[idx] = true
	}

	// 遍历所有波次，计算每个旗帜波的位置
	completedFlags := 0
	completedNormalWaves := 0

	for waveIdx := 0; waveIdx < totalWaves; waveIdx++ {
		if flagWaveSet[waveIdx] {
			// 这是旗帜波，计算其位置（刷新前的进度）
			flagProgress := float64(completedFlags) * flagSegment
			normalProgress := float64(completedNormalWaves) * progressPerNormalWave
			position := (flagProgress + normalProgress) / totalLength

			// 进度条从右到左显示，旗帜段在左边，需要反转位置
			// 原始位置 92% -> 显示位置 8%（1 - 0.92 = 0.08）
			position = 1.0 - position

			// 限制在 [0, 1] 范围内
			if position < 0 {
				position = 0
			}
			positions = append(positions, position)

			completedFlags++
		} else {
			completedNormalWaves++
		}
	}

	return positions
}

// UpdateProgressBar 更新进度条进度
//
// Story 11.5: 实现原版进度条的双重进度计算和虚拟/现实追踪
// 每帧调用，更新游戏时钟、虚拟进度和现实进度
func (s *LevelSystem) UpdateProgressBar() {
	if s.progressBarEntityID == 0 {
		return
	}

	progressBar, ok := ecs.GetComponent[*components.LevelProgressBarComponent](s.entityManager, s.progressBarEntityID)
	if !ok {
		log.Printf("[ProgressBar] ERROR: failed to get progress bar component for entity %d", s.progressBarEntityID)
		return
	}

	// === 废弃逻辑（向后兼容） ===
	// 统计当前击杀的僵尸数（通过 GameState.ZombiesKilled）
	killedZombies := s.gameState.ZombiesKilled
	progressBar.KilledZombies = killedZombies

	// === Story 11.5: 原版进度条机制 ===

	// 1. 更新游戏时钟（厘秒）
	s.updateGameTickCS(progressBar)

	// 2. 更新血量追踪
	s.updateWaveHealthTracking(progressBar)

	// 记录旧的虚拟进度用于调试
	oldVirtual := progressBar.VirtualProgress
	oldReal := progressBar.RealProgress

	// 3. 计算虚拟进度
	s.calculateVirtualProgress(progressBar)

	// 4. 更新现实进度（平滑追踪）
	s.updateRealProgress(progressBar)

	// 调试日志：每秒输出一次（检查整秒边界）
	currentSecond := progressBar.GameTickCS / 100
	if currentSecond > 0 && progressBar.GameTickCS%100 < 10 {
		log.Printf("[ProgressBar] wave=%d, gameTickCS=%d, virtual=%.4f->%.4f, real=%.4f->%.4f, timeProgress=%.4f, dmgProgress=%.4f, WaveInitialDelay=%.2f",
			progressBar.CurrentWaveNum, progressBar.GameTickCS,
			oldVirtual, progressBar.VirtualProgress,
			oldReal, progressBar.RealProgress,
			s.calculateTimeProgress(progressBar),
			s.calculateDamageProgress(progressBar),
			progressBar.WaveInitialDelay)
	}

	// 5. 同步到废弃字段（向后兼容）
	progressBar.ProgressPercent = progressBar.RealProgress
	if progressBar.ProgressPercent > 1.0 {
		progressBar.ProgressPercent = 1.0
	}
}

// ShowProgressBar 显示完整进度条（第一波僵尸生成后调用）
func (s *LevelSystem) ShowProgressBar() {
	if s.progressBarEntityID == 0 {
		return
	}

	progressBar, ok := ecs.GetComponent[*components.LevelProgressBarComponent](s.entityManager, s.progressBarEntityID)
	if !ok {
		return
	}

	// 切换到完整显示模式
	progressBar.ShowLevelTextOnly = false
	log.Println("[LevelSystem] Progress bar now showing full display (background + progress + flags)")
}

// ========================================
// Story 11.5: 原版进度条机制实现
// ========================================

// initProgressBarStructure 初始化进度条双段式结构
//
// Story 11.5 Task 2: 实现双段式进度条分配逻辑
// - 计算总波次数
// - 计算红字波段和普通波段长度
// - 初始化波次追踪状态
//
// 参数:
//   - pb: 进度条组件
func (s *LevelSystem) initProgressBarStructure(pb *components.LevelProgressBarComponent) {
	if s.gameState.CurrentLevel == nil {
		return
	}

	levelConfig := s.gameState.CurrentLevel

	// 1. 设置进度条总长度
	pb.TotalProgressLength = config.ProgressBarTotalLength
	pb.FlagSegmentLength = config.ProgressBarFlagSegmentLength

	// 2. 计算总波次数
	pb.TotalWaves = len(levelConfig.Waves)
	if pb.TotalWaves == 0 {
		pb.TotalWaves = 1 // 防止除零
	}

	// 3. 计算红字波数量和普通波段长度
	flagCount := len(levelConfig.FlagWaves)
	pb.NormalSegmentBase = pb.TotalProgressLength - (flagCount * pb.FlagSegmentLength)
	if pb.NormalSegmentBase < 0 {
		pb.NormalSegmentBase = 0
	}

	// 4. 初始化波次追踪状态
	pb.CurrentWaveNum = 0 // 游戏开始前为 0，第一波激活后为 1
	pb.FlagWaveCount = 0

	// 5. 初始化时间和血量追踪
	pb.WaveStartTime = 0
	pb.WaveInitialDelay = 0
	pb.WaveInitialHealth = 0
	pb.WaveCurrentHealth = 0
	pb.WaveRequiredDamage = 0

	// 6. 初始化虚拟/现实进度
	pb.VirtualProgress = 0
	pb.RealProgress = 0

	// 7. 初始化游戏时钟
	pb.GameTickCS = 0
	pb.LastTrackUpdateCS = 0
}

// updateGameTickCS 更新游戏时钟（厘秒）
//
// Story 11.5 Task 7: 游戏时钟计数器
// 将游戏时间（秒）转换为厘秒累加
//
// 参数:
//   - pb: 进度条组件
func (s *LevelSystem) updateGameTickCS(pb *components.LevelProgressBarComponent) {
	// 从 GameState 获取当前关卡时间（秒）
	levelTime := s.gameState.LevelTime

	// 转换为厘秒
	pb.GameTickCS = int(levelTime * config.CentisecondsPerSecond)
}

// updateWaveHealthTracking 更新血量追踪
//
// Story 11.5 Task 4: 实现血量追踪系统
// 每帧查询所有僵尸的当前血量总和
//
// 参数:
//   - pb: 进度条组件
func (s *LevelSystem) updateWaveHealthTracking(pb *components.LevelProgressBarComponent) {
	// 查询所有僵尸实体的血量
	totalHealth := 0.0

	zombieEntities := ecs.GetEntitiesWith2[
		*components.BehaviorComponent,
		*components.HealthComponent,
	](s.entityManager)

	for _, entityID := range zombieEntities {
		behavior, ok := ecs.GetComponent[*components.BehaviorComponent](s.entityManager, entityID)
		if !ok {
			continue
		}

		// 只统计僵尸类型
		if !isZombieType(behavior.Type) {
			continue
		}

		health, ok := ecs.GetComponent[*components.HealthComponent](s.entityManager, entityID)
		if !ok {
			continue
		}

		if health.CurrentHealth > 0 {
			totalHealth += float64(health.CurrentHealth)
		}
	}

	pb.WaveCurrentHealth = totalHealth
}

// calculateVirtualProgress 计算虚拟进度
//
// Story 11.5 Task 3: 实现双重进度计算逻辑
// - 红字波段：每个红字波贡献 12 格
// - 普通波段：根据左端点/右端点和波进度计算
// - 波进度 = max(时间进度, 血量削减进度)
//
// Story 11.5 Task 5: 进度允许超过右端点（不限制在 1.0 内）
//
// 参数:
//   - pb: 进度条组件
func (s *LevelSystem) calculateVirtualProgress(pb *components.LevelProgressBarComponent) {
	if pb.TotalProgressLength == 0 {
		return
	}

	// 1. 红字波段进度
	flagProgress := float64(pb.FlagWaveCount * pb.FlagSegmentLength)

	// 2. 普通波段进度
	normalProgress := 0.0

	if pb.TotalWaves > 1 && pb.CurrentWaveNum > 0 {
		// 计算左端点和右端点（整数除法向下取整）
		leftEndpoint := float64((pb.CurrentWaveNum - 1) * pb.NormalSegmentBase / (pb.TotalWaves - 1))
		rightEndpoint := float64(pb.CurrentWaveNum * pb.NormalSegmentBase / (pb.TotalWaves - 1))

		// 计算波进度（取时间进度和血量削减进度的最大值）
		waveProgress := s.calculateWaveProgress(pb)

		// 限制波进度不超过 1.0，防止进度条溢出
		// 只有在波次真正切换时（通过 OnWaveActivated），才会进入下一段
		if waveProgress > 1.0 {
			waveProgress = 1.0
		}

		// 计算普通波段内的进度
		// Story 11.5 Task 5: 不限制 waveProgress，允许超过 1.0
		normalProgress = (rightEndpoint-leftEndpoint)*waveProgress + leftEndpoint
	} else if pb.TotalWaves == 1 && pb.CurrentWaveNum > 0 {
		// 单波次关卡：普通波段进度直接等于波进度
		waveProgress := s.calculateWaveProgress(pb)
		if waveProgress > 1.0 {
			waveProgress = 1.0
		}
		normalProgress = float64(pb.NormalSegmentBase) * waveProgress
	}

	// 3. 计算总虚拟进度（允许超过 1.0）
	pb.VirtualProgress = (flagProgress + normalProgress) / float64(pb.TotalProgressLength)
}

// calculateWaveProgress 计算当前波进度
//
// Story 11.5 Task 3: 双重进度计算
// 波进度 = max(时间进度, 血量削减进度)
//
// 参数:
//   - pb: 进度条组件
//
// 返回:
//   - float64: 波进度（0.0 到 1.0+，可超过 1.0）
func (s *LevelSystem) calculateWaveProgress(pb *components.LevelProgressBarComponent) float64 {
	timeProgress := s.calculateTimeProgress(pb)
	damageProgress := s.calculateDamageProgress(pb)

	// 取两者最大值
	if timeProgress > damageProgress {
		return timeProgress
	}
	return damageProgress
}

// calculateTimeProgress 计算时间进度
//
// Story 11.5 Task 3: 时间进度 = 本波已存在时间 / 本波初始刷新倒计时
//
// 参数:
//   - pb: 进度条组件
//
// 返回:
//   - float64: 时间进度（0.0 到 1.0+）
func (s *LevelSystem) calculateTimeProgress(pb *components.LevelProgressBarComponent) float64 {
	if pb.WaveInitialDelay <= 0 {
		return 0
	}

	// 计算本波已存在时间
	elapsed := s.gameState.LevelTime - pb.WaveStartTime
	if elapsed < 0 {
		elapsed = 0
	}

	// 时间进度 = 已存在时间 / 初始倒计时
	return elapsed / pb.WaveInitialDelay
}

// calculateDamageProgress 计算血量削减进度
//
// Story 11.5 Task 3: 血量削减进度 = 已削减血量 / 所需削减血量
//
// 参数:
//   - pb: 进度条组件
//
// 返回:
//   - float64: 血量削减进度（0.0 到 1.0+）
func (s *LevelSystem) calculateDamageProgress(pb *components.LevelProgressBarComponent) float64 {
	if pb.WaveRequiredDamage <= 0 {
		return 0
	}

	// 计算已削减血量
	damageDealt := pb.WaveInitialHealth - pb.WaveCurrentHealth
	if damageDealt < 0 {
		damageDealt = 0
	}

	// 血量削减进度 = 已削减血量 / 所需削减血量
	return damageDealt / pb.WaveRequiredDamage
}

// updateRealProgress 更新现实进度（平滑追踪）
//
// Story 11.5 Task 6: 虚拟/现实双层追踪机制
// - 落后 1-6 格：每 20cs (0.2秒) 前进一格
// - 落后 7+ 格：每 5cs (0.05秒) 前进一格
// - 不落后时：不更新
//
// 参数:
//   - pb: 进度条组件
func (s *LevelSystem) updateRealProgress(pb *components.LevelProgressBarComponent) {
	// 计算虚拟与现实的差距（格数）
	diff := pb.VirtualProgress - pb.RealProgress
	if diff <= 0 {
		// 不落后或现实超前（回退情况），直接同步
		if pb.RealProgress > pb.VirtualProgress {
			pb.RealProgress = pb.VirtualProgress
		}
		return
	}

	// 转换为格数
	diffInUnits := int(diff * float64(pb.TotalProgressLength))
	if diffInUnits < 1 {
		return // 差距小于 1 格，不更新
	}

	// 计算每格对应的进度增量
	stepSize := 1.0 / float64(pb.TotalProgressLength)

	// 根据差距选择追踪速度
	if diffInUnits >= config.FastTrackThreshold {
		// 落后 7+ 格：快速追踪（每 5cs 前进一格）
		// 检查是否跨过了时间间隔倍数（避免因帧率波动跳过精确的倍数帧）
		if pb.GameTickCS/config.FastTrackIntervalCS > pb.LastTrackUpdateCS/config.FastTrackIntervalCS {
			pb.RealProgress += stepSize
			pb.LastTrackUpdateCS = pb.GameTickCS
		}
	} else {
		// 落后 1-6 格：慢速追踪（每 20cs 前进一格）
		// 检查是否跨过了时间间隔倍数
		if pb.GameTickCS/config.SlowTrackIntervalCS > pb.LastTrackUpdateCS/config.SlowTrackIntervalCS {
			pb.RealProgress += stepSize
			pb.LastTrackUpdateCS = pb.GameTickCS
		}
	}

	// 确保现实进度不超过虚拟进度
	if pb.RealProgress > pb.VirtualProgress {
		pb.RealProgress = pb.VirtualProgress
	}
}

// OnWaveActivated 波次激活时的回调
//
// Story 11.5: 更新进度条的波次追踪状态
// - 更新当前波次号
// - 记录本波开始时间和初始倒计时
// - 如果是红字波，增加红字波计数并立即推进虚拟进度
// - 初始化本波血量追踪
//
// 参数:
//   - waveIndex: 激活的波次索引（0-based）
//   - initialDelay: 下一波的初始倒计时（秒）
func (s *LevelSystem) OnWaveActivated(waveIndex int, initialDelay float64) {
	log.Printf("[LevelSystem] OnWaveActivated called: waveIndex=%d, initialDelay=%.2f, progressBarEntityID=%d",
		waveIndex, initialDelay, s.progressBarEntityID)

	if s.progressBarEntityID == 0 {
		log.Printf("[LevelSystem] OnWaveActivated: progressBarEntityID is 0, skipping")
		return
	}

	progressBar, ok := ecs.GetComponent[*components.LevelProgressBarComponent](s.entityManager, s.progressBarEntityID)
	if !ok {
		log.Printf("[LevelSystem] OnWaveActivated: failed to get progress bar component")
		return
	}

	// 更新当前波次号（从 1 开始）
	oldWaveNum := progressBar.CurrentWaveNum
	progressBar.CurrentWaveNum = waveIndex + 1

	// 记录本波开始时间
	progressBar.WaveStartTime = s.gameState.LevelTime

	// 记录下一波的初始倒计时（用于时间进度计算）
	progressBar.WaveInitialDelay = initialDelay

	log.Printf("[LevelSystem] OnWaveActivated: updated CurrentWaveNum %d->%d, WaveStartTime=%.2f, WaveInitialDelay=%.2f",
		oldWaveNum, progressBar.CurrentWaveNum, progressBar.WaveStartTime, progressBar.WaveInitialDelay)

	// 检查是否是红字波
	if s.gameState.CurrentLevel != nil {
		for _, flagWaveIndex := range s.gameState.CurrentLevel.FlagWaves {
			if flagWaveIndex == waveIndex {
				// 是红字波，增加红字波计数
				progressBar.FlagWaveCount++
				log.Printf("[LevelSystem] Flag wave %d activated, flagWaveCount=%d", waveIndex+1, progressBar.FlagWaveCount)
				break
			}
		}
	}

	// 初始化本波血量追踪
	s.initializeProgressBarWaveHealth(progressBar, waveIndex)
}

// initializeProgressBarWaveHealth 初始化进度条的波次血量追踪
//
// Story 11.5 Task 4: 在波次激活时记录本波僵尸总血量
//
// 参数:
//   - pb: 进度条组件
//   - waveIndex: 波次索引
func (s *LevelSystem) initializeProgressBarWaveHealth(pb *components.LevelProgressBarComponent, waveIndex int) {
	if s.gameState.CurrentLevel == nil {
		return
	}

	levelConfig := s.gameState.CurrentLevel
	if waveIndex < 0 || waveIndex >= len(levelConfig.Waves) {
		return
	}

	// 获取本波僵尸配置
	waveConfig := levelConfig.Waves[waveIndex]
	zombieList := s.extractZombieSpawnInfo(&waveConfig)

	// 计算本波僵尸总血量
	totalHealth := 0.0
	for _, zombie := range zombieList {
		// 获取僵尸类型的血量
		zombieHealth := s.getZombieTypeHealth(zombie.Type)
		totalHealth += float64(zombie.Count) * zombieHealth
	}

	pb.WaveInitialHealth = totalHealth
	pb.WaveCurrentHealth = totalHealth

	// 设置所需削减血量（用于血量进度计算）
	// 通常等于总血量，但可以根据关卡配置调整
	pb.WaveRequiredDamage = totalHealth

	log.Printf("[LevelSystem] Wave %d health initialized: initial=%.0f, required=%.0f",
		waveIndex+1, pb.WaveInitialHealth, pb.WaveRequiredDamage)
}

// getZombieTypeHealth 获取僵尸类型的血量
//
// 参数:
//   - zombieType: 僵尸类型字符串
//
// 返回:
//   - float64: 僵尸总血量（本体 + 饰品）
func (s *LevelSystem) getZombieTypeHealth(zombieType string) float64 {
	// 如果有僵尸配置，从配置中获取
	if s.zombieStatsConfig != nil {
		stats, ok := s.zombieStatsConfig.GetZombieStats(zombieType)
		if ok && stats != nil {
			// 总血量 = 本体血量 + I类饰品血量 + II类饰品血量
			return float64(stats.BaseHealth + stats.Tier1AccessoryHealth + stats.Tier2AccessoryHealth)
		}
	}

	// 默认血量（基于类型）
	switch zombieType {
	case "basic":
		return 200
	case "conehead":
		return 560
	case "buckethead":
		return 1300
	case "polevaulter":
		return 335
	case "newspaperzombie":
		return 331
	case "screendoor":
		return 1400
	case "footballzombie":
		return 1670
	case "dancingzombie":
		return 335
	case "backupdancer":
		return 200
	case "snorkel":
		return 200
	case "zomboni":
		return 1350
	case "bobsled":
		return 200
	case "dolphinrider":
		return 200
	case "jackinthebox":
		return 500
	case "balloon":
		return 200
	case "digger":
		return 200
	case "pogo":
		return 200
	case "yeti":
		return 1350
	case "bungee":
		return 450
	case "ladder":
		return 500
	case "catapult":
		return 850
	case "gargantuar":
		return 3000
	case "imp":
		return 70
	case "drzomboss":
		return 40000
	default:
		return 200 // 默认普通僵尸血量
	}
}
