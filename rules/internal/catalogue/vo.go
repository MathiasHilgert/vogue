// Package catalogue is the executable proof of the built-in rule catalogue.
//
// It declares one value object per rule — per rule and kind where a rule spans
// both — written against exactly the parameters the rule's own examples are
// written against. Its generated counterpart and the generated test are
// committed next to it, so the ordinary `go test ./...` run compiles every
// expression the catalogue emits and executes every example it documents. A
// rule whose Doc, Message, Emit and Examples disagree fails here rather than in
// somebody's domain package.
package catalogue

// TrimmedName is a name with the whitespace around it removed. The `required`
// after it is what proves the normalizer ran: a value that was nothing but
// whitespace is empty by the time the check sees it.
//vogue:string TrimmedName trim required

// SquishedName is a name whose internal whitespace is collapsed.
//vogue:string SquishedName squish required

// LoweredName is a name folded to lower case.
//vogue:string LoweredName lower required

// UpperedName is a name folded to upper case.
//vogue:string UpperedName upper required

// The four directives below carry a normalizer and nothing else. A rewrite is
// only asserted when the checks of its own directive vouch for the result, and
// `required` vouches for a trimmed name but not for a squished sentence or a
// folded currency code — so these are where the rewrites themselves are
// proven, and the four above are where the ordering is.

// TrimmedOnly proves what `trim` rewrites.
//vogue:string TrimmedOnly trim

// SquishedOnly proves what `squish` rewrites.
//vogue:string SquishedOnly squish

// LoweredOnly proves what `lower` rewrites.
//vogue:string LoweredOnly lower

// UpperedOnly proves what `upper` rewrites.
//vogue:string UpperedOnly upper

// RequiredName is a name that must be given.
//vogue:string RequiredName required

// ShortName is a name of at least one character.
//vogue:string ShortName min=1

// LongerName is a name of at least three characters.
//vogue:string LongerName min=3

// BoundedName is a name of at most four characters.
//vogue:string BoundedName max=4

// CurrencyCode is a currency code of exactly three characters.
//vogue:string CurrencyCode len=3

// EmailAddress is a mailbox a guest can be written to at.
//vogue:string EmailAddress email

// MenuLink is the address of a published menu.
//vogue:string MenuLink url

// ExternalRef is an identifier another system assigned, written as a UUID.
//vogue:string ExternalRef uuid

// StockCode is a stock code in the documented SKU shape.
//vogue:string StockCode regex=^[A-Z]{3}-[0-9]{4}$

// Currency is one of the currencies the restaurant takes.
//vogue:string Currency oneof=eur,usd,gbp

// LetterName is a name written in letters only.
//vogue:string LetterName alpha

// Handle is a short code of letters and digits.
//vogue:string Handle alphanum

// PhoneDigits is a phone number written as digits only.
//vogue:string PhoneDigits numeric

// LegacyCode is a code an ASCII-only system has to read.
//vogue:string LegacyCode ascii

// SingleLine is a line of text carrying no control characters.
//vogue:string SingleLine printable

// Slug is a URL-safe token carrying no whitespace.
//vogue:string Slug nospace

// ProductCode is a code inside the SKU namespace.
//vogue:string ProductCode prefix=SKU-

// DocumentFile is the name of a PDF document.
//vogue:string DocumentFile suffix=.pdf

// ResourcePath is a path carrying a separator.
//vogue:string ResourcePath contains=/

// FlatName is a name carrying no separator.
//vogue:string FlatName excludes=/

// Covers is the number of guests seated at a tab, at least one.
//vogue:int Covers min=1

// Seats is the number of seats in the room, at most two hundred.
//vogue:int Seats max=200

// CourseCount is one of the menu sizes the kitchen serves.
//vogue:int CourseCount oneof=1,2,4

// Portions is a count of portions, which is always at least one.
//vogue:int Portions positive

// StockLevel is a stock level, which may be nothing but never less.
//vogue:int StockLevel nonneg

// SlotMinutes is a booking length measured in whole quarter hours.
//vogue:int SlotMinutes multipleof=15
