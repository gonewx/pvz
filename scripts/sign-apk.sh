#!/bin/bash
# APK 签名脚本 - 用于生成可发布的签名 APK
# 使用: ./scripts/sign-apk.sh <unsigned-apk>

set -e

UNSIGNED_APK="$1"
ANDROID_HOME="${ANDROID_HOME:-/home/decker/app/android/sdk}"
BUILD_TOOLS="$ANDROID_HOME/build-tools/36.1.0"

# 颜色输出
GREEN='\033[0;32m'
RED='\033[0;31m'
NC='\033[0m'

echo_info() { echo -e "${GREEN}==>${NC} $1"; }
echo_error() { echo -e "${RED}错误:${NC} $1"; }

if [ -z "$UNSIGNED_APK" ]; then
    echo_error "用法: $0 <unsigned-apk>"
    exit 1
fi

if [ ! -f "$UNSIGNED_APK" ]; then
    echo_error "文件不存在: $UNSIGNED_APK"
    exit 1
fi

KEYSTORE="build/pvz-release.keystore"
APK_DIR=$(dirname "$UNSIGNED_APK")
ALIGNED_APK="${APK_DIR}/.aligned-temp.apk"
SIGNED_APK="${APK_DIR}/pvz.apk"

# 1. 生成密钥库（如果不存在）
if [ ! -f "$KEYSTORE" ]; then
    echo_info "生成新的密钥库..."
    keytool -genkeypair -v \
        -keystore "$KEYSTORE" \
        -alias pvz \
        -keyalg RSA \
        -keysize 2048 \
        -validity 10000 \
        -storepass android \
        -keypass android \
        -dname "CN=PVZ, OU=Game, O=Decker, L=Beijing, S=Beijing, C=CN"
fi

# 2. 对齐 APK
echo_info "对齐 APK..."
"$BUILD_TOOLS/zipalign" -v -p 4 "$UNSIGNED_APK" "$ALIGNED_APK"

# 3. 签名 APK
echo_info "签名 APK..."
"$BUILD_TOOLS/apksigner" sign \
    --ks "$KEYSTORE" \
    --ks-key-alias pvz \
    --ks-pass pass:android \
    --key-pass pass:android \
    --out "$SIGNED_APK" \
    "$ALIGNED_APK"

# 4. 验证签名
echo_info "验证签名..."
"$BUILD_TOOLS/apksigner" verify --verbose "$SIGNED_APK"

# 清理临时文件
rm -f "$ALIGNED_APK"

echo_info "✅ APK 签名完成: $SIGNED_APK"
echo ""
echo "安装方法:"
echo "  adb install -r $SIGNED_APK"
