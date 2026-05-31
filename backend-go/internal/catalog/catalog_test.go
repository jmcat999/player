package catalog

import "testing"

func TestOreBlockIDsMatchJavaCatalog(t *testing.T) {
	rejected := []string{
		"minecraft:lit_redstone_ore",
		"minecraft:lit_deepslate_redstone_ore",
		"minecraft:nether_quartz_ore",
	}
	for _, blockID := range rejected {
		if oreType, ok := OreTypeFromBlock(blockID); ok {
			t.Fatalf("%s mapped to %s, Java catalog does not include this alias", blockID, oreType)
		}
	}

	accepted := map[string]string{
		"minecraft:redstone_ore":           "REDSTONE_ORE",
		"minecraft:deepslate_redstone_ore": "DEEPSLATE_REDSTONE_ORE",
		"minecraft:quartz_ore":             "QUARTZ_ORE",
	}
	for blockID, want := range accepted {
		got, ok := OreTypeFromBlock(blockID)
		if !ok || got != want {
			t.Fatalf("%s mapped to (%s, %v), want %s", blockID, got, ok, want)
		}
	}
}

func TestSaplingLabelsMatchJavaCatalog(t *testing.T) {
	if got := SaplingLabel("MANGROVE_PROPAGULE"); got != "红树木繁殖体" {
		t.Fatalf("MANGROVE_PROPAGULE label = %q, want Java label", got)
	}
}
