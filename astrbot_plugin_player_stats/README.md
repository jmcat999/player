# astrbot_plugin_player_stats

AstrBot 玩家统计插件，用来查询玩家方块统计。

## 命令

```text
/绑定游戏id Steve
/我的游戏信息
/肝帝榜
/活跃榜
```

`/我的游戏信息` 会显示主服和 2服数据，并包含首次记录时间、统计数据截止日期和后台设置有效期的详情链接。

```text
详细足迹：http://10.0.0.2:8080/share/xxxx
复制到浏览器打开，链接2小时内有效
```

`/肝帝榜` 会显示主服和 2服排行榜，按破坏 + 放置合计排序。

`/活跃榜` 会显示最近 N 天活跃玩家，N 在插件配置 `active_days` 里设置，默认 7 天。

如果不想开放榜单命令，可以在 AstrBot 插件配置里关闭：

```text
enable_rankings = false
enable_active_rankings = false
```

Vue 后台的“矿透分析详情”页面可以提交“发送到Q群”任务。插件配置好 `enable_xray_group_send`
和 `xray_group_id` 后，会把风险提示发送到指定群，详情链接使用同一个 `share_base_url`：

```text
【风险提示】
服务器：主服
分析日期：2025/11/23 - 2025/11/29
玩家：Feb1236
风险分：76/100(高)
稀有矿：1342 个
追矿证据：19 次
十分钟挖取峰值：95 个
查看详情：http://服务器IP:9493/xray-share/xxxx
链接 1 天内有效
```

## 安装

把整个 `astrbot_plugin_player_stats` 目录放到 AstrBot 插件目录：

```text
AstrBot/data/plugins/astrbot_plugin_player_stats
```

然后在 AstrBot WebUI 重载插件。

## 配置

`api_base_url` 是机器人请求后端接口用的地址，不要加 `/api`，不要以 `/` 结尾。

如果 AstrBot 和这个项目在同一台服务器上，可以填：

```text
http://127.0.0.1:9493
```

`share_base_url` 是发到群里的详细足迹链接地址。它只用于生成：

```text
详细足迹：http://10.0.0.2:8080/share/xxxx
```

这里要填玩家手机和电脑能打开的地址，例如公网 IP、内网可访问 IP 或域名。不要加 `/api`，不要以 `/` 结尾。

Docker 默认部署时，`api_base_url` 和 `share_base_url` 都填公开入口，例如：

```text
http://服务器IP:9493
```

这个入口只开放玩家详情和插件必要接口，不暴露管理后台。如果 AstrBot 走本机或内网访问后端，但详情链接要发给群友打开，就把 `api_base_url` 填本机/内网地址，把 `share_base_url` 填群友能访问的地址。

`api_key` 填 Vue 管理后台“系统设置”里的 `AstrBot 插件密钥`。

详情链接有效期在 Vue 管理后台“系统设置”的“分享链接有效期（分钟）”里调整，AstrBot 会读取后端返回的有效期并显示正确文案。

矿透分析群发送需要在 AstrBot 插件配置中设置：

```text
enable_xray_group_send = true
xray_group_id = 123456789
```

`xray_group_id` 直接填写 QQ 群号即可。插件会优先通过 OneBot/aiocqhttp 的 `send_group_msg`
接口发送；如果你已经知道完整 `unified_msg_origin`，也可以直接填完整 UMO 作为高级兜底。

绑定示例：

```text
/绑定游戏id Steve
```

注意 `/绑定游戏id` 和 `Steve` 中间要有空格。绑定成功后再发送 `/我的游戏信息`。

绑定时插件会先请求统计后端检查这个游戏 ID 是否在主服或 2服出现过。后端直接查询导入时维护的玩家档案表，
不会重新统计破坏/放置数量；只有后端能查到玩家时才会保存绑定。如果两个服都没有记录，会提示等待数据更新后再尝试绑定。
这个检查不使用 `from_date` / `to_date` 过滤，避免老玩家因为当前统计日期范围太窄而无法绑定。


## WebSocket group sending

`enable_xray_group_ws = true` is enabled by default. When Vue creates a Send to QQ group task, the plugin receives it through WebSocket first. If WebSocket disconnects or the backend is temporarily unreachable, the plugin retries the WebSocket connection. Already-created tasks stay in the backend memory queue for up to 10 minutes and are sent after reconnect.

If AstrBot runs in Docker, do not set `api_base_url` to `http://127.0.0.1:9493` unless the backend is in the same container. Use the backend service name, host LAN IP, or another address reachable from the AstrBot container, for example:

```text
http://player-backend:8080
http://HOST_LAN_IP:9493
```

Do not append `/api` or a trailing slash to the backend address.
