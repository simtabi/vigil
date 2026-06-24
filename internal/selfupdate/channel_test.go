package selfupdate

import "testing"

func TestDetectChannel(t *testing.T) {
	cases := []struct {
		path string
		want Channel
	}{
		{"/opt/homebrew/bin/mta", Homebrew},
		{"/usr/local/Cellar/mta/0.1.0/bin/mta", Homebrew},
		{`C:\Users\me\scoop\apps\mta\current\mta.exe`, Scoop},
		{"/usr/bin/mta", SystemPackage},
		{"/bin/mta", SystemPackage},
		{"/home/me/.local/bin/mta", Standalone},
		{"/usr/local/bin/mta", Standalone},
		{`C:\Tools\mta\mta.exe`, Standalone},
	}
	for _, tc := range cases {
		if got := DetectChannel(tc.path); got != tc.want {
			t.Errorf("DetectChannel(%q) = %v, want %v", tc.path, got, tc.want)
		}
	}
}

func TestSelfUpdatableAndAdvice(t *testing.T) {
	if !Standalone.SelfUpdatable() {
		t.Fatal("standalone should be self-updatable")
	}
	for _, c := range []Channel{Homebrew, Scoop, SystemPackage} {
		if c.SelfUpdatable() {
			t.Errorf("%v should not be self-updatable", c)
		}
		if c.Advice() == "" {
			t.Errorf("%v should provide update advice", c)
		}
	}
	if Standalone.Advice() != "" {
		t.Error("standalone should have no package-manager advice")
	}
}

func TestIsDev(t *testing.T) {
	for _, v := range []string{"", "dev", "dev-abc123", "0.0.0-dev+sha", "1.0.0-snapshot"} {
		if !IsDev(v) {
			t.Errorf("IsDev(%q) should be true", v)
		}
	}
	for _, v := range []string{"1.2.3", "v1.2.3", "0.1.0"} {
		if IsDev(v) {
			t.Errorf("IsDev(%q) should be false", v)
		}
	}
}
