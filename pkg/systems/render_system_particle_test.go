package systems

import (
	"math"
	"testing"

	"github.com/gonewx/pvz/pkg/components"
	"github.com/gonewx/pvz/pkg/ecs"
	"github.com/hajimehoshi/ebiten/v2"
)

// TestBuildParticleVertices_BasicPositionMapping 测试基本位置映射（世界坐标 → 屏幕坐标）
func TestBuildParticleVertices_BasicPositionMapping(t *testing.T) {
	em := ecs.NewEntityManager()
	rs := NewRenderSystem(em)

	// 创建测试图片（32x32）
	testImage := ebiten.NewImage(32, 32)

	// 创建测试粒子组件
	particle := &components.ParticleComponent{
		Image:      testImage,
		Rotation:   0,   // 无旋转
		Scale:      1.0, // 无缩放
		Red:        1.0, // 白色
		Green:      1.0,
		Blue:       1.0,
		Alpha:      1.0, // 完全不透明
		Brightness: 1.0, // 无亮度调整
		Additive:   false,
	}

	// 创建测试位置组件（世界坐标 400, 300）
	pos := &components.PositionComponent{
		X: 400,
		Y: 300,
	}

	// 摄像机位置
	cameraX := 100.0

	// 生成顶点
	vertices := rs.buildParticleVertices(particle, pos, cameraX)

	// 验证返回 4 个顶点
	if len(vertices) != 4 {
		t.Fatalf("Expected 4 vertices, got %d", len(vertices))
	}

	// 验证屏幕坐标 = 世界坐标 - cameraX
	// 粒子中心在屏幕坐标 (300, 300)
	expectedScreenX := 400.0 - 100.0 // = 300
	expectedScreenY := 300.0

	// 图片大小 32x32，中心对齐，所以顶点范围应该是:
	// 左上: (300 - 16, 300 - 16) = (284, 284)
	// 右上: (300 + 16, 300 - 16) = (316, 284)
	// 左下: (300 - 16, 300 + 16) = (284, 316)
	// 右下: (300 + 16, 300 + 16) = (316, 316)

	tolerance := 0.01

	// 验证左上角
	if math.Abs(float64(vertices[0].DstX)-(expectedScreenX-16)) > tolerance {
		t.Errorf("Left-top X: expected %.2f, got %.2f", expectedScreenX-16, vertices[0].DstX)
	}
	if math.Abs(float64(vertices[0].DstY)-(expectedScreenY-16)) > tolerance {
		t.Errorf("Left-top Y: expected %.2f, got %.2f", expectedScreenY-16, vertices[0].DstY)
	}

	// 验证右上角
	if math.Abs(float64(vertices[1].DstX)-(expectedScreenX+16)) > tolerance {
		t.Errorf("Right-top X: expected %.2f, got %.2f", expectedScreenX+16, vertices[1].DstX)
	}

	// 验证左下角
	if math.Abs(float64(vertices[2].DstY)-(expectedScreenY+16)) > tolerance {
		t.Errorf("Left-bottom Y: expected %.2f, got %.2f", expectedScreenY+16, vertices[2].DstY)
	}

	// 验证右下角
	if math.Abs(float64(vertices[3].DstX)-(expectedScreenX+16)) > tolerance {
		t.Errorf("Right-bottom X: expected %.2f, got %.2f", expectedScreenX+16, vertices[3].DstX)
	}
	if math.Abs(float64(vertices[3].DstY)-(expectedScreenY+16)) > tolerance {
		t.Errorf("Right-bottom Y: expected %.2f, got %.2f", expectedScreenY+16, vertices[3].DstY)
	}
}

// TestBuildParticleVertices_RotationTransform 测试旋转变换正确性
func TestBuildParticleVertices_RotationTransform(t *testing.T) {
	em := ecs.NewEntityManager()
	rs := NewRenderSystem(em)

	// 创建测试图片（32x32）
	testImage := ebiten.NewImage(32, 32)

	testCases := []struct {
		name     string
		rotation float64 // 旋转角度（度）
	}{
		{"No Rotation", 0},
		{"90 Degrees", 90},
		{"180 Degrees", 180},
		{"270 Degrees", 270},
		{"45 Degrees", 45},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			particle := &components.ParticleComponent{
				Image:      testImage,
				Rotation:   tc.rotation,
				Scale:      1.0,
				Red:        1.0,
				Green:      1.0,
				Blue:       1.0,
				Alpha:      1.0,
				Brightness: 1.0,
				Additive:   false,
			}

			pos := &components.PositionComponent{X: 100, Y: 100}
			cameraX := 0.0

			vertices := rs.buildParticleVertices(particle, pos, cameraX)

			// 验证返回 4 个顶点
			if len(vertices) != 4 {
				t.Fatalf("Expected 4 vertices, got %d", len(vertices))
			}

			// 验证旋转后顶点不全相同（除非旋转0度）
			if tc.rotation != 0 {
				// 检查顶点确实发生了变换
				if vertices[0].DstX == vertices[1].DstX && vertices[0].DstY == vertices[1].DstY {
					t.Error("Rotation did not transform vertices")
				}
			}

			// 验证旋转中心在粒子位置（顶点应该围绕中心旋转）
			// 计算顶点的中心点
			centerX := (vertices[0].DstX + vertices[1].DstX + vertices[2].DstX + vertices[3].DstX) / 4
			centerY := (vertices[0].DstY + vertices[1].DstY + vertices[2].DstY + vertices[3].DstY) / 4

			expectedCenterX := float32(pos.X - cameraX)
			expectedCenterY := float32(pos.Y)

			tolerance := float32(0.1)
			if math.Abs(float64(centerX-expectedCenterX)) > float64(tolerance) {
				t.Errorf("Rotation center X: expected %.2f, got %.2f", expectedCenterX, centerX)
			}
			if math.Abs(float64(centerY-expectedCenterY)) > float64(tolerance) {
				t.Errorf("Rotation center Y: expected %.2f, got %.2f", expectedCenterY, centerY)
			}
		})
	}
}

// TestBuildParticleVertices_ScaleTransform 测试缩放变换正确性
func TestBuildParticleVertices_ScaleTransform(t *testing.T) {
	em := ecs.NewEntityManager()
	rs := NewRenderSystem(em)

	// 创建测试图片（32x32）
	testImage := ebiten.NewImage(32, 32)

	testCases := []struct {
		name          string
		scale         float64
		expectedWidth float64 // 预期宽度（像素）
	}{
		{"Half Scale", 0.5, 16},
		{"Normal Scale", 1.0, 32},
		{"Double Scale", 2.0, 64},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			particle := &components.ParticleComponent{
				Image:      testImage,
				Rotation:   0,
				Scale:      tc.scale,
				Red:        1.0,
				Green:      1.0,
				Blue:       1.0,
				Alpha:      1.0,
				Brightness: 1.0,
				Additive:   false,
			}

			pos := &components.PositionComponent{X: 100, Y: 100}
			cameraX := 0.0

			vertices := rs.buildParticleVertices(particle, pos, cameraX)

			// 验证返回 4 个顶点
			if len(vertices) != 4 {
				t.Fatalf("Expected 4 vertices, got %d", len(vertices))
			}

			// 计算顶点宽度（右边 - 左边）
			width := vertices[1].DstX - vertices[0].DstX

			tolerance := 0.1
			if math.Abs(float64(width)-tc.expectedWidth) > tolerance {
				t.Errorf("Scale %f: expected width %.2f, got %.2f", tc.scale, tc.expectedWidth, width)
			}
		})
	}
}

// TestBuildParticleVertices_ColorCalculation 测试颜色计算（RGB、Alpha、Brightness）
func TestBuildParticleVertices_ColorCalculation(t *testing.T) {
	em := ecs.NewEntityManager()
	rs := NewRenderSystem(em)

	// 创建测试图片
	testImage := ebiten.NewImage(32, 32)

	testCases := []struct {
		name       string
		red        float64
		green      float64
		blue       float64
		alpha      float64
		brightness float64
		expectedR  float32
		expectedG  float32
		expectedB  float32
		expectedA  float32
	}{
		{
			name:       "White Full Opacity",
			red:        1.0,
			green:      1.0,
			blue:       1.0,
			alpha:      1.0,
			brightness: 1.0,
			expectedR:  1.0,
			expectedG:  1.0,
			expectedB:  1.0,
			expectedA:  1.0,
		},
		{
			name:       "Red Color",
			red:        1.0,
			green:      0.0,
			blue:       0.0,
			alpha:      1.0,
			brightness: 1.0,
			expectedR:  1.0,
			expectedG:  0.0,
			expectedB:  0.0,
			expectedA:  1.0,
		},
		{
			name:       "Half Transparent",
			red:        1.0,
			green:      1.0,
			blue:       1.0,
			alpha:      0.5,
			brightness: 1.0,
			expectedR:  1.0,
			expectedG:  1.0,
			expectedB:  1.0,
			expectedA:  0.5,
		},
		{
			name:       "Double Brightness",
			red:        0.5,
			green:      0.5,
			blue:       0.5,
			alpha:      1.0,
			brightness: 2.0,
			expectedR:  1.0, // 0.5 * 2.0 = 1.0
			expectedG:  1.0,
			expectedB:  1.0,
			expectedA:  1.0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			particle := &components.ParticleComponent{
				Image:      testImage,
				Rotation:   0,
				Scale:      1.0,
				Red:        tc.red,
				Green:      tc.green,
				Blue:       tc.blue,
				Alpha:      tc.alpha,
				Brightness: tc.brightness,
				Additive:   false,
			}

			pos := &components.PositionComponent{X: 100, Y: 100}
			cameraX := 0.0

			vertices := rs.buildParticleVertices(particle, pos, cameraX)

			// 验证所有顶点颜色相同
			for i, v := range vertices {
				tolerance := float32(0.01)

				if math.Abs(float64(v.ColorR-tc.expectedR)) > float64(tolerance) {
					t.Errorf("Vertex %d: ColorR expected %.2f, got %.2f", i, tc.expectedR, v.ColorR)
				}
				if math.Abs(float64(v.ColorG-tc.expectedG)) > float64(tolerance) {
					t.Errorf("Vertex %d: ColorG expected %.2f, got %.2f", i, tc.expectedG, v.ColorG)
				}
				if math.Abs(float64(v.ColorB-tc.expectedB)) > float64(tolerance) {
					t.Errorf("Vertex %d: ColorB expected %.2f, got %.2f", i, tc.expectedB, v.ColorB)
				}
				if math.Abs(float64(v.ColorA-tc.expectedA)) > float64(tolerance) {
					t.Errorf("Vertex %d: ColorA expected %.2f, got %.2f", i, tc.expectedA, v.ColorA)
				}
			}
		})
	}
}

// TestDrawParticles_VertexArrayReuse 测试顶点数组复用（验证每帧重置切片长度）
func TestDrawParticles_VertexArrayReuse(t *testing.T) {
	em := ecs.NewEntityManager()
	rs := NewRenderSystem(em)

	// 验证预分配容量
	if cap(rs.particleVertices) != 4000 {
		t.Errorf("Expected particleVertices capacity 4000, got %d", cap(rs.particleVertices))
	}
	if cap(rs.particleIndices) != 6000 {
		t.Errorf("Expected particleIndices capacity 6000, got %d", cap(rs.particleIndices))
	}

	// 创建测试粒子实体（不带 UIComponent，作为游戏世界粒子）
	testImage := ebiten.NewImage(32, 32)

	for i := 0; i < 5; i++ {
		entityID := em.CreateEntity()
		em.AddComponent(entityID, &components.PositionComponent{X: 100, Y: 100})
		em.AddComponent(entityID, &components.ParticleComponent{
			Image:      testImage,
			Rotation:   0,
			Scale:      1.0,
			Red:        1.0,
			Green:      1.0,
			Blue:       1.0,
			Alpha:      1.0,
			Brightness: 1.0,
			Additive:   false,
		})
	}

	// 创建测试屏幕
	screen := ebiten.NewImage(800, 600)

	// 第一次渲染（使用 DrawGameWorldParticles，因为粒子没有 UIComponent）
	rs.DrawGameWorldParticles(screen, 0)

	// 验证顶点数量（5个粒子 * 4个顶点 = 20个顶点）
	expectedVertexCount := 5 * 4
	if len(rs.particleVertices) != expectedVertexCount {
		t.Errorf("First draw: expected %d vertices, got %d", expectedVertexCount, len(rs.particleVertices))
	}

	// 第二次渲染（应该重置切片长度）
	rs.DrawGameWorldParticles(screen, 0)

	// 验证顶点数量仍然正确（没有累积）
	if len(rs.particleVertices) != expectedVertexCount {
		t.Errorf("Second draw: expected %d vertices, got %d (array not reset)", expectedVertexCount, len(rs.particleVertices))
	}

	// 验证容量未被超出
	if cap(rs.particleVertices) < expectedVertexCount {
		t.Errorf("Vertex capacity exceeded: capacity %d < length %d", cap(rs.particleVertices), len(rs.particleVertices))
	}
}

// TestDrawParticles_BlendModeSelection 测试混合模式选择（Additive vs Normal）
func TestDrawParticles_BlendModeSelection(t *testing.T) {
	em := ecs.NewEntityManager()
	rs := NewRenderSystem(em)

	// 创建测试图片
	testImage := ebiten.NewImage(32, 32)

	// 创建 Normal 混合模式粒子（不带 UIComponent，作为游戏世界粒子）
	entityNormal := em.CreateEntity()
	em.AddComponent(entityNormal, &components.PositionComponent{X: 100, Y: 100})
	em.AddComponent(entityNormal, &components.ParticleComponent{
		Image:      testImage,
		Rotation:   0,
		Scale:      1.0,
		Red:        1.0,
		Green:      1.0,
		Blue:       1.0,
		Alpha:      1.0,
		Brightness: 1.0,
		Additive:   false, // Normal 混合
	})

	// 创建 Additive 混合模式粒子
	entityAdditive := em.CreateEntity()
	em.AddComponent(entityAdditive, &components.PositionComponent{X: 200, Y: 200})
	em.AddComponent(entityAdditive, &components.ParticleComponent{
		Image:      testImage,
		Rotation:   0,
		Scale:      1.0,
		Red:        1.0,
		Green:      1.0,
		Blue:       1.0,
		Alpha:      1.0,
		Brightness: 1.0,
		Additive:   true, // Additive 混合
	})

	// 创建测试屏幕
	screen := ebiten.NewImage(800, 600)

	// 执行渲染（使用 DrawGameWorldParticles，因为粒子没有 UIComponent）
	rs.DrawGameWorldParticles(screen, 0)

	// 注意：由于我们无法直接检查 DrawTrianglesOptions，
	// 这个测试主要验证渲染不会崩溃，并且两种混合模式都能正常处理

	// 验证顶点数量（2个粒子）
	// 注意：由于分批渲染，最后一个批次的顶点会保留在数组中
	// 所以这里我们只验证渲染成功完成
	if len(rs.particleVertices) == 0 {
		t.Error("No vertices generated for particles")
	}
}

// TestBuildParticleVertices_NilImage 测试空图片处理
func TestBuildParticleVertices_NilImage(t *testing.T) {
	em := ecs.NewEntityManager()
	rs := NewRenderSystem(em)

	particle := &components.ParticleComponent{
		Image:      nil, // 空图片
		Rotation:   0,
		Scale:      1.0,
		Red:        1.0,
		Green:      1.0,
		Blue:       1.0,
		Alpha:      1.0,
		Brightness: 1.0,
		Additive:   false,
	}

	pos := &components.PositionComponent{X: 100, Y: 100}
	cameraX := 0.0

	vertices := rs.buildParticleVertices(particle, pos, cameraX)

	// 验证返回 nil
	if vertices != nil {
		t.Errorf("Expected nil for nil image, got %d vertices", len(vertices))
	}
}

// TestDrawParticles_NoParticles 测试没有粒子时的渲染
func TestDrawParticles_NoParticles(t *testing.T) {
	em := ecs.NewEntityManager()
	rs := NewRenderSystem(em)

	screen := ebiten.NewImage(800, 600)

	// 执行渲染（应该快速返回，不崩溃）
	rs.DrawParticles(screen, 0)

	// 验证顶点数组为空
	if len(rs.particleVertices) != 0 {
		t.Errorf("Expected 0 vertices, got %d", len(rs.particleVertices))
	}
}

// BenchmarkDrawParticles100 性能测试：100个粒子渲染
func BenchmarkDrawParticles100(b *testing.B) {
	em := ecs.NewEntityManager()
	rs := NewRenderSystem(em)

	// 创建测试图片
	testImage := ebiten.NewImage(32, 32)

	// 创建 100 个粒子实体
	for i := 0; i < 100; i++ {
		entityID := em.CreateEntity()
		em.AddComponent(entityID, &components.PositionComponent{
			X: float64(100 + i*10),
			Y: float64(100 + (i%5)*20),
		})
		em.AddComponent(entityID, &components.ParticleComponent{
			Image:      testImage,
			Rotation:   float64(i % 360),
			Scale:      1.0 + float64(i%5)*0.1,
			Red:        1.0,
			Green:      0.8,
			Blue:       0.6,
			Alpha:      0.9,
			Brightness: 1.2,
			Additive:   i%2 == 0, // 交替使用加法混合和普通混合
		})
	}

	// 创建测试屏幕
	screen := ebiten.NewImage(800, 600)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rs.DrawParticles(screen, 0)
	}
}

// BenchmarkDrawParticles1000 性能测试：1000个粒子渲染（目标性能测试）
func BenchmarkDrawParticles1000(b *testing.B) {
	em := ecs.NewEntityManager()
	rs := NewRenderSystem(em)

	// 创建测试图片
	testImage := ebiten.NewImage(32, 32)

	// 创建 1000 个粒子实体
	for i := 0; i < 1000; i++ {
		entityID := em.CreateEntity()
		em.AddComponent(entityID, &components.PositionComponent{
			X: float64(100 + i*5),
			Y: float64(100 + (i%10)*15),
		})
		em.AddComponent(entityID, &components.ParticleComponent{
			Image:      testImage,
			Rotation:   float64(i % 360),
			Scale:      0.5 + float64(i%10)*0.1,
			Red:        float64(i%100) / 100.0,
			Green:      float64((i+33)%100) / 100.0,
			Blue:       float64((i+66)%100) / 100.0,
			Alpha:      0.5 + float64(i%50)/100.0,
			Brightness: 0.8 + float64(i%20)/50.0,
			Additive:   i%3 == 0, // 1/3 使用加法混合
		})
	}

	// 创建测试屏幕
	screen := ebiten.NewImage(800, 600)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rs.DrawParticles(screen, 0)
	}
	b.StopTimer()

	// 报告每次操作的分配次数
	b.ReportAllocs()
}

// BenchmarkBuildParticleVertices 性能测试：单个粒子顶点生成
func BenchmarkBuildParticleVertices(b *testing.B) {
	em := ecs.NewEntityManager()
	rs := NewRenderSystem(em)

	testImage := ebiten.NewImage(32, 32)
	particle := &components.ParticleComponent{
		Image:      testImage,
		Rotation:   45.0,
		Scale:      1.5,
		Red:        1.0,
		Green:      0.8,
		Blue:       0.6,
		Alpha:      0.9,
		Brightness: 1.2,
		Additive:   false,
	}
	pos := &components.PositionComponent{X: 400, Y: 300}
	cameraX := 100.0

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rs.buildParticleVertices(particle, pos, cameraX)
	}
}
