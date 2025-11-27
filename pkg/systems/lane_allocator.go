package systems

import (
	"log"
	"math/rand"

	"github.com/decker502/pvz/pkg/components"
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
//
// 返回:
//   - 选中的行号（1-6）
func (la *LaneAllocator) SelectLane(zombieType string, sceneType string) int {
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

	// 提取权重
	laneWeights := make([]float64, len(laneStates))
	for i, state := range laneStates {
		laneWeights[i] = state.Weight
	}

	// 计算权重占比
	weightP := CalculateWeightP(laneWeights)

	// 计算平滑权重
	smoothWeights := make([]float64, len(laneStates))
	for i, state := range laneStates {
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
			return laneStates[i].LaneIndex
		}
	}

	// 理论上不应该到达这里，但作为保险返回最后一行
	return laneStates[len(laneStates)-1].LaneIndex
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
//
// 返回:
//   - 合法行的索引列表
//
// 注意: 本故事仅实现基本的行号验证和全零权重处理
// 详细的合法行判定（水路、舞王等）将在 Story 17.5 中完成
func FilterLegalLanes(laneStates []*components.LaneStateComponent, zombieType string, sceneType string) []int {
	legalLanes := make([]int, 0, len(laneStates))

	for i, state := range laneStates {
		// 基本不合法条件：行号不在 1~6 之间
		if state.LaneIndex < 1 || state.LaneIndex > 6 {
			continue
		}

		// 权重为 0 的行不合法
		if state.Weight <= 0 {
			continue
		}

		// TODO: Story 17.5 将实现详细的合法行判定：
		// - 水路僵尸（潜水、海豚）只能在水路行（泳池第 3、4 行）
		// - 非水路僵尸不能在水路行
		// - 舞王禁止在屋顶场景出现
		// - 无草皮之地关卡外的裸地行限制

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
