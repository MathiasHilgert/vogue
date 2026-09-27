package rulecheck

import "sort"

// TimeZone reports whether value is the canonical name of a zone of the IANA
// time zone database, such as "America/Argentina/Buenos_Aires" or "UTC".
//
// The name is looked up, case-exactly, in the list of zones the Go toolchain's
// zone database holds (zones.go, written by internal/zonelist), rather than
// loaded with time.LoadLocation: LoadLocation reads the system's database
// first, which on a case-insensitive file system accepts "europe/madrid", and
// it accepts the database's own files — Factory, localtime, posixrules — that
// name no place. The list needs no zone database at run time; converting the
// value to a time.Location still does, so a binary in a minimal container
// image should import time/tzdata. The empty string and "Local" are not in the
// list. A lookup is a binary search over a sorted array, so nothing is cached.
func TimeZone(value string) bool {
	index := sort.SearchStrings(zones[:], value)
	return index < len(zones) && zones[index] == value
}
