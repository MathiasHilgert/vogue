package parse_test

import (
	"testing"

	"github.com/MathiasHilgert/vogue/parse"
	"github.com/stretchr/testify/assert"
)

func TestFieldName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "single word", in: "Title", want: "title"},
		{name: "single letter", in: "A", want: "a"},
		{name: "initialism alone", in: "ID", want: "id"},
		{name: "trailing initialism keeps its case", in: "TabID", want: "tabID"},
		{name: "initialism after two words", in: "GeoNamesID", want: "geoNamesID"},
		{name: "initialism in the middle", in: "SubdivisionISOCode", want: "subdivisionISOCode"},
		{name: "plural initialism", in: "UserIDs", want: "userIDs"},
		{name: "leading initialism", in: "HTTPStatus", want: "httpStatus"},
		{name: "leading short initialism", in: "URLPath", want: "urlPath"},
		{name: "long leading initialism", in: "CUITNumber", want: "cuitNumber"},
		{name: "multi word", in: "UserName", want: "userName"},
		{name: "digits stay in word", in: "Address2Line", want: "address2Line"},
		{name: "empty", in: "", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Arrange / Act.
			got := parse.FieldName(tt.in)

			// Assert.
			assert.Equal(t, tt.want, got)
		})
	}
}
