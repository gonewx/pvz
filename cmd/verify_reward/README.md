# Reward Animation Verifier

快速验证关卡奖励动画系统的工具，无需完成整个关卡即可测试卡片包掉落、弹跳和奖励面板显示效果。

## 用途

- ✅ 快速测试奖励动画，无需完成关卡（2秒 vs 5分钟）
- ✅ 调试抛物线轨迹和弹跳动画效果
- ✅ 验证奖励面板布局和文本显示
- ✅ 调整动画时序和参数

## 使用方法

### 基本使用

```bash
# 测试向日葵奖励（默认）
go run cmd/verify_reward/main.go

# 测试指定植物奖励
go run cmd/verify_reward/main.go --plant=peashooter

# 测试樱桃炸弹奖励
go run cmd/verify_reward/main.go --plant=cherrybomb

# 详细日志
go run cmd/verify_reward/main.go --verbose
```

### 快捷键

- **Space / Click** - 点击卡片包展开 / 关闭奖励面板
- **R** - 重新开始动画
- **Q** - 退出程序

## 命令行参数

| 参数 | 默认值 | 说明 |
|------|--------|------|
| `--plant` | `sunflower` | 要解锁的植物ID（如 sunflower, peashooter, cherrybomb, wallnut） |
| `--verbose` | `false` | 启用详细日志输出 |

## 动画阶段说明

验证程序会完整演示以下 5 个阶段：

### 1. Dropping（掉落阶段）
- 卡片包从屏幕右上方以抛物线轨迹掉落
- 使用真实物理公式（重力 800 px/s²）
- 自动过渡到弹跳阶段

### 2. Bouncing（弹跳阶段）
- 卡片包落地后播放 3 次弹跳动画
- 使用 sin 曲线模拟弹跳效果
- 振幅递减（每次 × 0.6）
- 自动过渡到展开阶段

### 3. Expanding（展开阶段）
- 等待玩家点击卡片包
- **按 Space 键或点击鼠标** 触发展开
- 点击后过渡到显示阶段

### 4. Showing（显示阶段）
- 显示奖励面板（背景 + 植物卡片 + 文本）
- 卡片从小到大缩放动画（0.5x → 1.5x）
- 显示植物名称和描述
- **按 Space 键或点击** 关闭面板

### 5. Closing（关闭阶段）
- 淡出动画
- 清理奖励实体
- 动画完成

## 示例场景

### 场景 1: 快速验证多种植物奖励

```bash
# 测试不同植物的奖励效果
go run cmd/verify_reward/main.go --plant=sunflower    # 向日葵
go run cmd/verify_reward/main.go --plant=peashooter   # 豌豆射手
go run cmd/verify_reward/main.go --plant=cherrybomb   # 樱桃炸弹
go run cmd/verify_reward/main.go --plant=wallnut      # 坚果墙
```

### 场景 2: 调试抛物线轨迹

```bash
# 启动验证程序，观察卡片包掉落轨迹
go run cmd/verify_reward/main.go --verbose
# 按 R 键重复观察，调整参数直到满意
```

### 场景 3: 测试交互流程

```bash
# 完整测试用户交互
go run cmd/verify_reward/main.go
# 等待卡片包落地弹跳后，按 Space 键展开
# 观察奖励面板显示，再按 Space 键关闭
# 按 R 键重新测试
```

## 技术细节

### 加载的系统

- ✅ RewardAnimationSystem - 奖励动画逻辑（掉落、弹跳、展开）
- ✅ RewardPanelRenderSystem - 奖励面板渲染
- ✅ RenderSystem - 基础渲染支持
- ❌ LevelSystem - 不需要（直接触发奖励）
- ❌ WaveSpawnSystem - 不需要（无僵尸波次）

### 调试信息显示

屏幕左上角显示：
- 当前植物ID和名称
- 当前动画阶段
- 系统激活状态（Active/Completed）
- 操作提示

### 物理参数（可在代码中调整）

```go
const (
    RewardGravity          = 800.0  // 重力加速度（像素/秒²）
    RewardInitialVelocityX = -200.0 // 初始水平速度（向左）
    RewardInitialVelocityY = -400.0 // 初始垂直速度（向上）

    RewardBounceFrequency  = 8.0    // 弹跳频率（Hz）
    RewardBounceAmplitude  = 30.0   // 初始弹跳振幅（像素）
    RewardBounceDecay      = 0.6    // 振幅衰减系数
    RewardMaxBounces       = 3      // 弹跳次数
)
```

## 支持的植物ID

当前系统支持的植物ID（根据 PlantUnlockManager）：

- `sunflower` - 向日葵
- `peashooter` - 豌豆射手
- `cherrybomb` - 樱桃炸弹
- `wallnut` - 坚果墙
- `snowpea` - 寒冰射手
- `chomper` - 大嘴花
- `repeater` - 双发射手

## 注意事项

1. **资源文件**：需要以下资源文件才能完整显示（如未找到会使用占位符）：
   - `assets/images/AwardScreen_Back.jpg` - 奖励背景
   - `assets/images/SeedPacket_Larger.png` - 大尺寸植物卡片
   - `assets/particles/AwardRays*.png` - 光芒特效图片（可选）

2. **字体渲染**：当前使用 `ebitenutil.DebugPrint` 占位符，后续优化为 TrueType 字体

3. **粒子特效**：Award.xml 粒子效果暂未集成，待 ParticleSystem 完善后添加

## 故障排查

### 问题：卡片包不显示

**原因**：缺少 SpriteComponent 或图片资源
**解决**：检查 `assets/images/` 目录是否包含卡片包图片

### 问题：奖励面板文本乱码

**原因**：字体文件未加载或编码问题
**解决**：当前使用占位符文本，待字体系统完善后自动修复

### 问题：动画太快/太慢

**原因**：物理参数不合适
**解决**：在 `pkg/systems/reward_animation_system.go` 中调整常量值

## 相关文件

- **Story 文档**: `docs/stories/8.3.story.md`
- **系统实现**: `pkg/systems/reward_animation_system.go`
- **面板渲染**: `pkg/systems/reward_panel_render_system.go`
- **组件定义**:
  - `pkg/components/reward_animation_component.go`
  - `pkg/components/reward_panel_component.go`

## 与完整游戏的区别

| 特性 | 验证程序 | 完整游戏 |
|------|---------|---------|
| 启动时间 | < 2 秒 | ~ 5 分钟（需完成关卡） |
| 测试范围 | 仅奖励动画 | 完整游戏流程 |
| 调试效率 | 高（即时重启） | 低（需重新打关卡） |
| 粒子效果 | 待集成 | 完整 Award.xml 特效 |
| 场景切换 | 无（固定奖励界面） | 有（返回主菜单/关卡地图） |

## 性能测试

- **帧率目标**: 60 FPS
- **内存占用**: < 100 MB
- **启动时间**: < 2 秒
- **重启时间**: < 0.5 秒

## 未来优化

- [ ] 集成 Award.xml 粒子特效（12个光芒发射器）
- [ ] 添加卡片包图片渲染
- [ ] 优化字体渲染（替换 DebugPrint）
- [ ] 支持命令行参数调整物理参数
- [ ] 添加慢动作模式（--slow-motion）
- [ ] 导出动画序列为 GIF（--export-gif）
