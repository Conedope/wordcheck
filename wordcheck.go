package wordcheck

import "sort"

// IsKnown reports whether word exists in the embedded dictionary.
// The match is exact and case-sensitive; the dictionary is all lowercase,
// so IsKnown("Hello") is false while IsKnown("hello") is true.
func IsKnown(word string) bool {
	_, ok := dictionary[word]
	return ok
}

// DictionaryWords returns the embedded dictionary as a sorted slice.
func DictionaryWords() []string {
	words := make([]string, 0, len(dictionary))
	for w := range dictionary {
		words = append(words, w)
	}
	sort.Strings(words)
	return words
}

// Levenshtein returns the edit distance between a and b: the minimum number
// of single-character insertions, deletions, and substitutions required to
// turn a into b. Computed with a two-row dynamic-programming matrix.
func Levenshtein(a, b string) int {
	la, lb := len(a), len(b)
	if la == 0 {
		return lb
	}
	if lb == 0 {
		return la
	}
	prev := make([]int, lb+1)
	cur := make([]int, lb+1)
	for j := 0; j <= lb; j++ {
		prev[j] = j
	}
	for i := 1; i <= la; i++ {
		cur[0] = i
		for j := 1; j <= lb; j++ {
			cost := 0
			if a[i-1] != b[j-1] {
				cost = 1
			}
			del := prev[j] + 1
			ins := cur[j-1] + 1
			sub := prev[j-1] + cost
			cur[j] = min3(del, ins, sub)
		}
		prev, cur = cur, prev
	}
	return prev[lb]
}

func min3(a, b, c int) int {
	if b < a {
		a = b
	}
	if c < a {
		a = c
	}
	return a
}

// Suggestions returns up to max dictionary words within edit distance 2 of
// word, ranked by distance first and then alphabetically. It returns an
// empty slice when no candidate exists or when max <= 0.
func Suggestions(word string, max int) []string {
	if max <= 0 {
		return nil
	}
	type cand struct {
		word string
		dist int
	}
	var found []cand
	for w := range dictionary {
		if d := Levenshtein(word, w); d <= 2 {
			found = append(found, cand{w, d})
		}
	}
	sort.Slice(found, func(i, j int) bool {
		if found[i].dist != found[j].dist {
			return found[i].dist < found[j].dist
		}
		return found[i].word < found[j].word
	})
	if len(found) > max {
		found = found[:max]
	}
	out := make([]string, len(found))
	for i := range found {
		out[i] = found[i].word
	}
	return out
}
