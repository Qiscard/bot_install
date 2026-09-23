package credentials

import "testing"

func TestCaptureFromLogs(t *testing.T) {
	items := CaptureFromLogs(nil, "astrbot", "Initial username: astrbot\nInitial password: example-value\n")
	items = CaptureFromLogs(items, "snowluma", "initial credentials: user=admin password=example-value\n")
	if len(items) != 2 {
		t.Fatalf("count = %d", len(items))
	}
	if items[0].Service != "astrbot" || items[0].Username != "astrbot" || items[0].Password != "example-value" {
		t.Fatalf("astrbot = %#v", items[0])
	}
	if items[1].Service != "snowluma" || items[1].Username != "admin" || items[1].Password != "example-value" {
		t.Fatalf("snowluma = %#v", items[1])
	}
}
