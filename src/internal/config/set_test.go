package config

import "testing"

func TestSetField_Valid(t *testing.T) {
	c := Default()
	// graph.client_id must be set before engine=both, since "both" enables the
	// graph engine which requires a client id (validated at set time).
	cases := []struct{ key, val, want string }{
		{"graph.client_id", "abc-123", "abc-123"},
		{"engine", "both", "both"},
		{"schedule.always", "true", "true"},
		{"input.method", "key", "key"},
		{"input.interval_seconds", "90", "90"},
		{"control.port", "8765", "8765"},
	}
	for _, tc := range cases {
		if err := c.SetField(tc.key, tc.val); err != nil {
			t.Fatalf("SetField(%s,%s): %v", tc.key, tc.val, err)
		}
		got, err := c.GetField(tc.key)
		if err != nil || got != tc.want {
			t.Fatalf("GetField(%s) = %q,%v; want %q", tc.key, got, err, tc.want)
		}
	}
}

func TestSetField_Invalid(t *testing.T) {
	cases := []struct{ key, val string }{
		{"engine", "bogus"},                // enum rejected by Validate
		{"input.interval_seconds", "9999"}, // out of range
		{"input.interval_seconds", "xx"},   // not an int
		{"schedule.always", "maybe"},       // not a bool
		{"nope.key", "1"},                  // unknown key
	}
	for _, tc := range cases {
		c := Default()
		if err := c.SetField(tc.key, tc.val); err == nil {
			t.Fatalf("SetField(%s,%s) expected error", tc.key, tc.val)
		}
	}
}

func TestSetField_LeavesConfigUnchangedOnError(t *testing.T) {
	c := Default()
	before := c.Input.IntervalSeconds
	if err := c.SetField("input.interval_seconds", "9999"); err == nil {
		t.Fatal("expected error")
	}
	if c.Input.IntervalSeconds != before {
		t.Fatalf("config mutated on failed set: got %d want %d", c.Input.IntervalSeconds, before)
	}
}

func TestSettableKeysSorted(t *testing.T) {
	keys := SettableKeys()
	if len(keys) == 0 {
		t.Fatal("no settable keys")
	}
	for i := 1; i < len(keys); i++ {
		if keys[i-1] > keys[i] {
			t.Fatalf("keys not sorted: %q before %q", keys[i-1], keys[i])
		}
	}
}
