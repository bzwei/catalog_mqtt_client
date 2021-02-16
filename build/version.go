package build

var (
	// Version of the release
	Version string

	// Sha1 is the sha of source commit for the release
	Sha1 string

	// Build is the built date for the release
	Build string
)

func init() {
	if Version == "" {
		Version = "development"
	}
	if Sha1 == "" {
		Sha1 = "unknown"
	}
	if Build == "" {
		Build = "development"
	}
}
