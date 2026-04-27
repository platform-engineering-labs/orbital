package ops

import (
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/blang/semver/v4"
)

type Version struct {
	Timestamp time.Time

	Major uint64 `pkl:"major"`
	Minor uint64 `pkl:"minor"`
	Patch uint64 `pkl:"patch"`

	PreRelease string `pkl:"preRelease"`
	Build      string `pkl:"build"`
}

func (v *Version) Parse(version string) error {
	split := strings.Split(version, "_")
	var err error = nil

	if len(split) < 1 {
		return errors.New("ops.Version: error parsing version")
	}

	sv, err := semver.Make(split[0])
	if err != nil {
		return errors.New("ops.Version: error parsing version (semver)")
	}
	v.Major = sv.Major
	v.Minor = sv.Minor
	v.Patch = sv.Patch
	v.PreRelease = serializePre(sv.Pre)
	v.Build = strings.Join(sv.Build, ".")

	if len(split) == 2 {
		v.Timestamp, err = time.Parse("20060102T150405Z", split[1])
	}

	return err
}

// serializePre joins a semver pre-release identifier list into the canonical
// dot-separated form ("dev.1", "rc.4", etc.).
func serializePre(pre []semver.PRVersion) string {
	if len(pre) == 0 {
		return ""
	}
	parts := make([]string, len(pre))
	for i, p := range pre {
		parts[i] = p.String()
	}
	return strings.Join(parts, ".")
}

func (v *Version) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.String())
}

func (v *Version) UnmarshalJSON(data []byte) error {
	var str string
	if err := json.Unmarshal(data, &str); err != nil {
		return err
	}

	return v.Parse(str)
}

func (v *Version) Semver() semver.Version {
	sv := semver.Version{
		Major: v.Major,
		Minor: v.Minor,
		Patch: v.Patch,
	}
	if v.PreRelease != "" {
		for _, p := range strings.Split(v.PreRelease, ".") {
			pr, err := semver.NewPRVersion(p)
			if err != nil {
				continue
			}
			sv.Pre = append(sv.Pre, pr)
		}
	}
	if v.Build != "" {
		sv.Build = strings.Split(v.Build, ".")
	}
	return sv
}

func (v *Version) Short() string {
	return v.Semver().String()
}

func (v *Version) String() string {
	s := []string{v.Semver().String(), v.Timestamp.Format("20060102T150405Z")}
	return strings.Join(s, "_")
}

func (v *Version) Compare(ve *Version) int {
	if v.Semver().GT(ve.Semver()) {
		return 1
	}

	if v.Semver().LT(ve.Semver()) {
		return -1
	}

	if v.Timestamp.After(ve.Timestamp) && !ve.Timestamp.IsZero() {
		return 1
	}

	if v.Timestamp.Before(ve.Timestamp) {
		return -1
	}

	if v.Timestamp.Equal(ve.Timestamp) {
		return 2
	}

	return 0
}

func (v *Version) EQ(ve *Version) bool {
	compare := v.Compare(ve)
	
	return compare == 0 || compare == 2
}

func (v *Version) EXQ(ve *Version) bool {
	return v.Compare(ve) == 2
}

func (v *Version) GT(ve *Version) bool {
	return v.Compare(ve) == 1
}

func (v *Version) GTE(ve *Version) bool {
	return v.Compare(ve) >= 0
}

func (v *Version) LT(ve *Version) bool {
	return v.Compare(ve) == -1
}

func (v *Version) LTE(ve *Version) bool {
	return v.Compare(ve) <= 0
}

func (v *Version) NEQ(ve *Version) bool {
	return v.Compare(ve) != 0
}
