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

### Docker（本地构建）

`Release` 不附带 Docker 镜像，可在本地自行构建：

```shell
docker build -t livepurer .

# 启动
docker run --name livepurer -p <HOST_PORT>:8800 -d --restart=always livepurer:latest

# 或添加 -v 参数
docker run --name livepurer -p <HOST_PORT>:8800 -v /HOST/PATH/DATA:/data -v /HOST/PATH/LOG:/log -d --restart=always livepurer:latest

# 查看 log
docker logs -f livepurer

# 设置账户/服务器配置文件
docker cp PATH/TO/account.yaml livepurer:/config/account.yaml
docker cp PATH/TO/server.yaml livepurer:/config/server.yaml
docker restart livepurer
```

### 二进制部署

下载 [Release](https://github.com/zhangchaosd/livepurer/releases) 的最新打包文件，解压后重命名 `config` 目录下的 `server.yaml.example` 为 `server.yaml`、`account.yaml.example` 为 `account.yaml`，填写相关信息。

```shell
chmod +x ./pure-live
./pure-live run
```

打开 `localhost:<port>`（默认 `8800`）即可使用。

> `pure-live` 的初衷是本地或局域网的直播流推送，对 `websocket` 推送没有做压缩或优化处理。
> 将 `pure-live` 运行在局域网内的 `NAS` 或其他小型服务器上，即可让整个局域网享受其支持。

### 前端（Web 界面）

本仓库（纯 core 版）的 `Release` **不内置前端页面**，仅提供 CLI 二进制与后端 API。

如需 Web 界面，可将任意符合 [API 文档](./docs/API.md) 的前端构建产物放入程序运行目录下的 `static` 文件夹中；原前端仓库（Vue）可参考：https://github.com/iyear/pure-live-frontend

## ⚙️ 命令行

查看版本：

```shell
./pure-live -v
```

```
v0.1.4
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
