# **Epic 6: 动画系统迁移 - 原版 Reanim 骨骼动画系统 (Animation System Migration - Reanim)**

**史诗目标:** 将当前基于简单帧数组的动画系统直接替换为原版 PVZ 的 Reanim 骨骼动画系统，实现 100% 还原原版动画效果，支持部件变换和复杂动画表现。

---

## 背景与动机

**现有系统：**
- 简单的帧数组动画系统（`AnimationComponent` + `AnimationSystem`）
- 适用于简单精灵表动画，但无法实现复杂的部件动画
- 无法还原原版 PVZ 的精细动画效果

**目标系统：**
- 原版 PVZ 的 Reanim 骨骼动画系统
- 支持部件（head, body, leaf 等）独立变换
- 支持位置、缩放、倾斜等复杂变换
- 已有解析后的资源（`assets/reanim/*.reanim` 和部件图片）
- 已有参考实现（`pvzwine_reverse/test_animation_viewer.go`）

**迁移策略：**
- **直接替换**：删除旧系统，不考虑向后兼容性
- **一次性完成**：所有使用动画的实体一起迁移
- **简化架构**：避免双系统并存的复杂性

---

## Stories

**Story 6.1: Reanim 基础设施（解析器和资源加载）**
> **As a** 开发者,
> **I want** to parse Reanim XML files and load sprite parts,
> **so that** I can use original PVZ animation data.

**Acceptance Criteria:**
1. 实现 `ReanimXML`, `Track`, `Frame` 数据结构（对应原版格式）
2. 实现 XML 解析器 `internal/reanim/parser.go`
3. 解析器能成功解析至少 3 个 Reanim 文件（PeaShooter, Sunflower, Wallnut）
4. 实现资源加载器扩展，支持按 Reanim 定义加载部件图片
5. 单元测试覆盖率 ≥ 80%
6. 与参考实现 `test_animation_viewer.go` 的数据结构对比验证

---

**Story 6.2: ReanimComponent 和 ReanimSystem（核心动画逻辑）**
> **As a** 开发者,
> **I want** to replace the old animation system with Reanim system,
> **so that** entities can use complex skeletal animations.

**Acceptance Criteria:**
1. **删除**：`pkg/components/animation.go` 文件（旧动画组件）
2. **删除**：`pkg/systems/animation_system.go` 文件（旧动画系统）
3. **创建**：`pkg/components/reanim_component.go`（新 Reanim 组件）
   - 包含部件图片映射 `map[string]*ebiten.Image`
   - 包含当前动画状态（动画名、帧号、帧缓存）
   - 组件纯数据，无方法
4. **创建**：`pkg/systems/reanim_system.go`（新 Reanim 系统）
   - 实现 `Update()` 方法：推进动画帧
   - 实现动画播放逻辑：帧推进、循环、FPS 控制
   - 实现帧缓存机制：处理空帧继承（原版特性）
5. 单元测试：验证动画播放逻辑正确（帧推进、循环、缓存）
6. 单元测试覆盖率 ≥ 80%

---

**Story 6.3: 渲染系统改造和实体迁移**
> **As a** 开发者,
> **I want** to update the render system and all entities to use Reanim,
> **so that** the game displays animations with original PVZ quality.

**Acceptance Criteria:**
1. **修改**：`pkg/systems/render_system.go`
   - 删除旧的 `AnimationComponent` 渲染逻辑
   - 实现新的 `ReanimComponent` 渲染逻辑
   - 支持部件变换：位置（X, Y）、缩放（ScaleX, ScaleY）、倾斜（SkewX, SkewY）
   - 按轨道顺序渲染部件（保证 Z-order 正确）
2. **更新**：所有实体工厂函数（`pkg/entities/`）
   - 豌豆射手使用 `ReanimComponent`
   - 向日葵使用 `ReanimComponent`
   - 坚果墙使用 `ReanimComponent`
   - 普通僵尸使用 `ReanimComponent`
   - 路障/铁桶僵尸使用 `ReanimComponent`
3. **更新**：游戏场景初始化，加载 Reanim 资源
4. 集成测试：游戏可正常启动，显示主菜单和游戏场景
5. 集成测试：关卡 1-1 可正常运行，所有动画流畅播放
6. 性能测试：游戏运行稳定在 60 FPS，渲染时间 < 5ms/frame
7. 视觉验证：动画效果与参考实现 `test_animation_viewer` 一致

---

## 技术约束

- ECS 架构保持不变
- 渲染系统 API 签名保持一致（`Draw(screen *ebiten.Image)` 方法）
- 性能目标：60 FPS，每帧渲染时间 < 5ms
- 资源位置：`assets/reanim/*.reanim` 和 `assets/reanim/images/*.png`

---

## 风险与缓解

**主要风险：**
- 一次性替换可能导致整个游戏动画系统失效

**缓解措施：**
- 使用功能分支 `feature/animation-migration` 开发（已存在）
- 每个 Story 完成后确保代码可编译和运行
- 参考 `pvzwine_reverse/test_animation_viewer.go` 验证每个阶段
- 创建 git tag `v0.x-before-reanim-migration` 用于快速回滚

**回滚计划：**
- 如果迁移失败，回滚整个分支到 main
- 预计回滚时间：< 5 分钟（简单的 git 操作）

---

## Definition of Done

- ✅ 所有 3 个 Story 完成并通过验收标准
- ✅ 旧的 `AnimationComponent` 和 `AnimationSystem` 已删除
- ✅ 所有游戏实体（植物、僵尸）使用 Reanim 动画正常运行
- ✅ 动画播放流畅，游戏运行稳定在 60 FPS
- ✅ 所有单元测试通过，覆盖率 ≥ 80%
- ✅ 代码符合项目编码规范（`gofmt`, `golangci-lint`）
- ✅ 文档更新：`CLAUDE.md` 添加 Reanim 系统说明
- ✅ 游戏可正常启动和玩关卡 1-1
- ✅ 无编译错误或警告

---

## 参考资源

- **参考实现**：`/mnt/disk0/project/game/pvz/ck/pvzwine_reverse/test_animation_viewer.go`
- **文档**：`/mnt/disk0/project/game/pvz/ck/pvzwine_reverse/GO_EBITENGINE_USAGE.md`
- **文档**：`/mnt/disk0/project/game/pvz/ck/pvzwine_reverse/REANIM_QUICKSTART.md`
- **资源**：`assets/reanim/*.reanim`（143 个动画文件）
- **资源**：`assets/reanim/images/*.png`（部件图片）
