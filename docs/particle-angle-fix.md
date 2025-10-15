# 粒子系统角度坐标系修复

## 问题描述

在实现粒子系统时，发现 `ZombieHead.xml` 中的 `LaunchAngle [150 185]` 配置导致僵尸头部向左飞，而不是预期的向右飞（被豌豆击中后应该向后飞）。

## 根本原因

PvZ 使用的角度坐标系与标准数学坐标系不同：

### PvZ 角度坐标系
- **0° = 向左**（僵尸前进方向）
- **90° = 向下**
- **180° = 向右**（僵尸后方）
- **270° = 向上**

### 标准数学/屏幕坐标系
- **0° = 向右**
- **90° = 向下**（屏幕Y轴向下）
- **180° = 向左**
- **270° = 向上**

**两者相差 180°！**

## 解决方案

在 `pkg/systems/particle_system.go` 的 `spawnParticle()` 函数中添加角度转换：

```go
// Story 7.6 修复：PvZ 角度坐标系转换
// PvZ 使用的坐标系：0° = 向左（僵尸前进方向），180° = 向右（僵尸后方）
// 屏幕坐标系：0° = 向右，180° = 向左
// 转换公式：screenAngle = (pvzAngle + 180) % 360
screenAngle := angle + 180.0
if screenAngle >= 360.0 {
    screenAngle -= 360.0
}

// Convert angle to radians and calculate velocity components
angleRad := screenAngle * math.Pi / 180.0
velocityX := speed * math.Cos(angleRad)
velocityY := -speed * math.Sin(angleRad) // 取反以适配屏幕坐标系（Y轴向下为正）
```

## 验证结果

### ZombieHead [150 185]
- **PvZ 角度**: 150° - 185°（接近僵尸后方）
- **转换后**: 330° - 5°（向右的扇形）
- **效果**: ✅ 头部向右飞出

### MoweredZombieHead [190 220]
- **PvZ 角度**: 190° - 220°（超过僵尸后方）
- **转换后**: 10° - 40°（向右下的扇形）
- **效果**: ✅ 头部向右下飞得更远

### PottedPlantGlow [90 270]
- **PvZ 角度**: 90° - 270°（从下到上）
- **转换后**: 270° - 90°（从下到上的右侧半圆）
- **效果**: ✅ 光芒向右侧半圆散发

## 测试覆盖

新增测试文件 `pkg/systems/particle_system_angle_test.go`：

- ✅ `TestAngleConversion`: 验证基本角度转换
- ✅ `TestZombieHeadAngleRange`: 验证 ZombieHead 角度范围
- ✅ `TestMoweredZombieHeadAngleRange`: 验证 MoweredZombieHead 角度范围
- ✅ `TestPottedPlantGlowAngleRange`: 验证 PottedPlantGlow 角度范围

所有测试通过率：**100%**

## 关键发现

1. **注释是正确的**：`.meta/particles/ZombieHead.md` 中的注释准确描述了 PvZ 的角度系统
2. **所有粒子效果统一**：所有粒子效果都使用相同的 PvZ 角度坐标系
3. **无需特殊处理**：不需要根据粒子类型区分坐标系

## 影响范围

- ✅ 所有使用 `LaunchAngle` 的粒子效果
- ✅ 僵尸相关粒子（ZombieHead, MoweredZombieHead, ZombieArm 等）
- ✅ 环境粒子（PottedPlantGlow, Planting 等）
- ✅ 特效粒子（Explosion, Splash 等）

## 弹跳效果验证

### 物理模拟结果

使用 ZombieHead 配置进行物理模拟：

```
初始状态:
  位置: Y=300.0
  地面: Y=390.0 (GroundConstraint Y=90)
  速度: vy=71.4 (向下)
  重力: 17.0 像素/秒²
  反弹系数: 0.3

弹跳记录:
  弹跳 #1 (t=1.12s): 速度 90.4 → -27.1 (损失 70%)
  弹跳 #2 (t=4.30s): 速度 27.0 → -8.1 (损失 70%)

结果: ✅ 2次明显弹跳，符合预期
```

### 完整的运动过程

1. **初始发射** (t=0s)
   - 头部以 330 像素/秒向右飞出
   - 初始向下速度 71.4 像素/秒

2. **抛物线下落** (t=0-1.12s)
   - 重力加速度 17 像素/秒² 使其加速下落
   - 速度从 71.4 增加到 90.4

3. **第一次弹跳** (t=1.12s)
   - 撞击地面，反弹速度 = 30% × 原速度
   - 向上弹起，最大高度约为原来的 9%

4. **第二次弹跳** (t=4.30s)
   - 再次落地，反弹速度更小
   - 弹起高度进一步降低

5. **最终静止** (t≈5s)
   - 反弹速度 < 5 像素/秒，停止弹跳
   - 静止在地面上

## 相关文件

- `pkg/systems/particle_system.go` - 核心修复（角度转换 + 弹跳物理）
- `pkg/systems/particle_system_angle_test.go` - 测试验证
- `.meta/particles/ZombieHead.md` - 文档说明


