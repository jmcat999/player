import asyncio
import inspect
import json
from datetime import date, datetime, timedelta, timezone
from typing import Any

import httpx
import websockets
from astrbot.api import AstrBotConfig, logger
from astrbot.api.event import AstrMessageEvent, MessageChain, filter
from astrbot.api.star import Context, Star, register
from astrbot.core.star.filter.command import GreedyStr


@register(
    "player_stats",
    "Codex",
    "查询 Minecraft 玩家在主服和 2服的方块统计",
    "0.11.0",
)
class PlayerStatsPlugin(Star):
    SERVERS = (
        ("main", "主服"),
        ("sub", "2服"),
    )

    def __init__(self, context: Context, config: AstrBotConfig):
        super().__init__(context)
        self.config = config
        self._xray_group_task = None
        self._last_xray_ws_error = ""
        self._log_query_usage: dict[str, list[datetime]] = {}
        try:
            self._xray_group_task = asyncio.create_task(self._xray_group_sender_loop())
        except RuntimeError as ex:
            logger.warning(f"start xray group sender loop failed: {ex}")

    @filter.command("绑定游戏id", alias={"绑定游戏ID", "绑定id", "绑定ID"})
    async def bind_game_id(self, event: AstrMessageEvent, game_id: GreedyStr):
        """绑定 QQ 用户和游戏 ID。例如 /绑定游戏id Steve。"""
        game_id = self._normalize_game_id(str(game_id))
        if not game_id:
            yield event.plain_result("请输入游戏 ID，例如：/绑定游戏id Steve\n注意：/绑定游戏id 和 Steve 中间要有空格。")
            return

        check = await self._check_player_exists(game_id)
        if not check["ok"]:
            yield event.plain_result(check["message"])
            return

        canonical_game_id = check.get("player_name") or game_id
        await self.put_kv_data(self._binding_key(event), canonical_game_id)
        servers = "、".join(check.get("servers") or [])
        yield event.plain_result(
            f"已绑定游戏 ID：{canonical_game_id}\n"
            f"已在服务器中找到：{servers}\n"
            "之后发送 /我的游戏信息 即可查询。"
        )

    @filter.command("我的游戏信息", alias={"我的信息", "我的统计", "方块统计"})
    async def my_game_stats(self, event: AstrMessageEvent):
        """查询自己的主服和 2服统计。"""
        bound_game_id = await self.get_kv_data(self._binding_key(event), "")
        if not bound_game_id:
            yield event.plain_result(
                "还没有绑定游戏 ID\n"
                "请先发送：/绑定游戏id 游戏id\n"
                "使用示例：/绑定游戏id Steve\n"
                "注意：/绑定游戏id 和 Steve 中间要有空格\n"
                "绑定成功后再发送：/我的游戏信息"
            )
            return

        yield event.plain_result(await self._query_and_format_all(bound_game_id))

    @filter.command("肝帝榜", alias={"排行榜", "方块排行榜"})
    async def player_rankings(self, event: AstrMessageEvent):
        """查询主服和 2服玩家方块行为排行榜。"""
        if not self._config_bool("enable_rankings", True):
            yield event.plain_result("肝帝榜命令已关闭。")
            return
        yield event.plain_result(await self._query_and_format_rankings())

    @filter.command("活跃榜", alias={"活跃玩家", "最近活跃"})
    async def active_rankings(self, event: AstrMessageEvent):
        """查询最近 N 天的主服和 2服活跃玩家。"""
        if not self._config_bool("enable_active_rankings", True):
            yield event.plain_result("活跃榜命令已关闭。")
            return
        yield event.plain_result(await self._query_and_format_active_rankings())

    @filter.command("查日志", alias={"坐标日志", "查坐标日志"})
    async def query_coordinate_logs(self, event: AstrMessageEvent, args: GreedyStr):
        """按交互坐标查询公开日志。例如 /查日志 -191 -34 750。"""
        coords = self._parse_coordinate_args(str(args))
        if coords is None:
            yield event.plain_result(self._log_query_usage_text())
            return

        bound_game_id = await self.get_kv_data(self._binding_key(event), "")
        if not bound_game_id:
            yield event.plain_result(
                "请先绑定游戏 ID 后再查询日志。\n"
                "绑定示例：/绑定游戏id Steve\n"
                f"{self._log_query_usage_text()}"
            )
            return

        presence = await self._query_bound_player_presence(bound_game_id)
        if not presence["ok"]:
            yield event.plain_result(presence["message"])
            return

        quota = self._consume_log_query_quota(presence["player_name"])
        if not quota["ok"]:
            yield event.plain_result(
                f"查询次数已用完：每个玩家 1 小时内最多查询 {quota['limit']} 次。\n"
                f"请约 {quota['reset_minutes']} 分钟后再试。"
            )
            return

        x, y, z = coords
        limit = self._public_log_result_limit()
        days = self._public_log_recent_days()
        servers = [server for server in presence["servers"] if server.get("serverId")]
        tasks = [
            self._fetch_public_coordinate_logs(server["serverId"], x, y, z, limit, days)
            for server in servers
        ]
        results = await asyncio.gather(*tasks, return_exceptions=True)
        error = self._first_error_message(list(results))
        if error:
            yield event.plain_result(error)
            return

        lines = [
            f"交互坐标日志：{self._format_coord_triplet(x, y, z)}",
            f"绑定玩家：{presence['player_name']}",
            f"查询范围：最近 {days} 天",
            f"本小时剩余查询次数：{quota['remaining']}/{quota['limit']}",
        ]
        for server, result in zip(servers, results, strict=True):
            lines.extend(["", self._format_public_log_result(server, result)])
        yield event.plain_result("\n".join(lines))

    async def _query_and_format_all(self, game_id: str) -> str:
        tasks = [self._fetch_player_stats(game_id, server_id) for server_id, _ in self.SERVERS]
        results = await asyncio.gather(*tasks, return_exceptions=True)

        for result in results:
            if isinstance(result, httpx.ConnectError):
                return "查询失败：统计后端没有连接上，请确认 player-stats 后端正在运行。"
            if isinstance(result, httpx.TimeoutException):
                return "查询失败：统计后端响应超时。"
            if isinstance(result, httpx.HTTPStatusError) and result.response.status_code != 404:
                logger.warning(f"player stats api error: {result.response.status_code} {result.response.text}")
                if result.response.status_code == 401:
                    return "查询失败：统计后端拒绝访问，请在 AstrBot 插件配置里填写插件密钥 api_key。"
                return f"查询失败：统计后端返回 {result.response.status_code}。"
            if isinstance(result, Exception) and not isinstance(result, httpx.HTTPStatusError):
                logger.exception(f"query player stats failed: {result}")
                return "查询失败：机器人插件内部错误，请看 AstrBot 日志。"

        lines = [f"玩家：{game_id}"]
        has_data = False
        for (server_id, server_name), result in zip(self.SERVERS, results, strict=True):
            if isinstance(result, httpx.HTTPStatusError) and result.response.status_code == 404:
                lines.extend(["", f"{server_name}：暂无数据"])
                continue
            has_data = True
            lines.extend(["", self._format_server_stats(server_name, result)])
        if has_data:
            share_info = await self._create_share_link(game_id)
            if share_info:
                lines.extend(["", f"详细足迹：{share_info['link']}", f"链接{share_info['ttl_text']}内有效"])
        return "\n".join(lines)

    async def _query_and_format_rankings(self) -> str:
        limit = self._ranking_limit()
        from_date = str(self.config.get("from_date", "")).strip()
        to_date = str(self.config.get("to_date", "")).strip()

        share_info = await self._create_ranking_share_link("total", limit, from_date, to_date)
        if not share_info:
            return "查询失败：无法生成排行榜链接，请稍后重试。"

        return "\n".join([
            f"肝帝榜 Top {limit}",
            "按破坏+放置合计排序",
            f"查看排名：{share_info['link']}",
            f"链接{share_info['ttl_text']}内有效",
        ])

    async def _query_and_format_active_rankings(self) -> str:
        limit = self._ranking_limit()
        active_days = self._active_days()
        today = date.today()
        from_date = today - timedelta(days=active_days - 1)
        to_date = today

        share_info = await self._create_ranking_share_link(
            "active", limit, from_date.isoformat(), to_date.isoformat()
        )
        if not share_info:
            return "查询失败：无法生成活跃榜链接，请稍后重试。"

        return "\n".join([
            f"最近 {active_days} 天活跃玩家 Top {limit}",
            f"统计范围：{from_date.isoformat()} 至 {to_date.isoformat()}",
            "按破坏+放置合计排序",
            f"查看排名：{share_info['link']}",
            f"链接{share_info['ttl_text']}内有效",
        ])

    async def _fetch_player_stats(self, game_id: str, server_id: str) -> dict[str, Any]:
        base_url = self._api_base_url()
        timeout_seconds = float(self.config.get("timeout_seconds", 8))
        params: dict[str, str] = {
            "serverId": server_id,
            "playerName": game_id,
        }

        from_date = str(self.config.get("from_date", "")).strip()
        to_date = str(self.config.get("to_date", "")).strip()
        if from_date:
            params["from"] = from_date
        if to_date:
            params["to"] = to_date

        async with httpx.AsyncClient(timeout=timeout_seconds) as client:
            response = await client.get(
                f"{base_url}/api/stats/player",
                params=params,
                headers=self._auth_headers(),
            )
            response.raise_for_status()
            return response.json()

    async def _check_player_exists(self, game_id: str) -> dict[str, Any]:
        try:
            result = await self._fetch_player_presence(game_id)
        except httpx.HTTPStatusError as ex:
            if ex.response.status_code == 404:
                return {
                    "ok": False,
                    "message": (
                        f"绑定失败!没有这个玩家信息：{game_id}\n"
                        "请确认大小写和游戏内名字是否正确，或等待数据更新后再尝试绑定。"
                        "(数据每天凌晨0~1点自动更新)"
                    ),
                }
            logger.warning(f"check player presence api error: {ex.response.status_code} {ex.response.text}")
            if ex.response.status_code == 401:
                return {
                    "ok": False,
                    "message": "绑定失败：统计后端拒绝访问，请在 AstrBot 插件配置里填写插件密钥 api_key。",
                }
            return {
                "ok": False,
                "message": f"绑定失败：统计后端返回 {ex.response.status_code}。",
            }
        except httpx.ConnectError:
            return {
                "ok": False,
                "message": "绑定失败：统计后端没有连接上，请稍后再试。",
            }
        except httpx.TimeoutException:
            return {
                "ok": False,
                "message": "绑定失败：统计后端响应超时，请稍后再试。",
            }
        except Exception as ex:
            logger.exception(f"check player exists failed: {ex}")
            return {
                "ok": False,
                "message": "绑定失败：机器人插件内部错误，请看 AstrBot 日志。",
            }

        servers = result.get("servers") or []
        found_servers = [
            str(server.get("serverName") or server.get("serverId") or "").strip()
            for server in servers
        ]
        found_servers = [server for server in found_servers if server]
        if not found_servers:
            return {
                "ok": False,
                "message": (
                    f"绑定失败!没有这个玩家信息：{game_id}\n"
                    "请确认大小写和游戏内名字是否正确，或等待数据更新后再尝试绑定。"
                    "(数据每天凌晨0~1点自动更新)"
                ),
            }
        canonical_name = str(result.get("playerName") or "").strip()
        return {
            "ok": True,
            "player_name": canonical_name or game_id,
            "servers": found_servers,
        }

    async def _query_bound_player_presence(self, game_id: str) -> dict[str, Any]:
        try:
            result = await self._fetch_player_presence(game_id)
        except httpx.HTTPStatusError as ex:
            if ex.response.status_code == 404:
                return {
                    "ok": False,
                    "message": (
                        f"查询失败：绑定的游戏 ID 没有这个玩家信息：{game_id}\n"
                        "请重新绑定，或等待数据更新后再尝试查询。"
                        "(数据每天凌晨0~1点自动更新)"
                    ),
                }
            logger.warning(f"check bound player presence api error: {ex.response.status_code} {ex.response.text}")
            if ex.response.status_code == 401:
                return {
                    "ok": False,
                    "message": "查询失败：统计后端拒绝访问，请在 AstrBot 插件配置里填写插件密钥 api_key。",
                }
            return {
                "ok": False,
                "message": f"查询失败：统计后端返回 {ex.response.status_code}。",
            }
        except httpx.ConnectError:
            return {
                "ok": False,
                "message": "查询失败：统计后端没有连接上，请稍后再试。",
            }
        except httpx.TimeoutException:
            return {
                "ok": False,
                "message": "查询失败：统计后端响应超时，请稍后再试。",
            }
        except Exception as ex:
            logger.exception(f"check bound player exists failed: {ex}")
            return {
                "ok": False,
                "message": "查询失败：机器人插件内部错误，请看 AstrBot 日志。",
            }

        servers = []
        for server in result.get("servers") or []:
            server_id = str(server.get("serverId") or "").strip()
            if not server_id:
                continue
            servers.append({
                "serverId": server_id,
                "serverName": str(server.get("serverName") or server_id).strip(),
            })
        if not servers:
            return {
                "ok": False,
                "message": (
                    f"查询失败：绑定的游戏 ID 没有这个玩家信息：{game_id}\n"
                    "请重新绑定，或等待数据更新后再尝试查询。"
                    "(数据每天凌晨0~1点自动更新)"
                ),
            }
        return {
            "ok": True,
            "player_name": str(result.get("playerName") or game_id).strip(),
            "servers": servers,
        }

    async def _fetch_player_presence(self, game_id: str) -> dict[str, Any]:
        base_url = self._api_base_url()
        timeout_seconds = float(self.config.get("timeout_seconds", 8))
        params: dict[str, str] = {
            "playerName": game_id,
        }

        async with httpx.AsyncClient(timeout=timeout_seconds) as client:
            response = await client.get(
                f"{base_url}/api/stats/player-presence",
                params=params,
                headers=self._auth_headers(),
            )
            response.raise_for_status()
            return response.json()

    async def _fetch_public_coordinate_logs(
        self, server_id: str, x: float, y: float, z: float, limit: int, days: int
    ) -> dict[str, Any]:
        base_url = self._api_base_url()
        timeout_seconds = float(self.config.get("timeout_seconds", 8))
        params: dict[str, str] = {
            "serverId": server_id,
            "x": self._format_coord_value(x),
            "y": self._format_coord_value(y),
            "z": self._format_coord_value(z),
            "limit": str(limit),
            "days": str(days),
        }

        async with httpx.AsyncClient(timeout=timeout_seconds) as client:
            response = await client.get(
                f"{base_url}/api/stats/public-coordinate-logs",
                params=params,
                headers=self._auth_headers(),
            )
            response.raise_for_status()
            return response.json()

    async def _fetch_rankings(
        self,
        server_id: str,
        limit: int,
        from_date_override: str | None = None,
        to_date_override: str | None = None,
    ) -> list[dict[str, Any]]:
        base_url = self._api_base_url()
        timeout_seconds = float(self.config.get("timeout_seconds", 8))
        params: dict[str, str] = {
            "serverId": server_id,
            "limit": str(limit),
        }

        from_date = from_date_override or str(self.config.get("from_date", "")).strip()
        to_date = to_date_override or str(self.config.get("to_date", "")).strip()
        if from_date:
            params["from"] = from_date
        if to_date:
            params["to"] = to_date

        async with httpx.AsyncClient(timeout=timeout_seconds) as client:
            response = await client.get(
                f"{base_url}/api/stats/players",
                params=params,
                headers=self._auth_headers(),
            )
            response.raise_for_status()
            return response.json()

    async def _create_share_link(self, game_id: str) -> dict[str, str]:
        base_url = self._api_base_url()
        share_base_url = self._share_base_url()
        timeout_seconds = float(self.config.get("timeout_seconds", 8))
        try:
            async with httpx.AsyncClient(timeout=timeout_seconds) as client:
                response = await client.post(
                    f"{base_url}/api/share/tokens",
                    json={"playerName": game_id},
                    headers=self._auth_headers(),
                )
                response.raise_for_status()
                data = response.json()
        except Exception as ex:
            logger.warning(f"create player share link failed: {ex}")
            return {}

        link = ""
        share_path = str(data.get("sharePath") or "").strip()
        if share_path:
            link = f"{share_base_url}/{share_path.lstrip('/')}"
        else:
            token = str(data.get("token") or "").strip()
            if token:
                link = f"{share_base_url}/share/{token}"
        if not link:
            return {}
        return {
            "link": link,
            "ttl_text": self._share_ttl_text(data),
        }

    async def _create_ranking_share_link(
        self, ranking_type: str, limit: int, from_date: str, to_date: str
    ) -> dict[str, str]:
        base_url = self._api_base_url()
        share_base_url = self._share_base_url()
        timeout_seconds = float(self.config.get("timeout_seconds", 8))
        body: dict[str, object] = {"type": ranking_type, "limit": limit}
        if from_date:
            body["fromDate"] = from_date
        if to_date:
            body["toDate"] = to_date
        try:
            async with httpx.AsyncClient(timeout=timeout_seconds) as client:
                response = await client.post(
                    f"{base_url}/api/share/ranking-tokens",
                    json=body,
                    headers=self._auth_headers(),
                )
                response.raise_for_status()
                data = response.json()
        except Exception as ex:
            logger.warning(f"create ranking share link failed: {ex}")
            return {}

        link = ""
        share_path = str(data.get("sharePath") or "").strip()
        if share_path:
            link = f"{share_base_url}/{share_path.lstrip('/')}"
        else:
            token = str(data.get("token") or "").strip()
            if token:
                link = f"{share_base_url}/share/ranking/{token}"
        if not link:
            return {}
        return {
            "link": link,
            "ttl_text": self._share_ttl_text(data),
        }

    async def _xray_group_sender_loop(self):
        while True:
            if not self._config_bool("enable_xray_group_send", False):
                await asyncio.sleep(self._xray_group_poll_seconds())
                continue
            if not self._xray_group_target():
                await asyncio.sleep(self._xray_group_poll_seconds())
                continue
            if self._config_bool("enable_xray_group_ws", True):
                try:
                    await self._xray_group_ws_loop()
                except asyncio.CancelledError:
                    raise
                except Exception as ex:
                    error_text = str(ex)
                    if error_text != self._last_xray_ws_error:
                        logger.warning(f"xray group websocket disconnected, will retry: {ex}")
                        self._last_xray_ws_error = error_text
                    else:
                        logger.debug(f"xray group websocket still unavailable: {ex}")
                    await asyncio.sleep(max(30, self._xray_group_poll_seconds()))
            else:
                await self._send_pending_xray_group_messages()
                await asyncio.sleep(self._xray_group_poll_seconds())

    async def _xray_group_ws_loop(self):
        headers = self._auth_headers()
        if not headers.get("X-Player-Stats-Key"):
            raise RuntimeError("api_key is required for xray group websocket")
        connect_kwargs = {
            "ping_interval": 20,
            "ping_timeout": 20,
            "open_timeout": float(self.config.get("timeout_seconds", 8)),
        }
        header_param = "additional_headers"
        try:
            if "additional_headers" not in inspect.signature(websockets.connect).parameters:
                header_param = "extra_headers"
        except (TypeError, ValueError):
            pass
        connect_kwargs[header_param] = headers
        async with websockets.connect(self._xray_group_ws_url(), **connect_kwargs) as websocket:
            await self._send_pending_xray_group_messages()
            async for raw_message in websocket:
                try:
                    message = json.loads(raw_message)
                except json.JSONDecodeError as ex:
                    logger.warning(f"invalid xray websocket message: {ex}")
                    continue
                if isinstance(message, dict):
                    await self._deliver_xray_group_message(message)

    async def _send_pending_xray_group_messages(self):
        try:
            messages = await self._fetch_pending_xray_group_messages()
        except Exception as ex:
            logger.warning(f"fetch xray group messages failed: {ex}")
            return

        for message in messages:
            await self._deliver_xray_group_message(message)

    async def _deliver_xray_group_message(self, message: dict[str, Any]):
        message_id = message.get("id")
        try:
            text = self._format_xray_group_message(message)
            await self._send_xray_group_text(text)
            if message_id is not None:
                await self._mark_xray_group_message(message_id, True, "")
        except Exception as ex:
            logger.warning(f"send xray group message failed: {ex}")
            if message_id is not None:
                try:
                    await self._mark_xray_group_message(message_id, False, str(ex))
                except Exception as mark_ex:
                    logger.warning(f"mark xray group message failed: {mark_ex}")

    async def _fetch_pending_xray_group_messages(self) -> list[dict[str, Any]]:
        base_url = self._api_base_url()
        timeout_seconds = float(self.config.get("timeout_seconds", 8))
        async with httpx.AsyncClient(timeout=timeout_seconds) as client:
            response = await client.get(
                f"{base_url}/api/share/xray-group-messages/pending",
                params={"limit": "5"},
                headers=self._auth_headers(),
            )
            response.raise_for_status()
            data = response.json()
            return data if isinstance(data, list) else []

    async def _mark_xray_group_message(self, message_id: int, success: bool, error_message: str):
        base_url = self._api_base_url()
        timeout_seconds = float(self.config.get("timeout_seconds", 8))
        async with httpx.AsyncClient(timeout=timeout_seconds) as client:
            response = await client.post(
                f"{base_url}/api/share/xray-group-messages/{message_id}/delivery",
                json={"success": success, "errorMessage": error_message},
                headers=self._auth_headers(),
            )
            response.raise_for_status()

    def _format_xray_group_message(self, message: dict[str, Any]) -> str:
        ttl_minutes = int(message.get("ttlMinutes") or 1440)
        risk_level = str(message.get("riskLevel") or "-").strip() or "-"
        return "\n".join([
            "【风险提示】",
            f"服务器：{message.get('serverName') or message.get('serverId') or '-'}",
            f"分析日期：{self._format_xray_date_range(message.get('fromTime'), message.get('toTime'))}",
            f"玩家：{message.get('playerName') or '-'}",
            f"风险分：{int(message.get('riskScore') or 0)}/100({risk_level})",
            f"稀有矿：{int(message.get('miningSessionRareOreBreaks') or 0)} 个",
            f"追矿证据：{int(message.get('trackingEvidenceCount') or 0)} 次",
            f"十分钟挖取峰值：{int(message.get('peakRareOreWindowCount') or 0)} 个",
            f"查看详情：{self._xray_share_link(message)}",
            f"链接 {self._format_duration_minutes_spaced(ttl_minutes)}内有效",
        ])

    def _xray_share_link(self, message: dict[str, Any]) -> str:
        share_path = str(message.get("sharePath") or "").strip()
        if share_path:
            return f"{self._share_base_url()}/{share_path.lstrip('/')}"
        return self._share_base_url()

    def _xray_group_target(self) -> str:
        return str(self.config.get("xray_group_id", "")).strip()

    async def _send_xray_group_text(self, text: str):
        target = self._xray_group_target()
        if not target:
            raise RuntimeError("xray_group_id is empty")

        if ":" in target:
            await self.context.send_message(target, MessageChain().message(text))
            return

        await self._send_onebot_group_text(target, text)

    async def _send_onebot_group_text(self, group_id: str, text: str):
        errors: list[str] = []
        for client in self._discover_onebot_clients():
            try:
                await self._call_onebot_send_group_msg(client, group_id, text)
                return
            except Exception as ex:
                errors.append(f"{type(client).__name__}: {ex}")

        detail = "; ".join(errors[-3:]) if errors else "no OneBot/aiocqhttp client found"
        raise RuntimeError(f"send_group_msg failed for group {group_id}: {detail}")

    def _discover_onebot_clients(self) -> list[Any]:
        clients: list[Any] = []
        seen: set[int] = set()

        def add_client(candidate: Any):
            if candidate is None:
                return
            marker = id(candidate)
            if marker in seen:
                return
            if self._looks_like_onebot_client(candidate):
                seen.add(marker)
                clients.append(candidate)

        platform_manager = getattr(self.context, "platform_manager", None)
        get_insts = getattr(platform_manager, "get_insts", None)
        if callable(get_insts):
            try:
                raw_platforms = get_insts() or []
                platforms = raw_platforms.values() if isinstance(raw_platforms, dict) else raw_platforms
            except Exception as ex:
                logger.warning(f"discover AstrBot platforms failed: {ex}")
                platforms = []

            for platform in platforms:
                add_client(platform)
                get_client = getattr(platform, "get_client", None)
                if callable(get_client):
                    try:
                        add_client(get_client())
                    except Exception as ex:
                        logger.debug(f"get AstrBot platform client failed: {ex}")
                add_client(getattr(platform, "bot", None))
                add_client(getattr(platform, "client", None))

        add_client(getattr(self.context, "bot", None))
        add_client(getattr(self.context, "client", None))
        return clients

    def _looks_like_onebot_client(self, candidate: Any) -> bool:
        if hasattr(candidate, "call_action"):
            return True
        class_name = type(candidate).__name__.lower()
        return "onebot" in class_name or "cqhttp" in class_name

    async def _call_onebot_send_group_msg(self, client: Any, group_id: str, text: str):
        target_group_id = self._onebot_group_id(group_id)
        message = [{"type": "text", "data": {"text": text}}]

        call_action = getattr(client, "call_action", None)
        if callable(call_action):
            attempts = [
                ("send_group_msg", {"group_id": target_group_id, "message": message}),
                ("send_group_msg", {"group_id": target_group_id, "message": text}),
                ("send_msg", {"message_type": "group", "group_id": target_group_id, "message": text}),
            ]
            last_error: Exception | None = None
            for action, payload in attempts:
                try:
                    await self._maybe_await(call_action(action, **payload))
                    return
                except Exception as ex:
                    last_error = ex
            if last_error:
                raise last_error

        send_group_msg = getattr(client, "send_group_msg", None)
        if callable(send_group_msg):
            await self._maybe_await(send_group_msg(group_id=target_group_id, message=text))
            return

        send_msg = getattr(client, "send_msg", None)
        if callable(send_msg):
            await self._maybe_await(send_msg(
                message_type="group",
                group_id=target_group_id,
                message=text,
            ))
            return

        raise RuntimeError("client has no OneBot send method")

    def _onebot_group_id(self, group_id: str) -> int | str:
        group_id = str(group_id).strip()
        if group_id.isdigit():
            return int(group_id)
        return group_id

    async def _maybe_await(self, value: Any) -> Any:
        if inspect.isawaitable(value):
            return await value
        return value

    def _xray_group_poll_seconds(self) -> int:
        try:
            return max(3, min(int(self.config.get("xray_group_poll_seconds", 5)), 60))
        except (TypeError, ValueError):
            return 5

    def _format_xray_date_range(self, from_time: str | None, to_time: str | None) -> str:
        return f"{self._format_short_date(from_time)} - {self._format_short_date(to_time)}"

    def _format_short_date(self, value: str | None) -> str:
        if not value:
            return "-"
        text = str(value)
        try:
            parsed = datetime.fromisoformat(text.replace("Z", "+00:00"))
            return parsed.strftime("%Y/%m/%d")
        except ValueError:
            return text[:10].replace("-", "/")

    def _api_base_url(self) -> str:
        raw_base_url = str(self.config.get("api_base_url", "http://127.0.0.1:9493")).strip()
        return (raw_base_url or "http://127.0.0.1:9493").rstrip("/")

    def _xray_group_ws_url(self) -> str:
        base_url = self._api_base_url()
        if base_url.startswith("https://"):
            return "wss://" + base_url[len("https://"):] + "/api/share/xray-group-messages/ws"
        if base_url.startswith("http://"):
            return "ws://" + base_url[len("http://"):] + "/api/share/xray-group-messages/ws"
        return base_url.rstrip("/") + "/api/share/xray-group-messages/ws"

    def _share_base_url(self) -> str:
        raw_share_url = str(self.config.get("share_base_url", "http://127.0.0.1:9493")).strip()
        return (raw_share_url or "http://127.0.0.1:9493").rstrip("/")

    def _first_error_message(self, results: list[Any]) -> str:
        for result in results:
            if isinstance(result, httpx.ConnectError):
                return "查询失败：统计后端没有连接上，请确认 player-stats 后端正在运行。"
            if isinstance(result, httpx.TimeoutException):
                return "查询失败：统计后端响应超时。"
            if isinstance(result, httpx.HTTPStatusError):
                logger.warning(f"player stats api error: {result.response.status_code} {result.response.text}")
                if result.response.status_code == 401:
                    return "查询失败：统计后端拒绝访问，请在 AstrBot 插件配置里填写插件密钥 api_key。"
                return f"查询失败：统计后端返回 {result.response.status_code}。"
            if isinstance(result, Exception):
                logger.exception(f"query player stats failed: {result}")
                return "查询失败：机器人插件内部错误，请看 AstrBot 日志。"
        return ""

    def _auth_headers(self) -> dict[str, str]:
        api_key = str(self.config.get("api_key", "")).strip()
        if api_key:
            return {"X-Player-Stats-Key": api_key}

        token = str(self.config.get("api_token", "")).strip()
        if not token:
            return {}
        if token.lower().startswith("bearer "):
            token = token[7:].strip()
        return {"Authorization": f"Bearer {token}"}

    def _format_server_stats(self, fallback_server_name: str, data: dict[str, Any]) -> str:
        server_name = data.get("serverName") or fallback_server_name
        broken_count = int(data.get("brokenCount", data.get("digCount", 0)))
        placed_count = int(data.get("placedCount", 0))
        total_count = int(data.get("totalCount", broken_count + placed_count))

        lines = [
            f"{server_name}：",
            f"首次记录：{self._format_time(data.get('firstSeenAt'))}",
            f"统计数据截止于：{self._format_date(data.get('latestLogDate'))}",
            f"破坏方块：{broken_count:,}",
            f"放置方块：{placed_count:,}",
            f"合计行为：{total_count:,}",
        ]

        from_date = data.get("from")
        to_date = data.get("to")
        if from_date or to_date:
            lines.append(f"统计范围：{from_date or '最早'} 至 {to_date or '最新'}")

        return "\n".join(lines)

    def _format_ranking(self, server_name: str, rows: list[dict[str, Any]]) -> str:
        if not rows:
            return f"{server_name}：暂无数据"

        lines = [f"{server_name}："]
        for index, row in enumerate(rows, start=1):
            player_name = row.get("playerName") or "-"
            broken_count = int(row.get("brokenCount", 0))
            placed_count = int(row.get("placedCount", 0))
            total_count = int(row.get("totalCount", broken_count + placed_count))
            lines.append(
                f"{index}. {player_name} 破坏 {broken_count:,} / 放置 {placed_count:,} / 合计 {total_count:,}"
            )
        return "\n".join(lines)

    def _format_public_log_result(self, server: dict[str, str], data: dict[str, Any]) -> str:
        server_name = data.get("serverName") or server.get("serverName") or server.get("serverId") or "-"
        matched_rows = int(data.get("matchedRows") or 0)
        rows = data.get("rows") or []
        if not matched_rows:
            return f"{server_name}：没有匹配日志"

        lines = [f"{server_name}：匹配 {matched_rows:,} 条，显示最近 {len(rows):,} 条"]
        for index, row in enumerate(rows, start=1):
            when = " ".join(part for part in [str(row.get("date") or ""), str(row.get("time") or "")] if part).strip()
            actor = row.get("playerName") or "-"
            action = row.get("action") or "-"
            detail = self._row_detail(row)
            coord = self._row_interaction_coord(row)
            lines.append(f"{index}. {when} {actor} {action} {detail} @ {coord}".strip())
        return "\n".join(lines)

    def _row_detail(self, row: dict[str, Any]) -> str:
        for key in ("detail1", "detail2"):
            value = str(row.get(key) or "").strip()
            if value and value != "-":
                return value
        return "-"

    def _row_interaction_coord(self, row: dict[str, Any]) -> str:
        x = str(row.get("x2") or "-")
        y = str(row.get("y2") or "-")
        z = str(row.get("z2") or "-")
        dimension = str(row.get("dimension2") or "").strip()
        if dimension and dimension != "-":
            return f"{x}, {y}, {z} · {dimension}"
        return f"{x}, {y}, {z}"

    def _parse_coordinate_args(self, raw: str) -> tuple[float, float, float] | None:
        normalized = (raw or "").replace(",", " ").replace("，", " ").strip()
        parts = [part for part in normalized.split() if part]
        if len(parts) != 3:
            return None
        try:
            return (float(parts[0]), float(parts[1]), float(parts[2]))
        except ValueError:
            return None

    def _log_query_usage_text(self) -> str:
        return (
            "用法：/查日志 X Y Z\n"
            "示例：/查日志 -191 -34 750\n"
            "说明：查询的是交互坐标，也就是被点击、破坏或放置的方块坐标。"
        )

    def _consume_log_query_quota(self, game_id: str) -> dict[str, Any]:
        limit = self._public_log_hourly_limit()
        key = (game_id or "").strip().lower()
        now = datetime.now(timezone.utc)
        cutoff = now - timedelta(hours=1)
        bucket = [value for value in self._log_query_usage.get(key, []) if value > cutoff]
        if len(bucket) >= limit:
            reset_seconds = max(60, int((bucket[0] + timedelta(hours=1) - now).total_seconds()))
            self._log_query_usage[key] = bucket
            return {
                "ok": False,
                "limit": limit,
                "remaining": 0,
                "reset_minutes": max(1, (reset_seconds + 59) // 60),
            }
        bucket.append(now)
        self._log_query_usage[key] = bucket
        return {
            "ok": True,
            "limit": limit,
            "remaining": max(0, limit - len(bucket)),
            "reset_minutes": 0,
        }

    def _ranking_limit(self) -> int:
        try:
            return max(1, min(int(self.config.get("ranking_limit", 10)), 20))
        except (TypeError, ValueError):
            return 10

    def _public_log_hourly_limit(self) -> int:
        try:
            return max(1, min(int(self.config.get("log_query_hourly_limit", 5)), 60))
        except (TypeError, ValueError):
            return 5

    def _public_log_result_limit(self) -> int:
        try:
            return max(1, min(int(self.config.get("log_query_result_limit", 8)), 20))
        except (TypeError, ValueError):
            return 8

    def _public_log_recent_days(self) -> int:
        try:
            return max(1, min(int(self.config.get("log_query_recent_days", 7)), 365))
        except (TypeError, ValueError):
            return 7

    def _active_days(self) -> int:
        try:
            return max(1, min(int(self.config.get("active_days", 7)), 90))
        except (TypeError, ValueError):
            return 7

    def _share_ttl_text(self, data: dict[str, Any]) -> str:
        try:
            ttl_minutes = int(data.get("ttlMinutes") or 0)
            if ttl_minutes > 0:
                return self._format_duration_minutes(ttl_minutes)
        except (TypeError, ValueError):
            pass

        expires_at = str(data.get("expiresAt") or "").strip()
        if expires_at:
            try:
                expires = datetime.fromisoformat(expires_at.replace("Z", "+00:00"))
                now = datetime.now(expires.tzinfo or timezone.utc)
                ttl_minutes = max(1, int((expires - now).total_seconds() + 59) // 60)
                return self._format_duration_minutes(ttl_minutes)
            except ValueError:
                pass
        return "1小时"

    def _format_duration_minutes(self, minutes: int) -> str:
        safe_minutes = max(1, int(minutes))
        if safe_minutes % 1440 == 0:
            return f"{safe_minutes // 1440}天"
        if safe_minutes % 60 == 0:
            return f"{safe_minutes // 60}小时"
        if safe_minutes > 60:
            hours = safe_minutes // 60
            rest_minutes = safe_minutes % 60
            return f"{hours}小时{rest_minutes}分钟"
        return f"{safe_minutes}分钟"

    def _format_duration_minutes_spaced(self, minutes: int) -> str:
        safe_minutes = max(1, int(minutes))
        if safe_minutes % 1440 == 0:
            return f"{safe_minutes // 1440} 天"
        if safe_minutes % 60 == 0:
            return f"{safe_minutes // 60} 小时"
        if safe_minutes > 60:
            hours = safe_minutes // 60
            rest_minutes = safe_minutes % 60
            return f"{hours} 小时 {rest_minutes} 分钟"
        return f"{safe_minutes} 分钟"

    def _format_coord_triplet(self, x: float, y: float, z: float) -> str:
        return f"{self._format_coord_value(x)}, {self._format_coord_value(y)}, {self._format_coord_value(z)}"

    def _format_coord_value(self, value: float) -> str:
        if float(value).is_integer():
            return str(int(value))
        return f"{value:g}"

    def _config_bool(self, key: str, default: bool) -> bool:
        value = self.config.get(key, default)
        if isinstance(value, bool):
            return value
        if isinstance(value, str):
            return value.strip().lower() not in {"false", "0", "no", "off", "关闭"}
        return bool(value)

    def _format_date(self, value: str | None) -> str:
        return value or "暂无记录"

    def _format_time(self, value: str | None) -> str:
        if not value:
            return "暂无记录"
        try:
            return datetime.fromisoformat(value.replace("Z", "+00:00")).strftime("%Y-%m-%d %H:%M:%S")
        except ValueError:
            return value

    def _binding_key(self, event: AstrMessageEvent) -> str:
        sender_id = event.get_sender_id()
        return f"player_stats_binding:{sender_id}"

    def _normalize_game_id(self, game_id: str | None) -> str:
        normalized = (game_id or "").strip()
        if len(normalized) >= 2 and normalized[0] == normalized[-1] and normalized[0] in {'"', "'"}:
            normalized = normalized[1:-1].strip()
        return normalized

    async def terminate(self):
        """插件卸载时调用。"""
        if self._xray_group_task:
            self._xray_group_task.cancel()
            try:
                await self._xray_group_task
            except asyncio.CancelledError:
                pass
