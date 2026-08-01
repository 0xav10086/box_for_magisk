# PROJECT NOTES / 项目进度与目标

> 本文件记录本项目当前的目标、已完成的工作、以及已理解的实现细节。
> 用于跨设备（VPS / 本地电脑 / Android 手机）接续工作时快速恢复上下文。

## 项目背景

`box_for_magisk` 是 **Box for Root (BFR)** v1.10.2 的 fork
（上游：[taamarin/box_for_magisk](https://github.com/taamarin/box_for_magisk)）。
它是一个 Android 透明代理 Magisk 模块，支持 Magisk / KernelSU / APatch 三种 root 框架。

### 原始三步走计划
1. **分析** `~/box_for_magisk/` 中 clash 相关文件与代码 —— 已完成
2. **创建** `mihomo_for_kernelSU/` 项目，把 mihomo 移植到拥有 root 环境的 Android 手机上 —— 进行中
3. **测试** 新项目 —— 未开始

### 当前实际方向
- 测试设备：OP60EBL1（Android，已 root，装有 KernelSU/Magisk）
- 在设备上通过 box_for_magisk 运行 mihomo
- 测试节点配置：`flow_collect_server/templates/flashfox_nodes.yaml`（含 62 个 vless 节点）

## 已理解 / 已实现

### 1. box_for_magisk 架构（BFR v1.10.2）
- 透明代理原理：iptables 把设备流量重定向到本地代理内核
- 多内核：clash(mihomo) / sing-box / xray / v2fly / hysteria（`settings.ini` 的 `bin_name` 切换，默认 clash）
- 多框架：Magisk / KernelSU / APatch
- 5 种网络模式：tproxy / redirect / mixed / enhance / tun
- 核心脚本（`box/scripts/`）：
  - `box.service` (923 行)：start/stop/restart、进程管理、cgroup
  - `box.iptables` (891 行)：iptables 规则、应用 UID/GID/网卡黑白名单
  - `box.tool` (1256 行)：`upkernel` 下载内核、`upsubs` 更新订阅、`geox` 更新 GeoIP
  - `start.sh`：开机入口，等数据分区就绪后启动服务 + 启用 iptables
  - inotify 系列：WiFi 切换 / 路由表变化 / 模块 enable-disable 监控

### 2. `box.tool upkernel` 机制（clash/mihomo 分支）
- 流程：备份旧内核 → 按架构定 `arch` → 探测版本 → 拼 URL 下载 → gunzip → 装到
  `bin/xclash/mihomo` → 软链 `bin/clash` → 重启服务
- 版本来源：`mihomo_stable="enable"` 时从 GitHub `releases` API 探测最新**稳定版**
  （实测返回 `v1.19.29`，与 `/releases/latest` 一致，探测逻辑正确）
- **不自动更新**：upkernel 需手动执行；crontab 默认关闭（`run_crontab=false`），
  且默认定时任务不含 upkernel
- 下载失败会静默回滚到 `.bak`（`upfile()`），不会破坏旧内核

### 3. 设备 TLS 问题（已解决的工作流）
- 现象：设备 curl 报 `curl: (35) TLS connect error ... OPENSSL_internal:invalid library`，
  无法连接 ghfast.top，导致 `box.tool upkernel` 失败
- 原因：设备 curl 链接的 OpenSSL/BoringSSL 版本过旧
- 解决流程：**本地电脑下载 → adb push → 手机安装**
  1. 本地下载：`curl -L -o mihomo.gz "https://github.com/MetaCubeX/mihomo/releases/download/v1.19.29/mihomo-android-arm64-v8-v1.19.29.gz"`
  2. 本地 `gunzip mihomo.gz`
  3. `adb push mihomo /sdcard/mihomo`
  4. `adb shell su -c "cp /sdcard/mihomo /data/adb/box/bin/xclash/mihomo"`
     - 注意：**先停服务再 cp**，否则报 `Text file busy`
     - 停服务：`adb shell su -c "/data/adb/box/scripts/box.service stop"`
  5. 设权限/软链：`chown root:net_admin`、`chmod 6755`、`ln -sf` 到 `bin/clash`
  6. `box.service start` 重启，`box/bin/clash -v` 验证版本
  7. 清理 `/sdcard/mihomo`

### 4. 配置简化（flashfox_nodes.yaml）
- 节点名去掉国旗 emoji 和中文，只留缩写+序号：`🇭🇰 香港 HK-01` → `HK-01`（全 62 节点）
- 分组名按 clash 惯例改英文：`🚀 节点选择` → `PROXY`、`自动选择` → `AUTO`、`故障转移` → `FAILOVER`
- `proxies` 的 `name:` 与 `proxy-groups` 引用、rules 三处同步修改，配置保持自洽

### 5. Android 上测单个节点的命令（不用 curl 直连 vless）
- curl 不支持 vless 协议，需先用独立 mihomo 暴露本地端口
- 方法：写一个只含该节点的最小 mihomo 配置 → 用 `/data/adb/box/bin/clash -d /data/local/tmp -f 配置.yaml`
  独立启动（端口错开，如 7891，避免与 box 服务冲突）→ `curl -x http://127.0.0.1:7891 https://api.ipify.org`
- 测完 `killall clash`

## 待办 / 下一步

- [ ] 创建 `mihomo_for_kernelSU/`：精简 fork box_for_magisk，只保留 clash/mihomo + KernelSU
      （参考设计决策：形态、功能范围、mihomo 二进制来源）
- [ ] 设计可靠的 mihomo 自动更新机制（启动时对比版本 / 定时任务 / 多镜像 / 离线更新入口）
      —— 解决设备 TLS 旧导致 upkernel 失败的问题
- [ ] 全节点批量测试：从 flashfox_nodes.yaml 自动生成各节点最小配置，批量测出口 IP/延迟并汇总

## 关键路径速查
- 模块根目录：`~/box_for_magisk/`
- 主配置：`~/box_for_magisk/box/settings.ini`
- clash 配置模板：`~/box_for_magisk/box/clash/config.yaml`
- 核心脚本：`~/box_for_magisk/box/scripts/box.service` / `box.iptables` / `box.tool`
- 节点测试配置：`~/flow_collect_server/templates/flashfox_nodes.yaml`
