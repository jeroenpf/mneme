package cli

// version is the build version reported by `mneme --version`. It defaults to
// "dev" for local/unreleased builds and is overridden at link time by
// GoReleaser via:
//
//	-ldflags "-X github.com/jeroenpf/mneme/internal/cli.version=<tag>"
//
// Keep this a plain package-level var (not a const) so the linker can set it.
var version = "dev"
