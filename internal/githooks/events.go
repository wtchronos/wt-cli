package githooks

// SupportedGitEvents returns all git hook events that wt manages.
var SupportedGitEvents = []string{
	"post-checkout",
	"post-commit",
	"post-merge",
}

// IsSupported returns true if the given event name is one we manage.
func IsSupported(event string) bool {
	for _, e := range SupportedGitEvents {
		if e == event {
			return true
		}
	}
	return false
}
