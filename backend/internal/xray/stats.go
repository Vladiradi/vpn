package xray

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// TrafficStat holds uplink/downlink traffic for a single device.
type TrafficStat struct {
	UUID     string
	Uplink   int64
	Downlink int64
}

// StatsResponse is the JSON returned by `xray api statsquery`.
type StatsResponse struct {
	Stat []struct {
		Name  string `json:"name"`
		Value int64  `json:"value"`
	} `json:"stat"`
}

// ParseStatsResponse parses the raw output of `xray api statsquery`.
// It expects names like "user>>>uuid@traffic->uplink".
func ParseStatsResponse(raw string) ([]TrafficStat, error) {
	var resp StatsResponse
	if err := json.Unmarshal([]byte(raw), &resp); err != nil {
		return nil, fmt.Errorf("unmarshal stats: %w", err)
	}

	stats := make(map[string]*TrafficStat)
	for _, s := range resp.Stat {
		uuid, direction, ok := parseStatName(s.Name)
		if !ok {
			continue
		}
		st, exists := stats[uuid]
		if !exists {
			st = &TrafficStat{UUID: uuid}
			stats[uuid] = st
		}
		switch direction {
		case "uplink":
			st.Uplink = s.Value
		case "downlink":
			st.Downlink = s.Value
		}
	}

	result := make([]TrafficStat, 0, len(stats))
	for _, st := range stats {
		result = append(result, *st)
	}
	return result, nil
}

// parseStatName extracts UUID and direction from xray stats names like:
// "user>>>550e8400-e29b-41d4-a716-446655440000@traffic->uplink"
func parseStatName(name string) (uuid, direction string, ok bool) {
	if !strings.HasPrefix(name, "user>>>") {
		return "", "", false
	}
	name = strings.TrimPrefix(name, "user>>>")
	parts := strings.Split(name, "@")
	if len(parts) != 2 {
		return "", "", false
	}
	uuid = parts[0]
	suffix := parts[1]
	if !strings.HasPrefix(suffix, "traffic->") {
		return "", "", false
	}
	direction = strings.TrimPrefix(suffix, "traffic->")
	return uuid, direction, true
}

// AggregateTraffic sums uplink/downlink across all stats.
func AggregateTraffic(stats []TrafficStat) (uplink, downlink int64) {
	for _, s := range stats {
		uplink += s.Uplink
		downlink += s.Downlink
	}
	return
}

// FormatInt64 returns a string representation of an int64 value.
// It is a helper for CLI argument building.
func FormatInt64(v int64) string {
	return strconv.FormatInt(v, 10)
}
