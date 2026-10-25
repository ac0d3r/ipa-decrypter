// Package version holds the build-time version string. It is overridden
// at link time with -ldflags "-X .../internal/version.Version=<tag>".
package version

// Version is set by the build system; "dev" when built from source
// without injection.
var Version = "dev"
