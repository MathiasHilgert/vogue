package rulecheck

import "time"

// TimeZone reports whether value names a zone of the IANA time zone database, such
// as "America/Argentina/Buenos_Aires" or "UTC".
//
// The name is resolved with time.LoadLocation, so it is only as current as
// the zone database the process can read: the system's, or the one embedded by
// importing time/tzdata, which a binary running in a minimal container needs.
// "Local" and the empty string, which LoadLocation maps to the process's own
// zone and to UTC, are rejected, since neither names a place a stored value
// could mean.
func TimeZone(value string) bool {
	if value == "" || value == "Local" {
		return false
	}
	_, err := time.LoadLocation(value)
	return err == nil
}
