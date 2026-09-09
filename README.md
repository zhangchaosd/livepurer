# LivePurer

> 让直播回归纯粹

[![Go](https://img.shields.io/github/go-mod/go-version/zhangchaosd/livepurer?style=flat-square)](https://github.com/zhangchaosd/livepurer)
![License](https://img.shields.io/badge/license-AGPL-lightgrey.svg?style=flat-square)
![Release](https://img.shields.io/github/v/release/zhangchaosd/livepurer?color=red&style=flat-square)
![Last Commit](https://img.shields.io/github/last-commit/zhangchaosd/livepurer?style=flat-square)
[![Go Report Card](https://goreportcard.com/badge/github.com/zhangchaosd/livepurer)](https://goreportcard.com/report/github.com/zhangchaosd/livepurer)

**该项目仅供学习，请勿用于商业用途。任何使用该项目造成的后果由使用者自行承担。**

没有礼物、粉丝团、弹窗，只有直播、弹幕。

一个开源的多平台直播聚合工具（Go 编写）：获取直播间信息、直播流地址、弹幕流，支持 WebSocket 弹幕转发与本地直播流转发。

## ✨ 特性

- 🔎 直播间信息、直播流（最高画质）、弹幕流
- ⌛ 平台 `Websocket` 弹幕协议封装，支持转发弹幕消息、直播间热度消息
- 🗝️ 解决跨域问题，支持直播流本地转发
- 📂 简易的收藏夹功能支持
- 🧬 跨平台支持（Linux / Windows / macOS，支持 386 / amd64 / arm / arm64）
- 🔨 支持设置 `Socks5` 代理
- ⚙️ 同时也是一个简单的命令行工具（取流 / 下载弹幕）

## 🖥️ 支持平台

| 平台 | 平台参数 | 直播间信息 | 直播流 | 弹幕 | 发送弹幕 |
| :--: | :--: | :--: | :--: | :--: | :--: |
| 哔哩哔哩 | `bilibili` | ✅ | ✅ | ✅ | ✅ |
| 虎牙 | `huya` | ✅ | ✅ | ✅ | ❌ |
| 斗鱼 | `douyu` | ✅ | ✅ | ✅ | ❌ |
| 映客 | `inke` | ✅ | ✅ | ✅ | ❌ |

> 注：虎牙 / 斗鱼的取流与弹幕功能已按平台最新协议适配（2025 年新版签名方案）。
> 取流画质固定为最高档：斗鱼 `rate=0`（原画/蓝光，主播开 2K 时为原画 2K60）、虎牙 `ratio=0`（原画）。

## 🛠️ 部署

### 二进制部署

下载 [Release](https://github.com/zhangchaosd/livepurer/releases) 的最新打包文件，解压后重命名 `config` 目录下的 `server.yaml.example` 为 `server.yaml`、`account.yaml.example` 为 `account.yaml`，填写相关信息。

```shell
chmod +x ./pure-live
./pure-live run
```

打开 `localhost:<port>`（默认 `8800`）即可使用。

> `pure-live` 的初衷是本地或局域网的直播流推送，对 `websocket` 推送没有做压缩或优化处理。
> 将 `pure-live` 运行在局域网内的 `NAS` 或其他小型服务器上，即可让整个局域网享受其支持。

### 📺 局域网 IPTV 播放列表（APTV 等）

可配置多个平台的直播间，汇总为一个固定的 IPTV 播放列表（m3u），供局域网内的 APTV、PotPlayer、VLC 等播放器加载。

1. 在 `config` 目录新建 `channels.yaml`，声明要观看的频道（参考 `config/channels.yaml.example`）：

```yaml
channels:
  - plat: douyu      # 平台名: bilibili / huya / douyu / inke
    room: "9999"     # 房间号
    name: "yyf斗鱼"  # 可选, 自定义频道名; 不填则使用直播间标题
  - plat: huya
    room: "226046"
  - plat: bilibili
    room: "6"
```

2. 重启程序后，在播放器中添加播放列表地址：

```
http://<局域网IP>:8800/api/v1/live/m3u
```

- 频道流地址指向稳定的 `/api/v1/live/play?plat=&room=`，由服务端在每次播放时动态解析最新流地址，**无需担心签名过期**
- 可选参数：`?online=1` 仅输出在线频道；`?refresh=1` 跳过缓存强制刷新
- 在线频道排在列表前面

### 前端（Web 界面）

Release 已内置 Web 管理界面；启动服务后访问 `http://<服务器地址>:<端口>/` 即可使用。

界面包含总览、直播播放、我的收藏、IPTV 频道与设置五个页面，支持最近打开记录、收藏搜索、频道草稿校验/排序/撤销、复制与下载 M3U、系统状态和分组设置。订阅与分享地址自动使用本机局域网 IP（优先 192.168 网段），无需手动替换回环地址。桌面与手机均可使用，支持键盘操作和对话框焦点管理。前端资源被嵌入可执行文件，启动时会释放到数据目录的 `static` 文件夹，因此运行不需要 Node.js、npm 或额外前端文件。

### 从源码构建二进制

构建需要 Go 1.25+ 和 Node.js 24+。运行只需要二进制程序及配置，不需要 Node.js 或前端资源目录。

```shell
make build
cp config/server.yaml.example config/server.yaml
cp config/account.yaml.example config/account.yaml
./bin/pure-live run
```

已有配置时请保留原文件，不要重复复制。`make build` 构建当前操作系统与架构，输出为 `bin/pure-live`。也可使用 `make build BINARY=bin/pure-live.exe` 自定义输出文件名。跨平台发行包由 Release 工作流生成；本地安装 GoReleaser 后可运行 `make release`。

### 界面开发

开发前端时运行：

```shell
cd web
npm ci
npm run dev
```

## ⚙️ 命令行

查看版本：

```shell
./pure-live -v
```

```
v0.2.0
go1.25.14 darwin/arm64
```

查看帮助：

```shell
./pure-live -h
./pure-live run -h
./pure-live get -h
./pure-live export -h
```

### run —— 启动本地服务器

`-s`：服务器配置文件路径，默认为 `config/server.yaml`

`-a`：账号配置文件路径，默认为 `config/account.yaml`

```shell
./pure-live run
./pure-live run -s myserver.yml
./pure-live run -s my/myserver.yml -a my/myaccount.yml
```

### get —— 获取直播信息、直播流、弹幕流

`-p`：平台名（`bilibili` / `huya` / `douyu` / `inke`）

`-r`：房间号，长短号均可

`--stream`：下载直播流到 `.flv` 文件（仅支持 flv），不传入则不下载；如需更精细的控制请使用 `ffmpeg`

`--danmaku`：抓取弹幕流，以 `xlsx` 格式保存，不传入则不抓取

`--roll`：抓取弹幕时是否在终端滚动显示弹幕内容

```shell
./pure-live get -p bilibili -r 6
./pure-live get -p douyu -r 6556593 --stream b.flv
./pure-live get -p huya -r 226046 --stream b.flv --danmaku dm.xlsx --roll
```

成功获得相关信息：

```
Room: 6556593
Upper: 熊猫大G
Title: 送死流老司机！送皮肤！
Link: https://www.douyu.com/6556593
Stream: https://stream-shanghai-ct-61-172-246-235.edgesrv.com:443/live/6556593ryETvmvct.flv?wsAuth=......
```

### export —— 导出收藏及收藏夹信息

`-d`：数据库路径，默认 `data/data.db`

`-p`：导出路径，默认 `export.xlsx`

```shell
./pure-live export
./pure-live export -d mydata/data.db
./pure-live export -d mydata/data.db -p mydata.xlsx
```

## 🚀 自动发版

仓库已配置 GitHub Actions 自动发版：推送 `v*` 格式的 tag 即触发 [GoReleaser](https://goreleaser.com) 构建多平台二进制并自动发布 Release。

```shell
git tag v0.2.0
git push origin v0.2.0
```

## 📝 文档

- 如何写一个自己的前端？[API 文档](./docs/API.md)
- 如何添加新的平台支持？[Client 文档](./docs/Client.md)

## 🔩 贡献

请使用 `issue` 发起任何问题，非重要事情请勿私聊：

- 提出新的特性帮助 `livepurer` 成长，特性的支持效率取决于其重要程度
- 提出 `BUG` 解决使用中的问题，`BUG` 的修复将优先考虑

## 🔌 TODO

- [ ] 虎牙 / 斗鱼发送弹幕
- [ ] 取流画质可配置（目前固定最高画质）
- [ ] 网易CC
- [ ] Twitch（等待第三方库支持 `m3u8` 拉流）
- [ ] 弹幕 JSON / ASS 导出

## 🧑 贡献者

<a href="https://github.com/zhangchaosd/livepurer/graphs/contributors">
  <img src="https://contrib.rocks/image?repo=zhangchaosd/livepurer"  alt="contrib"/>
</a>

## 🗒️ 参考

- https://github.com/wbt5/real-url
- https://github.com/flxxyz/douyudm
- https://github.com/BacooTang/huya-danmu

## 🔖 LICENSE

AGPL-3.0 License

### 控制台与播放维护

- 直播页支持停止、重新连接、复制稳定播放地址，以及加入 IPTV 频道草稿。
- 收藏夹中的直播间可以直接播放；IPTV 页支持排序、重复检查、保存及复制/下载 M3U。
- IPTV 草稿需要点击保存才会更新电视订阅。通过局域网 IP 打开控制台，可直接复制适用于电视的地址。
- 切换页面会释放 Web 播放连接。转发使用有界缓冲和绝对媒体时钟，避免上游等待与帧间等待叠加，并按音视频轨道分别处理重叠帧。
- 上游连续 15 秒没有数据会结束本次拉流，由播放转发循环重新解析地址；主动停止会取消正在建立的 HTTP 连接。
- 若仍卡顿，请记录平台、房间号、播放器和发生时间，结合 `log` 下日志定位。最高画质对 Wi-Fi 带宽及电视解码能力仍有要求。

验证命令：`go test ./...`、`go test -race ./pkg/forwarder ./app/server/internal/api/v1 ./app/server/internal/config`、`cd web && npm run build`。前端修改后需重新构建 Go 可执行文件并重启。

### v0.1.6 虎牙卡顿修复

虎牙取流改用平台提供的原生 FLV 线路及两层摘要签名，避免旧移动端 HLS 地址改写和旧签名导致频繁返回短流。优先使用 HS、AL 线路，保留 TLS 证书校验。

2026-09-06 对房间 `226046` 的本机对照采样：旧版 45 秒输出 340 个音视频标签；修复版 90 秒输出 9,186 个标签，时间戳零回退，超过 250ms 的输出间隔仅一次（537ms）。网页已验证实际画面。数据反映本次网络环境，电视解码与局域网性能仍应以实机播放为准。

### UI 设计与验证

交互设计见 [Figma 设计稿](https://www.figma.com/design/Ufe0pLRyK97YtUwi6UQs2y)。实现与交互说明见 [UI 设计说明](docs/UI.md)。

```shell
npm --prefix web run typecheck
npm --prefix web test
```

首次运行浏览器测试先执行 `cd web && npx playwright install chromium`。测试使用隔离的模拟数据，不会修改实际收藏与配置。
