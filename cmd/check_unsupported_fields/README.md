# 粒子系统字段检查工具

## 功能说明

这个工具用于检查 `assets/effect/particles/` 目录下所有粒子配置文件（XML），找出我们的粒子系统不支持的字段。

## 使用方法

### 1. 编译并运行

```bash
# 在项目根目录下运行
go run cmd/check_unsupported_fields/main.go
```

### 2. 或者编译后运行

```bash
# 编译
go build -o bin/check_unsupported_fields cmd/check_unsupported_fields/main.go

# 运行
./bin/check_unsupported_fields
```

## 输出说明

### 成功情况

如果所有字段都已支持，输出：
```
✅ 所有粒子配置文件中的字段都已支持！
```

### 发现不支持字段

如果发现不支持的字段，输出示例：
```
❌ 发现 3 个不支持的字段:

字段: TrackType
  使用次数: 5
  出现文件:
    - Award.xml
    - Splash.xml
    - ZombieHead.xml

字段: ParticleFlags
  使用次数: 2
  出现文件:
    - Explosion.xml
    - Fire.xml

=== 汇总 ===
不支持的字段列表 (按使用频率排序):
1. TrackType (使用 5 次)
2. ParticleFlags (使用 2 次)
```

## 支持的字段列表

当前粒子系统支持的字段（基于 `internal/particle/types.go`）：

### Emitter 基本属性
- `Name` - 发射器名称

### Spawn 属性（控制粒子发射）
- `SpawnMinActive` - 最小活跃粒子数
- `SpawnMaxActive` - 最大活跃粒子数
- `SpawnMaxLaunched` - 最大发射粒子总数
- `SpawnRate` - 每秒发射粒子数

### Particle 属性（粒子视觉属性）
- `ParticleDuration` - 生命周期（毫秒）
- `ParticleAlpha` - 透明度（0-1）
- `ParticleScale` - 缩放倍数
- `ParticleSpinAngle` - 初始旋转角度
- `ParticleSpinSpeed` - 旋转速度（度/秒）
- `ParticleRed` - 红色通道（0-1）
- `ParticleGreen` - 绿色通道（0-1）
- `ParticleBlue` - 蓝色通道（0-1）
- `ParticleBrightness` - 亮度乘数
- `ParticleLoops` - 动画循环次数
- `ParticleStretch` - 拉伸效果
- `ParticlesDontFollow` - 不跟随发射器移动

### Launch 属性（发射参数）
- `LaunchSpeed` - 初始速度
- `LaunchAngle` - 发射方向（度）
- `AlignLaunchSpin` - 旋转对齐发射方向
- `RandomLaunchSpin` - 随机初始旋转
- `RandomStartTime` - 随机动画起始时间

### Emitter 属性（发射器配置）
- `EmitterBoxX` - 发射区域宽度（水平）
- `EmitterBoxY` - 发射区域高度（垂直）
- `EmitterRadius` - 发射半径（圆形发射器）
- `EmitterType` - 发射器形状类型
- `EmitterSkewX` - 水平倾斜
- `EmitterOffsetX` - 水平偏移
- `EmitterOffsetY` - 垂直偏移

### System 属性（系统级设置）
- `SystemDuration` - 总效果持续时间（毫秒）
- `SystemAlpha` - 系统级透明度修改器
- `SystemLoops` - 系统重复次数
- `SystemField` - 系统级力场效果

### Image 属性（贴图配置）
- `Image` - 粒子纹理资源 ID
- `ImageFrames` - 动画帧数
- `ImageRow` - 精灵图行
- `ImageCol` - 精灵图列
- `Animated` - 启用帧动画（0 或 1）
- `AnimationRate` - 每秒帧数

### Rendering 属性（渲染模式）
- `Additive` - 加法混合（0 或 1）
- `FullScreen` - 全屏效果（0 或 1）
- `HardwareOnly` - 需要硬件加速
- `ClipTop` - 顶部裁剪

### 其他属性
- `CrossFadeDuration` - 过渡淡入淡出时间
- `DieIfOverloaded` - 系统过载时销毁效果
- `CollisionReflect` - 反弹系数
- `CollisionSpin` - 碰撞旋转

### Field 子标签（力场）
- `Field` - 力场容器
- `FieldType` - 力场类型
- `X` - 水平力分量
- `Y` - 垂直力分量

## 如何添加新字段支持

如果发现不支持的字段需要添加：

1. 在 `internal/particle/types.go` 的 `EmitterConfig` 结构体中添加字段定义
2. 在 `cmd/check_unsupported_fields/main.go` 的 `supportedFields` map 中添加字段名
3. 在 `pkg/entities/particle_factory.go` 中实现字段的解析逻辑
4. 在 `pkg/systems/particle_system.go` 中实现字段的运行时行为
5. 重新运行此工具验证

## 相关文档

- Story 7.2: `docs/stories/7.2.story.md` - 粒子系统核心实现
- Story 7.3: `docs/stories/7.3.story.md` - 粒子渲染系统
- 粒子测试指南: `docs/particle-testing-guide.md`




