package ops

import (
	"testing"
)

func TestParse_PreReleaseAndBuildPopulated(t *testing.T) {
	cases := []struct {
		input      string
		major      uint64
		minor      uint64
		patch      uint64
		preRelease string
		build      string
	}{
		{"0.1.0", 0, 1, 0, "", ""},
		{"0.1.0-dev.1", 0, 1, 0, "dev.1", ""},
		{"1.2.3-rc.4", 1, 2, 3, "rc.4", ""},
		{"0.1.0+build.42", 0, 1, 0, "", "build.42"},
		{"0.1.0-dev.1+build.42", 0, 1, 0, "dev.1", "build.42"},
	}

	for _, tc := range cases {
		t.Run(tc.input, func(t *testing.T) {
			v := &Version{}
			if err := v.Parse(tc.input); err != nil {
				t.Fatalf("Parse(%q) returned error: %v", tc.input, err)
			}
			if v.Major != tc.major || v.Minor != tc.minor || v.Patch != tc.patch {
				t.Errorf("Parse(%q): got %d.%d.%d, want %d.%d.%d",
					tc.input, v.Major, v.Minor, v.Patch, tc.major, tc.minor, tc.patch)
			}
			if v.PreRelease != tc.preRelease {
				t.Errorf("Parse(%q): PreRelease = %q, want %q", tc.input, v.PreRelease, tc.preRelease)
			}
			if v.Build != tc.build {
				t.Errorf("Parse(%q): Build = %q, want %q", tc.input, v.Build, tc.build)
			}
		})
	}
}

func TestSemver_IncludesPreReleaseAndBuild(t *testing.T) {
	v := &Version{}
	if err := v.Parse("0.1.0-dev.1+build.42"); err != nil {
		t.Fatalf("Parse: %v", err)
	}
	got := v.Semver().String()
	want := "0.1.0-dev.1+build.42"
	if got != want {
		t.Errorf("Semver().String() = %q, want %q", got, want)
	}
}

func TestShort_RetainsPreReleaseAndBuild(t *testing.T) {
	v := &Version{}
	if err := v.Parse("0.1.0-dev.1"); err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if got, want := v.Short(), "0.1.0-dev.1"; got != want {
		t.Errorf("Short() = %q, want %q", got, want)
	}
}

func TestComparePrereleasesAreOrdered(t *testing.T) {
	older := &Version{}
	if err := older.Parse("0.1.0-dev.1"); err != nil {
		t.Fatalf("Parse older: %v", err)
	}
	newer := &Version{}
	if err := newer.Parse("0.1.0-dev.2"); err != nil {
		t.Fatalf("Parse newer: %v", err)
	}
	// 0.1.0-dev.2 should be greater than 0.1.0-dev.1.
	if !newer.GT(older) {
		t.Errorf("expected 0.1.0-dev.2 > 0.1.0-dev.1, got Compare = %d", newer.Compare(older))
	}
	// Pre-release versions are < the GA release per semver.
	ga := &Version{}
	if err := ga.Parse("0.1.0"); err != nil {
		t.Fatalf("Parse GA: %v", err)
	}
	if !older.LT(ga) {
		t.Errorf("expected 0.1.0-dev.1 < 0.1.0, got Compare = %d", older.Compare(ga))
	}
}
