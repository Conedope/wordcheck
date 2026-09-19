package wordcheck

import "strings"

// TokenizeOpts controls tokenization and case handling.
type TokenizeOpts struct {
	// IgnoreCase lowercases every word token so downstream checks are
	// case-insensitive. Number tokens are unaffected.
	IgnoreCase bool
	// IgnoreNumbers drops number tokens from the output entirely.
	IgnoreNumbers bool
}

// Token is a single word or number found in the input text.
// Start and End are byte offsets into the original string; Line is 1-based.
type Token struct {
	Word  string
	Start int // byte offset of the first byte
	End   int // byte offset just past the last byte
	Line  int // 1-based line number
	Num   bool
}

func isLetter(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
}

func isDigit(c byte) bool {
	return c >= '0' && c <= '9'
}

// TokenizeWords splits s into word and number tokens.
//
// A word is a run of letters (a-z, A-Z) plus apostrophes that are
// surrounded by letters on both sides (e.g. "don't", "John's").
// A leading or trailing apostrophe is punctuation, not part of the word.
//
// A number is a run of digits that may contain '.' or ',' separators that
// are followed by another digit (e.g. 100,000 or 3.14). Number tokens are
// marked Num. When opts.IgnoreNumbers is true they are not emitted.
func TokenizeWords(s string, opts TokenizeOpts) []Token {
	var tokens []Token
	line := 1
	i := 0
	for i < len(s) {
		c := s[i]
		switch {
		case c == '\n':
			line++
			i++
		case isLetter(c):
			start := i
			j := i
			for j < len(s) {
				ch := s[j]
				if isLetter(ch) {
					j++
					continue
				}
				if ch == '\'' && j+1 < len(s) && isLetter(s[j+1]) {
					j++
					continue
				}
				break
			}
			tok := s[start:j]
			if opts.IgnoreCase {
				tok = strings.ToLower(tok)
			}
			tokens = append(tokens, Token{Word: tok, Start: start, End: j, Line: line})
			i = j
		case isDigit(c):
			start := i
			j := i
			for j < len(s) {
				ch := s[j]
				if isDigit(ch) {
					j++
					continue
				}
				if (ch == '.' || ch == ',') && j+1 < len(s) && isDigit(s[j+1]) {
					j++
					continue
				}
				break
			}
			if !opts.IgnoreNumbers {
				tokens = append(tokens, Token{Word: s[start:j], Start: start, End: j, Line: line, Num: true})
			}
			i = j
		default:
			i++
		}
	}
	return tokens
}
