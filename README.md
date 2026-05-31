# 玩家日志统计

基岩版 Minecraft 玩家日志统计工具。项目读取 `PlayerLogger.js` 生成的 CSV，支持本地目录或 SMB 远程复制，导入 MySQL 后提供玩家统计、矿物统计、日志查询、矿透分析、分享页面和 AstrBot 插件接口。

当前后端是 Go：`backend-go`。前端目录是 Vue 3：`frontend-vue`。

## Docker 部署

GitHub Actions 会在 `main` 分支推送后构建并推送两个 Docker 镜像：

```text
ghcr.io/jmcat999/player-backend:latest
ghcr.io/jmcat999/player-frontend:latest
```

服务器只需要 Docker / Docker Compose，不需要 clone 整个项目。新建一个空目录，直接下载部署文件：

```bash
mkdir -p /vol1/1000/docker/player
cd /vol1/1000/docker/player
curl -fsSL https://raw.githubusercontent.com/jmcat999/player/main/docker-compose.yml -o docker-compose.yml
mkdir -p docker-data/synced-logs/main docker-data/synced-logs/sub docker-data/mysql
docker compose pull
docker compose up -d
```

如果你想保留源码再部署，也可以 clone，但要在空目录的上一级执行：

```bash
cd /vol1/1000/docker
git clone https://github.com/jmcat999/player.git
cd player
docker compose pull
docker compose up -d
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

生产环境请先修改 `docker-compose.yml` 里的 `APP_ADMIN_PASSWORD`、`APP_ADMIN_JWT_SECRET`、MySQL 密码和对外端口。账号创建后，再改 `APP_ADMIN_PASSWORD` 不会覆盖旧密码，后续请在管理后台改密码。

如果 `docker compose pull` 提示 `unauthorized`，通常是两个原因之一：

- GitHub Actions 还没成功构建并推送 `latest` 镜像。
- GHCR 镜像包默认是 Private，还没有设置成 Public。

处理方式：先打开 GitHub 仓库的 Actions 页面，确认最新的 `Build` 已成功；然后在 GitHub 仓库的 Packages 里把 `player-backend` 和 `player-frontend` 两个镜像设为 Public。也可以不公开镜像，在服务器登录 GHCR：

```bash
echo YOUR_GITHUB_TOKEN | docker login ghcr.io -u jmcat999 --password-stdin
```

这个 token 至少需要 `read:packages` 权限。

## 更新部署

如果你是免 clone 部署，更新时执行：

```bash
cd /vol1/1000/docker/player
curl -fsSL https://raw.githubusercontent.com/jmcat999/player/main/docker-compose.yml -o docker-compose.yml
docker compose pull
docker compose up -d
```

如果你是 clone 源码部署，更新时执行：

```bash
git pull
docker compose pull
docker compose up -d
```

`docker-data/` 是持久化数据目录，不要删除。

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
PLAYER_LOGS_SMB_ENABLED: true
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

Vue 前端：

```powershell
cd frontend-vue
npm install
npm run dev
```

生产构建验证：

```powershell
cd backend-go
go build -trimpath ./cmd/player-stats

cd ../frontend-vue
npm run build
```

## 矿透分析

矿透分析采用保守策略：

- 拐弯追矿：直线通道至少 8 格，转向角度大于 85 度，拐弯后 2-5 格内命中钻石或远古残骸，路径贴合度至少 95%。
- 矿脉直达：两个矿脉间距至少 8 格，时间 10-120 秒，中间普通方块不超过 6 个，并去重已使用矿脉。
- 绿宝石仍计入数量统计，但不作为追矿证据。
- 只有短直线弱证据时严格限分，避免高效正常玩家误判。
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
frontend-vue/                Vue 3 前端
astrbot_plugin_player_stats/ AstrBot 插件
PlayerLogger.js              基岩版日志插件脚本
docker-compose.yml           拉取 GHCR 镜像部署
.github/workflows/build.yml  GitHub Actions 构建和推送镜像
```
