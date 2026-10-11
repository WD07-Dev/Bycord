#!/bin/bash

version_name="$(sed -n 's/^versionName=//p' version.properties)"
[[ -n "$version_name" ]]
go_ldflags="-X bycord/core.Version=$version_name"

mkdir -p dist/linux dist/windows dist/android platform/android/app/libs
rm -f dist/linux/bycord.zip dist/windows/bycord.exe dist/android/bycord.apk
go test ./core ./platform/desktop ./platform/android/bind ./platform/linux
go build -trimpath -ldflags "$go_ldflags" -o dist/linux/bycord ./platform/linux
cp icon.png dist/linux/bycord.png
cp platform/linux/bycord.desktop dist/linux/bycord.desktop
zip -q -FS -j dist/linux/bycord-linux.zip dist/linux/bycord dist/linux/bycord.png dist/linux/bycord.desktop
rm -f dist/linux/bycord dist/linux/bycord.png dist/linux/bycord.desktop
GOOS=windows GOARCH=amd64 go build -trimpath -ldflags "-H=windowsgui $go_ldflags" -o dist/windows/bycord-window.exe ./platform/windows

android_sdk="${ANDROID_SDK_ROOT:-${ANDROID_HOME:-}}"
if [[ -n "$android_sdk" ]] && [[ -f "$android_sdk/platforms/android-34/android.jar" ]] && command -v gradle >/dev/null 2>&1; then
    android_ndk="${ANDROID_NDK_HOME:-}"
    if [[ -z "$android_ndk" ]]; then
        for ndk_candidate in "$android_sdk"/ndk/*; do
            [[ -d "$ndk_candidate" ]] && android_ndk="$ndk_candidate"
        done
    fi
    [[ -n "$android_ndk" ]]
    ANDROID_HOME="$android_sdk" ANDROID_NDK_HOME="$android_ndk" go tool gomobile bind \
    -target=android -androidapi=29 -javapkg=com.ourgram.bycord.core -trimpath \
    -ldflags "$go_ldflags" \
    -o platform/android/app/libs/bycord-core.aar ./platform/android/bind
    gradle -p platform/android :app:assembleDebug
    cp platform/android/app/build/outputs/apk/debug/app-debug.apk dist/android/bycord-android.apk
fi