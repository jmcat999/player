package catalog

import (
	"strings"
)

type BlockType struct {
	Type     string
	Label    string
	BlockIDs []string
}

type MilestoneType struct {
	Type        string
	Label       string
	FoundText   string
	MissingText string
}

var OreTypes = []BlockType{
	{Type: "COAL_ORE", Label: "煤矿石", BlockIDs: []string{"minecraft:coal_ore"}},
	{Type: "DEEPSLATE_COAL_ORE", Label: "深层煤矿石", BlockIDs: []string{"minecraft:deepslate_coal_ore"}},
	{Type: "IRON_ORE", Label: "铁矿石", BlockIDs: []string{"minecraft:iron_ore"}},
	{Type: "DEEPSLATE_IRON_ORE", Label: "深层铁矿石", BlockIDs: []string{"minecraft:deepslate_iron_ore"}},
	{Type: "LAPIS_ORE", Label: "青金石矿石", BlockIDs: []string{"minecraft:lapis_ore"}},
	{Type: "DEEPSLATE_LAPIS_ORE", Label: "深层青金石矿石", BlockIDs: []string{"minecraft:deepslate_lapis_ore"}},
	{Type: "COPPER_ORE", Label: "铜矿石", BlockIDs: []string{"minecraft:copper_ore"}},
	{Type: "DEEPSLATE_COPPER_ORE", Label: "深层铜矿石", BlockIDs: []string{"minecraft:deepslate_copper_ore"}},
	{Type: "GOLD_ORE", Label: "金矿石", BlockIDs: []string{"minecraft:gold_ore"}},
	{Type: "DEEPSLATE_GOLD_ORE", Label: "深层金矿石", BlockIDs: []string{"minecraft:deepslate_gold_ore"}},
	{Type: "REDSTONE_ORE", Label: "红石矿石", BlockIDs: []string{"minecraft:redstone_ore"}},
	{Type: "DEEPSLATE_REDSTONE_ORE", Label: "深层红石矿石", BlockIDs: []string{"minecraft:deepslate_redstone_ore"}},
	{Type: "DIAMOND_ORE", Label: "钻石矿石", BlockIDs: []string{"minecraft:diamond_ore"}},
	{Type: "DEEPSLATE_DIAMOND_ORE", Label: "深层钻石矿石", BlockIDs: []string{"minecraft:deepslate_diamond_ore"}},
	{Type: "EMERALD_ORE", Label: "绿宝石矿石", BlockIDs: []string{"minecraft:emerald_ore"}},
	{Type: "DEEPSLATE_EMERALD_ORE", Label: "深层绿宝石矿石", BlockIDs: []string{"minecraft:deepslate_emerald_ore"}},
	{Type: "QUARTZ_ORE", Label: "下界石英矿石", BlockIDs: []string{"minecraft:quartz_ore"}},
	{Type: "NETHER_GOLD_ORE", Label: "下界金矿石", BlockIDs: []string{"minecraft:nether_gold_ore"}},
	{Type: "ANCIENT_DEBRIS", Label: "远古残骸", BlockIDs: []string{"minecraft:ancient_debris"}},
}

var WoodTypes = []BlockType{
	{Type: "OAK_LOG", Label: "橡木原木", BlockIDs: []string{"minecraft:oak_log"}},
	{Type: "SPRUCE_LOG", Label: "云杉原木", BlockIDs: []string{"minecraft:spruce_log"}},
	{Type: "BIRCH_LOG", Label: "白桦原木", BlockIDs: []string{"minecraft:birch_log"}},
	{Type: "JUNGLE_LOG", Label: "丛林原木", BlockIDs: []string{"minecraft:jungle_log"}},
	{Type: "ACACIA_LOG", Label: "金合欢原木", BlockIDs: []string{"minecraft:acacia_log"}},
	{Type: "DARK_OAK_LOG", Label: "深色橡木原木", BlockIDs: []string{"minecraft:dark_oak_log"}},
	{Type: "MANGROVE_LOG", Label: "红树原木", BlockIDs: []string{"minecraft:mangrove_log"}},
	{Type: "CHERRY_LOG", Label: "樱花原木", BlockIDs: []string{"minecraft:cherry_log"}},
	{Type: "PALE_OAK_LOG", Label: "苍白橡木原木", BlockIDs: []string{"minecraft:pale_oak_log"}},
	{Type: "CRIMSON_STEM", Label: "绯红菌柄", BlockIDs: []string{"minecraft:crimson_stem"}},
	{Type: "WARPED_STEM", Label: "诡异菌柄", BlockIDs: []string{"minecraft:warped_stem"}},
}

var SaplingTypes = []BlockType{
	{Type: "OAK_SAPLING", Label: "橡树树苗", BlockIDs: []string{"minecraft:oak_sapling"}},
	{Type: "SPRUCE_SAPLING", Label: "云杉树苗", BlockIDs: []string{"minecraft:spruce_sapling"}},
	{Type: "BIRCH_SAPLING", Label: "白桦树苗", BlockIDs: []string{"minecraft:birch_sapling"}},
	{Type: "JUNGLE_SAPLING", Label: "丛林树苗", BlockIDs: []string{"minecraft:jungle_sapling"}},
	{Type: "ACACIA_SAPLING", Label: "金合欢树苗", BlockIDs: []string{"minecraft:acacia_sapling"}},
	{Type: "DARK_OAK_SAPLING", Label: "深色橡树树苗", BlockIDs: []string{"minecraft:dark_oak_sapling"}},
	{Type: "MANGROVE_PROPAGULE", Label: "红树木繁殖体", BlockIDs: []string{"minecraft:mangrove_propagule"}},
	{Type: "CHERRY_SAPLING", Label: "樱花树苗", BlockIDs: []string{"minecraft:cherry_sapling"}},
	{Type: "PALE_OAK_SAPLING", Label: "苍白橡树树苗", BlockIDs: []string{"minecraft:pale_oak_sapling"}},
	{Type: "CRIMSON_FUNGUS", Label: "绯红菌", BlockIDs: []string{"minecraft:crimson_fungus"}},
	{Type: "WARPED_FUNGUS", Label: "诡异菌", BlockIDs: []string{"minecraft:warped_fungus"}},
}

var MilestoneTypes = []MilestoneType{
	{Type: "FIRST_WOOD", Label: "第一次砍到木头", FoundText: "你第一次砍下木头。", MissingText: "木头暂时还没有记录。"},
	{Type: "FIRST_STONE", Label: "第一次挖到石头", FoundText: "你第一次挖到石头。", MissingText: "石头暂时还没有记录。"},
	{Type: "FIRST_COAL", Label: "第一次发现煤炭", FoundText: "你第一次发现煤炭。", MissingText: "煤炭暂时还没有记录。"},
	{Type: "FIRST_IRON", Label: "第一次挖到铁矿", FoundText: "你第一次挖到铁矿。", MissingText: "还没有挖到铁矿。"},
	{Type: "FIRST_DIAMOND", Label: "第一次挖到钻石", FoundText: "你第一次挖到钻石。", MissingText: "还没有挖到钻石。"},
	{Type: "FIRST_ANCIENT_DEBRIS", Label: "第一次找到远古残骸", FoundText: "你第一次找到远古残骸。", MissingText: "远古残骸暂时还没有记录。"},
}

var (
	oreByBlockID       = indexBlockIDs(OreTypes)
	woodByBlockID      = indexBlockIDs(WoodTypes)
	saplingByBlockID   = indexBlockIDs(SaplingTypes)
	oreLabelByType     = indexLabels(OreTypes)
	woodLabelByType    = indexLabels(WoodTypes)
	saplingLabelByType = indexLabels(SaplingTypes)
)

func NormalizeBlockID(raw string) string {
	value := strings.TrimSpace(strings.ToLower(raw))
	if value == "" || value == "-" {
		return ""
	}
	if strings.Contains(value, ":") {
		return value
	}
	return "minecraft:" + value
}

func OreTypeFromBlock(raw string) (string, bool) {
	item, ok := oreByBlockID[NormalizeBlockID(raw)]
	if !ok {
		return "", false
	}
	return item.Type, true
}

func WoodTypeFromBlock(raw string) (string, bool) {
	item, ok := woodByBlockID[NormalizeBlockID(raw)]
	if !ok {
		return "", false
	}
	return item.Type, true
}

func SaplingTypeFromBlock(raw string) (string, bool) {
	item, ok := saplingByBlockID[NormalizeBlockID(raw)]
	if !ok {
		return "", false
	}
	return item.Type, true
}

func MilestoneTypeFromDestroyedBlock(raw string) (string, bool) {
	blockID := NormalizeBlockID(raw)
	if blockID == "" {
		return "", false
	}
	if _, ok := woodByBlockID[blockID]; ok {
		return "FIRST_WOOD", true
	}
	if blockID == "minecraft:stone" || blockID == "minecraft:deepslate" {
		return "FIRST_STONE", true
	}
	oreType, ok := OreTypeFromBlock(blockID)
	if !ok {
		return "", false
	}
	switch oreType {
	case "COAL_ORE", "DEEPSLATE_COAL_ORE":
		return "FIRST_COAL", true
	case "IRON_ORE", "DEEPSLATE_IRON_ORE":
		return "FIRST_IRON", true
	case "DIAMOND_ORE", "DEEPSLATE_DIAMOND_ORE":
		return "FIRST_DIAMOND", true
	case "ANCIENT_DEBRIS":
		return "FIRST_ANCIENT_DEBRIS", true
	default:
		return "", false
	}
}

func OreLabel(oreType string) string {
	return labelOrType(oreLabelByType, oreType)
}

func WoodLabel(woodType string) string {
	return labelOrType(woodLabelByType, woodType)
}

func SaplingLabel(saplingType string) string {
	return labelOrType(saplingLabelByType, saplingType)
}

func IsDiamond(oreType string) bool {
	return oreType == "DIAMOND_ORE" || oreType == "DEEPSLATE_DIAMOND_ORE"
}

func IsEmerald(oreType string) bool {
	return oreType == "EMERALD_ORE" || oreType == "DEEPSLATE_EMERALD_ORE"
}

func IsAncientDebris(oreType string) bool {
	return oreType == "ANCIENT_DEBRIS"
}

func IsRareOre(oreType string) bool {
	return IsDiamond(oreType) || IsEmerald(oreType) || IsAncientDebris(oreType)
}

func IsTrackingTargetOre(oreType string) bool {
	return IsDiamond(oreType) || IsAncientDebris(oreType)
}

func indexBlockIDs(items []BlockType) map[string]BlockType {
	result := make(map[string]BlockType)
	for _, item := range items {
		for _, blockID := range item.BlockIDs {
			result[NormalizeBlockID(blockID)] = item
		}
	}
	return result
}

func indexLabels(items []BlockType) map[string]string {
	result := make(map[string]string)
	for _, item := range items {
		result[item.Type] = item.Label
	}
	return result
}

func labelOrType(labels map[string]string, typ string) string {
	if label, ok := labels[typ]; ok {
		return label
	}
	return typ
}
