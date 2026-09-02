package xray

import "testing"

func TestParseStatsResponse(t *testing.T) {
	raw := `{"stat":[{"name":"user>>>550e8400-e29b-41d4-a716-446655440000@traffic->uplink","value":1024},{"name":"user>>>550e8400-e29b-41d4-a716-446655440000@traffic->downlink","value":2048},{"name":"inbound>>>vless->traffic->uplink","value":999}]}`

	stats, err := ParseStatsResponse(raw)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(stats) != 1 {
		t.Fatalf("expected 1 stat, got %d", len(stats))
	}
	st := stats[0]
	if st.UUID != "550e8400-e29b-41d4-a716-446655440000" {
		t.Errorf("unexpected uuid: %s", st.UUID)
	}
	if st.Uplink != 1024 {
		t.Errorf("unexpected uplink: %d", st.Uplink)
	}
	if st.Downlink != 2048 {
		t.Errorf("unexpected downlink: %d", st.Downlink)
	}
}
