package wordcheck

// Miss is a flagged word together with its ranked candidate corrections.
type Miss struct {
	Token       Token
	Suggestions []string
}

// Check tokenizes text and returns one Miss for every token that is not a
// number and not present in the dictionary. Suggestions are computed from
// the token word as produced by TokenizeWords (so with opts.IgnoreCase they
// are derived from the lowercased form).
func Check(text string, opts TokenizeOpts) []Miss {
	var misses []Miss
	for _, t := range TokenizeWords(text, opts) {
		if t.Num || IsKnown(t.Word) {
			continue
		}
		misses = append(misses, Miss{Token: t, Suggestions: Suggestions(t.Word, 3)})
	}
	return misses
}
