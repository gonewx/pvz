#!/bin/bash
# ECS 架构合规性检查脚本
# 检测系统间的直接调用（违反零耦合原则）

echo "🔍 检查 ECS 架构合规性..."

# 检查是否有系统直接调用其他系统的方法
violations=0

# 检查 ReanimSystem 的违规调用
echo "检查 ReanimSystem 违规调用..."
grep -rn "\.reanimSystem\." pkg/systems/*.go 2>/dev/null | grep -v "_test.go" | grep -v "reanim_system.go" > /tmp/violations.txt

if [ -s /tmp/violations.txt ]; then
    echo "❌ 发现 ReanimSystem 违规调用:"
    cat /tmp/violations.txt
    violations=$((violations + $(wc -l < /tmp/violations.txt)))
else
    echo "✅ 未发现 ReanimSystem 违规调用"
fi

# 检查其他系统间调用（可扩展）
echo ""
echo "检查其他系统间违规调用模式..."

# 检查系统构造函数是否接收其他系统作为参数（除了允许的依赖）
# 允许的依赖: EntityManager, GameState, ResourceManager
echo "检查系统构造函数依赖..."
suspicious_deps=$(grep -rn "func New.*System.*\*.*System" pkg/systems/*.go | \
    grep -v "_test.go" | \
    grep -v "EntityManager" | \
    grep -v "GameState" | \
    grep -v "ResourceManager" | \
    grep -v "CameraSystem" | \
    grep -v "LawnGridSystem" | \
    wc -l)

if [ "$suspicious_deps" -gt 0 ]; then
    echo "⚠️  发现 $suspicious_deps 个可疑的系统依赖:"
    grep -rn "func New.*System.*\*.*System" pkg/systems/*.go | \
        grep -v "_test.go" | \
        grep -v "EntityManager" | \
        grep -v "GameState" | \
        grep -v "ResourceManager" | \
        grep -v "CameraSystem" | \
        grep -v "LawnGridSystem"
    violations=$((violations + suspicious_deps))
else
    echo "✅ 系统构造函数依赖正常"
fi

# 总结
echo ""
echo "================================================"
if [ $violations -eq 0 ]; then
    echo "✅ 架构合规性检查通过！"
    echo "   - 未发现系统间直接调用"
    echo "   - 符合 ECS 零耦合原则"
    exit 0
else
    echo "❌ 发现 $violations 处违规，请修复"
    echo "   - 系统间不应直接调用"
    echo "   - 请使用组件通信（如 AnimationCommandComponent）"
    echo "   - 详见: docs/architecture/coding-standards.md"
    exit 1
fi
