package catalogue_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/MathiasHilgert/vogue/rules/internal/catalogue"
)

// nothing names the empty input several of the tables below share.
const nothing = "the empty string"

// accepts runs a constructor against every case and compares whether it
// accepted the input with what the case expects.
func accepts[T any](t *testing.T, construct func(string) (T, error), cases []acceptanceCase) {
	t.Helper()

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			// Act
			_, err := construct(testCase.in)

			// Assert
			assert.Equal(t, testCase.want, err == nil)
		})
	}
}

// acceptanceCase is one input and whether the rule under test accepts it.
type acceptanceCase struct {
	name string
	in   string
	want bool
}

func TestEmailAddress_Acceptance(t *testing.T) {
	t.Parallel()

	accepts(t, catalogue.NewEmailAddress, []acceptanceCase{
		{name: "an ordinary address", in: "waiter@example.com", want: true},
		{name: "a subaddressed mailbox", in: "orders+tab7@example.com", want: true},
		{name: "a subdomain", in: "a@mail.example.com", want: true},
		{name: nothing, in: "", want: false},
		{name: "a bare local part", in: "waiter", want: false},
		{name: "no domain", in: "waiter@", want: false},
		{name: "two at signs", in: "a@b@c.test", want: false},
		{name: "a display name, which a stored address must not carry", in: "Waiter <a@b.test>", want: false},
		{name: "surrounding whitespace", in: " a@b.test ", want: false},
		{name: "an address list", in: "a@b.test, c@d.test", want: false},
	})
}

func TestMenuLink_Acceptance(t *testing.T) {
	t.Parallel()

	accepts(t, catalogue.NewMenuLink, []acceptanceCase{
		{name: "an https url", in: "https://example.com/menu", want: true},
		{name: "an http url with a port and a query", in: "http://example.com:8080/menu?tab=7", want: true},
		{name: nothing, in: "", want: false},
		{name: "a bare host without a scheme", in: "example.com", want: false},
		{name: "a relative path", in: "/menu", want: false},
		{name: "a scheme that is not http or https", in: "ftp://example.com", want: false},
		{name: "a scheme without a host", in: "https://", want: false},
		{name: "a control character", in: "https://example.com/\n", want: false},
	})
}

func TestExternalRef_Acceptance(t *testing.T) {
	t.Parallel()

	accepts(t, catalogue.NewExternalRef, []acceptanceCase{
		{name: "a version 4 uuid", in: "9b2b4f52-1c2d-4e5a-9f3b-6d7c8e9f0a1b", want: true},
		{name: "a version 7 uuid", in: "018f3a2b-7c4d-7e8f-9a0b-1c2d3e4f5a6b", want: true},
		{name: "the nil uuid", in: "00000000-0000-0000-0000-000000000000", want: true},
		{name: "upper case hex", in: "9B2B4F52-1C2D-4E5A-9F3B-6D7C8E9F0A1B", want: true},
		{name: nothing, in: "", want: false},
		{name: "a truncated uuid", in: "9b2b4f52-1c2d-4e5a-9f3b", want: false},
		{name: "a non-hex rune", in: "9b2b4f52-1c2d-4e5a-9f3b-6d7c8e9f0a1z", want: false},
		{name: "a hyphen where a digit belongs", in: "9b2b4f52-1c2d-4e5a-9f3b-6d7c8e9f0a-b", want: false},
		{name: "a digit where a hyphen belongs", in: "9b2b4f521c2d-4e5a-9f3b-6d7c8e9f0a1b1", want: false},
		{name: "the undashed form, which stored ids never use", in: "9b2b4f521c2d4e5a9f3b6d7c8e9f0a1b", want: false},
		{name: "surrounding whitespace", in: " 9b2b4f52-1c2d-4e5a-9f3b-6d7c8e9f0a1b ", want: false},
	})
}

func TestZoneName_Acceptance(t *testing.T) {
	t.Parallel()

	accepts(t, catalogue.NewZoneName, []acceptanceCase{
		{name: "a zone with a city", in: "America/Argentina/Buenos_Aires", want: true},
		{name: "another continent", in: "Europe/Madrid", want: true},
		{name: "UTC", in: "UTC", want: true},
		{name: "the first zone of the list", in: "Africa/Abidjan", want: true},
		{name: "the last zone of the list", in: "Zulu", want: true},
		{name: "a link the database keeps for compatibility", in: "US/Eastern", want: true},
		{name: "a zone that does not exist", in: "Mars/Olympus_Mons", want: false},
		{name: "the process-local zone, which names no place", in: "Local", want: false},
		{name: "the empty string, which LoadLocation reads as UTC", in: "", want: false},
		{name: "a path escaping the zone database", in: "../etc/passwd", want: false},
		{name: "a zone in the wrong case, which a case-insensitive file system would load", in: "europe/madrid", want: false},
		{name: "a zone shouted", in: "EUROPE/MADRID", want: false},
		{name: "utc in lower case", in: "utc", want: false},
		{name: "the placeholder zone of the database", in: "Factory", want: false},
		{name: "the system's local zone file", in: "localtime", want: false},
		{name: "the database's rules file", in: "posixrules", want: false},
		{name: "a directory of the database", in: "America", want: false},
	})
}

func TestStockCode_Acceptance(t *testing.T) {
	t.Parallel()

	accepts(t, catalogue.NewStockCode, []acceptanceCase{
		{name: "a code in the documented shape", in: "SKU-0042", want: true},
		{name: "a value outside the pattern", in: "sku-0042", want: false},
		{name: "a value with a suffix, which the anchors reject", in: "SKU-0042x", want: false},
		{name: nothing, in: "", want: false},
	})
}
