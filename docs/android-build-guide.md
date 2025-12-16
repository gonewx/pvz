# Android APK 构建快速参考

## 📦 一键构建 APK

```bash
# 1. 安装 JDK（如果未安装）
sudo apt install openjdk-17-jdk

# 2. 设置环境变量
export ANDROID_HOME=/path/android/sdk
export ANDROID_NDK_HOME=$ANDROID_HOME/ndk/27.2.12479018

# 3. 构建 APK
make build-apk

# 4. 安装到设备
adb install -r build/pvz-unsigned.apk
```

## 📋 构建流程

| 步骤 | 命令 | 输出 |
|------|------|------|
| 1. 构建 AAR | `make build-android` | `build/android/pvz.aar` |
| 2. 构建 APK | `make build-apk` | `build/pvz-unsigned.apk` |
| 3. 签名 APK | `make sign-apk APK=build/pvz-unsigned.apk` | `build/pvz-unsigned-signed.apk` |

## 🔧 环境检查清单

- [ ] JDK 已安装 (`javac -version`)
- [ ] Android SDK 已设置 (`echo $ANDROID_HOME`)
- [ ] Android NDK 已安装 (`ls $ANDROID_NDK_HOME`)
- [ ] ebitenmobile 已安装 (`ebitenmobile version`)
- [ ] adb 可用 (`adb devices`)

## 🚨 常见错误

### 错误: `javac: command not found`
```bash
sudo apt install openjdk-17-jdk
```

### 错误: `Android SDK not found`
```bash
export ANDROID_HOME=/path/to/android-sdk
```

### 错误: `ebitenutil.NewImageFromFile undefined`
✅ 已修复！使用 `embedded.LoadImage()` 替代

## 📱 APK 类型

| 类型 | 用途 | 签名 | 安装方式 |
|------|------|------|----------|
| **unsigned** | 测试 | ❌ | adb install |
| **signed** | 发布 | ✅ | adb install / 商店 |

## 🎯 快速命令

```bash
# 完整构建（AAR + APK）
make build-apk

# 仅构建 AAR
make build-android

# 签名 APK
make sign-apk APK=build/pvz-unsigned.apk

# 清理构建
make clean

# 安装到设备
adb install -r build/pvz.apk

# 卸载应用
adb uninstall com.decker.pvz

# 查看日志
adb logcat | grep pvz
```

## 📐 APK 尺寸优化

| 优化方法 | 说明 | 预期减小 |
|---------|------|---------|
| ProGuard | 代码混淆压缩 | ~30% |
| 资源优化 | 移除未使用资源 | ~20% |
| Native 库优化 | 仅保留目标架构 | ~40% |

当前默认构建包含 4 种架构：`armeabi-v7a`, `arm64-v8a`, `x86`, `x86_64`

## 🔐 密钥管理

**测试密钥** (自动生成):
- 位置: `build/pvz-release.keystore`
- 密码: `android`
- 别名: `pvz`

**生产密钥** (需手动创建):
```bash
keytool -genkeypair -v \
    -keystore release.keystore \
    -alias pvz-release \
    -keyalg RSA \
    -keysize 2048 \
    -validity 10000
```

⚠️ **警告**: 妥善保管生产密钥，丢失后无法更新应用！
