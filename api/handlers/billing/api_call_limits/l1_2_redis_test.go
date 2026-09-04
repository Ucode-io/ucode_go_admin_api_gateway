package apilimits

import (
	"testing"

	"ucode/ucode_go_api_gateway/config"
)

func TestUsageFieldRoundTrip(t *testing.T) {
	cases := []usageKey{
		{source: "client", method: "GET", route: "/v2/items/:collection", collection: "orders"},
		{source: "admin", method: "POST", route: "/v1/pricing/all", collection: ""},
		{source: "client", method: "DELETE", route: "/v2/items/:collection", collection: "we|ird"},
	}

	for _, in := range cases {
		source, method, route, collection, ok := parseUsageField(in.field())
		if !ok {
			t.Fatalf("field %q did not parse", in.field())
		}
		if source != in.source || method != in.method || route != in.route {
			t.Errorf("round trip lost data: got %q/%q/%q want %q/%q/%q",
				source, method, route, in.source, in.method, in.route)
		}
		if want := sanitize(in.collection); collection != want {
			t.Errorf("collection: got %q want %q", collection, want)
		}
	}
}

func TestParseUsageFieldRejectsTotal(t *testing.T) {
	if _, _, _, _, ok := parseUsageField(config.KeyUsageTotalField); ok {
		t.Fatal("the total field must not parse as a detail field")
	}
}

func TestTrackerDetailsSumToTotal(t *testing.T) {
	tr := NewTracker(nil, 0)

	add := func(collection string, n int) {
		for i := 0; i < n; i++ {
			tr.add(usageKey{
				projectID: "p1", source: "client", method: "GET",
				route: "/v2/items/:collection", collection: collection,
			})
		}
	}
	add("orders", 5)
	add("users", 3)
	tr.add(usageKey{projectID: "p2", source: "admin", method: "POST", route: "/v1/fare"})

	batch := tr.take()

	var p1 int64
	for k, v := range batch {
		if k.projectID == "p1" {
			p1 += v
		}
	}
	if p1 != 8 {
		t.Errorf("p1 total: got %d want 8", p1)
	}
	if len(batch) != 3 {
		t.Errorf("distinct keys: got %d want 3", len(batch))
	}
	if got := tr.take(); got != nil {
		t.Errorf("take() must empty the map, got %v", got)
	}
}

func TestCollectDetailsFoldsTailIntoOther(t *testing.T) {
	fields := map[string]string{config.KeyUsageTotalField: "0"}
	var want int64

	for i := 0; i < maxDetailRowsPerFlush+25; i++ {
		k := usageKey{source: "client", method: "GET", route: "/v2/items/:collection",
			collection: string(rune('a'+i%26)) + string(rune('a'+i/26))}
		fields[k.field()] = "2"
		want += 2
	}

	details, sent := collectDetails(fields)

	if len(details) != maxDetailRowsPerFlush+1 {
		t.Errorf("rows: got %d want %d", len(details), maxDetailRowsPerFlush+1)
	}
	if len(sent) != maxDetailRowsPerFlush+25 {
		t.Errorf("sent map must cover every consumed field: got %d", len(sent))
	}

	var got int64
	for _, d := range details {
		got += d.GetCount()
	}
	if got != want {
		t.Errorf("folding lost counts: got %d want %d", got, want)
	}
}
