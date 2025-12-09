package entities

import (
	"testing"

	"github.com/decker502/pvz/internal/reanim"
	"github.com/decker502/pvz/pkg/components"
	"github.com/decker502/pvz/pkg/config"
	"github.com/decker502/pvz/pkg/ecs"
	"github.com/hajimehoshi/ebiten/v2"
)

// mockResourceLoaderForBowling 模拟资源加载器
type mockResourceLoaderForBowling struct {
	reanimXML  *reanim.ReanimXML
	partImages map[string]*ebiten.Image
}

func (m *mockResourceLoaderForBowling) LoadImage(path string) (*ebiten.Image, error) {
	return ebiten.NewImage(64, 64), nil
}

func (m *mockResourceLoaderForBowling) GetReanimXML(name string) *reanim.ReanimXML {
	return m.reanimXML
}

func (m *mockResourceLoaderForBowling) GetReanimPartImages(name string) map[string]*ebiten.Image {
	return m.partImages
}

func (m *mockResourceLoaderForBowling) LoadSoundEffect(path string) (interface{}, error) {
	return nil, nil
}

// createMockResourceLoaderForBowling 创建模拟资源加载器
func createMockResourceLoaderForBowling() *mockResourceLoaderForBowling {
	// 创建模拟的 anim_face 轨道数据（滚动动画）
	// 模拟 30 帧可见：17 帧摇摆 + 13 帧滚动
	// x 值范围：-8 到 96（直径约 104，周长约 327）
	minX := -8.0
	maxX := 96.0
	frames := make([]reanim.Frame, 56) // 56 帧总数

	// 帧 0-16：摇摆帧（可见）
	for i := 0; i <= 16; i++ {
		x := 7.0 // 摇摆帧 x 值稳定
		frames[i] = reanim.Frame{X: &x}
	}

	// 帧 17：隐藏帧
	hidden := -1
	frames[17] = reanim.Frame{FrameNum: &hidden}

	// 帧 18-42：继承隐藏
	for i := 18; i <= 42; i++ {
		frames[i] = reanim.Frame{}
	}

	// 帧 43-55：滚动帧（可见）
	visible := 0
	frames[43] = reanim.Frame{FrameNum: &visible, X: &minX}
	for i := 44; i <= 54; i++ {
		x := minX + (maxX-minX)*float64(i-43)/12.0
		frames[i] = reanim.Frame{X: &x}
	}
	frames[55] = reanim.Frame{X: &maxX}

	return &mockResourceLoaderForBowling{
		reanimXML: &reanim.ReanimXML{
			FPS: 12,
			Tracks: []reanim.Track{
				{Name: "anim_face", Frames: frames},
			},
		},
		partImages: map[string]*ebiten.Image{
			"test_part": ebiten.NewImage(64, 64),
		},
	}
}

// TestNewBowlingNutEntity_Normal 测试普通坚果创建
func TestNewBowlingNutEntity_Normal(t *testing.T) {
	em := ecs.NewEntityManager()
	rm := createMockResourceLoaderForBowling()

	entityID, err := NewBowlingNutEntity(em, rm, 2, 1, false)

	if err != nil {
		t.Fatalf("NewBowlingNutEntity failed: %v", err)
	}
	if entityID == 0 {
		t.Error("EntityID should not be 0")
	}

	// 验证 PositionComponent 存在
	posComp, ok := ecs.GetComponent[*components.PositionComponent](em, entityID)
	if !ok {
		t.Fatal("PositionComponent not found")
	}

	// 验证位置计算
	expectedX := config.GridWorldStartX + float64(1)*config.CellWidth + config.CellWidth/2
	expectedY := config.GridWorldStartY + float64(2)*config.CellHeight + config.CellHeight/2
	if posComp.X < expectedX-0.1 || posComp.X > expectedX+0.1 {
		t.Errorf("Position X = %f, want %f", posComp.X, expectedX)
	}
	if posComp.Y < expectedY-0.1 || posComp.Y > expectedY+0.1 {
		t.Errorf("Position Y = %f, want %f", posComp.Y, expectedY)
	}

	// 验证 BowlingNutComponent 存在
	nutComp, ok := ecs.GetComponent[*components.BowlingNutComponent](em, entityID)
	if !ok {
		t.Fatal("BowlingNutComponent not found")
	}

	if nutComp.IsExplosive {
		t.Error("IsExplosive should be false for normal nut")
	}
	if !nutComp.IsRolling {
		t.Error("IsRolling should be true")
	}
	if nutComp.Row != 2 {
		t.Errorf("Row = %d, want 2", nutComp.Row)
	}
	// 验证速度是动态计算的，应该在合理范围内（150-400 像素/秒）
	if nutComp.VelocityX < 150 || nutComp.VelocityX > 400 {
		t.Errorf("VelocityX = %f, should be in range [150, 400]", nutComp.VelocityX)
	}
}

// TestNewBowlingNutEntity_Explosive 测试爆炸坚果创建
func TestNewBowlingNutEntity_Explosive(t *testing.T) {
	em := ecs.NewEntityManager()
	rm := createMockResourceLoaderForBowling()

	entityID, err := NewBowlingNutEntity(em, rm, 3, 2, true)

	if err != nil {
		t.Fatalf("NewBowlingNutEntity failed: %v", err)
	}

	// 验证 BowlingNutComponent
	nutComp, ok := ecs.GetComponent[*components.BowlingNutComponent](em, entityID)
	if !ok {
		t.Fatal("BowlingNutComponent not found")
	}

	if !nutComp.IsExplosive {
		t.Error("IsExplosive should be true for explosive nut")
	}
}

// TestNewBowlingNutEntity_ComponentsExist 测试所有组件存在
func TestNewBowlingNutEntity_ComponentsExist(t *testing.T) {
	em := ecs.NewEntityManager()
	rm := createMockResourceLoaderForBowling()

	entityID, err := NewBowlingNutEntity(em, rm, 0, 0, false)
	if err != nil {
		t.Fatalf("NewBowlingNutEntity failed: %v", err)
	}

	// 验证 PositionComponent
	if _, ok := ecs.GetComponent[*components.PositionComponent](em, entityID); !ok {
		t.Error("PositionComponent should exist")
	}

	// 验证 BowlingNutComponent
	if _, ok := ecs.GetComponent[*components.BowlingNutComponent](em, entityID); !ok {
		t.Error("BowlingNutComponent should exist")
	}

	// 验证 ReanimComponent
	if _, ok := ecs.GetComponent[*components.ReanimComponent](em, entityID); !ok {
		t.Error("ReanimComponent should exist")
	}

	// 验证 AnimationCommandComponent
	if _, ok := ecs.GetComponent[*components.AnimationCommandComponent](em, entityID); !ok {
		t.Error("AnimationCommandComponent should exist")
	}

	// 验证 CollisionComponent
	if _, ok := ecs.GetComponent[*components.CollisionComponent](em, entityID); !ok {
		t.Error("CollisionComponent should exist")
	}
}

// TestNewBowlingNutEntity_PositionCalculation 测试位置计算正确性
func TestNewBowlingNutEntity_PositionCalculation(t *testing.T) {
	em := ecs.NewEntityManager()
	rm := createMockResourceLoaderForBowling()

	testCases := []struct {
		row       int
		col       int
		expectedX float64
		expectedY float64
	}{
		{0, 0, config.GridWorldStartX + config.CellWidth/2, config.GridWorldStartY + config.CellHeight/2},
		{1, 1, config.GridWorldStartX + config.CellWidth + config.CellWidth/2, config.GridWorldStartY + config.CellHeight + config.CellHeight/2},
		{4, 8, config.GridWorldStartX + 8*config.CellWidth + config.CellWidth/2, config.GridWorldStartY + 4*config.CellHeight + config.CellHeight/2},
	}

	for _, tc := range testCases {
		entityID, err := NewBowlingNutEntity(em, rm, tc.row, tc.col, false)
		if err != nil {
			t.Fatalf("NewBowlingNutEntity failed for row=%d, col=%d: %v", tc.row, tc.col, err)
		}

		posComp, _ := ecs.GetComponent[*components.PositionComponent](em, entityID)
		if posComp.X < tc.expectedX-0.1 || posComp.X > tc.expectedX+0.1 {
			t.Errorf("Row=%d, Col=%d: X = %f, want %f", tc.row, tc.col, posComp.X, tc.expectedX)
		}
		if posComp.Y < tc.expectedY-0.1 || posComp.Y > tc.expectedY+0.1 {
			t.Errorf("Row=%d, Col=%d: Y = %f, want %f", tc.row, tc.col, posComp.Y, tc.expectedY)
		}
	}
}

// TestNewBowlingNutEntity_CollisionComponent 测试碰撞组件配置
func TestNewBowlingNutEntity_CollisionComponent(t *testing.T) {
	em := ecs.NewEntityManager()
	rm := createMockResourceLoaderForBowling()

	entityID, _ := NewBowlingNutEntity(em, rm, 0, 0, false)

	collComp, ok := ecs.GetComponent[*components.CollisionComponent](em, entityID)
	if !ok {
		t.Fatal("CollisionComponent not found")
	}

	if collComp.Width != config.BowlingNutCollisionWidth {
		t.Errorf("CollisionWidth = %f, want %f", collComp.Width, config.BowlingNutCollisionWidth)
	}
	if collComp.Height != config.BowlingNutCollisionHeight {
		t.Errorf("CollisionHeight = %f, want %f", collComp.Height, config.BowlingNutCollisionHeight)
	}
}

// TestNewBowlingNutEntity_ReanimComponent 测试动画组件配置
func TestNewBowlingNutEntity_ReanimComponent(t *testing.T) {
	em := ecs.NewEntityManager()
	rm := createMockResourceLoaderForBowling()

	entityID, _ := NewBowlingNutEntity(em, rm, 0, 0, false)

	reanimComp, ok := ecs.GetComponent[*components.ReanimComponent](em, entityID)
	if !ok {
		t.Fatal("ReanimComponent not found")
	}

	if reanimComp.ReanimName != "Wallnut" {
		t.Errorf("ReanimName = %s, want 'Wallnut'", reanimComp.ReanimName)
	}
}

// TestNewBowlingNutEntity_AnimationCommand 测试动画命令配置
func TestNewBowlingNutEntity_AnimationCommand(t *testing.T) {
	em := ecs.NewEntityManager()
	rm := createMockResourceLoaderForBowling()

	entityID, _ := NewBowlingNutEntity(em, rm, 0, 0, false)

	animCmd, ok := ecs.GetComponent[*components.AnimationCommandComponent](em, entityID)
	if !ok {
		t.Fatal("AnimationCommandComponent not found")
	}

	if animCmd.AnimationName != "anim_face" {
		t.Errorf("AnimationName = %s, want 'anim_face'", animCmd.AnimationName)
	}
	if animCmd.Processed {
		t.Error("Processed should be false initially")
	}
	// 验证起始帧使用配置值
	if animCmd.StartFrame != config.BowlingNutRollingStartFrame {
		t.Errorf("StartFrame = %d, want %d (from config)", animCmd.StartFrame, config.BowlingNutRollingStartFrame)
	}
}

// TestNewBowlingNutEntity_FailsWithNilReanimData 测试缺少资源时失败
func TestNewBowlingNutEntity_FailsWithNilReanimData(t *testing.T) {
	em := ecs.NewEntityManager()
	rm := &mockResourceLoaderForBowling{
		reanimXML:  nil, // 没有 Reanim 数据
		partImages: nil,
	}

	_, err := NewBowlingNutEntity(em, rm, 0, 0, false)

	if err == nil {
		t.Error("Expected error when Reanim resources are nil")
	}
}

// TestNewBowlingNutEntity_InitialBounceCount 测试初始弹射次数
func TestNewBowlingNutEntity_InitialBounceCount(t *testing.T) {
	em := ecs.NewEntityManager()
	rm := createMockResourceLoaderForBowling()

	entityID, _ := NewBowlingNutEntity(em, rm, 0, 0, false)

	nutComp, _ := ecs.GetComponent[*components.BowlingNutComponent](em, entityID)
	if nutComp.BounceCount != 0 {
		t.Errorf("Initial BounceCount = %d, want 0", nutComp.BounceCount)
	}
}

// TestNewBowlingNutEntity_InitialSoundState 测试初始音效状态
func TestNewBowlingNutEntity_InitialSoundState(t *testing.T) {
	em := ecs.NewEntityManager()
	rm := createMockResourceLoaderForBowling()

	entityID, _ := NewBowlingNutEntity(em, rm, 0, 0, false)

	nutComp, _ := ecs.GetComponent[*components.BowlingNutComponent](em, entityID)
	if nutComp.SoundPlaying {
		t.Error("Initial SoundPlaying should be false")
	}
}

