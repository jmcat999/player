# 玩家日志统计

基岩版 Minecraft 玩家日志统计工具。项目读取 `PlayerLogger.js` 生成的 CSV，支持本地目录或 SMB 远程复制，导入 MySQL 后提供玩家统计、矿物统计、日志查询、矿透分析、分享页面和 AstrBot 插件接口。

当前 Docker 默认使用 Go 后端：`backend-go`。

## 快速启动

要求：

- Docker / Docker Compose
- 可选：Go 1.25+ 用于本地后端开发
- 可选：Node.js 20+ 用于本地前端开发

一键启动：

```powershell
docker compose up -d --build
```

默认入口：

```text
公开页面 / AstrBot 接口：http://服务器IP:9493
管理后台：http://服务器IP:8843
```

默认管理员：

```text
账号：admin
密码：admin123456
```

生产环境请修改 `docker-compose.yml` 里的 `APP_ADMIN_PASSWORD`、`APP_ADMIN_JWT_SECRET`、MySQL 密码和对外端口。

## CSV 目录

Docker 会把宿主机目录挂载到容器：

```text
./docker-data/synced-logs/main  -> 主服 CSV
./docker-data/synced-logs/sub   -> 2服 CSV
./docker-data/mysql             -> MySQL 数据
```

CSV 文件名保持插件格式，例如：

```text
player_actions_2026-05-02.csv
```

默认会跳过当天或未来日期的 CSV，避免读取仍在写入的日志。

## SMB 同步

后台“系统设置”里可以配置 SMB，也可以直接修改环境变量：

```yaml
PLAYER_LOGS_SMB_HOST: "192.168.1.10"
PLAYER_LOGS_SMB_PORT: 445
PLAYER_LOGS_SMB_DOMAIN: ""
PLAYER_LOGS_SMB_USERNAME: "your-user"
PLAYER_LOGS_SMB_PASSWORD: "your-password"
PLAYER_LOGS_SMB_SHARE: "mcshare"
PLAYER_LOGS_SMB_DIRECTORY: "logs"
PLAYER_LOGS_SMB_FILE_GLOB: "player_actions_*.csv"
PLAYER_LOGS_SMB_RECURSIVE: false
```

SMB 复制会先把远程 CSV 拉到本地归档目录，再从本地文件解析入库。

## 本地开发

Go 后端：

```powershell
cd backend-go
go test ./...
go vet ./...
go run ./cmd/player-stats
```

前端：

```powershell
cd frontend
npm install
npm run dev
```

生产构建验证：

```powershell
cd backend-go
go build -trimpath ./cmd/player-stats

cd ../frontend
npm run build
```

## 矿透分析

矿透分析采用保守策略：

- 拐弯追矿：直线通道至少 8 格，转向角度大于 85 度，拐弯后 2-5 格内命中钻石或远古残骸，路径贴合度至少 95%。
- 矿脉直达：两个矿脉间距至少 8 格，时间 10-120 秒，中间普通方块不超过 6 个，并去重已使用矿脉。
- 绿宝石仍计入数量统计，但不作为追矿证据。
- 只有短直线弱证据时会严格限分，避免高效正常玩家误判。
- 下界远古残骸包含床炸场景的单独保守证据。

分析结果默认只保留最近 1 次。

## AstrBot 插件

插件目录：

```text
astrbot_plugin_player_stats
```

把该目录复制到 AstrBot 插件目录后，在插件配置里填写：

```text
api_base_url: http://服务器IP:9493
share_base_url: http://服务器IP:9493
api_key: 管理后台“系统设置”里的 AstrBot 插件密钥
```

命令示例：

```text
/绑定游戏id Steve
/我的游戏信息
```

## 项目结构

```text
backend-go/                  Go 后端
frontend/                    Vue 3 前端
astrbot_plugin_player_stats/ AstrBot 插件
docs/                        说明文档
PlayerLogger.js              基岩版日志插件脚本
docker-compose.yml           一键部署配置
```
