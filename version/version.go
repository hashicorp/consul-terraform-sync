package version

import (
	"fmt"
	"strings"
)

var (
	Name = "consul-terraform-sync"

	// GitCommit is the git commit that was compiled. These will be filled in by
	// the compiler.
	GitCommit   string
	GitDescribe string

	// GitDirty is dirty if the working tree has local modifications from HEAD.
	// These will be filled in by the compiler.
	GitDirty string

	// The main version number that is being run at the moment.
	//
	// Version must conform to the format expected by
	// github.com/hashicorp/go-version for tests to work.
	Version = "0.5.2"

	// VersionPrerelease is a pre-release marker for the version. If this is ""
	// (empty string) then it means that it is a final release. Otherwise, this
	// is a pre-release such as "dev" (in development), "beta", "rc1", etc.
	VersionPrerelease = ""

	VersionMetadata = ""
)

// GetHumanVersion composes the parts of the version in a way that's suitable
// for displaying to humans.
func GetHumanVersion() string {
	version := Version
	if GitDescribe != "" && VersionPrerelease == "" {
		version = GitDescribe
	}

	release := VersionPrerelease
	if GitDescribe == "" && release == "" {
		release = "dev"
	}
	if release != "" {
		version += fmt.Sprintf("-%s", release)
	}

	metadata := VersionMetadata
	if metadata != "" {
		version += fmt.Sprintf("+%s", metadata)
	}

	if GitCommit != "" && GitDirty != "" {
		version += fmt.Sprintf(" (%s dirty)", GitCommit)
	} else if GitCommit != "" {
		version += fmt.Sprintf(" (%s)", GitCommit)
	}

	// Strip off any single quotes added by the git information.
	return strings.Replace(version, "'", "", -1)
}
