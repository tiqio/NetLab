package compliance

import "strings"

const (
	ExitOK                  = 0
	ExitMalformed           = 2
	ExitInvalidReference    = 3
	ExitStaleContradiction  = 4
	ExitDeploymentAuthority = 5
	ExitProhibitedContent   = 6
)

func ExitCode(err error) int {
	if err == nil {
		return ExitOK
	}
	message := strings.ToLower(err.Error())
	switch {
	case strings.Contains(message, "prohibited"), strings.Contains(message, "redaction"):
		return ExitProhibitedContent
	case strings.Contains(message, "authoritative instance"), strings.Contains(message, "deployment authority"):
		return ExitDeploymentAuthority
	case strings.Contains(message, "non-accepted evidence"), strings.Contains(message, "stale"), strings.Contains(message, "contradict"):
		return ExitStaleContradiction
	case strings.Contains(message, "references missing"), strings.Contains(message, "invalid exception"):
		return ExitInvalidReference
	default:
		return ExitMalformed
	}
}
