# MyGo Calculator

用 [MyGo](https://mygo.run/zh) 构建的跨平台桌面计算器，界面采用 Liquid Glass（液态玻璃）风格。

## 功能

- 四则运算 `+ − × ÷`，支持小数、正负号 `±`、百分比 `%`
- 超大结果自动用科学计数法显示（≥ 1e10 或 < 1e-6）
- 计算记录：右上角时钟图标查看全部记录，可清空；归零（C）时清除当前表达式
- Liquid Glass 材质：显示屏、按钮、记录弹窗均使用玻璃效果
- 响应式布局：按钮和显示区域随窗口尺寸同步缩放，默认窗口 430×850，可在最小/最大边界内自由调整

## 平台

| 平台 | 状态 | 说明 |
|------|------|------|
| Windows | 支持 | exe 与 NSIS 安装程序 |
| macOS | 支持 | 应用包与 DMG（CI 上为临时签名） |
| Linux | 支持 | 可执行文件、Deb / RPM 包 |
| Android | 实验性 | 可构建 APK/AAB，玻璃着色器在手机上不生效 |

## 本地开发

需要 Go 1.27 或更高版本。原生 UI 项目不需要 Bun 或前端工具。

```powershell
# 运行
go run .

# 开发模式（热重载）
go tool mygo dev

# 测试
go test ./...

# 构建本机应用
go tool mygo build
```

CLI 通过 `go.mod` 的 tool 指令固定版本，用 `go tool mygo` 调用。

## 构建与发布

推送 `v*` 标签会触发 GitHub Actions：创建草稿 Release，并为 macOS / Windows / Linux 构建、签名并上传安装包与自动更新文件。检查草稿后发布即可。

```powershell
git tag v1.0.0
git push origin v1.0.0
```

更新签名：`mygo keygen` 生成的 `mygo-update.key` 必须作为仓库机密 `MYGO_UPDATER_PRIVATE_KEY` 提供，公钥写在 `mygo.json` 的 `updates.publicKey`。

## 项目结构

```
main.go          应用状态、计算逻辑与视图
main_test.go     响应式布局回归测试
mygo.json        应用配置（名称、版本、图标、更新、Android）
icons/           记录图标 SVG
resources/       应用图标 PNG
```

## 许可证

MIT
