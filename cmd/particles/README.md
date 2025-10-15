# 粒子效果验证工具

用于测试和调试 PvZ 游戏粒子系统的独立工具。

## 功能特性

- 自动扫描并加载所有 106 个粒子效果
- 实时界面内搜索/筛选
- 灵活的导航和快速跳转
- 显示粒子和发射器实时统计
- 支持自动循环播放模式

## 使用方法

### 编译

```bash
go build -o bin/particles cmd/particles/main.go
```

### 运行

```bash
# 基本使用
./bin/particles

# 初始筛选（只显示包含 "Pea" 的效果）
./bin/particles --filter=Pea

# 从特定效果开始
./bin/particles --effect=PeaSplat

# 自动循环播放模式
./bin/particles --auto-play
```

## 键盘控制

### 导航
- **←/→**: 上一个/下一个效果
- **Page Up/Down**: 向前/向后跳转 10 个效果
- **Home/End**: 跳到第一个/最后一个效果
- **1-9**: 快速跳转到第 1-9 个效果（0 = 第 10 个）

### 操作
- **鼠标点击**: 在鼠标位置生成粒子效果
- **空格键**: 在屏幕中心生成粒子效果
- **R**: 清除所有活跃粒子和发射器
- **F 或 /**: 进入搜索模式
- **Q 或 Escape**: 退出程序

### 搜索模式（按 F 或 / 进入）
- **输入字母/数字**: 实时筛选效果名称
- **Backspace**: 删除最后一个字符
- **Enter 或 Escape**: 退出搜索模式

## 界面说明

```
Particle Viewer - Effect 3/15              [当前效果索引/总数]
Filter: "Pea" (15/106 effects)            [当前筛选条件和结果数]
Effect: PeaSplat                           [当前效果名称]
Active Particles: 128                      [活跃粒子数量]
Active Emitters: 2                         [活跃发射器数量]

SEARCH: pea_                               [搜索模式指示器]
(Type to filter, Backspace to delete...)   [搜索提示]
```

## 测试工作流

### 1. 快速浏览所有效果
```bash
# 启动自动循环播放
./bin/particles --auto-play
```

### 2. 测试特定类型效果
```bash
# 测试豌豆相关效果
./bin/particles --filter=Pea

# 进入后按 Space 或点击鼠标查看效果
```

### 3. 调试单个效果
```bash
# 直接跳到特定效果
./bin/particles --effect=PeaSplat

# 或启动后按 F 进入搜索模式，输入 "pea" 筛选
```

### 4. 对比多个效果
```bash
# 启动后按 ← → 快速切换
# 每次点击鼠标生成当前选中的效果
# 按 R 清除所有粒子后切换到下一个
```

## 常见用途

### QA 测试
- 逐个验证所有 106 个效果是否正常显示
- 检查粒子颜色、透明度、旋转等属性
- 验证发射器是否按预期生成粒子

### 性能测试
- 同时生成多个效果，观察粒子数量
- 监控活跃粒子和发射器数量
- 测试大量粒子时的帧率

### 视觉调试
- 对比原版 PvZ 效果与当前实现
- 调整粒子参数后快速验证效果
- 截图保存效果对比

## 故障排查

### 程序无法启动
```bash
# 检查是否在项目根目录
pwd  # 应该显示 .../pvz3

# 检查资源文件是否存在
ls assets/effect/particles/*.xml | wc -l  # 应该显示 106

# 检查配置文件
ls assets/config/resources.yaml
```

### 效果无法显示
- 检查终端日志中的错误信息
- 确认粒子图片资源已加载（日志中会显示）
- 尝试按 R 清除粒子后重新生成

### 搜索无结果
- 检查搜索关键词是否正确（区分大小写）
- 按 Escape 退出搜索模式，查看总效果数量
- 使用 `--filter` 命令行参数验证筛选逻辑

## 开发说明

### 代码结构
```
cmd/particles/main.go
├── ParticleViewerGame          # 主游戏循环
├── updateSearchMode()          # 搜索模式输入处理
├── updateNormalMode()          # 正常模式输入处理
├── filterEffects()             # 效果筛选逻辑
└── drawUI()                    # 界面渲染
```

### 扩展建议
- 添加效果列表侧边栏
- 支持效果参数实时调整
- 导出效果截图/录像功能
- 添加效果对比模式

## 相关文档

- [Story 7.4: 粒子效果实际应用](../../docs/stories/7.4.story.md)
- [CLAUDE.md - 粒子系统使用指南](../../CLAUDE.md#粒子系统使用指南)
