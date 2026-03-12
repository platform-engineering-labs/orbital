package runtime

import "runtime/debug"

func BuildCommitRev() string {
	info, ok := debug.ReadBuildInfo()
	if ok {
		for _, setting := range info.Settings {
			if setting.Key == "vcs.revision" {
				return setting.Value
			}
		}
	}

	return "unknown"
}

func BuildVersion() string {
	info, ok := debug.ReadBuildInfo()
	if ok {
		return info.Main.Version
	}

	return "v0.0.0"
}
