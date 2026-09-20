package gen

// declSet collects the package-level declarations the rules of one generated
// file ask for through [vogue.Rule.Declare]. Declarations are deduplicated by
// their own source, so two directives compiling the same regular expression
// share one variable, and kept in first-seen order so the generated file is
// byte-stable across runs.
type declSet struct {
	seen  map[string]struct{}
	order []string
}

// newDeclSet returns an empty set.
func newDeclSet() *declSet {
	return &declSet{seen: map[string]struct{}{}}
}

// add records a declaration, ignoring an empty one and a repeat of one already
// recorded.
func (s *declSet) add(decl string) {
	if decl == "" {
		return
	}
	if _, ok := s.seen[decl]; ok {
		return
	}
	s.seen[decl] = struct{}{}
	s.order = append(s.order, decl)
}

// all returns the recorded declarations in first-seen order.
func (s *declSet) all() []string { return s.order }
