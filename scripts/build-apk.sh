#!/bin/bash
# 构建 PVZ Android APK 的完整脚本
# 使用: ./scripts/build-apk.sh

set -e  # 遇到错误立即退出

# 配置变量
ANDROID_HOME="${ANDROID_HOME:-/home/decker/app/android/sdk}"
ANDROID_NDK_HOME="${ANDROID_NDK_HOME:-$ANDROID_HOME/ndk/27.2.12479018}"
BUILD_TOOLS_VERSION="36.1.0"
PLATFORM_VERSION="android-36"
APP_NAME="pvz"
PACKAGE_NAME="com.decker.pvz"
MIN_SDK=23
TARGET_SDK=36

# 颜色输出
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

echo_info() { echo -e "${GREEN}==>${NC} $1"; }
echo_warn() { echo -e "${YELLOW}警告:${NC} $1"; }
echo_error() { echo -e "${RED}错误:${NC} $1"; }

# 1. 检查依赖
echo_info "检查构建依赖..."

if [ ! -d "$ANDROID_HOME" ]; then
    echo_error "Android SDK 未找到: $ANDROID_HOME"
    echo "请设置 ANDROID_HOME 环境变量"
    exit 1
fi

if ! command -v java &> /dev/null; then
    echo_error "Java 未安装"
    echo "请安装 JDK: sudo apt install openjdk-17-jdk"
    exit 1
fi

echo_info "环境检查通过"
echo "  - ANDROID_HOME: $ANDROID_HOME"
echo "  - ANDROID_NDK_HOME: $ANDROID_NDK_HOME"
echo "  - Java: $(java -version 2>&1 | head -n 1)"

# 确保环境变量被 make 子进程继承
export ANDROID_HOME
export ANDROID_NDK_HOME

# 2. 构建 AAR 库
echo_info "步骤 1/4: 构建 AAR 库..."
ANDROID_HOME="$ANDROID_HOME" ANDROID_NDK_HOME="$ANDROID_NDK_HOME" make build-android

AAR_FILE="build/android/${APP_NAME}.aar"
if [ ! -f "$AAR_FILE" ]; then
    echo_error "AAR 文件未生成: $AAR_FILE"
    exit 1
fi

# 3. 创建 Android 项目结构
echo_info "步骤 2/4: 创建 Android 项目..."
ANDROID_PROJECT="build/android-project"
rm -rf "$ANDROID_PROJECT"
mkdir -p "$ANDROID_PROJECT"/{app/src/main/{java/${PACKAGE_NAME//.//},res/{values,drawable,mipmap-hdpi,mipmap-mdpi,mipmap-xhdpi,mipmap-xxhdpi,mipmap-xxxhdpi}},app/libs,gradle/wrapper}

# 4. 生成 AndroidManifest.xml
cat > "$ANDROID_PROJECT/app/src/main/AndroidManifest.xml" <<EOF
<?xml version="1.0" encoding="utf-8"?>
<manifest xmlns:android="http://schemas.android.com/apk/res/android"
    package="$PACKAGE_NAME"
    android:versionCode="1"
    android:versionName="1.0">

    <uses-sdk
        android:minSdkVersion="$MIN_SDK"
        android:targetSdkVersion="$TARGET_SDK" />

    <uses-permission android:name="android.permission.INTERNET" />
    <uses-permission android:name="android.permission.VIBRATE" />

    <application
        android:allowBackup="true"
        android:icon="@mipmap/ic_launcher"
        android:label="Plants vs. Zombies"
        android:theme="@android:style/Theme.NoTitleBar.Fullscreen">

        <activity
            android:name=".MainActivity"
            android:configChanges="orientation|keyboardHidden|screenSize"
            android:screenOrientation="landscape"
            android:exported="true">
            <intent-filter>
                <action android:name="android.intent.action.MAIN" />
                <category android:name="android.intent.category.LAUNCHER" />
            </intent-filter>
        </activity>
    </application>
</manifest>
EOF

# 5. 生成 MainActivity.java
cat > "$ANDROID_PROJECT/app/src/main/java/${PACKAGE_NAME//.//}/MainActivity.java" <<EOF
package $PACKAGE_NAME;

import android.app.Activity;
import android.os.Bundle;
import ${PACKAGE_NAME}.ebitenmobileview.Ebitenmobileview;

public class MainActivity extends Activity {
    @Override
    protected void onCreate(Bundle savedInstanceState) {
        super.onCreate(savedInstanceState);
        setContentView(Ebitenmobileview.createGameView(this));
    }
}
EOF

# 6. 生成 build.gradle
cat > "$ANDROID_PROJECT/app/build.gradle" <<EOF
plugins {
    id 'com.android.application'
}

android {
    compileSdkVersion $TARGET_SDK

    defaultConfig {
        applicationId "$PACKAGE_NAME"
        minSdkVersion $MIN_SDK
        targetSdkVersion $TARGET_SDK
        versionCode 1
        versionName "1.0"
    }

    buildTypes {
        release {
            minifyEnabled false
        }
    }

    compileOptions {
        sourceCompatibility JavaVersion.VERSION_1_8
        targetCompatibility JavaVersion.VERSION_1_8
    }
}

dependencies {
    implementation files('libs/${APP_NAME}.aar')
}
EOF

# 7. 生成根 build.gradle
cat > "$ANDROID_PROJECT/build.gradle" <<EOF
buildscript {
    repositories {
        google()
        mavenCentral()
    }
    dependencies {
        classpath 'com.android.tools.build:gradle:8.1.0'
    }
}

allprojects {
    repositories {
        google()
        mavenCentral()
    }
}
EOF

# 8. 生成 settings.gradle
cat > "$ANDROID_PROJECT/settings.gradle" <<EOF
rootProject.name = "PVZ"
include ':app'
EOF

# 9. 生成 gradle.properties
cat > "$ANDROID_PROJECT/gradle.properties" <<EOF
org.gradle.jvmargs=-Xmx2048m
android.useAndroidX=true
android.enableJetifier=true
EOF

# 10. 复制 AAR 文件
cp "$AAR_FILE" "$ANDROID_PROJECT/app/libs/"

# 11. 创建简单图标 (可选)
echo_info "步骤 3/4: 生成应用图标..."
# 这里使用 Android SDK 默认图标，你可以替换为自己的图标
for dpi in hdpi mdpi xhdpi xxhdpi xxxhdpi; do
    cp -f "$ANDROID_HOME/platforms/$PLATFORM_VERSION/data/res/drawable-$dpi/sym_def_app_icon.png" \
       "$ANDROID_PROJECT/app/src/main/res/mipmap-$dpi/ic_launcher.png" 2>/dev/null || true
done

# 12. 使用 Gradle Wrapper 构建 APK
echo_info "步骤 4/4: 构建 APK (这可能需要几分钟)..."

# 下载 Gradle Wrapper
cd "$ANDROID_PROJECT"
if ! command -v gradle &> /dev/null; then
    echo_warn "Gradle 未安装，下载 Gradle Wrapper..."
    # 使用代理下载（如果配置了）
    export http_proxy="${http_proxy:-http://127.0.0.1:2080}"
    export https_proxy="${https_proxy:-http://127.0.0.1:2080}"
    curl -L https://services.gradle.org/distributions/gradle-8.1-bin.zip -o gradle.zip || {
        echo_error "下载 Gradle 失败，请检查网络连接或代理设置"
        echo "提示: 如需使用代理，设置环境变量: export http_proxy=http://127.0.0.1:2080"
        exit 1
    }
    unzip -q gradle.zip
    GRADLE_CMD="./gradle-8.1/bin/gradle"
else
    GRADLE_CMD="gradle"
fi

# 生成 Gradle Wrapper
$GRADLE_CMD wrapper --gradle-version 8.1

# 构建 APK
./gradlew assembleRelease

cd - > /dev/null

# 13. 输出结果
APK_FILE="$ANDROID_PROJECT/app/build/outputs/apk/release/app-release-unsigned.apk"
if [ -f "$APK_FILE" ]; then
    # 复制到 build 目录
    cp "$APK_FILE" "build/${APP_NAME}-unsigned.apk"

    echo_info "✅ APK 构建成功!"
    echo ""
    echo "生成文件:"
    echo "  - 未签名 APK: build/${APP_NAME}-unsigned.apk"
    echo ""
    echo_warn "注意: 此 APK 未签名，仅供测试使用"
    echo ""
    echo "安装方法 (需要 adb):"
    echo "  adb install -r build/${APP_NAME}-unsigned.apk"
    echo ""
    echo "如需发布，请使用以下命令签名:"
    echo "  ./scripts/sign-apk.sh build/${APP_NAME}-unsigned.apk"
else
    echo_error "APK 构建失败"
    exit 1
fi
