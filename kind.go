package vogue

import (
	"fmt"
	"strings"
)

// Kind enumerates the value-object shapes vogue can generate. It is the first
// token of a directive (`//vogue:string Title ...`) and it constrains which
// rules may be applied to a value object: a rule declares the kinds it
// understands through its [Rule.Kinds] set.
type Kind uint8

const (
	// String is a value object wrapping a single string field.
	String Kind = iota
	// Int is a value object wrapping a single int field.
	Int
	// Enum is a closed set of string constants with Values, String and parsing.
	Enum
	// ID is a UUID-backed identifier value object.
	ID
)

// kindNames maps a Kind to its directive spelling. The slice index is the Kind
// value, so it doubles as the set of known kinds.
var kindNames = [...]string{
	String: "string",
	Int:    "int",
	Enum:   "enum",
	ID:     "id",
}

// String returns the directive spelling of the kind, or a Kind(<n>) placeholder
// when the receiver is not one of the declared kinds.
func (k Kind) String() string {
	if int(k) < len(kindNames) {
		return kindNames[k]
	}
	return "Kind(" + fmt.Sprint(uint8(k)) + ")"
}

// valid reports whether k is one of the kinds declared by this package.
func (k Kind) valid() bool { return int(k) < len(kindNames) }

// ParseKind resolves the directive spelling of a kind. Parsing is
// case-sensitive because directives are source code, not user input, and a
// silent fold would hide typos the generator should report.
func ParseKind(s string) (Kind, error) {
	for i, name := range kindNames {
		if name == s {
			return Kind(i), nil
		}
	}
	return 0, fmt.Errorf("vogue: unknown kind %q (want one of: %s)", s, strings.Join(kindNames[:], ", "))
}

// KindSet is an immutable bitmask of [Kind] values. The zero value is the empty
// set, which lets a Rule literal omit the field and still be rejected by
// [Rule.Validate] with a descriptive message.
type KindSet uint32

// Kinds builds a KindSet from the given kinds. Duplicates are collapsed and
// kinds outside the declared range are ignored, so the result always describes
// a set the generator can act on.
func Kinds(kinds ...Kind) KindSet {
	var set KindSet
	for _, k := range kinds {
		if k.valid() {
			set |= 1 << k
		}
	}
	return set
}

// Has reports whether the set contains the given kind.
func (s KindSet) Has(k Kind) bool { return k.valid() && s&(1<<k) != 0 }

// Empty reports whether the set contains no kind.
func (s KindSet) Empty() bool { return s == 0 }

// Kinds returns the members of the set in declaration order. The result is
// freshly allocated and safe for the caller to retain.
func (s KindSet) Kinds() []Kind {
	out := make([]Kind, 0, len(kindNames))
	for i := range kindNames {
		if k := Kind(i); s.Has(k) {
			out = append(out, k)
		}
	}
	return out
}

// String renders the set as a comma-separated list in declaration order, for
// use in generator diagnostics such as "rule min applies to: string, int".
func (s KindSet) String() string {
	var b strings.Builder
	for i := range kindNames {
		if k := Kind(i); s.Has(k) {
			if b.Len() > 0 {
				b.WriteString(", ")
			}
			b.WriteString(k.String())
		}
	}
	return b.String()
}
