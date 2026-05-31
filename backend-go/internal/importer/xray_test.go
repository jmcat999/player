package importer

import (
	"testing"
	"time"
)

func TestCollectMetricsCountsDiamondAndAncientDebrisLikeJava(t *testing.T) {
	base := time.Date(2026, 5, 23, 7, 45, 0, 0, time.UTC)
	events := []xrayEvent{
		{
			row:        LogQueryRow{Dimension2: "overworld"},
			happenedAt: base,
			oreType:    "DEEPSLATE_DIAMOND_ORE",
			point:      point3{x: 0, y: 20, z: 0},
			hasPoint:   true,
		},
		{
			row:        LogQueryRow{Dimension2: "overworld"},
			happenedAt: base.Add(time.Second),
			oreType:    "DEEPSLATE_DIAMOND_ORE",
			rareOre:    true,
			point:      point3{x: 1, y: -54, z: 0},
			hasPoint:   true,
		},
		{
			row:        LogQueryRow{Dimension2: "nether"},
			happenedAt: base.Add(2 * time.Second),
			oreType:    "ANCIENT_DEBRIS",
			point:      point3{x: 2, y: 40, z: 0},
			hasPoint:   true,
		},
	}

	metrics := collectMetrics(events)
	if metrics.diamondOreBreaks != 2 {
		t.Fatalf("diamondOreBreaks = %d, want 2", metrics.diamondOreBreaks)
	}
	if metrics.ancientDebrisBreaks != 1 {
		t.Fatalf("ancientDebrisBreaks = %d, want 1", metrics.ancientDebrisBreaks)
	}
	if metrics.rareOreBreaks != 3 {
		t.Fatalf("rareOreBreaks = %d, want 3", metrics.rareOreBreaks)
	}
	if metrics.suspiciousRareOreBreaks != 1 {
		t.Fatalf("suspiciousRareOreBreaks = %d, want 1", metrics.suspiciousRareOreBreaks)
	}
	if got := metrics.oreCounts["DEEPSLATE_DIAMOND_ORE"]; got != 2 {
		t.Fatalf("oreCounts[DEEPSLATE_DIAMOND_ORE] = %d, want 2", got)
	}
	if got := metrics.rareOreCounts["DEEPSLATE_DIAMOND_ORE"]; got != 2 {
		t.Fatalf("rareOreCounts[DEEPSLATE_DIAMOND_ORE] = %d, want 2", got)
	}
	if got := metrics.rareOreCounts["ANCIENT_DEBRIS"]; got != 1 {
		t.Fatalf("rareOreCounts[ANCIENT_DEBRIS] = %d, want 1", got)
	}
	if len(metrics.rareRows) != int(metrics.rareOreBreaks) {
		t.Fatalf("rareRows len = %d, rareOreBreaks = %d", len(metrics.rareRows), metrics.rareOreBreaks)
	}
	if metrics.peakRareOreCount != 3 {
		t.Fatalf("peakRareOreCount = %d, want 3", metrics.peakRareOreCount)
	}
	if metrics.suspiciousPeakRareOreCount != 1 {
		t.Fatalf("suspiciousPeakRareOreCount = %d, want 1", metrics.suspiciousPeakRareOreCount)
	}
}

func TestRecentTunnelEventsStopsAtInterruptedPath(t *testing.T) {
	base := time.Date(2026, 5, 23, 7, 45, 0, 0, time.UTC)
	events := []xrayEvent{
		tunnelEvent(base, 0, -54, 0),
		{
			row:        LogQueryRow{Dimension2: "overworld"},
			happenedAt: base.Add(time.Second),
			point:      point3{x: 1, y: -54, z: 0},
			hasPoint:   true,
		},
		tunnelEvent(base.Add(2*time.Second), 2, -54, 0),
	}
	target := xrayEvent{
		row:        LogQueryRow{Dimension2: "overworld"},
		happenedAt: base.Add(3 * time.Second),
		oreType:    "DEEPSLATE_DIAMOND_ORE",
		rareOre:    true,
		point:      point3{x: 3, y: -54, z: 0},
		hasPoint:   true,
	}

	window := recentTunnelEvents(events, target, 180)
	if len(window) != 1 {
		t.Fatalf("recentTunnelEvents len = %d, want 1", len(window))
	}
	if window[0].point.x != 2 {
		t.Fatalf("recentTunnelEvents kept x = %.0f, want 2", window[0].point.x)
	}
}

func tunnelEvent(t time.Time, x, y, z float64) xrayEvent {
	return xrayEvent{
		row:         LogQueryRow{Dimension2: "overworld"},
		happenedAt:  t,
		blockID:     "minecraft:deepslate",
		tunnelBlock: true,
		point:       point3{x: x, y: y, z: z},
		hasPoint:    true,
	}
}
