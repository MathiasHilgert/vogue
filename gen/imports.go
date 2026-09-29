package gen

import (
	"fmt"
	"sort"
	"strings"
)

// Import paths the generated code uses, named once so a template change and an
// import change never drift apart.
const (
	importDriver  = "database/sql/driver"
	importErrors  = "errors"
	importFmt     = "fmt"
	importStrconv = "strconv"
	importUUID    = "github.com/google/uuid"
	importDecimal = "github.com/govalues/decimal"
)

// importSet collects the import paths of one generated file, keyed by the
// identifier they will be referenced through. Aliasing is deliberately out of
// scope: two paths claiming the same identifier are a generate-time error, so
// the generated file never contains a name the author did not write.
type importSet struct {
	byIdent map[string]string
	skip    string
}

// newImportSet returns an empty set. Paths equal to self are dropped, which is
// how a rule whose function lives in the package being generated is called
// without importing that package into itself.
func newImportSet(self string) *importSet {
	return &importSet{byIdent: map[string]string{}, skip: self}
}

// add records an import path, reporting a collision when another path already
// claims the same identifier.
func (s *importSet) add(path string) error {
	if path == "" || (s.skip != "" && path == s.skip) {
		return nil
	}
	ident := packageIdent(path)
	switch existing, ok := s.byIdent[ident]; {
	case !ok:
		s.byIdent[ident] = path
	case existing != path:
		return fmt.Errorf("gen: import collision on identifier %q between %q and %q: aliasing is not supported, import one of them through a wrapper package",
			ident, existing, path)
	}
	return nil
}

// groups returns the collected import paths sorted and split into the standard
// library group and everything else, which is how goimports would lay them out.
func (s *importSet) groups() (std, ext []string) {
	for _, path := range s.byIdent {
		if strings.Contains(strings.Split(path, "/")[0], ".") {
			ext = append(ext, path)
			continue
		}
		std = append(std, path)
	}
	sort.Strings(std)
	sort.Strings(ext)
	return std, ext
}

// packageIdent derives the identifier an import path is referenced through,
// skipping a semantic-import-versioning element such as "/v2".
func packageIdent(path string) string {
	parts := strings.Split(path, "/")
	last := parts[len(parts)-1]
	if len(parts) > 1 && isMajorVersion(last) {
		last = parts[len(parts)-2]
	}
	return last
}

// isMajorVersion reports whether a path element is a major-version suffix.
func isMajorVersion(s string) bool {
	if len(s) < 2 || s[0] != 'v' {
		return false
	}
	for i := 1; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return true
}
