# Model Quota (大模型额度与余额查询)

<p align="center">
  <a href="#english">English</a> | <a href="#中文">简体中文</a>
</p>

---

<span id="中文"></span>

## 中文说明

**Model Quota (`model-quota`)** 是专为 [CLIProxyAPI](https://github.com/router-for-me/CLIProxyAPI) 及 [EasyCLIProxyAPI](https://github.com/router-for-me/EasyCLIProxyAPI) 开发的原生动态库插件。

它在桌面端或网页端提供一个优雅聚合的监控面板，帮助开发者实时掌握各个大模型供应商的账户余额、Coding Plan 套餐用量、5 小时/每周额度进度条与重置倒计时。

### ✨ 特性

- 📊 **同供应商深度聚合**：智谱 GLM 的 5 小时额度、每周额度、月度 MCP 调用次数全部整合于同一张卡片，清晰直观。
- ⏳ **精确重置倒计时**：清晰显示下次额度重置的具体日期时间以及相对倒计时（如 `重置于 10/18 23:30 (2小时45分后)`）。
- 🎨 **官方矢量 LOGO 与原生适配**：内联智谱 GLM 彩色几何渐变 LOGO 与 DeepSeek 蓝鲸 LOGO，完全适配 EasyCLIProxyAPI 的浅色与深色主题。
- 🔑 **标准插件配置支持**：API Key 通过官方生命周期配置钩子（`plugins.configs.model-quota`）安全传入，可在客户端界面配置，也可在配置文件中按需指定。
- 🛡️ **严格遵循 CPA 路由安全规范**：静态界面挂载于资源路由，动态额度查询通过受管理密钥鉴权的管理路由执行，安全可靠。
- ⚡ **零额外依赖**：纯静态资源全嵌入，原生 C ABI 交互，通过内核宿主回调（`host.http.do`）发起网络查询，完美复用本地代理设置。

### 📦 支持的供应商与指标

| 供应商 | 查询指标 | 数据来源 |
|---|---|---|
| **智谱 GLM** | 5 小时额度比例、每周额度比例、MCP 每月调用、套餐档位 (Lite/Pro 等)、下次重置时间 | `https://open.bigmodel.cn/api/monitor/usage/quota/limit` |
| **DeepSeek** | 账户总余额、币种、可用状态 | `https://api.deepseek.com/user/balance` |
| **OpenCode Go** | 滚动 (5 小时)/每周/每月额度比例与重置时间 | `https://opencode.ai/zen/go/v1/usage` |
| *更多供应商* | *架构高度模块化，持续拓展中* | *统一接口抽象* |

### 🚀 安装与使用

#### 方式一：通过 EasyCLIProxyAPI 插件商店安装（推荐）

1. 在客户端打开 **「插件」->「插件商店」**。
2. 搜索 **`model-quota`** 或 **`大模型额度`**，点击安装。
3. 安装完成后重启内核，侧边栏或插件列表中将直接出现 **「额度查询」** 按钮。

#### 方式二：手动离线安装

1. 从 [Releases](https://github.com/ZRZ-0805/cpa-plugin-model-quota/releases) 下载适合您操作系统的 ZIP 压缩包（如 `model-quota_0.3.0_windows_amd64.zip`）。
2. 解压并将动态库（Windows 为 `model-quota.dll`，Linux 为 `model-quota.so`，macOS 为 `model-quota.dylib`）放入内核的 `plugins/<GOOS>/<GOARCH>/` 或 `plugins/` 目录。
3. 在 `config.yaml` 中配置启用：
   ```yaml
   plugins:
     enabled: true
     dir: "plugins"
     configs:
       model-quota:
         enabled: true
         priority: 1
         glm_api_key: "你的智谱API密钥"
         deepseek_api_key: "你的DeepSeek密钥"
         opencode_api_key: "你的OpenCode密钥"
   ```

---

<span id="english"></span>

## English

**Model Quota (`model-quota`)** is an official-compliant native dynamic library plugin for [CLIProxyAPI](https://github.com/router-for-me/CLIProxyAPI) and [EasyCLIProxyAPI](https://github.com/router-for-me/EasyCLIProxyAPI).

It provides an intuitive, consolidated dashboard to monitor your AI provider accounts, including GLM Coding Plan quotas, 5-hour/weekly usage progress, reset countdowns, DeepSeek balances, and OpenCode Go plan usage.

### ✨ Features

- 📊 **Consolidated Provider View**: All limits for the same provider (e.g. GLM 5-hour limit, weekly limit, MCP calls) are grouped into a single card.
- ⏳ **Reset Time Countdowns**: Live countdowns indicating when your quota window will replenish (e.g. `Resets at 23:30 (in 2h 45m)`).
- 🎨 **Official Vector Logos & Dark Mode**: Inline SVG graphics for GLM and DeepSeek, automatically adapting to light and dark themes.
- 🔑 **Standard Plugin Configuration**: Upstream API keys are configured securely via standard plugin configurations (`plugins.configs.model-quota`).
- 🛡️ **CPA Architecture Compliant**: Strictly adheres to CPA route boundary specifications. Dynamic queries are secured under `/v0/management/...`, and UI assets are cleanly served as static resources.
- ⚡ **Zero External Dependencies**: Embedded UI resources, native C ABI invocation, and requests issued securely through host callbacks (`host.http.do`).

### 📄 License

MIT License © 2026 ZRZ-0805
