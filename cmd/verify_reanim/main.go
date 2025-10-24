// verify_reanim.go - Reanim 动画系统验证程序
// 根据 .meta/reanim/reanim.md 技术文档验证当前实现
// 使用项目的 ResourceManager 加载资源
package main

import (
	"flag"
	"fmt"
	"image/color"
	"log"
	"os"
	"sort"
	"strings"

	"github.com/decker502/pvz/pkg/components"
	"github.com/decker502/pvz/pkg/ecs"
	"github.com/decker502/pvz/pkg/game"
	"github.com/decker502/pvz/pkg/systems"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"gopkg.in/yaml.v3"
)

// ========== 验证报告结构 ==========

type ValidationReport struct {
	TestName string
	Passed   bool
	Message  string
}

var validationReports []ValidationReport

func addReport(testName string, passed bool, message string) {
	validationReports = append(validationReports, ValidationReport{
		TestName: testName,
		Passed:   passed,
		Message:  message,
	})
	status := "✗ FAIL"
	if passed {
		status = "✓ PASS"
	}
	log.Printf("%s | %-30s | %s", status, testName, message)
}

// ========== YAML 配置结构 ==========

type AnimationConfigEntry struct {
	OverlayTracks []string `yaml:"overlay_tracks"`
}

type AnimationConfigs struct {
	Version string                          `yaml:"version"`
	Plants  map[string]AnimationConfigEntry `yaml:"plants"`
	Zombies map[string]AnimationConfigEntry `yaml:"zombies"`
	Effects map[string]AnimationConfigEntry `yaml:"effects"`
}

// LoadAnimationConfigs 从 YAML 文件加载动画配置
func LoadAnimationConfigs(path string) (*AnimationConfigs, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read %s: %w", path, err)
	}
	var configs AnimationConfigs
	if err := yaml.Unmarshal(data, &configs); err != nil {
		return nil, fmt.Errorf("failed to unmarshal yaml: %w", err)
	}
	return &configs, nil
}

// ========== 验证函数（根据技术文档） ==========

// validateTrackClassification 验证轨道类型分类（黄金法则）
func validateTrackClassification(rm *game.ResourceManager, reanimName string) {
	reanimXML := rm.GetReanimXML(reanimName)
	if reanimXML == nil {
		addReport("轨道分类（黄金法则）", false, "无法获取 Reanim 数据")
		return
	}

	visualCount := 0
	logicalCount := 0

	log.Printf("\n  轨道分类详情:")
	for _, track := range reanimXML.Tracks {
		hasImage := false
		for _, frame := range track.Frames {
			if frame.ImagePath != "" {
				hasImage = true
				break
			}
		}

		if hasImage {
			visualCount++
			log.Printf("    [视觉] %s (包含 <i> 标签)", track.Name)
		} else {
			logicalCount++
			log.Printf("    [逻辑] %s (无 <i> 标签)", track.Name)
		}
	}

	addReport("轨道分类（黄金法则）", true,
		fmt.Sprintf("%d 视觉 + %d 逻辑", visualCount, logicalCount))
}

// validateAnimationStyle 验证动画文件风格识别
func validateAnimationStyle(rm *game.ResourceManager, reanimName string) {
	reanimXML := rm.GetReanimXML(reanimName)
	if reanimXML == nil {
		addReport("风格识别", false, "无法获取数据")
		return
	}

	hasLogicalTrack := false
	for _, track := range reanimXML.Tracks {
		isVisual := false
		for _, frame := range track.Frames {
			if frame.ImagePath != "" {
				isVisual = true
				break
			}
		}
		if !isVisual {
			hasLogicalTrack = true
			break
		}
	}

	if hasLogicalTrack {
		addReport("风格识别", true, "风格 A (共享时间轴)")
	} else {
		addReport("风格识别", true, "风格 B (自包含动画)")
	}
}

// validateResourceLoading 验证资源加载
func validateResourceLoading(rm *game.ResourceManager, reanimName string) {
	reanimXML := rm.GetReanimXML(reanimName)
	if reanimXML == nil {
		addReport("资源加载-XML", false, "加载失败")
		return
	}
	addReport("资源加载-XML", true, fmt.Sprintf("FPS=%d, 轨道=%d", reanimXML.FPS, len(reanimXML.Tracks)))

	partImages := rm.GetReanimPartImages(reanimName)
	if partImages == nil || len(partImages) == 0 {
		addReport("资源加载-图片", false, "加载失败")
		return
	}
	addReport("资源加载-图片", true, fmt.Sprintf("%d 张图片", len(partImages)))
}

// validateOverlayConfig 验证叠加层配置
func validateOverlayConfig(configs *AnimationConfigs, reanimName string) {
	// 检查 plants 分组
	if entry, ok := configs.Plants[reanimName]; ok {
		count := len(entry.OverlayTracks)
		if count > 0 {
			addReport("叠加层配置", true, fmt.Sprintf("%d 个叠加轨道: %v", count, entry.OverlayTracks))
		} else {
			addReport("叠加层配置", true, "无叠加轨道")
		}
		return
	}

	// 检查 effects 分组
	if entry, ok := configs.Effects[reanimName]; ok {
		count := len(entry.OverlayTracks)
		if count > 0 {
			addReport("叠加层配置", true, fmt.Sprintf("%d 个叠加轨道", count))
		} else {
			addReport("叠加层配置", true, "无叠加轨道")
		}
		return
	}

	addReport("叠加层配置", true, "未配置（默认无叠加）")
}

// ========== 图形界面游戏循环 ==========

type VerificationGame struct {
	entityManager *ecs.EntityManager
	reanimSystem  *systems.ReanimSystem
	renderSystem  *systems.RenderSystem

	testEntity     ecs.EntityID
	reanimName     string
	availableAnims []string
	currentAnimIdx int
}

func (g *VerificationGame) Update() error {
	// 方向键切换动画
	if inpututil.IsKeyJustPressed(ebiten.KeyArrowRight) || inpututil.IsKeyJustPressed(ebiten.KeyArrowDown) {
		g.currentAnimIdx = (g.currentAnimIdx + 1) % len(g.availableAnims)
		animName := g.availableAnims[g.currentAnimIdx]
		if err := g.reanimSystem.PlayAnimation(g.testEntity, animName); err != nil {
			log.Printf("切换动画失败: %v", err)
		}
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyArrowLeft) || inpututil.IsKeyJustPressed(ebiten.KeyArrowUp) {
		g.currentAnimIdx--
		if g.currentAnimIdx < 0 {
			g.currentAnimIdx = len(g.availableAnims) - 1
		}
		animName := g.availableAnims[g.currentAnimIdx]
		if err := g.reanimSystem.PlayAnimation(g.testEntity, animName); err != nil {
			log.Printf("切换动画失败: %v", err)
		}
	}

	// B 键触发叠加动画
	if inpututil.IsKeyJustPressed(ebiten.KeyB) {
		if err := g.reanimSystem.PlayAnimationOverlay(g.testEntity, "anim_blink", true); err != nil {
			log.Printf("触发叠加失败: %v", err)
		}
	}

	// 更新系统
	deltaTime := 1.0 / 60.0
	g.reanimSystem.Update(deltaTime)

	return nil
}

func (g *VerificationGame) Draw(screen *ebiten.Image) {
	screen.Fill(color.RGBA{135, 206, 235, 255})

	// 渲染动画
	g.renderSystem.Draw(screen, 0)

	// 获取组件信息
	reanimComp, _ := ecs.GetComponent[*components.ReanimComponent](g.entityManager, g.testEntity)

	// 显示信息
	info := fmt.Sprintf("=== Reanim 验证（ResourceManager）===\n")
	info += fmt.Sprintf("资源: %s\n", g.reanimName)
	info += fmt.Sprintf("动画: %s (%d/%d)\n",
		g.availableAnims[g.currentAnimIdx], g.currentAnimIdx+1, len(g.availableAnims))

	if reanimComp != nil {
		info += fmt.Sprintf("帧: %d/%d\n", reanimComp.CurrentFrame, reanimComp.VisibleFrameCount)
		info += fmt.Sprintf("FPS: %d | 循环: %v\n", reanimComp.Reanim.FPS, reanimComp.IsLooping)

		if len(reanimComp.OverlayAnims) > 0 {
			info += "叠加: "
			for _, overlay := range reanimComp.OverlayAnims {
				info += fmt.Sprintf("%s ", overlay.AnimName)
			}
			info += "\n"
		}
	}

	info += "\n操作:\n"
	info += "  ← → ↑ ↓ : 切换动画\n"
	info += "  B        : 触发叠加(anim_blink)\n"
	info += "\n验证:\n"

	passCount := 0
	for _, r := range validationReports {
		if r.Passed {
			passCount++
		}
	}
	info += fmt.Sprintf("  通过: %d/%d 项", passCount, len(validationReports))

	ebitenutil.DebugPrint(screen, info)
}

func (g *VerificationGame) Layout(outsideWidth, outsideHeight int) (int, int) {
	return 800, 600
}

// ========== 主函数 ==========

func main() {
	reanimName := flag.String("anim", "PeaShooterSingle", "动画名称（如 PeaShooterSingle, SunFlower）")
	runGUI := flag.Bool("gui", true, "是否运行图形界面")
	flag.Parse()

	log.SetOutput(os.Stdout)
	log.SetFlags(log.Ltime)

	log.Printf("====== Reanim 系统验证（ResourceManager）======")
	log.Printf("动画: %s", *reanimName)
	log.Printf("文档: .meta/reanim/reanim.md")
	log.Println()

	// 1. 创建 ResourceManager
	audioContext := audio.NewContext(48000)
	rm := game.NewResourceManager(audioContext)

	// 2. 加载资源配置
	if err := rm.LoadResourceConfig("assets/config/resources.yaml"); err != nil {
		log.Fatalf("加载资源配置失败: %v", err)
	}

	// 3. 加载 Reanim 资源
	log.Println(">>> 步骤 1: 加载 Reanim 资源")
	if err := rm.LoadReanimResources(); err != nil {
		log.Fatalf("加载失败: %v", err)
	}

	// 4. 验证资源加载
	log.Println("\n>>> 步骤 2: 验证资源加载")
	validateResourceLoading(rm, *reanimName)

	// 5. 验证轨道分类
	log.Println("\n>>> 步骤 3: 验证轨道分类（黄金法则）")
	validateTrackClassification(rm, *reanimName)

	// 6. 验证动画风格
	log.Println("\n>>> 步骤 4: 验证动画文件风格")
	validateAnimationStyle(rm, *reanimName)

	// 7. 验证叠加层配置
	log.Println("\n>>> 步骤 5: 验证叠加层配置")
	configs, err := LoadAnimationConfigs("cmd/verify_reanim/animation_config.yaml")
	if err != nil {
		log.Printf("警告: 无法加载配置 %v", err)
		configs = &AnimationConfigs{}
	}
	validateOverlayConfig(configs, *reanimName)

	// 输出验证报告
	printFinalReport()

	if !*runGUI {
		return
	}

	// 8. 创建测试实体
	log.Println("\n>>> 步骤 6: 启动图形界面")
	em := ecs.NewEntityManager()
	rs := systems.NewReanimSystem(em)
	renderSys := systems.NewRenderSystem(em)

	testEntity := em.CreateEntity()
	ecs.AddComponent(em, testEntity, &components.PositionComponent{X: 400, Y: 300})

	reanimXML := rm.GetReanimXML(*reanimName)
	partImages := rm.GetReanimPartImages(*reanimName)
	if reanimXML == nil || partImages == nil {
		log.Fatalf("无法获取 Reanim 数据")
	}

	ecs.AddComponent(em, testEntity, &components.ReanimComponent{
		Reanim:     reanimXML,
		PartImages: partImages,
	})

	// 9. 获取可用动画
	availableAnims := []string{}
	for _, track := range reanimXML.Tracks {
		// 检查是否为逻辑轨道
		isLogical := true
		for _, frame := range track.Frames {
			if frame.ImagePath != "" {
				isLogical = false
				break
			}
		}
		if isLogical && strings.HasPrefix(track.Name, "anim_") {
			availableAnims = append(availableAnims, track.Name)
		}
	}

	// 如果没有逻辑轨道，使用直接渲染
	if len(availableAnims) == 0 {
		log.Printf("未发现逻辑轨道，使用直接渲染模式")
		if err := rs.InitializeDirectRender(testEntity); err != nil {
			log.Fatalf("初始化失败: %v", err)
		}
		availableAnims = []string{"direct_render"}
	} else {
		sort.Strings(availableAnims)
		if err := rs.PlayAnimation(testEntity, availableAnims[0]); err != nil {
			log.Fatalf("播放失败: %v", err)
		}
	}

	// 10. 运行游戏
	game := &VerificationGame{
		entityManager:  em,
		reanimSystem:   rs,
		renderSystem:   renderSys,
		testEntity:     testEntity,
		reanimName:     *reanimName,
		availableAnims: availableAnims,
		currentAnimIdx: 0,
	}

	ebiten.SetWindowSize(800, 600)
	ebiten.SetWindowTitle(fmt.Sprintf("Reanim 验证 - %s", *reanimName))
	if err := ebiten.RunGame(game); err != nil {
		log.Fatal(err)
	}
}

func printFinalReport() {
	log.Println("\n========================================")
	log.Println("         验证报告摘要")
	log.Println("========================================")

	passCount := 0
	for _, r := range validationReports {
		status := "✗"
		if r.Passed {
			status = "✓"
			passCount++
		}
		log.Printf("%s | %-30s | %s", status, r.TestName, r.Message)
	}

	log.Println("========================================")
	log.Printf("总计: %d 通过, %d 失败", passCount, len(validationReports)-passCount)

	if passCount == len(validationReports) {
		log.Println("🎉 所有验证通过！")
	} else {
		log.Println("⚠️  部分验证失败")
	}
	log.Println("========================================")
}
