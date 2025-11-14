#!/bin/bash
# 测试 Animation Showcase 单个模式修复
#
# 问题: Story 16.3 重构后，单个模式时动画没有显示在虚拟显示区域内
# 原因: Render 方法期望中心坐标，但调用时传入了左上角坐标
# 修复: 修改调用处，传入中心坐标而非左上角坐标

echo "=== 编译 Animation Showcase ==="
go build -o /tmp/animation_showcase \
    cmd/animation_showcase/main.go \
    cmd/animation_showcase/config.go \
    cmd/animation_showcase/grid_layout.go \
    cmd/animation_showcase/animation_cell.go

if [ $? -ne 0 ]; then
    echo "❌ 编译失败"
    exit 1
fi

echo "✅ 编译成功"
echo ""
echo "=== 运行 Animation Showcase ==="
echo "用法: /tmp/animation_showcase --config=cmd/animation_showcase/config.yaml --verbose"
echo ""
echo "验证步骤:"
echo "1. 运行程序后，在网格模式下点击任意单元格"
echo "2. 按 Enter 键切换到单个模式"
echo "3. 确认动画显示在虚拟显示区域（800x600 灰色边框）的中心"
echo "4. 按 Enter 返回网格模式，确认动画仍然在单元格中心"
echo ""
echo "预期结果:"
echo "✅ 单个模式：动画显示在虚拟显示区域（800x600）的中心"
echo "✅ 网格模式：动画显示在单元格的中心"
