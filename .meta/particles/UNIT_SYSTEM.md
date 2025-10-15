# PvZ 粒子系统单位转换说明

## 问题背景

原版《植物大战僵尸》使用固定时间步长 **0.01秒（1厘秒）** 作为物理更新基准（相当于100FPS）。粒子配置文件中的某些值基于这个时间步长定义，而非真实的"每秒"单位。

## 单位系统分类

### 1. 直接使用的值（已是"每秒"单位）

这些参数在配置文件中已经是标准的"每秒"单位，**无需转换**：

| 参数 | 配置值示例 | 含义 | 说明 |
|------|-----------|------|------|
| `LaunchSpeed` | 330 | 330像素/秒 | 粒子发射速度 |
| `ParticleSpinSpeed` | 720 | 720度/秒 | 粒子旋转速度（2圈/秒） |
| `CollisionSpin` | -3 | -3度/秒 | 碰撞时增加的旋转速度 |
| `ParticleDuration` | 180 | 180厘秒 = 1.8秒 | 粒子持续时间（已转换） |
| `SystemDuration` | 180 | 180厘秒 = 1.8秒 | 系统持续时间（已转换） |

### 2. 基于固定时间步长的值（需要转换）

这些参数是基于0.01秒时间步长的**增量值**，需要转换为真实的"每秒"单位：

| 参数 | 配置值示例 | 原始含义 | 转换后 | 转换公式 |
|------|-----------|---------|--------|---------|
| `Acceleration` (Field) | 17 | 每0.01秒增加17像素/秒 | 1700像素/秒² | `value / 0.01` |
| `Friction` (Field) | 0.005 | 每0.01秒衰减0.5% | 0.5/秒 | `value / 0.01` |

## 转换实现

### 代码位置

`pkg/systems/particle_system.go` 中的 `applyFields()` 函数：

```go
// PopCap's original fixed physics time step (centiseconds)
const OriginalTimeStep = 0.01 // 1 centisecond = 0.01 seconds

case "Acceleration":
    // Unit conversion: Config values are "velocity delta per 0.01s"
    // Convert to true acceleration (pixels/second²)
    ax = ax / OriginalTimeStep // pixels/centisecond → pixels/second²
    ay = ay / OriginalTimeStep

    // Apply acceleration to velocity
    p.VelocityX += ax * dt
    p.VelocityY += ay * dt

case "Friction":
    // Unit conversion: Config values are "velocity decay per 0.01s"
    // Convert to per-second friction coefficient
    frictionX = frictionX / OriginalTimeStep
    frictionY = frictionY / OriginalTimeStep

    // Apply friction (velocity decay)
    p.VelocityX *= (1 - frictionX*dt)
    p.VelocityY *= (1 - frictionY*dt)
```

## 验证示例：ZombieHead

### 配置值
```xml
<LaunchSpeed>330</LaunchSpeed>          <!-- 330像素/秒 ✓ -->
<LaunchAngle>[150 185]</LaunchAngle>    <!-- 150-185度 ✓ -->
<Field>
    <FieldType>Acceleration</FieldType>
    <Y>17</Y>                           <!-- 17像素/厘秒 → 1700像素/秒² -->
</Field>
<ParticleSpinSpeed>[-720 720]</ParticleSpinSpeed>  <!-- ±720度/秒 ✓ -->
<CollisionSpin>[-3 -6]</CollisionSpin>  <!-- -3到-6度/秒 ✓ -->
```

### 物理验证

#### 修复前（错误）
- 加速度：17像素/秒²
- 落地时间：**11.01秒** ❌ 太慢！
- 反弹高度：**27.4像素** ❌ 太低！

#### 修复后（正确）
- 加速度：1700像素/秒²
- 落地时间：**~0.38秒** ✓ 合理
- 反弹高度：**~8像素** ✓ 合理
- 弹跳在0.72秒内完成 ✓ 符合配置预期

## 常见问题

### Q1: 为什么速度不需要转换？
A: 配置文件中的速度值已经是"每秒"单位。原版游戏在导出配置时已经做了转换。

### Q2: 为什么加速度需要特殊处理？
A: 加速度配置值表示的是"每个固定时间步（0.01秒）内速度的增量"，本质是 `Δv/Δt`，需要除以时间步长才能得到真实加速度 `a = Δv/Δt / Δt = Δv/Δt²`。

### Q3: 如何判断一个参数是否需要转换？
A: 规则：
- 如果是Field类型（Acceleration、Friction等力场效果）→ **需要转换**
- 其他参数（速度、角度、旋转速度等）→ **不需要转换**

### Q4: 测试用例如何适配？
A: 测试用例中的配置值需要调整为PopCap单位：
```go
// 修复前（错误）
{FieldType: "Acceleration", Y: "100"} // 期望100像素/秒²

// 修复后（正确）
{FieldType: "Acceleration", Y: "1"}   // 1像素/厘秒 = 100像素/秒²
```

## 参考资料

- 原版PvZ反编译分析（推测）
- ZombieHead.xml 配置文件注释（`.meta/particles/ZombieHead.md`）
- 单位转换分析（`/tmp/unit_analysis.md`）
