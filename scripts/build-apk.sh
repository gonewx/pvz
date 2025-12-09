#!/bin/bash
# 构建 PVZ Android APK 的完整脚本
# 使用: ./scripts/build-apk.sh

set -e  # 遇到错误立即退出

# 获取项目根目录的绝对路径
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

# 配置变量
ANDROID_HOME="${ANDROID_HOME:-${HOME}/Android/Sdk}"
ANDROID_NDK_HOME="${ANDROID_NDK_HOME:-$ANDROID_HOME/ndk/27.2.12479018}"
BUILD_TOOLS_VERSION="34.0.0"
PLATFORM_VERSION="android-34"
APP_NAME="pvz"
PACKAGE_NAME="com.decker.pvz"
MIN_SDK=23
TARGET_SDK=34

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
cd "$PROJECT_ROOT"
ANDROID_HOME="$ANDROID_HOME" ANDROID_NDK_HOME="$ANDROID_NDK_HOME" make build-android

AAR_FILE="$PROJECT_ROOT/build/android/${APP_NAME}.aar"
if [ ! -f "$AAR_FILE" ]; then
    echo_error "AAR 文件未生成: $AAR_FILE"
    exit 1
fi

# 3. 创建 Android 项目结构
echo_info "步骤 2/4: 创建 Android 项目..."
ANDROID_PROJECT="$PROJECT_ROOT/build/android-project"
rm -rf "$ANDROID_PROJECT"
mkdir -p "$ANDROID_PROJECT/app/src/main/java/${PACKAGE_NAME//.//}"
mkdir -p "$ANDROID_PROJECT/app/src/main/res"
mkdir -p "$ANDROID_PROJECT/app/src/main/res/values"
mkdir -p "$ANDROID_PROJECT/app/src/main/res/drawable"
for dpi in hdpi mdpi xhdpi xxhdpi xxxhdpi; do
    mkdir -p "$ANDROID_PROJECT/app/src/main/res/mipmap-$dpi"
done
mkdir -p "$ANDROID_PROJECT/app/libs"
mkdir -p "$ANDROID_PROJECT/gradle/wrapper"

# 4. 生成 AndroidManifest.xml
cat > "$ANDROID_PROJECT/app/src/main/AndroidManifest.xml" <<EOF
<?xml version="1.0" encoding="utf-8"?>
<manifest xmlns:android="http://schemas.android.com/apk/res/android"
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
import android.view.ViewGroup;
import android.view.WindowManager;
import android.widget.FrameLayout;
import ${PACKAGE_NAME}.mobile.EbitenView;
import go.Seq;

public class MainActivity extends Activity {
    private EbitenView view;

    @Override
    protected void onCreate(Bundle savedInstanceState) {
        super.onCreate(savedInstanceState);

        // Initialize Go context - MUST be called before any Go code runs
        Seq.setContext(getApplicationContext());

        // Keep screen on
        getWindow().addFlags(WindowManager.LayoutParams.FLAG_KEEP_SCREEN_ON);

        view = new EbitenView(this);

        // Explicitly set layout params to MATCH_PARENT
        FrameLayout.LayoutParams params = new FrameLayout.LayoutParams(
            ViewGroup.LayoutParams.MATCH_PARENT,
            ViewGroup.LayoutParams.MATCH_PARENT
        );
        view.setLayoutParams(params);

        setContentView(view);
    }

    @Override
    protected void onPause() {
        super.onPause();
        view.suspendGame();
    }

    @Override
    protected void onResume() {
        super.onResume();
        view.resumeGame();
    }
}
EOF

# 6. 生成 build.gradle
cat > "$ANDROID_PROJECT/app/build.gradle" <<EOF
plugins {
    id 'com.android.application'
}

android {
    namespace "$PACKAGE_NAME"
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
        maven { url 'https://maven.aliyun.com/repository/google' }
        maven { url 'https://maven.aliyun.com/repository/public' }
        google()
        mavenCentral()
    }
    dependencies {
        classpath 'com.android.tools.build:gradle:8.1.0'
    }
}

allprojects {
    repositories {
        maven { url 'https://maven.aliyun.com/repository/google' }
        maven { url 'https://maven.aliyun.com/repository/public' }
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

# 11. 复制应用图标
echo_info "步骤 3/4: 复制应用图标..."
ICON_DIR="$PROJECT_ROOT/assets/icons/android/res"

if [ -d "$ICON_DIR" ]; then
    echo_info "使用项目图标: $ICON_DIR"
    for dpi in hdpi mdpi xhdpi xxhdpi xxxhdpi; do
        SRC_ICON="$ICON_DIR/mipmap-$dpi/ic_launcher.png"
        if [ -f "$SRC_ICON" ]; then
            cp -f "$SRC_ICON" "$ANDROID_PROJECT/app/src/main/res/mipmap-$dpi/ic_launcher.png"
        fi
    done
else
    # 兜底：使用旧图标路径
    echo_warn "新图标目录未找到，尝试使用旧图标..."
    ICON_SOURCE="$PROJECT_ROOT/assets/images/Store_PvZIcon.png"
    if [ -f "$ICON_SOURCE" ]; then
        for dpi in hdpi mdpi xhdpi xxhdpi xxxhdpi; do
            cp -f "$ICON_SOURCE" "$ANDROID_PROJECT/app/src/main/res/mipmap-$dpi/ic_launcher.png"
        done
    fi
fi

# 验证图标是否存在
if [ ! -f "$ANDROID_PROJECT/app/src/main/res/mipmap-hdpi/ic_launcher.png" ]; then
    echo_error "图标复制失败，构建可能会失败"
fi

# 12. 使用 Gradle Wrapper 构建 APK
echo_info "步骤 4/4: 构建 APK (这可能需要几分钟)..."

# Gradle 缓存目录（使用绝对路径）
GRADLE_CACHE_DIR="$PROJECT_ROOT/build/gradle-cache"
GRADLE_VERSION="8.5"
GRADLE_DIR="$GRADLE_CACHE_DIR/gradle-$GRADLE_VERSION"

# 优先使用本地缓存的 Gradle 或下载指定版本
# 避免使用系统 Gradle（可能版本不兼容）
if [ -d "$GRADLE_DIR" ] && [ -f "$GRADLE_DIR/bin/gradle" ]; then
    echo_info "使用缓存的 Gradle $GRADLE_VERSION"
    GRADLE_CMD="$GRADLE_DIR/bin/gradle"
else
    echo_info "下载 Gradle $GRADLE_VERSION 到缓存目录..."
    mkdir -p "$GRADLE_CACHE_DIR"

    GRADLE_ZIP="$GRADLE_CACHE_DIR/gradle-$GRADLE_VERSION-bin.zip"
    curl -L "https://services.gradle.org/distributions/gradle-$GRADLE_VERSION-bin.zip" -o "$GRADLE_ZIP" || {
        echo_error "下载 Gradle 失败，请检查网络连接或代理设置"
        echo "提示: 如需使用代理，设置环境变量: export http_proxy=http://127.0.0.1:2080"
        exit 1
    }

    echo_info "解压 Gradle 到缓存目录..."
    unzip -q "$GRADLE_ZIP" -d "$GRADLE_CACHE_DIR"
    rm -f "$GRADLE_ZIP"  # 删除 zip 文件，节省空间

    GRADLE_CMD="$GRADLE_DIR/bin/gradle"
    echo_info "Gradle $GRADLE_VERSION 已缓存到: $GRADLE_DIR"
fi

# 生成 Gradle Wrapper
cd "$ANDROID_PROJECT"

# 隔离 Gradle 环境
unset GRADLE_HOME
export GRADLE_USER_HOME="$PROJECT_ROOT/build/gradle-user-home"
mkdir -p "$GRADLE_USER_HOME"

# 解析并配置代理
if [ -n "$http_proxy" ]; then
    PROXY_HOST=$(echo $http_proxy | sed -E 's|http://||; s|:| |g' | awk '{print $1}')
    PROXY_PORT=$(echo $http_proxy | sed -E 's|http://||; s|:| |g' | awk '{print $2}')
    GRADLE_PROXY_ARGS="-Dhttp.proxyHost=$PROXY_HOST -Dhttp.proxyPort=$PROXY_PORT -Dhttps.proxyHost=$PROXY_HOST -Dhttps.proxyPort=$PROXY_PORT"
    echo_info "应用代理设置: host=$PROXY_HOST port=$PROXY_PORT"
else
    GRADLE_PROXY_ARGS=""
fi

echo_info "生成 Gradle Wrapper..."
"$GRADLE_CMD" wrapper --gradle-version $GRADLE_VERSION --no-daemon $GRADLE_PROXY_ARGS

# 构建 APK
echo_info "执行 Gradle 构建..."
./gradlew assembleRelease --no-daemon --stacktrace

cd - > /dev/null

# 13. 输出结果
APK_FILE="$ANDROID_PROJECT/app/build/outputs/apk/release/app-release-unsigned.apk"
OUTPUT_APK="$PROJECT_ROOT/build/${APP_NAME}-unsigned.apk"
if [ -f "$APK_FILE" ]; then
    # 复制到 build 目录
    cp "$APK_FILE" "$OUTPUT_APK"

    echo_info "✅ APK 构建成功!"
    echo ""
    echo "生成文件:"
    echo "  - 未签名 APK: $OUTPUT_APK"
    echo ""
    echo_warn "注意: 此 APK 未签名，仅供测试使用"
    echo ""
    echo "安装方法 (需要 adb):"
    echo "  adb install -r $OUTPUT_APK"
    echo ""
    echo "如需发布，请使用以下命令签名:"
    echo "  ./scripts/sign-apk.sh $OUTPUT_APK"
else
    echo_error "APK 构建失败"
    exit 1
fi
