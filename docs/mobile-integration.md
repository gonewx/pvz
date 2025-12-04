# 移动端集成指南

本文档说明如何将植物大战僵尸游戏集成到 Android 和 iOS 原生应用中。

## 概述

游戏使用 [Ebitengine](https://ebitengine.org/) 开发，通过 `ebitenmobile` 工具可以将游戏打包为：

- **Android**: `.aar` (Android Archive)
- **iOS**: `.xcframework` (XCFramework)

这些包可以像普通的原生库一样集成到 Android Studio 或 Xcode 项目中。

## 环境要求

### 通用要求

- Go 1.21 或更高版本
- ebitenmobile 工具（通过 `make install-ebitenmobile` 安装）

### Android 构建

- Android SDK
- Android NDK（推荐 r21 或更高版本）
- 设置环境变量：
  ```bash
  export ANDROID_HOME=/path/to/android-sdk
  export ANDROID_NDK_HOME=$ANDROID_HOME/ndk/<version>
  ```

### iOS 构建

- macOS 操作系统（必需）
- Xcode 12 或更高版本
- Xcode Command Line Tools

## 构建移动端包

### 安装 ebitenmobile

```bash
make install-ebitenmobile
```

### 构建 Android AAR

```bash
make build-android
```

输出文件：`build/android/pvz.aar`

### 构建 iOS XCFramework

```bash
# 仅在 macOS 上可用
make build-ios
```

输出文件：`build/ios/PVZ.xcframework/`

### 构建所有移动端平台

```bash
make build-mobile
```

## Android 集成

### 1. 项目结构

创建一个新的 Android Studio 项目，推荐结构如下：

```
MyPvZGame/
├── app/
│   ├── src/main/
│   │   ├── java/com/example/mypvzgame/
│   │   │   └── MainActivity.java
│   │   ├── res/
│   │   │   └── layout/
│   │   │       └── activity_main.xml
│   │   └── AndroidManifest.xml
│   ├── libs/
│   │   └── pvz.aar          # 将构建产物复制到这里
│   └── build.gradle
├── settings.gradle
└── build.gradle
```

### 2. Gradle 配置

在 `app/build.gradle` 中添加依赖：

```gradle
android {
    defaultConfig {
        minSdk 23  // 与构建时的 ANDROID_API 一致
        // ...
    }
}

dependencies {
    implementation files('libs/pvz.aar')
    // 或者使用以下方式引用所有 aar 文件
    // implementation fileTree(dir: 'libs', include: ['*.aar'])
}
```

### 3. Activity 实现

创建 `MainActivity.java`：

```java
package com.example.mypvzgame;

import android.os.Bundle;
import androidx.appcompat.app.AppCompatActivity;
import go.Seq;
import com.decker.pvz.Pvz;  // 根据 -javapkg 参数确定

public class MainActivity extends AppCompatActivity {
    @Override
    protected void onCreate(Bundle savedInstanceState) {
        super.onCreate(savedInstanceState);

        // 初始化 Go 运行时
        Seq.setContext(getApplicationContext());

        // 创建游戏视图
        Pvz.EbitenView gameView = new Pvz.EbitenView(this);
        setContentView(gameView);
    }
}
```

### 4. AndroidManifest.xml

确保添加必要的权限和配置：

```xml
<?xml version="1.0" encoding="utf-8"?>
<manifest xmlns:android="http://schemas.android.com/apk/res/android"
    package="com.example.mypvzgame">

    <application
        android:allowBackup="true"
        android:label="@string/app_name"
        android:supportsRtl="true"
        android:theme="@style/Theme.AppCompat.NoActionBar">

        <activity
            android:name=".MainActivity"
            android:configChanges="orientation|screenSize|keyboardHidden"
            android:exported="true"
            android:screenOrientation="landscape">
            <intent-filter>
                <action android:name="android.intent.action.MAIN" />
                <category android:name="android.intent.category.LAUNCHER" />
            </intent-filter>
        </activity>

    </application>
</manifest>
```

## iOS 集成

### 1. 项目结构

创建一个新的 Xcode 项目，结构如下：

```
MyPvZGame/
├── MyPvZGame/
│   ├── AppDelegate.swift
│   ├── SceneDelegate.swift
│   ├── ViewController.swift
│   ├── Info.plist
│   └── Assets.xcassets/
├── Frameworks/
│   └── PVZ.xcframework/     # 将构建产物复制到这里
└── MyPvZGame.xcodeproj
```

### 2. 添加 XCFramework

1. 将 `build/ios/PVZ.xcframework` 复制到项目的 `Frameworks` 目录
2. 在 Xcode 中，选择项目 → Build Phases → Link Binary With Libraries
3. 点击 "+" 添加 `PVZ.xcframework`
4. 确保 Embed 设置为 "Embed & Sign"

### 3. ViewController 实现

修改 `ViewController.swift`：

```swift
import UIKit
import PVZ  // 导入游戏框架

class ViewController: UIViewController {
    override func viewDidLoad() {
        super.viewDidLoad()

        // 创建游戏视图
        let gameView = MobileBind.ebitenViewController()

        // 添加为子视图控制器
        addChild(gameView)
        view.addSubview(gameView.view)
        gameView.view.frame = view.bounds
        gameView.view.autoresizingMask = [.flexibleWidth, .flexibleHeight]
        gameView.didMove(toParent: self)
    }

    override var prefersStatusBarHidden: Bool {
        return true
    }

    override var supportedInterfaceOrientations: UIInterfaceOrientationMask {
        return .landscape
    }
}
```

### 4. Info.plist 配置

添加以下配置以支持全屏和横屏模式：

```xml
<key>UIRequiresFullScreen</key>
<true/>
<key>UISupportedInterfaceOrientations</key>
<array>
    <string>UIInterfaceOrientationLandscapeLeft</string>
    <string>UIInterfaceOrientationLandscapeRight</string>
</array>
<key>UISupportedInterfaceOrientations~ipad</key>
<array>
    <string>UIInterfaceOrientationLandscapeLeft</string>
    <string>UIInterfaceOrientationLandscapeRight</string>
</array>
```

## 常见问题

### Q: Android 构建失败，提示找不到 NDK

**A**: 确保设置了正确的环境变量：

```bash
export ANDROID_HOME=/path/to/android-sdk
export ANDROID_NDK_HOME=$ANDROID_HOME/ndk/25.2.9519653  # 使用实际版本号
```

### Q: iOS 构建失败，提示 "not on macOS"

**A**: iOS 构建必须在 macOS 上执行。如果您使用 Linux 或 Windows，只能构建 Android 版本。

### Q: 游戏启动后黑屏

**A**: 检查以下几点：
1. 确保 `.aar` 或 `.xcframework` 正确导入
2. 检查 Activity/ViewController 是否正确初始化游戏视图
3. 查看日志输出，确认资源加载是否成功

### Q: 如何调整游戏分辨率？

**A**: 游戏默认使用 800x600 的逻辑分辨率，会自动适应设备屏幕。如需修改，可以在 `pkg/app/app.go` 的 `Layout()` 方法中调整。

### Q: 如何启用调试日志？

**A**: 移动端默认禁用详细日志。如需启用，修改 `mobile/mobile.go` 中的配置：

```go
cfg := app.Config{
    Verbose: true,  // 改为 true
    // ...
}
```

## 构建配置

### 自定义 Android API 版本

编辑 `Makefile` 中的 `ANDROID_API` 变量：

```makefile
ANDROID_API := 23  # 修改为目标 API 版本
```

### 自定义 Java 包名

编辑 `Makefile` 中的 `JAVA_PKG` 变量：

```makefile
JAVA_PKG := com.yourcompany.pvz
```

## 参考资源

- [Ebitengine 官方文档](https://ebitengine.org/en/documents/mobile.html)
- [ebitenmobile 命令参考](https://pkg.go.dev/github.com/hajimehoshi/ebiten/v2/cmd/ebitenmobile)
- [Android AAR 文档](https://developer.android.com/studio/projects/android-library)
- [Apple XCFramework 文档](https://developer.apple.com/documentation/xcode/creating-a-multi-platform-binary-framework-bundle)
