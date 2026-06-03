package buildinfo

import "runtime/debug"

var (
	Version      = "dev"
	BuildRFC3339 = "1970-01-01T00:00:00Z"
	Revision     = "local"
)

const (
	defaultVersion      = "dev"
	defaultBuildRFC3339 = "1970-01-01T00:00:00Z"
	defaultRevision     = "local"
)

func PopulateFromBuildInfo() {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return
	}

	if Version == defaultVersion && info.Main.Version != "" && info.Main.Version != "(devel)" {
		Version = info.Main.Version
	}

	for _, setting := range info.Settings {
		switch setting.Key {
		case "vcs.revision":
			if Revision == defaultRevision && setting.Value != "" {
				Revision = setting.Value
			}
		case "vcs.time":
			if BuildRFC3339 == defaultBuildRFC3339 && setting.Value != "" {
				BuildRFC3339 = setting.Value
			}
		}
	}
}
