package importer

import (
	"strings"
	"testing"
	"time"
)

func TestParserCountsChineseBlockActions(t *testing.T) {
	location := time.FixedZone("CST", 8*60*60)
	input := strings.NewReader(strings.Join([]string{
		"日期,时间,玩家,行为,x,y,z,维度,维度2,a,b,c,方块",
		"2026-04-01,19:19:00,Alex,破坏方块,0,10,0,overworld,overworld,,,,minecraft:deepslate_diamond_ore",
		"2026-04-01,19:20:00,Alex,破坏方块,1,10,0,overworld,overworld,,,,minecraft:diamond_ore",
		"2026-04-01,19:21:00,Alex,放置方块,1,64,0,overworld,overworld,,,,minecraft:oak_sapling",
	}, "\n"))

	parsed, err := newParser(location).parse(input)
	if err != nil {
		t.Fatal(err)
	}
	if parsed.rowCount != 3 {
		t.Fatalf("rowCount = %d, want 3", parsed.rowCount)
	}
	key := statKey{statDate: time.Date(2026, 4, 1, 0, 0, 0, 0, location), playerName: "Alex"}
	if got := parsed.stats[key].broken; got != 2 {
		t.Fatalf("broken count = %d, want 2", got)
	}
	if got := parsed.stats[key].placed; got != 1 {
		t.Fatalf("placed count = %d, want 1", got)
	}
	if got := parsed.oreCounts[typedPlayerKey{playerName: "Alex", typ: "DEEPSLATE_DIAMOND_ORE"}]; got != 1 {
		t.Fatalf("deepslate diamond count = %d, want 1", got)
	}
	if got := parsed.oreCounts[typedPlayerKey{playerName: "Alex", typ: "DIAMOND_ORE"}]; got != 1 {
		t.Fatalf("diamond count = %d, want 1", got)
	}
	if got := parsed.saplingCounts[typedPlayerKey{playerName: "Alex", typ: "OAK_SAPLING"}]; got != 1 {
		t.Fatalf("oak sapling count = %d, want 1", got)
	}
}
