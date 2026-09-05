package apilimits

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"ucode/ucode_go_api_gateway/config"
)

const testBucket = "2026-09-04 14:30:00"

func TestUsageFieldRoundTrip(t *testing.T) {
	cases := []usageKey{
		{source: "client", authType: "api_key", actorID: "k-1", actorName: "Mobile App", method: "GET", route: "/v2/items/:collection", collection: "orders"},
		{source: "admin", authType: "bearer", actorID: "u-7", method: "POST", route: "/v1/pricing/all", collection: ""},
		{source: "client", authType: "", method: "DELETE", route: "/v2/items/:collection", collection: "we|ird"},
	}

	for _, in := range cases {
		got, ok := parseUsageField(in.field(testBucket))
		if !ok {
			t.Fatalf("field %q did not parse", in.field(testBucket))
		}
		if got.bucket != testBucket {
			t.Errorf("bucket: got %q want %q", got.bucket, testBucket)
		}
		if got.source != in.source || got.authType != in.authType ||
			got.actorID != in.actorID || got.actorName != in.actorName ||
			got.method != in.method || got.route != in.route {
			t.Errorf("round trip lost data: %+v vs %+v", got, in)
		}
		if want := sanitize(in.collection); got.collection != want {
			t.Errorf("collection: got %q want %q", got.collection, want)
		}
	}
}

func TestParseUsageFieldRejectsTotal(t *testing.T) {
	if _, ok := parseUsageField(config.KeyUsageTotalField); ok {
		t.Fatal("the total field must not parse as a detail field")
	}
}

func TestTrackerDetailsSumToTotal(t *testing.T) {
	tr := NewTracker(nil, 0)

	add := func(collection string, n int) {
		for i := 0; i < n; i++ {
			tr.add(usageKey{
				projectID: "p1", source: "client", authType: "api_key",
				method: "GET", route: "/v2/items/:collection", collection: collection,
			})
		}
	}
	add("orders", 5)
	add("users", 3)
	tr.add(usageKey{projectID: "p2", source: "admin", authType: "bearer", method: "POST", route: "/v1/fare"})

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

// Auth type must not be collapsed away: the same route reached with a key and
// with a user token has to stay two separate rows.
func TestTrackerKeepsAuthTypesApart(t *testing.T) {
	tr := NewTracker(nil, 0)
	base := usageKey{projectID: "p1", source: "client", method: "GET", route: "/v2/items/:collection", collection: "deal"}

	byKey := base
	byKey.authType = config.UsageAuthApiKey
	byToken := base
	byToken.authType = config.UsageAuthBearer

	tr.add(byKey)
	tr.add(byKey)
	tr.add(byToken)

	batch := tr.take()
	if len(batch) != 2 {
		t.Fatalf("expected 2 rows, got %d: %v", len(batch), batch)
	}
	if batch[byKey] != 2 || batch[byToken] != 1 {
		t.Errorf("counts split wrong: key=%d token=%d", batch[byKey], batch[byToken])
	}
}

func TestGroupByBucketSplitsIntervals(t *testing.T) {
	early := usageKey{source: "client", authType: "api_key", method: "GET", route: "/v2/items/:collection", collection: "deal"}
	late := early

	fields := map[string]string{
		config.KeyUsageTotalField:          "9",
		early.field("2026-09-04 14:30:00"): "5",
		late.field("2026-09-04 14:45:00"):  "4",
		"garbage":                          "1",
	}

	grouped := groupByBucket(fields)
	if len(grouped) != 2 {
		t.Fatalf("expected 2 buckets, got %d", len(grouped))
	}
	if n := grouped["2026-09-04 14:30:00"][0].count; n != 5 {
		t.Errorf("14:30 bucket: got %d want 5", n)
	}
	if n := grouped["2026-09-04 14:45:00"][0].count; n != 4 {
		t.Errorf("14:45 bucket: got %d want 4", n)
	}
}

func TestDetailRowsFoldTailIntoOther(t *testing.T) {
	var (
		entries []bucketEntry
		want    int64
	)
	for i := 0; i < maxDetailRowsPerFlush+25; i++ {
		k := usageKey{
			source: "client", authType: "api_key", method: "GET",
			route: "/v2/items/:collection", collection: fmt.Sprintf("c%d", i),
		}
		parsed, _ := parseUsageField(k.field(testBucket))
		entries = append(entries, bucketEntry{field: k.field(testBucket), usageField: parsed, count: 2})
		want += 2
	}

	details, sent := detailRows(testBucket, entries)

	if len(details) != maxDetailRowsPerFlush+1 {
		t.Errorf("rows: got %d want %d", len(details), maxDetailRowsPerFlush+1)
	}
	if len(sent) != maxDetailRowsPerFlush+25 {
		t.Errorf("sent map must cover every consumed field: got %d", len(sent))
	}

	var got int64
	for _, d := range details {
		got += d.GetCount()
		if d.GetBucket() != testBucket {
			t.Errorf("row lost its bucket: %q", d.GetBucket())
		}
	}
	if got != want {
		t.Errorf("folding lost counts: got %d want %d", got, want)
	}
}

func TestHourOfTruncatesBucket(t *testing.T) {
	if got := hourOf("2026-09-04 14:45:00"); got != "2026-09-04 14:00:00" {
		t.Errorf("hourOf: got %q want %q", got, "2026-09-04 14:00:00")
	}
	// An unparseable bucket must still yield a usable hour rather than "".
	if got := hourOf("nonsense"); got == "" {
		t.Error("hourOf must fall back to the current hour")
	} else if _, err := time.Parse(bucketLayout, got); err != nil {
		t.Errorf("fallback is not a timestamp: %q", got)
	}
}

// Two different keys hitting the same route must stay two rows — that is the
// whole point of recording the actor.
func TestTrackerKeepsActorsApart(t *testing.T) {
	tr := NewTracker(nil, 0)
	base := usageKey{
		projectID: "p1", source: "client", authType: config.UsageAuthApiKey,
		method: "GET", route: "/v2/items/:collection", collection: "deal",
	}

	mobile := base
	mobile.actorID, mobile.actorName = "key-1", "Mobile App"
	backoffice := base
	backoffice.actorID, backoffice.actorName = "key-2", "1C"

	tr.add(mobile)
	tr.add(mobile)
	tr.add(backoffice)

	batch := tr.take()
	if len(batch) != 2 {
		t.Fatalf("expected 2 rows, got %d: %v", len(batch), batch)
	}
	if batch[mobile] != 2 || batch[backoffice] != 1 {
		t.Errorf("counts split wrong: mobile=%d backoffice=%d", batch[mobile], batch[backoffice])
	}
}

// The credential must never reach the stored key. actorID comes from the
// resolved record id (or user id), not from the X-API-KEY header.
func TestActorIsNeverTheCredential(t *testing.T) {
	const secret = "P-grV8bBFcJWQ5b34HIvaFLfy3kqCiJfce"

	k := usageKey{
		projectID: "p1", source: "client", authType: config.UsageAuthApiKey,
		actorID: "6f1c2d3e-key-record-id", actorName: "Mobile App",
		method: "GET", route: "/v2/items/:collection", collection: "deal",
	}

	if strings.Contains(k.field(testBucket), secret) {
		t.Fatal("the encoded field must never carry the api key secret")
	}
}
