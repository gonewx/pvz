# **Epic 7: 粒子特效系统 (Particle Effect System) - Brownfield Enhancement**

**史诗目标:** 实现完整的粒子特效系统，支持原版PVZ的所有视觉特效（僵尸肢体掉落、爆炸、溅射等），通过解析XML配置和高性能批量渲染技术，为游戏提供丰富的视觉反馈。

---

## 背景与动机

**现有系统上下文：**
- **项目架构**: ECS (Entity-Component-System) 架构
- **游戏引擎**: Ebitengine v2
- **渲染系统**: `RenderSystem`，基于`ReanimComponent`渲染游戏实体
- **资源管理**: `ResourceManager`，支持图片、音频、Reanim XML的加载和缓存
- **组件策略**: 游戏世界实体使用`ReanimComponent`，UI元素使用`SpriteComponent`

**现有问题：**
- 游戏缺少粒子特效，视觉反馈单调
- 无法展现原版PVZ的经典特效（如僵尸手臂掉落、樱桃爆炸、豌豆溅射等）
- 已有106个XML配置文件和79个粒子贴图资源，但尚未实现解析和渲染

**增强细节：**

**资源现状：**
- `assets/effect/particles/`: 106个粒子配置XML文件
- `assets/particles/`: 79个粒子贴图PNG文件
- XML配置包含40+种粒子属性（发射器、轨迹、生命周期、动画等）

**要实现的功能：**
1. **XML配置解析系统**: 解析所有粒子配置，支持发射器、粒子属性、力场等
2. **粒子引擎核心**: 发射器逻辑、粒子生命周期管理、动画插值、力场系统
3. **高性能渲染**: 使用Ebitengine的`DrawTriangles`批量渲染，支持加法混合
4. **ECS集成**: 创建粒子相关组件和系统，与现有架构无缝集成
5. **游戏事件触发**: 在僵尸死亡、爆炸、击中等游戏事件中触发粒子效果

**如何集成：**
- 扩展`ResourceManager`支持粒子XML和贴图加载
- 创建新的`ParticleComponent`和`EmitterComponent`（ECS组件）
- 创建新的`ParticleSystem`（更新粒子状态）
- 在`RenderSystem`中添加粒子批量渲染逻辑
- 在`BehaviorSystem`等现有系统中触发粒子效果

**成功标准：**
- 成功解析并加载106个粒子配置XML文件
- 支持所有核心粒子属性（位置、速度、透明度、缩放、旋转、颜色、力场等）
- 支持加法混合模式（Additive Blending）
- 性能目标：同屏1000+粒子保持60 FPS
- 僵尸死亡、爆炸、子弹击中等场景能正确触发粒子效果

---

## Stories

### **Story 7.1: 粒子配置解析系统**
> **As a** 开发者,
> **I want** to parse particle XML configuration files and load particle textures,
> **so that** I can use original PVZ particle effect data.

**Acceptance Criteria:**
1. 实现粒子配置数据结构：`ParticleConfig`, `EmitterConfig`, `Field`等
2. 实现XML解析器 `internal/particle/parser.go`
3. 解析器能成功解析至少5个代表性粒子配置（Award.xml, BossExplosion.xml, CabbageSplat.xml等）
4. 支持40+种粒子属性的解析（Spawn*, Particle*, Launch*, Field等）
5. 扩展`ResourceManager`，添加`LoadParticleConfig(name string)`方法
6. 粒子贴图加载集成到`ResourceManager`
7. 单元测试覆盖率 ≥ 80%
8. 解析结果与XML配置一致性验证

---

### **Story 7.2: 粒子发射器核心引擎**
> **As a** 开发者,
> **I want** to implement particle emitter logic and lifecycle management,
> **so that** particles can be spawned, updated, and destroyed correctly.

**Acceptance Criteria:**
1. **创建**: `pkg/components/particle_component.go`（粒子组件）
   - 存储单个粒子的状态（位置、速度、旋转、缩放、透明度、颜色、生命周期等）
   - 纯数据组件，无方法
2. **创建**: `pkg/components/emitter_component.go`（发射器组件）
   - 存储发射器配置引用和状态
   - 管理发射时机、活跃粒子数等
3. **创建**: `pkg/systems/particle_system.go`（粒子系统）
   - 实现`Update(dt float64)`方法：更新所有粒子和发射器
   - 粒子生命周期管理：生成、更新、销毁
   - 动画插值系统：支持线性插值和关键帧动画（Alpha, Scale, Spin等）
   - 力场系统：支持重力、摩擦、加速度等
4. 实现发射器逻辑：
   - 按配置生成粒子（SpawnMinActive, SpawnRate等）
   - 发射速度和角度（LaunchSpeed, LaunchAngle）
   - 发射区域（EmitterBox, EmitterRadius, EmitterType）
5. 单元测试：验证粒子生命周期、动画插值、力场计算正确性
6. 单元测试覆盖率 ≥ 80%

---

### **Story 7.3: 粒子渲染系统与ECS集成**
> **As a** 开发者,
> **I want** to render particles efficiently using Ebitengine DrawTriangles,
> **so that** the game can display thousands of particles at 60 FPS.

**Acceptance Criteria:**
1. **修改**: `pkg/systems/render_system.go`
   - 添加`DrawParticles(screen *ebiten.Image, cameraX float64)`方法
   - 使用`ebiten.DrawTriangles`批量渲染粒子（每个粒子2个三角形）
   - 支持加法混合模式（Additive Blending）：
     ```go
     op.Blend = ebiten.Blend{
         BlendFactorSourceRGB:      ebiten.BlendFactorOne,
         BlendFactorDestinationRGB: ebiten.BlendFactorOne,
         BlendOperationRGB:         ebiten.BlendOperationAdd,
         // ...
     }
     ```
   - 支持粒子属性：位置、旋转、缩放、透明度、颜色、亮度
   - 按粒子配置的混合模式渲染（Additive vs Normal）
2. **优化**: 复用顶点数组，避免每帧重新分配内存
3. **优化**: 关闭粒子贴图的mipmaps（如果需要）
4. **集成**: 粒子渲染插入到正确的渲染层级（在游戏世界和UI之间）
5. **创建**: 粒子实体工厂函数 `pkg/entities/particle_factory.go`
   - `CreateParticleEffect(name string, x, y float64)`
6. 性能测试：1000个粒子同屏渲染，保持60 FPS
7. 视觉验证：粒子渲染效果与原版PVZ一致（加法混合、颜色混合等）

---

### **Story 7.4: 游戏效果集成与事件触发**
> **As a** 玩家,
> **I want** to see particle effects when zombies die, explosions occur, and projectiles hit,
> **so that** I get rich visual feedback during gameplay.

**Acceptance Criteria:**
1. **修改**: `pkg/systems/behavior_system.go`
   - 僵尸死亡时触发粒子效果（手臂掉落、头掉落等）
   - 樱桃炸弹爆炸时触发爆炸粒子效果
2. **修改**: `pkg/systems/collision_system.go`（如存在）或相关系统
   - 豌豆击中僵尸时触发溅射粒子效果
   - 卷心菜击中时触发卷心菜溅射效果
3. **实现**: 至少5种游戏粒子效果：
   - 僵尸死亡效果（ZombieDeath.xml或类似）
   - 爆炸效果（BossExplosion.xml或类似）
   - 豌豆溅射（PeaSplat.xml或类似）
   - 卷心菜溅射（CabbageSplat.xml）
   - 奖励光效（Award.xml或类似）
4. **创建**: 粒子效果触发辅助函数：
   - `SpawnParticleEffect(effectName string, worldX, worldY float64)`
5. 集成测试：游戏运行时，上述场景能正确触发粒子效果
6. 集成测试：粒子效果播放完成后自动清理，无内存泄漏
7. 性能测试：战斗场景（多僵尸死亡、多爆炸）保持60 FPS
8. 视觉验证：效果与原版PVZ一致

---

## 兼容性要求

- ✅ **ECS架构保持不变**: 新增组件和系统，不修改核心ECS框架
- ✅ **现有渲染管线兼容**: 粒子渲染不影响植物、僵尸、UI的渲染
- ✅ **资源管理器扩展**: 通过添加新方法扩展`ResourceManager`，不破坏现有API
- ✅ **性能无回退**: 添加粒子系统后，游戏在无粒子场景下性能不受影响
- ✅ **无破坏性修改**: 不删除现有组件或系统，只添加新功能

---

## 风险与缓解

**主要风险：**
1. **性能风险**: 大量粒子可能导致帧率下降
2. **内存风险**: 粒子对象频繁创建/销毁可能导致GC压力
3. **复杂性风险**: 40+种粒子属性，解析和实现可能出错

**缓解措施：**
1. **性能缓解**:
   - 使用`ebiten.DrawTriangles`批量渲染（Ebitengine官方推荐）
   - 实现粒子对象池，复用粒子对象
   - 性能监控：每Story完成后测试帧率
2. **内存缓解**:
   - 粒子生命周期结束后立即回收到对象池
   - 限制同屏粒子最大数量（可配置）
3. **复杂性缓解**:
   - 分阶段实现：先支持核心属性，再逐步添加高级特性
   - 参考原版PVZ的实现和已有XML配置
   - 充分的单元测试和集成测试

**回滚计划：**
- 使用功能分支 `feature/particle-system` 开发
- 每个Story完成后确保游戏可正常运行（粒子可禁用）
- 如果性能不达标，可临时禁用粒子系统
- 预计回滚时间：< 5 分钟（git 分支切换）

---

## Definition of Done

- ✅ 所有4个Story完成并通过验收标准
- ✅ 成功解析106个粒子配置XML文件
- ✅ 支持40+种粒子属性和力场系统
- ✅ 至少5种游戏粒子效果正常触发和播放
- ✅ 性能测试通过：1000+粒子同屏保持60 FPS
- ✅ 所有单元测试通过，覆盖率 ≥ 80%
- ✅ 代码符合项目编码规范（`gofmt`, `golangci-lint`）
- ✅ 文档更新：`CLAUDE.md`添加粒子系统说明
- ✅ 游戏可正常启动和玩关卡1-1，粒子效果正常显示
- ✅ 无内存泄漏（长时间运行测试）
- ✅ 无编译错误或警告

---

## 技术约束

- **引擎**: 严格使用Ebitengine v2的API
- **渲染方法**: 使用`ebiten.DrawTriangles`而非`DrawImage`（性能考虑）
- **混合模式**: 支持加法混合（Additive）和普通混合（Normal）
- **性能目标**: 60 FPS，粒子渲染时间 < 3ms/frame（1000粒子）
- **内存目标**: 粒子系统总内存占用 < 50MB（峰值）
- **资源位置**:
  - 配置: `assets/effect/particles/*.xml`
  - 贴图: `assets/particles/*.png`

---

## 参考资源

- **Ebitengine官方文档**: DrawTriangles和Blend选项
- **DeepWiki查询结果**: Ebitengine粒子系统最佳实践
- **XML配置示例**:
  - `assets/effect/particles/Award.xml`（复杂多发射器）
  - `assets/effect/particles/BossExplosion.xml`（爆炸效果）
  - `assets/effect/particles/CabbageSplat.xml`（溅射效果）
- **粒子贴图**: `assets/particles/`（79个PNG文件）
- **项目架构文档**: `CLAUDE.md`（ECS架构、组件策略）

---

## 备注

**范围说明：**
本Epic包含4个Story，略超出brownfield-create-epic建议的1-3个Story范围。原因：
- 粒子系统功能完整，涉及解析、引擎、渲染、集成4个独立模块
- 每个Story职责单一，便于开发和测试
- 4个Story可在3-4个开发会话内完成，符合brownfield增强定位

如需调整为3个Story（合并Story 7.3和7.4），请告知。
