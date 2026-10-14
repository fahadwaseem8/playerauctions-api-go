package offers

import "regexp"

var uuidPattern = regexp.MustCompile(`(?i)^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

// ValidUUID checks the canonical UUID form used by PostgreSQL's JSON output.
func ValidUUID(value string) bool {
	return uuidPattern.MatchString(value)
}
