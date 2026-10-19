# AGENTS.md

本文件供 AI 编码代理（OpenCode 等）在本仓库工作时遵循。

## 构建与交叉编译原则

本工程是 CLIProxyAPI 的原生动态库插件（`import "C"` + `-buildmode=c-shared`），**任何平台产物都必须 CGO 编译**，且交叉编译目标必须与宿主运行环境的 libc 匹配。

### 基本规则

1. **CGO 必须显式开启**：`CGO_ENABLED=1`（Windows 环境 `go env` 默认为 0，cgo 源文件会被静默排除，导致 `undefined` 等诡异编译错误，连 `go test ./...` 也需要）。
2. **产物格式由目标平台决定**：Windows→`.dll`，Linux→`.so`，macOS→`.dylib`。
3. **构建命令**（本机构建）：
   ```powershell
   $env:CGO_ENABLED = "1"
   go build -buildvcs=false -trimpath -buildmode=c-shared -ldflags "-s -w" -o "dist\model-quota.dll" .
   ```
   生成后删除 `-buildmode=c-shared` 附带的 `.h` 文件。

### Windows 上交叉编译 Linux/macOS 产物：必须用 zig cc

MinGW-w64 只能产出 Windows PE 格式（`.dll`），无法产出 Linux ELF / macOS Mach-O。本机以 zig 作为交叉 C 工具链（`D:\Work\zig\zig.exe`，无需加入 PATH，编译时显式引用）：

```powershell
# Linux x64（glibc 基线 2.17，兼容绝大多数发行版）
$env:CGO_ENABLED = "1"; $env:GOOS = "linux"; $env:GOARCH = "amd64"
$env:CC = "D:\Work\zig\zig.exe cc -target x86_64-linux-gnu.2.17"
go build -buildvcs=false -trimpath -buildmode=c-shared -ldflags "-s -w" -o "dist\model-quota.so" .

# macOS（如需）
$env:CC = "D:\Work\zig\zig.exe cc -target x86_64-macos"    # Intel
$env:CC = "D:\Work\zig\zig.exe cc -target aarch64-macos"   # Apple Silicon
```

### ⚠️ 关键教训：glibc vs musl 必须与宿主镜像匹配

- **默认用 glibc 目标（`x86_64-linux-gnu.2.17`）**：Debian/Ubuntu/CentOS 等主流镜像都是 glibc，运行时只认带版本号的 `libc.so.6`。
- **禁止默认用 musl 目标**：musl 产物依赖无版本号的 `libc.so`，该文件在 glibc 系统上只存在于 `libc6-dev` 开发包，运行时 dlopen 直接报错：
  ```
  dlopen .../model-quota.so: libc.so: cannot open shared object file: No such file or directory
  ```
  这是本项目实际踩过的坑（CPA 官方 Docker 镜像为 glibc）。
- 只有确认宿主是 Alpine（musl 系统）时才用 `-target x86_64-linux-musl`。拿不准时请用户先在容器内执行 `cat /etc/os-release` 确认。

### 产物验证（zig 0.17 的 objdump 是占位实现，不可用）

用字节级检查代替：

```powershell
$bytes = [System.IO.File]::ReadAllBytes("dist\model-quota.so")
$text = [System.Text.Encoding]::ASCII.GetString($bytes)
# ELF 头：$bytes[4]=2 → ELF64；offset 16 (u16)=0x03 → ET_DYN 共享库；offset 18 (u16)=0x3E → x86-64
# libc 依赖：$text.Contains("libc.so.6`0") → glibc 产物正确
# musl 产物特征：精确匹配 "libc.so`0"（无版本号）
# 导出符号：检索 cliproxy_plugin_init / cliproxyPluginCall / cliproxyPluginFree / cliproxyPluginShutdown
```

### 代理环境注意

本机全局环境变量 `HTTP_PROXY`/`HTTPS_PROXY` 使用了 `https://` 方案，会导致 Git Credential Manager 报
`ServicePointManager 不支持具有 https 方案的代理` 而中断推送。需要 git 网络操作时，临时覆盖：

```powershell
$env:HTTP_PROXY = "http://127.0.0.1:7890"; $env:HTTPS_PROXY = "http://127.0.0.1:7890"
git push origin main
```
