package systems

import (
	"log"
	"math/rand"

	"github.com/decker502/pvz/pkg/components"
	"github.com/decker502/pvz/pkg/config"
	"github.com/decker502/pvz/pkg/ecs"
)

// LaneAllocator 行分配器系统
//
// 实现平滑权重行分配算法，确保僵尸在多个行之间的分布自然且避免连续重复
type LaneAllocator struct {
	entityManager *ecs.EntityManager
	laneEntities  []ecs.EntityID // 存储所有行实体的 ID（长度为 RowMax）
}

// NewLaneAllocator 创建新的行分配器系统
func NewLaneAllocator(em *ecs.EntityManager) *LaneAllocator {
	return &LaneAllocator{
		entityManager: em,
		laneEntities:  make([]ecs.EntityID, 0),
	}
}

// InitializeLanes 初始化所有行的状态组件
//
// 参数:
//   - rowMax: 最大行数（5 或 6）
//   - initialWeight: 初始权重（冒险模式为 1）
func (la *LaneAllocator) InitializeLanes(rowMax int, initialWeight float64) {
	la.laneEntities = make([]ecs.EntityID, rowMax)

	for i := 0; i < rowMax; i++ {
		entity := la.entityManager.CreateEntity()
		laneState := &components.LaneStateComponent{
			LaneIndex:        i + 1, // 行号从 1 开始
			Weight:           initialWeight,
			LastPicked:       0,
			SecondLastPicked: 0,
		}
		ecs.AddComponent(la.entityManager, entity, laneState)
		la.laneEntities[i] = entity
	}

	log.Printf("[LaneAllocator] Initialized %d lanes with initial weight %.2f", rowMax, initialWeight)
}

// SelectLane 为僵尸选择一个行
//
// 参数:
//   - zombieType: 僵尸类型（用于合法行判定）
//   - sceneType: 场景类型（用于合法行判定）
//   - spawnRules: 僵尸生成规则配置
//   - enabledLanes: 关卡启用的行列表（草地行）
//   - laneRestriction: 波次级行限制（可选，为空则不限制）
//
// 返回:
//   - 选中的行号（1-6）
func (la *LaneAllocator) SelectLane(
	zombieType string,
	sceneType string,
	spawnRules *config.SpawnRulesConfig,
	enabledLanes []int,
	laneRestriction []int,
) int {
	// 收集所有行的状态
	laneStates := make([]*components.LaneStateComponent, 0, len(la.laneEntities))
	for _, entity := range la.laneEntities {
		laneState, ok := ecs.GetComponent[*components.LaneStateComponent](la.entityManager, entity)
		if ok {
			laneStates = append(laneStates, laneState)
		}
	}

	if len(laneStates) == 0 {
		log.Printf("[LaneAllocator] WARNING: No lane states found, using default lane 6")
		return 6
	}

	// 过滤出合法行（在计算权重之前）
	legalLaneIndices := FilterLegalLanes(laneStates, zombieType, sceneType, spawnRules, enabledLanes, laneRestriction)

	// 如果没有合法行，返回默认第六行
	if len(legalLaneIndices) == 0 {
		log.Printf("[LaneAllocator] WARNING: No legal lanes for zombie type %s in scene %s, using default lane 6", zombieType, sceneType)
		return 6
	}

	// 仅对合法行计算平滑权重
	legalLaneStates := make([]*components.LaneStateComponent, len(legalLaneIndices))
	for i, idx := range legalLaneIndices {
		legalLaneStates[i] = laneStates[idx]
	}

	// 提取权重
	laneWeights := make([]float64, len(legalLaneStates))
	for i, state := range legalLaneStates {
		laneWeights[i] = state.Weight
	}

	// 计算权重占比
	weightP := CalculateWeightP(laneWeights)

	// 计算平滑权重
	smoothWeights := make([]float64, len(legalLaneStates))
	for i, state := range legalLaneStates {
		pLast := CalculatePLast(state.LastPicked, weightP[i])
		pSecondLast := CalculatePSecondLast(state.SecondLastPicked, weightP[i])
		smoothWeights[i] = CalculateSmoothWeight(weightP[i], pLast, pSecondLast)
	}

	// 累积和方式选择行
	totalWeight := 0.0
	for _, sw := range smoothWeights {
		totalWeight += sw
	}

	if totalWeight <= 0 {
		log.Printf("[LaneAllocator] WARNING: All smooth weights are zero, using default lane 6")
		return 6
	}

	randNum := rand.Float64() * totalWeight
	cumulativeWeight := 0.0
	for i, sw := range smoothWeights {
		cumulativeWeight += sw
		if cumulativeWeight >= randNum {
			return legalLaneStates[i].LaneIndex
		}
	}

	// 理论上不应该到达这里，但作为保险返回最后一行
	return legalLaneStates[len(legalLaneStates)-1].LaneIndex
}

// UpdateLaneCounters 更新选中行的计数器
//
// 参数:
//   - selectedLane: 选中的行号（1-6）
//
// 算法说明（原版 PvZ 插入事件）：
//  1. 如果 Weight[i] > 0，则 ∀ i，LastPicked[i] 和 SecondLastPicked[i] 均 +1
//  2. 将 LastPicked[j] 的值赋给 SecondLastPicked[j]（j 为选中行）
//  3. 将 LastPicked[j] 设为 0
func (la *LaneAllocator) UpdateLaneCounters(selectedLane int) {
	// 第一遍：递增所有权重 > 0 的行的计数器
	for _, entity := range la.laneEntities {
		laneState, ok := ecs.GetComponent[*components.LaneStateComponent](la.entityManager, entity)
		if !ok {
			continue
		}

		if laneState.Weight > 0 {
			laneState.LastPicked++
			laneState.SecondLastPicked++
		}
	}

	// 第二遍：重置选中行的计数器
	for _, entity := range la.laneEntities {
		laneState, ok := ecs.GetComponent[*components.LaneStateComponent](la.entityManager, entity)
		if !ok {
			continue
		}

		if laneState.LaneIndex == selectedLane {
			// LastPicked 已经在第一遍中递增了，所以 LastPicked-1 是选中前的值
			laneState.SecondLastPicked = laneState.LastPicked - 1
			laneState.LastPicked = 0
			break
		}
	}
}

// CalculateWeightP 计算权重占比
//
// 参数:
//   - laneWeights: 所有行的权重列表
//
// 返回:
//   - 权重占比列表
func CalculateWeightP(laneWeights []float64) []float64 {
	sum := 0.0
	for _, w := range laneWeights {
		sum += w
	}

	weightP := make([]float64, len(laneWeights))
	for i, w := range laneWeights {
		if sum > 0 {
			weightP[i] = w / sum
		} else {
			weightP[i] = 0
		}
	}
	return weightP
}

// CalculatePLast 计算影响因子 PLast
//
// 公式: PLast = (6 × LastPicked × WeightP + 6 × WeightP - 3) / 4
//
// 参数:
//   - lastPicked: 距离上次被选取的计数器
//   - weightP: 权重占比
//
// 返回:
//   - PLast 值
func CalculatePLast(lastPicked int, weightP float64) float64 {
	return (6.0*float64(lastPicked)*weightP + 6.0*weightP - 3.0) / 4.0
}

// CalculatePSecondLast 计算影响因子 PSecondLast
//
// 公式: PSecondLast = (SecondLastPicked × WeightP + WeightP - 1) / 4
//
// 参数:
//   - secondLastPicked: 距离上上次被选取的计数器
//   - weightP: 权重占比
//
// 返回:
//   - PSecondLast 值
func CalculatePSecondLast(secondLastPicked int, weightP float64) float64 {
	return (float64(secondLastPicked)*weightP + weightP - 1.0) / 4.0
}

// CalculateSmoothWeight 计算平滑权重
//
// 公式: SmoothWeight = WeightP × clamp(PLast + PSecondLast, 0.01, 100)
//
// 参数:
//   - weightP: 权重占比
//   - pLast: PLast 影响因子
//   - pSecondLast: PSecondLast 影响因子
//
// 返回:
//   - 平滑权重
func CalculateSmoothWeight(weightP float64, pLast float64, pSecondLast float64) float64 {
	if weightP < 1e-6 {
		return 0
	}

	// clamp(PLast + PSecondLast, 0.01, 100)
	sum := pLast + pSecondLast
	if sum < 0.01 {
		sum = 0.01
	} else if sum > 100.0 {
		sum = 100.0
	}

	return weightP * sum
}

// FilterLegalLanes 根据僵尸类型过滤合法行
//
// 参数:
//   - laneStates: 所有行的状态列表
//   - zombieType: 僵尸类型（用于特殊限制判定）
//   - sceneType: 场景类型（用于场景限制判定）
//   - spawnRules: 僵尸生成规则配置
//   - enabledLanes: 关卡启用的行列表（草地行）
//   - laneRestriction: 波次级行限制（可选，为空则不限制）
//
// 返回:
//   - 合法行的索引列表
func FilterLegalLanes(
	laneStates []*components.LaneStateComponent,
	zombieType string,
	sceneType string,
	spawnRules *config.SpawnRulesConfig,
	enabledLanes []int,
	laneRestriction []int,
) []int {
	legalLanes := make([]int, 0, len(laneStates))

	for i, state := range laneStates {
		// 1. 基本不合法条件
		if state.LaneIndex < 1 || state.LaneIndex > 6 {
			continue
		}
		if state.Weight <= 0 {
			continue
		}
		// 非泳池/浓雾关卡的第 6 行不合法
		if state.LaneIndex == 6 && sceneType != "pool" && sceneType != "fog" {
			continue
		}

		// 2. 波次级行限制（Story 17.2: laneRestriction）
		if len(laneRestriction) > 0 {
			found := false
			for _, lane := range laneRestriction {
				if lane == state.LaneIndex {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}

		// 3. 水路限制（如果配置可用）
		if spawnRules != nil {
			isWater := IsWaterLane(state.LaneIndex, sceneType, spawnRules.SceneTypeRestrictions.WaterLaneConfig)
			isWaterZombie := IsWaterZombie(zombieType, spawnRules.SceneTypeRestrictions.WaterZombies)

			// 水路行不允许非水路僵尸
			if isWater && !isWaterZombie {
				continue
			}
			// 非水路行不允许水路僵尸
			if !isWater && isWaterZombie {
				continue
			}

			// 4. 舞王限制
			if zombieType == "dancing" {
				// 屋顶场景禁止舞王
				if sceneType == "roof" {
					continue
				}
				// 非后院场景：检查上下相邻行
				if sceneType != "pool" && sceneType != "fog" {
					if len(enabledLanes) > 0 && !IsAdjacentLaneValid(state.LaneIndex, enabledLanes) {
						continue
					}
				}
			}
		}

		legalLanes = append(legalLanes, i)
	}

	// 无可选行时返回空列表（调用方应返回默认第六行）
	return legalLanes
}

// LogLaneWeights 输出所有行的权重分布（调试用）
//
// 参数:
//   - verbose: 是否启用详细日志模式
func (la *LaneAllocator) LogLaneWeights(verbose bool) {
	if !verbose {
		return
	}

	log.Printf("[LaneAllocator] === Lane Weights Distribution ===")
	for i, entity := range la.laneEntities {
		laneState, ok := ecs.GetComponent[*components.LaneStateComponent](la.entityManager, entity)
		if ok {
			log.Printf("[LaneAllocator] Lane %d: Weight=%.2f, LastPicked=%d, SecondLastPicked=%d",
				laneState.LaneIndex, laneState.Weight, laneState.LastPicked, laneState.SecondLastPicked)
		} else {
			log.Printf("[LaneAllocator] Lane %d: Component not found", i+1)
		}
	}
}

// LogLaneSelectionProbability 输出行选择概率（调试用）
//
// 参数:
//   - verbose: 是否启用详细日志模式
func (la *LaneAllocator) LogLaneSelectionProbability(verbose bool) {
	if !verbose {
		return
	}

	// 收集所有行的状态
	laneStates := make([]*components.LaneStateComponent, 0, len(la.laneEntities))
	for _, entity := range la.laneEntities {
		laneState, ok := ecs.GetComponent[*components.LaneStateComponent](la.entityManager, entity)
		if ok {
			laneStates = append(laneStates, laneState)
		}
	}

	if len(laneStates) == 0 {
		log.Printf("[LaneAllocator] WARNING: No lane states found")
		return
	}

	// 提取权重
	laneWeights := make([]float64, len(laneStates))
	for i, state := range laneStates {
		laneWeights[i] = state.Weight
	}

	// 计算权重占比
	weightP := CalculateWeightP(laneWeights)

	// 计算平滑权重
	smoothWeights := make([]float64, len(laneStates))
	totalSmoothWeight := 0.0
	for i, state := range laneStates {
		pLast := CalculatePLast(state.LastPicked, weightP[i])
		pSecondLast := CalculatePSecondLast(state.SecondLastPicked, weightP[i])
		smoothWeights[i] = CalculateSmoothWeight(weightP[i], pLast, pSecondLast)
		totalSmoothWeight += smoothWeights[i]
	}

	log.Printf("[LaneAllocator] === Lane Selection Probability ===")
	for i, state := range laneStates {
		probability := 0.0
		if totalSmoothWeight > 0 {
			probability = (smoothWeights[i] / totalSmoothWeight) * 100
		}
		log.Printf("[LaneAllocator] Lane %d: Weight=%.2f, LastPicked=%d, SmoothWeight=%.4f, Probability=%.2f%%",
			state.LaneIndex, state.Weight, state.LastPicked, smoothWeights[i], probability)
	}
}

// IsWaterLane 判断指定行是否为水路
//
// 参数:
//   - laneIndex: 行号（1-6）
//   - sceneType: 场景类型（如 "pool", "fog"）
//   - waterLaneConfig: 水路配置映射（场景 -> 水路行号列表）
//
// 返回:
//   - true 表示该行为水路
func IsWaterLane(laneIndex int, sceneType string, waterLaneConfig map[string][]int) bool {
	waterLanes, exists := waterLaneConfig[sceneType]
	if !exists {
		return false
	}
	for _, lane := range waterLanes {
		if lane == laneIndex {
			return true
		}
	}
	return false
}

// IsWaterZombie 判断僵尸类型是否为水路专属
//
// 参数:
//   - zombieType: 僵尸类型
//   - waterZombies: 水路专属僵尸列表
//
// 返回:
//   - true 表示该僵尸为水路专属
func IsWaterZombie(zombieType string, waterZombies []string) bool {
	for _, wz := range waterZombies {
		if wz == zombieType {
			return true
		}
	}
	return false
}

// IsAdjacentLaneValid 检查舞王僵尸的上下相邻行是否有效
//
// 参数:
//   - laneIndex: 当前行号（1-6）
//   - enabledLanes: 启用的行列表（草地行）
//
// 返回:
//   - true 表示上下至少有一行有效
func IsAdjacentLaneValid(laneIndex int, enabledLanes []int) bool {
	hasUpper := false
	hasLower := false

	for _, lane := range enabledLanes {
		if lane == laneIndex-1 {
			hasUpper = true
		}
		if lane == laneIndex+1 {
			hasLower = true
		}
	}

	// 至少上下有一行有效
	return hasUpper || hasLower
}
