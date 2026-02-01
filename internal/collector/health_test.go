package collector

import "testing"

func TestParseHealthFromInspectRaw(t *testing.T) {
	raw := []byte(`{
	  "State": {
	    "Health": {
	      "Status": "unhealthy",
	      "Log": [
	        { "Start": "2026-02-01T00:00:00.000000000Z", "End": "2026-02-01T00:00:01.000000000Z", "ExitCode": 0 },
	        { "Start": "2026-02-01T00:01:00.000000000Z", "End": "2026-02-01T00:01:01.000000000Z", "ExitCode": 1 }
	      ]
	    }
	  }
	}`)

	status, since := parseHealthFromInspectRaw(raw)
	if status != "unhealthy" {
		t.Fatalf("expected unhealthy, got %q", status)
	}
	if since.IsZero() {
		t.Fatalf("expected unhealthySince to be set")
	}
}

func TestParseHealthFromInspectRaw_NoneWhenMissing(t *testing.T) {
	raw := []byte(`{ "State": { } }`)
	status, since := parseHealthFromInspectRaw(raw)
	if status != "none" {
		t.Fatalf("expected none, got %q", status)
	}
	if !since.IsZero() {
		t.Fatalf("expected unhealthySince zero, got %v", since)
	}
}

