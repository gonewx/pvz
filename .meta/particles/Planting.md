# Planting 粒子效果分析与修复

## 问题描述

**原版效果**: 植物种植时，土粒溅起的效果，只发射 **8 个粒子** 然后停止

**Bug 现象**: 我们的实现中粒子数量远超 8 个，持续不断生成

## 根本原因

### Planting.xml 配置
```xml
<SpawnMinActive>8</SpawnMinActive>      <!-- 活跃粒子数量 = 8 -->
<SystemDuration>30</SystemDuration>     <!-- 系统持续 30 厘秒 = 0.3 秒 -->
<ParticleDuration>30</ParticleDuration> <!-- 粒子寿命 30 厘秒 = 0.3 秒 -->
<!-- 注意：没有配置 SpawnRate 和 SpawnMaxLaunched -->
```

### 旧代码逻辑（错误）

```go
if spawnRate == 0 {
    // 持续补充模式：如果活跃粒子数 < SpawnMinActive，就补充到目标数量
    for activeCount < spawnMinActive {
        ps.spawnParticle(...)  // 每帧都补充到 8 个
    }
}
```

**问题**: 粒子消失后，每一帧都会补充到 8 个，导致总粒子数远超 8 个

## 修复方案

### 新逻辑：区分两种模式

```go
if spawnRate == 0 {
    // 确定最大发射数量
    effectiveMaxLaunched := spawnMaxLaunched
    if effectiveMaxLaunched == 0 {
        // 未配置 SpawnMaxLaunched：默认等于 SpawnMinActive（一次性发射模式）
        effectiveMaxLaunched = spawnMinActive  // ← 关键修复
    }

    // 限制总发射数量
    for activeCount < spawnMinActive && emitter.TotalLaunched < effectiveMaxLaunched {
        ps.spawnParticle(...)
        emitter.TotalLaunched++
    }
}
```

### 语义说明

| 配置组合 | 语义 | 示例 |
|---------|------|------|
| `SpawnRate=0`, `SpawnMaxLaunched=0` | **一次性发射** SpawnMinActive 个粒子 | Planting (8个土粒) |
| `SpawnRate=0`, `SpawnMaxLaunched>0` | **持续补充** 到 SpawnMinActive 个活跃粒子 | Award (持续粒子流) |
| `SpawnRate>0` | **按时间间隔** 持续生成 | 其他大部分效果 |

## 预期效果

修复后，Planting 效果应该是：
1. ✅ 在 t=0 时刻一次性发射 8 个粒子
2. ✅ 每个粒子存活 0.3 秒后消失
3. ✅ 发射器在 0.3 秒后自动销毁
4. ✅ 总粒子数始终不超过 8 个

## 验证命令

```bash
go run cmd/particles/main.go --verbose --effect="Planting" > /tmp/planting.log 2>&1
```

检查日志中：
- `TotalLaunched` 应该最多到 8
- 粒子总数应该在 0-8 之间变化

## 相关文件

- **修复位置**: `pkg/systems/particle_system.go:210-240`
- **配置文件**: `data/particles/Planting.xml`
- **测试命令**: `cmd/particles/main.go`

## 设计原则

遵循 **原版游戏的配置语义**：
- `SpawnMaxLaunched=0` 不是"无限制"，而是"使用默认值（等于 SpawnMinActive）"
- 这符合原版游戏的设计哲学：简化配置，合理默认值
