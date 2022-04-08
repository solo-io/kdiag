package version

var (
	// Version is the main version number that is being run at the moment.
	Version = "dev"

	// VersionPrerelease is A pre-release marker for the Version. If this is ""
	// (empty string) then it means that it is a final release. Otherwise, this
	// is a pre-release such as "dev" (in development), "beta", "rc1", etc.
	VersionPrerelease = "dev"

	Commit = "HEAD"

	ImageRepo = "ghcr.io/solo-io/kdiag"
)
