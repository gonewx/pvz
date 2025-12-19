#!/bin/bash

# 备份存档
adb shell run-as com.decker.pvz cat /data/data/com.decker.pvz/saves/{username} > {username}

# 推送存档到本地
adb push {username} /data/local/tmp/

# 恢复存档
adb shell run-as com.decker.pvz cp /data/local/tmp/{username} /data/data/com.decker.pvz/saves/{username}

# 删除存档
adb shell run-as com.decker.pvz rm /data/data/com.decker.pvz/saves/{username}
adb shell run-as com.decker.pvz rm /data/data/com.decker.pvz/saves/{username}_battle