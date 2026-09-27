package validation

import "errors"

// The sentinel errors of the runtime. They are package-level variables because
// errors.Is compares by identity; they are never reassigned.
var (
	// ErrInvalid is matched by every [FieldError], and therefore by every
	// [Notification] holding at least one, so errors.Is(err, ErrInvalid)
	// answers "was this input rejected by a rule?" without naming the rule.
	ErrInvalid = errors.New("invalid value")

	// ErrUnsupportedSource is wrapped by a generated Scan that is handed a
	// source type it does not read, such as a bool for a text column.
	ErrUnsupportedSource = errors.New("unsupported source type")

	// ErrZeroValue is wrapped by the MarshalText of a value object's zero
	// value. The zero value was never constructed, and no text reads back as
	// it: an empty string or a 0 would come back as a constructed value
	// object. Its JSON form is null instead, and its SQL form NULL.
	ErrZeroValue = errors.New("the zero value has no text form")

	// ErrLossySource is wrapped by a generated Scan that refuses a source
	// which has already lost information, such as a binary float handed to an
	// exact decimal: converting it would silently store a different number.
	ErrLossySource = errors.New("source has already lost precision")
)
