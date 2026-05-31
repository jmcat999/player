// 插件基础信息
const PLUGIN_NAME = "PlayerLogger";

const CONFIG_PATH = "plugins/PlayerLogger/config.json";

let config;

// 初始化配置文件
function initConfig() {
    const defaultConfig = {
        events: {
            preJoin: true,    // 开始连接
            join: true,       // 加入游戏
            left: true,       // 离开游戏
            chat: true,       // 聊天消息
            death: true,      // 死亡
            respawn: true,    // 重生
            command: true,    // 命令执行
            changeDim: true,  // 维度切换
            attack: true,      // 攻击实体
            takeItem: true,   // 捡起物品
            dropItem: true,    // 丢弃物品
            consumeTotem: true,    // 消耗图腾
            placeBlock: true,      // 放置方块后
            destroyBlock: true,     // 破坏方块
            openContainer: true,    // 打开容器
            closeContainer: true,    // 关闭容器
            expAdd: true,          // 获得经验
            bedEnter: true,        // 上床睡觉
            fishing: true,          // 钓鱼
            blockInteract: true,  // 方块交互事件
            blockChange: true,  // 方块改变事件
            blockExplode: true,  // 方块爆炸事件
            respawnAnchorExplode: true,  // 重生锚爆炸
            blockExploded: true,         // 方块被爆炸破坏
            fireSpread: true,            // 火焰蔓延
            containerChange: true,  // 容器变化事件
            farmLandDecay: true,  // 耕地退化事件
            useFrameBlock: true,  // 物品展示框使用事件
            liquidFlow: true,     // 液体流动事件
            setArmor: true,  // 玩家改变盔甲栏事件开关
            cmdBlockExecute: true,  // 命令方块执行事件
            redStoneUpdate: true,   // 红石更新事件
            hopperSearch: true,    // 漏斗检测物品事件
            hopperPushOut: true,   // 漏斗输出物品事件
            pistonPush: true,      // 活塞推动事件
            mobDie: true,    // 生物死亡事件
            mobHurt: true,    // 生物受伤事件开关
            projectileHit: true,    // 弹射物击中方块事件
            attackBlock: true,     // 攻击方块事件
            useItem: true,         // 使用物品事件
            useItemOn: true,       // 对方块使用物品事件
            useBucketPlace: true,  // 使用桶放置事件
            useBucketTake: true,   // 使用桶装东西事件
            mobSpawned: true,           // 生物生成事件
            projectileHitEntity: true,  // 弹射物击中实体事件
            entityExplode: true,        // 实体爆炸事件
            witherBossDestroy: true,  // 凋灵破坏方块
            ride: true,               // 骑乘实体
            stepPressurePlate: true,  // 踩压力板
            projectileCreated: true,  // 弹射物创建
            npcCmd: true,             // NPC执行命令
            projectileHitEntity: true,    // 弹射物击中实体
            changeArmorStand: true,       // 操作盔甲架
            entityTransformation: true,   // 实体转变
            consoleCmd: true,      // 控制台命令执行
            scoreChanged: true,    // 计分板变化
            ate: true,              // 玩家食用物品
            effectAdded: true,      // 获得效果
            effectRemoved: true,    // 移除效果
            effectUpdated: true,    // 更新效果
            changeSprinting: true,  // 切换疾跑
            sneak: true,           // 切换潜行
            jump: true,            // 玩家跳跃 
            openContainerScreen: true,   // 打开容器界面
            inventoryChange: true,       // 物品栏变化
            signChange: true,//告示牌
            playerInteractEntity: true   // 玩家与实体交互事件
        },
        logPath: "logs/player_actions",
        // 屏蔽关键词配置，支持正则表达式。
        blockKeywords: []
    };

    // 确保配置文件目录存在
    const configDir = "plugins/PlayerLogger";
    if (!file.exists(configDir)) {
        file.mkdir(configDir);
    }

    // 如果配置文件不存在，创建默认配置
    if (!file.exists(CONFIG_PATH)) {
        file.writeLine(CONFIG_PATH, JSON.stringify(defaultConfig, null, 4));
    }

    // 读取配置文件
    try {
        const configContent = file.readFrom(CONFIG_PATH);
        config = JSON.parse(configContent);
    } catch (e) {
        logger.error("配置文件读取失败，使用默认配置");
        config = defaultConfig;
    }
}



// 修改获取实体名称的函数
function getEntityName(entity) {
    if (!entity) return "未知";
    // 如果是玩家类型，返回玩家名称，否则返回实体类型标识符
    if (entity.isPlayer()) {
        return `玩家${entity.name}`;
    }
    // 注意：这里直接使用entity.type而不是调用未定义的函数
    return entity.type;
}

// ② 新增工具函数：检查日志是否需要被屏蔽
function isLogBlocked(logLine) {
    if (!config || !config.blockKeywords || config.blockKeywords.length === 0) return false;
    for (const pattern of config.blockKeywords) {
        try {
            const regex = new RegExp(pattern);
            if (regex.test(logLine)) {
                return true;
            }
        } catch (e) {
            logger.error(`屏蔽关键词正则表达式错误: ${pattern}，错误：${e.message}`);
        }
    }
    return false;
}

// 整数化坐标输出
function formatPos(pos) {
    if (!pos) return "-,-,-,-";
    return `${Math.floor(pos.x)},${Math.floor(pos.y)},${Math.floor(pos.z)},${getDimensionName(pos.dimid)}`;
}

// 修改获取实体坐标的处理
function getEntityPosString(entity) {
    if (!entity || !entity.pos) return "-,-,-,-";
    return formatPos(entity.pos);
}
// 检查事件是否启用
function isEventEnabled(eventName) {
    return config?.events?.[eventName] ?? true; // 如果配置读取失败，默认启用
}

function getLogPath() {
    return config?.logPath ?? "logs/player_actions";
}

// 工具函数：获取当前日期字符串(YYYY-MM-DD格式)
function getDateString() {
    const date = new Date();
    return `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, '0')}-${String(date.getDate()).padStart(2, '0')}`;
}

// 工具函数：获取当前时间字符串 (HH:mm:ss格式) 
function getTimeString() {
    const date = new Date();
    return `${String(date.getHours()).padStart(2, '0')}:${String(date.getMinutes()).padStart(2, '0')}:${String(date.getSeconds()).padStart(2, '0')}`;
}

// 工具函数：获取维度名称
function getDimensionName(dimid) {
    switch(dimid) {
        case 0: return "主世界";
        case 1: return "下界";
        case 2: return "末地";
        default: return "未知";
    } 
}

// 工具函数：获取死亡原因描述
function getDeathSource(source) {
    if (!source) return "未知原因";
    
    // 获取实体类型名称
    const typeName = source.type;
    // 如果是玩家，返回玩家名称
    if (typeName === "minecraft:player") {
        return `玩家${source.name}`;
    }
    // 返回实体类型
    return typeName;
}

// 工具函数：获取玩家IP 
function getPlayerIP(player) {
    const dv = player.getDevice();
    return dv ? dv.ip : "unknown";
}

// 工具函数：获取位置信息字符串
function getPosString(obj) {
    if (!obj) return "-,-,-,-";
    
    let pos;
    let dimid;
    
    // 根据对象类型获取位置信息
    if (obj.blockPos) {
        // 对于玩家和实体
        pos = obj.blockPos;
        dimid = pos.dimid;
    } else if (obj.pos) {
        // 对于某些实体
        pos = obj.pos;
        dimid = pos.dimid;
    } else if (obj.dim) {
        // 对于方块
        pos = obj;
        dimid = obj.dim;
    } else {
        // 无法获取位置信息
        return "-,-,-,-";
    }
    
    const dim = getDimensionName(dimid);
    return `${pos.x},${pos.y},${pos.z},${dim}`; 
}

function getBlockPosString(block) {
    if (!block) return "-,-,-,-";
    return `${block.pos.x},${block.pos.y},${block.pos.z},${getDimensionName(block.pos.dimid)}`;
}

// 记录玩家行为到CSV文件（带坐标）
function logPlayerAction(player, action, detail1, detail2 = "-", pos2 = "-,-,-,-") {
    const dateStr = getDateString();
    const timeStr = getTimeString();
    const posStr = getPosString(player);
    const logLine = `${dateStr},${timeStr},${player.realName},${action},${posStr},${pos2},${detail1},${detail2}`;
    
    writeLogToFile(logLine);
}

// 记录玩家行为到CSV文件（不带坐标）
function logPlayerActionNoPos(player, action, detail1, detail2 = "-") {
    const dateStr = getDateString();
    const timeStr = getTimeString();
    const logLine = `${dateStr},${timeStr},${player.name},${action},-,-,-,-,-,-,-,-,${detail1},${detail2}`;
    
    writeLogToFile(logLine);
}

// 写入日志到文件
function writeLogToFile(logLine) {
    // 检查日志是否匹配屏蔽关键词，若匹配则不写入日志
    if (isLogBlocked(logLine)) {
        return;
    }
    
    const dateStr = getDateString();
    const fileName = `${getLogPath()}_${dateStr}.csv`;
    // 确保日志目录存在
    const logDir = fileName.substring(0, fileName.lastIndexOf('/'));
    if (!file.exists(logDir)) {
        file.mkdir(logDir);
    }
    
    // 如果文件不存在，先创建表头
    if (!file.exists(fileName)) {
        file.writeLine(fileName, "日期,时间,玩家名称,行为,X坐标,Y坐标,Z坐标,维度," + 
                                "X坐标2,Y坐标2,Z坐标2,维度2,详细信息1,详细信息2");
    }
    
    // 写入日志内容
    file.writeLine(fileName, logLine);
}

// 在插件加载时初始化配置
initConfig();

// 注册事件监听器 

mc.listen("onChat", (player, msg) => {
    if (!isEventEnabled("chat")) return;
    logPlayerAction(player, "发送聊天", msg);
});

mc.listen("onPlayerCmd", (player, cmd) => {
    if (!isEventEnabled("command")) return;
    logPlayerAction(player, "执行命令", cmd);
});

// 修改破坏方块监听
mc.listen("onDestroyBlock", (player, block) => {
    if (!isEventEnabled("destroyBlock")) return;
    
    try {
        // 获取方块的类型和位置
        const blockType = block.type;
        const blockPos = getBlockPosString(block);
        
        // 记录玩家破坏方块的信息，使用pos2记录被破坏方块的位置
        logPlayerAction(player, "破坏方块", blockType, "-", blockPos);
    } catch (e) {
        logger.error(`破坏方块事件处理错误: ${e}`);
    }
}) 

// 放置方块监听
mc.listen("afterPlaceBlock", (player, block) => {
    if (!isEventEnabled("placeBlock")) return;
    
    try {
        // 获取方块的类型和位置
        const blockType = block.type;
        const blockPos = getBlockPosString(block);
        
        // 记录玩家放置方块的信息，使用pos2记录放置方块的位置
        logPlayerAction(player, "放置方块", blockType, "-", blockPos);
    } catch (e) {
        logger.error(`放置方块事件处理错误: ${e}`);
    }
});

// 添加使用桶放置监听
mc.listen("onUseBucketPlace", (player, item, block) => {
    if (!isEventEnabled("useBucketPlace")) return;
    
    try {
        if (!player || !player.realName) return;
        // 获取方块位置信息
        const blockPos = `${block.pos.x},${block.pos.y},${block.pos.z},${getDimensionName(block.pos.dimid)}`;
        
        // 记录使用桶放置事件
        const dateStr = getDateString();
        const timeStr = getTimeString();
        const logLine = `${dateStr},${timeStr},${player.name},使用桶放置,${blockPos},-,-,-,${block.type},${item.type}`;
        
        writeLogToFile(logLine);
        
    } catch (e) {
        logger.error(`使用桶放置事件处理错误: ${e}`);
    }
    
    return true;
});

// 告示牌修改监听
mc.listen("onBlockChanged", (beforeBlock, afterBlock) => {
    // 检查是否启用
    if (!isEventEnabled("signChange")) return;
    
    try {
        // 检查是否是告示牌变更
        if (beforeBlock.type !== afterBlock.type || !afterBlock.type.includes('sign')) return;
        
        // 延迟执行以确保NBT数据已更新
        setTimeout(() => {
            try {
                // 获取告示牌的NBT数据
                const blockEntity = afterBlock.getBlockEntity();
                if (!blockEntity) return;
                
                const blockEntityNbt = blockEntity.getNbt();
                if (!blockEntityNbt) return;

                function processSignText(text) {
                    if(!text) return "空";
                    // 将换行符替换为空格
                    return text.replace(/\n/g, " ").replace(/\r/g, " ");
                }
                
                // 读取告示牌正面文本
                const frontText = processSignText(blockEntityNbt.getTag('FrontText')?.getData('Text'));
                // 读取并处理告示牌背面文本
                const backText = processSignText(blockEntityNbt.getTag('BackText')?.getData('Text'));
                
                // 获取修改者XUID并转换为玩家名
                const textOwnerXuid = blockEntityNbt.getTag('FrontText')?.getData('TextOwner') || "";
                const playerName = textOwnerXuid ? (data.xuid2name(textOwnerXuid) || "未知玩家") : "未知玩家";
                
                // 记录告示牌修改事件
                const dateStr = getDateString();
                const timeStr = getTimeString();
                const signPos = `${afterBlock.pos.x},${afterBlock.pos.y},${afterBlock.pos.z},${getDimensionName(afterBlock.pos.dimid)}`;
                const logLine = `${dateStr},${timeStr},${playerName},修改告示牌,${signPos},-,-,-,${afterBlock.type},正面:"${frontText}",背面:"${backText}"`;
                
                writeLogToFile(logLine);
                
            } catch (e) {
                logger.error(`告示牌NBT读取错误: ${e.message}`);
            }
        }, 10); // 延迟10ms执行
        
    } catch (e) {
        logger.error(`告示牌修改事件处理错误: ${e.message}`);
    }
});
