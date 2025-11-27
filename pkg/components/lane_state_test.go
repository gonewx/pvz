package components

import "testing"

func TestLaneStateComponent(t *testing.T) {
	// 测试组件初始化
	laneState := LaneStateComponent{
		LaneIndex:        3,
		Weight:           1.0,
		LastPicked:       0,
		SecondLastPicked: 0,
	}

	if laneState.LaneIndex != 3 {
		t.Errorf("Expected LaneIndex=3, got %d", laneState.LaneIndex)
	}
	if laneState.Weight != 1.0 {
		t.Errorf("Expected Weight=1.0, got %f", laneState.Weight)
	}
	if laneState.LastPicked != 0 {
		t.Errorf("Expected LastPicked=0, got %d", laneState.LastPicked)
	}
	if laneState.SecondLastPicked != 0 {
		t.Errorf("Expected SecondLastPicked=0, got %d", laneState.SecondLastPicked)
	}
}

func TestLaneStateComponentFieldModification(t *testing.T) {
	// 测试字段可以正常修改
	laneState := LaneStateComponent{
		LaneIndex:        1,
		Weight:           1.0,
		LastPicked:       0,
		SecondLastPicked: 0,
	}

	// 模拟选中后更新
	laneState.SecondLastPicked = laneState.LastPicked
	laneState.LastPicked = 0

	if laneState.LastPicked != 0 {
		t.Errorf("Expected LastPicked=0 after update, got %d", laneState.LastPicked)
	}
	if laneState.SecondLastPicked != 0 {
		t.Errorf("Expected SecondLastPicked=0 after update, got %d", laneState.SecondLastPicked)
	}

	// 模拟计数器递增
	laneState.LastPicked++
	laneState.SecondLastPicked++

	if laneState.LastPicked != 1 {
		t.Errorf("Expected LastPicked=1 after increment, got %d", laneState.LastPicked)
	}
	if laneState.SecondLastPicked != 1 {
		t.Errorf("Expected SecondLastPicked=1 after increment, got %d", laneState.SecondLastPicked)
	}
}

func TestLaneStateComponentZeroWeight(t *testing.T) {
	// 测试零权重场景
	laneState := LaneStateComponent{
		LaneIndex:        2,
		Weight:           0.0,
		LastPicked:       5,
		SecondLastPicked: 10,
	}

	if laneState.Weight != 0.0 {
		t.Errorf("Expected Weight=0.0, got %f", laneState.Weight)
	}
}
