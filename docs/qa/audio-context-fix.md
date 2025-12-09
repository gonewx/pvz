# 音频上下文重复创建问题修复

## 问题描述

运行 `go run main.go --verbose` 时出现 panic：

```
panic: audio: context is already created

goroutine 1 [running, locked to thread]:
github.com/hajimehoshi/ebiten/v2/audio.NewContext(0xbb80)
    /home/decker/go/pkg/mod/github.com/hajimehoshi/ebiten/v2@v2.9.0/audio/audio.go:90 +0x305
main.main()
    /mnt/disk0/project/game/pvz/pvz/main.go:50 +0x97
exit status 2
```

## 根本原因

在 `pkg/systems/test_helpers.go` 中，定义了一个全局变量：

```go
var testAudioContext = audio.NewContext(48000)
```

这个全局变量在**包初始化时**就会创建音频上下文。当 main.go 导入 `pkg/systems` 包（间接导入，通过其他包）时，这个全局变量会在 `main()` 函数执行之前就初始化。然后 `main.go` 的第 50 行再次创建音频上下文，导致冲突。

### Ebitengine 音频上下文限制

Ebitengine 的音频系统要求**全局只能创建一个音频上下文**。这是因为：
- 音频上下文管理系统级音频资源（音频设备、混音器等）
- 多个上下文会导致资源冲突
- 底层音频库（Oto）不支持多实例

## 解决方案

**采用延迟初始化（Lazy Initialization）模式**：

### Before（错误）

```go
// pkg/systems/test_helpers.go
var testAudioContext = audio.NewContext(48000)  // ❌ 包初始化时立即创建
```

### After（正确）

```go
// pkg/systems/test_helpers.go
var (
	testAudioContext     *audio.Context
	testAudioContextOnce sync.Once
)

// getTestAudioContext 获取测试音频上下文（延迟创建）
func getTestAudioContext() *audio.Context {
	testAudioContextOnce.Do(func() {
		testAudioContext = audio.NewContext(48000)
	})
	return testAudioContext
}
```

### 关键改进

1. **延迟初始化**：音频上下文不在包初始化时创建，而是在首次调用 `getTestAudioContext()` 时创建
2. **线程安全**：使用 `sync.Once` 确保只创建一次（即使多个测试并发调用）
3. **按需创建**：只有在运行测试时才会创建上下文，不影响 main.go

## 变更文件

### 1. pkg/systems/test_helpers.go

**变更内容**：
- 将全局变量 `testAudioContext` 改为延迟初始化
- 添加 `getTestAudioContext()` 函数
- 导入 `sync` 包

### 2. 所有测试文件（批量更新）

**变更文件**：
- `behavior_system_test.go`
- `input_system_test.go`
- `physics_system_test.go`
- `particle_system_test.go`
- `sun_spawn_system_test.go`

**变更内容**：
将所有 `testAudioContext` 替换为 `getTestAudioContext()`

**示例**：
```go
// Before
rm := game.NewResourceManager(testAudioContext)

// After
rm := game.NewResourceManager(getTestAudioContext())
```

## 测试验证

```bash
# 编译主程序
$ go build -o /tmp/pvz_final
✅ 编译成功

# 编译测试
$ go test ./pkg/systems -c -o /dev/null
✅ 测试编译成功

# 运行单个测试
$ go test ./pkg/systems -run TestSunClickStateChange -v
✅ PASS

# 运行所有测试
$ go test ./pkg/systems
✅ PASS
```

## 技术细节

### sync.Once 工作原理

```go
testAudioContextOnce.Do(func() {
    testAudioContext = audio.NewContext(48000)
})
```

`sync.Once.Do()` 保证：
- 回调函数只会执行**一次**
- 线程安全（即使多个 goroutine 同时调用）
- 后续调用会立即返回，不会重复执行

### 为什么不在 init() 函数中创建？

```go
// ❌ 仍然会失败
func init() {
    testAudioContext = audio.NewContext(48000)
}
```

`init()` 函数在包初始化时自动执行，与全局变量初始化一样，会在 main.main() 之前运行，仍然会导致冲突。

## 最佳实践

### 单例模式（Singleton Pattern）

这个修复实现了经典的单例模式：
1. **私有构造**：不暴露直接创建方式
2. **延迟初始化**：按需创建
3. **线程安全**：使用 sync.Once
4. **全局访问点**：通过 getTestAudioContext() 函数

### 适用场景

这种模式适用于：
- 系统级资源（音频设备、数据库连接）
- 全局配置对象
- 日志系统
- 资源管理器

## 兼容性

**向后兼容**：
- 测试代码需要批量更新（已完成）
- 主程序代码无需修改
- 不影响游戏功能

**Breaking Changes**：
- 测试代码必须使用 `getTestAudioContext()` 而不是 `testAudioContext`

## 后续工作

无，问题已完全修复。

## 参考资料

- Ebitengine 音频文档: https://pkg.go.dev/github.com/hajimehoshi/ebiten/v2/audio
- Go sync.Once 文档: https://pkg.go.dev/sync#Once
- 单例模式: https://refactoring.guru/design-patterns/singleton/go/example
