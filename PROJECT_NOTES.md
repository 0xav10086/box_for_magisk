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
- 测试节点配置：**设备上** `/data/adb/box/clash/config.yaml`（proxy-providers 配置）+
  `/data/adb/box/clash/Proxy-Providers/flashfox_nodes.yaml`（provider 拉取的文件，原始格式，
  节点名如 `🇭🇰 香港 HK-01`；2026-08-01 服务端已换成简化名 `HK-01`，共 64 节点）
  > 注：`flow_collect_server/templates/flashfox_nodes.yaml` 是上游开发机路径，非本机环境，
  > 本地 `C:\Users\23203\code\FlowCollect` 不是当前使用的订阅源

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

### 6. ⚠️ 测单节点的前提：独立实例必须以 root:net_admin（gid 3005）运行
- **背景**：box.iptables 的 nat OUTPUT `BOX_LOCAL` 链只排除 `uid-owner 0 --gid-owner 3005`（box 进程），
  其余进程 TCP 全部 REDIRECT 到 9797 → 被 box mihomo 按规则路由；DNS UDP:53 被劫持到 1053 fake-ip
- **坑**：`su -c` 直接启动的独立实例是 root:root（gid 0）→ 出站被透明代理截获 →
  节点连接失败（`connect error: context deadline exceeded`）是**假阴性**，不代表节点挂了
- **正确做法**（2026-08-01 验证）：Go 交叉编译一个 setgid 包装器（android/arm64，CGO_ENABLED=0），
  `syscall.Setgid(3005)` + `syscall.Exec(clash)`，再 nohup 启动；
  设备无 toybox setuidgid、magisk/ksu busybox 被 SELinux 挡（u:r:su:s0 域）
- **可选**：配置加 `dns:` 段（DoH 上游 `https://dns.alidns.com/dns-query`，`enhanced-mode: redir-host`）
  避免解析走被劫持路径；测完按 **PID** kill（勿 killall，会误杀主服务）
- 收尾：`adb forward --remove`、清 `/data/local/tmp`

### 7. HK-01 节点测试结果（2026-08-01）
- **结论：节点可用**。代理出口 `155.117.84.101`（Hong Kong, AS32167 LSHIY LLC），延迟 ~1.7-2.5s；
  直连出口对照 `103.54.154.41`（大陆运营商）；经代理访问 gstatic 204 正常（1.1s）
- provider 已强制刷新（mihomo API `PUT /providers/proxies/Global-ISP1`），64 节点简化名

### 8. ✅ box DNS 解析问题：根因 + 修复（2026-08-01 已解决）
- **根因**（box debug 日志实锤）：box 的 dns 段在大陆网络下无法解析非 CN 域名
  ① `fallback: [1.1.1.1, dns.cloudflare.com, dns.google]` 三个 DoH **全部被墙**（超时/RST）
  ② 国内 DoH（alidns/doh.pub）对节点域名 `*.llguanglisf.com` 返回 **NODATA**（空答案）
  → 节点连接在解析阶段超时（`dns resolve failed: context deadline exceeded`）→ 健康检查全挂 → 节点"假死"
- **修复**（已应用并验证）：
  ① dns 段加 `proxy-server-nameserver: [223.5.5.5, 114.114.114.114]`（节点域名走明文 UDP，可用）
  ② fallback 换成 `[dns.alidns.com/dns-query, doh.pub/dns-query]`
- **验证**：`/dns/query allapp5p-mhb5pb1m4h.llguanglisf.com` → 43.206.214.185；64 节点全 alive；
  box 7890 出口 154.64.226.169（0.78-0.97s）
- **⚠️ 写入 config.yaml 的 SELinux 技巧**（本设备实测，重要）：
  - box 文件标签是 `u:object_r:system_file:s0`，su 域（u:r:su:s0）对它们**直接写被拒**
    （`>` 重定向/cp/mv/rm 全 denied），但 **sed -i（temp+rename）允许**（box.service 自身
    就是用 sed -i/printf 写 config.yaml 的，mihomo 写 provider 也是 temp+rename）
  - 所以改 config 用：`sed -i -f <script> /data/adb/box/clash/config.yaml`（script 放 /data/local/tmp）
  - 改完用 mihomo API `PUT /configs?force=true`（body 带 `{"path":...}`）热重载，无需重启服务
  - box 文件 ls/stat 也被拒（getattr 限制），但 cat 可读
- **遗留小问题**：86DirectRules 中的域名被解析成 fake-ip 后 DIRECT 直连（198.18.x.x）超时 ——
  独立于本修复的规则/DNS 交互问题，暂不影响代理主链路

## 待办 / 下一步

- [x] ~~全节点批量测试~~（先单节点）：HK-01 已验证可用（见 §7），批量测试方法同 §5/§6
- [x] ~~排查 box 主服务 DNS 解析问题~~：已根因+修复+验证（见 §8）
- [ ] 创建 `mihomo_for_kernelSU/`：精简 fork box_for_magisk，只保留 clash/mihomo + KernelSU
      （参考设计决策：形态、功能范围、mihomo 二进制来源）
- [ ] 设计可靠的 mihomo 自动更新机制（启动时对比版本 / 定时任务 / 多镜像 / 离线更新入口）
      —— 解决设备 TLS 旧导致 upkernel 失败的问题

## 关键路径速查
- 模块根目录：`~/box_for_magisk/`
- 主配置：`~/box_for_magisk/box/settings.ini`
- clash 配置模板：`~/box_for_magisk/box/clash/config.yaml`
- 核心脚本：`~/box_for_magisk/box/scripts/box.service` / `box.iptables` / `box.tool`
- 节点测试配置：`~/flow_collect_server/templates/flashfox_nodes.yaml`
