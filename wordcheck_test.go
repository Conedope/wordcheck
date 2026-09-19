package wordcheck

import (
	"reflect"
	"sort"
	"testing"
)

func TestIsKnown(t *testing.T) {
	for _, w := range []string{"the", "hello", "spelling", "world", "quick", "lazy", "jumped", "wall"} {
		if !IsKnown(w) {
			t.Errorf("IsKnown(%q) = false, want true", w)
		}
	}
}

func TestIsKnownCaseSensitive(t *testing.T) {
	if IsKnown("Hello") {
		t.Error(`IsKnown("Hello") = true, want false (dictionary is lowercase)`)
	}
	if IsKnown("THE") {
		t.Error(`IsKnown("THE") = true, want false`)
	}
}

func TestIsKnownUnknown(t *testing.T) {
	for _, w := range []string{"zzqwxvjk", "speling", "quik", "lazzy", "aubergin", "plaughed"} {
		if IsKnown(w) {
			t.Errorf("IsKnown(%q) = true, want false", w)
		}
	}
}

func TestDictionaryWordsSortedUnique(t *testing.T) {
	words := DictionaryWords()
	if !sort.StringsAreSorted(words) {
		t.Fatal("DictionaryWords() is not sorted")
	}
	for i := 1; i < len(words); i++ {
		if words[i] == words[i-1] {
			t.Fatalf("duplicate word %q in dictionary", words[i])
		}
	}
}

func TestDictionaryWordsSize(t *testing.T) {
	if n := len(DictionaryWords()); n != 21082 {
		t.Fatalf("DictionaryWords() length = %d, want 21082", n)
	}
}

func TestDictionaryWordsContainsCommon(t *testing.T) {
	words := DictionaryWords()
	for _, w := range []string{"hello", "spelling", "the", "about"} {
		idx := sort.SearchStrings(words, w)
		if idx >= len(words) || words[idx] != w {
			t.Errorf("dictionary missing %q", w)
		}
	}
}

func TestLevenshtein(t *testing.T) {
	cases := []struct {
		a, b string
		want int
	}{
		{"", "", 0},
		{"", "ab", 2},
		{"ab", "", 2},
		{"a", "a", 0},
		{"abc", "abc", 0},
		{"a", "b", 1},
		{"abcd", "abcde", 1}, // insertion
		{"abcde", "abcd", 1}, // deletion
		{"kitten", "sitting", 3},
		{"flaw", "lawn", 2},
		{"saturday", "sunday", 3},
		{"intention", "execution", 5},
		{"book", "back", 2},
		{"abcdef", "zyxwvu", 6},
	}
	for _, c := range cases {
		if got := Levenshtein(c.a, c.b); got != c.want {
			t.Errorf("Levenshtein(%q, %q) = %d, want %d", c.a, c.b, got, c.want)
		}
	}
}

func TestLevenshteinSymmetric(t *testing.T) {
	pairs := [][2]string{{"kitten", "sitting"}, {"university", "universe"}, {"abcdef", ""}, {"hello", "halo"}}
	for _, p := range pairs {
		if Levenshtein(p[0], p[1]) != Levenshtein(p[1], p[0]) {
			t.Errorf("Levenshtein not symmetric for %q and %q", p[0], p[1])
		}
	}
}

func TestSuggestionsContainsSpelling(t *testing.T) {
	sugs := Suggestions("speling", 10)
	if len(sugs) == 0 {
		t.Fatal("Suggestions(\"speling\") returned nothing")
	}
	found := false
	for _, s := range sugs {
		if s == "spelling" {
			found = true
		}
	}
	if !found {
		t.Fatalf("Suggestions(\"speling\") = %v, want it to contain \"spelling\"", sugs)
	}
}

func TestSuggestionsFirstIsClosest(t *testing.T) {
	sugs := Suggestions("speling", 10)
	if sugs[0] != "spelling" {
		t.Errorf("Suggestions(\"speling\")[0] = %q, want \"spelling\"", sugs[0])
	}
	if d := Levenshtein("speling", sugs[0]); d != 1 {
		t.Errorf("closest suggestion \"%s\" is at distance %d, want 1", sugs[0], d)
	}
}

func TestSuggestionsRankedThenAlphabetical(t *testing.T) {
	sugs := Suggestions("helo", 5)
	want := []string{"halo", "held", "hell", "hello", "help"}
	if !reflect.DeepEqual(sugs, want) {
		t.Errorf("Suggestions(\"helo\", 5) = %v, want %v", sugs, want)
	}
}

func TestSuggestionsOrderingAndBounds(t *testing.T) {
	words := []string{"speling", "kitten", "helo", "quik", "apporpriate", "recieve"}
	for _, w := range words {
		sugs := Suggestions(w, 20)
		for i, s := range sugs {
			if d := Levenshtein(w, s); d > 2 {
				t.Fatalf("Suggestions(%q)[%d] = %q at distance %d (want <= 2)", w, i, s, d)
			}
			if i > 0 {
				prev := sugs[i-1]
				dPrev, dCur := Levenshtein(w, prev), Levenshtein(w, s)
				if dPrev > dCur || (dPrev == dCur && prev >= s) {
					t.Fatalf("Suggestions(%q) not sorted by (distance, word): ... %q, %q", w, prev, s)
				}
			}
		}
		if len(sugs) > 20 {
			t.Fatalf("Suggestions(%q) returned %d results, cap is 20", w, len(sugs))
		}
	}
}

func TestSuggestionsMaxCap(t *testing.T) {
	if got := Suggestions("helo", 1); len(got) != 1 {
		t.Errorf("Suggestions(\"helo\", 1) length = %d, want 1", len(got))
	}
	full := Suggestions("helo", 1000)
	if got := Suggestions("helo", 2); len(got) != 2 {
		t.Errorf("Suggestions(\"helo\", 2) length = %d, want 2 (full=%d)", len(got), len(full))
	}
	if got := Suggestions("helo", len(full)); len(got) != len(full) {
		t.Errorf("Suggestions with max == full count returned %d, want %d", len(got), len(full))
	}
}

func TestSuggestionsNonPositiveMax(t *testing.T) {
	for _, m := range []int{0, -1, -5} {
		if got := Suggestions("helo", m); len(got) != 0 {
			t.Errorf("Suggestions(\"helo\", %d) = %v, want empty", m, got)
		}
	}
}

func TestSuggestionsGibberishEmpty(t *testing.T) {
	for _, w := range []string{"zzqwxvjk", "qqqqqq", "xyzzyq"} {
		if got := Suggestions(w, 10); len(got) != 0 {
			t.Errorf("Suggestions(%q) = %v, want empty", w, got)
		}
	}
}

const checkFixture = "the quik brown dog jumped over the lazzy wall\n" +
	"hello 3.14 world 100,000\n"

func TestCheckMisses(t *testing.T) {
	misses := Check(checkFixture, TokenizeOpts{})
	if len(misses) != 2 {
		t.Fatalf("Check() returned %d misses (%v), want 2", len(misses), misses)
	}

	q := misses[0]
	if q.Token.Word != "quik" || q.Token.Line != 1 || q.Token.Start != 4 {
		t.Errorf("first miss = %#v, want word=quik line=1 start=4", q)
	}
	if len(q.Suggestions) == 0 || q.Suggestions[0] != "quick" {
		t.Errorf("suggestions for quik = %v, want first to be quick", q.Suggestions)
	}

	l := misses[1]
	if l.Token.Word != "lazzy" || l.Token.Line != 1 || l.Token.Start != 35 {
		t.Errorf("second miss = %#v, want word=lazzy line=1 start=35", l)
	}
	if len(l.Suggestions) == 0 || l.Suggestions[0] != "lazy" {
		t.Errorf("suggestions for lazzy = %v, want first to be lazy", l.Suggestions)
	}
}

func TestCheckCleanText(t *testing.T) {
	if misses := Check("the quick brown dog jumped over the lazy wall\n", TokenizeOpts{}); len(misses) != 0 {
		t.Errorf("Check() on clean text returned misses: %v", misses)
	}
}

func TestCheckNumbersNeverFlagged(t *testing.T) {
	misses := Check("12345 3.14 100,000 42 7", TokenizeOpts{})
	if len(misses) != 0 {
		t.Errorf("number tokens were flagged: %v", misses)
	}
}

func TestCheckIgnoreNumbers(t *testing.T) {
	misses := Check("the 42 quik", TokenizeOpts{IgnoreNumbers: true})
	if len(misses) != 1 || misses[0].Token.Word != "quik" {
		t.Errorf("Check with IgnoreNumbers = %v, want only quik", misses)
	}
}

func TestCheckCaseSensitive(t *testing.T) {
	text := "The Quick Brown Dog"
	if misses := Check(text, TokenizeOpts{}); len(misses) != 4 {
		t.Errorf("case-sensitive Check() on %q returned %d misses, want 4: %v", text, len(misses), misses)
	}
	if misses := Check(text, TokenizeOpts{IgnoreCase: true}); len(misses) != 0 {
		t.Errorf("case-insensitive Check() returned %d misses, want 0: %v", len(misses), misses)
	}
}

func TestCheckSuggestionsFromLowercasedWord(t *testing.T) {
	misses := Check("QUIK", TokenizeOpts{IgnoreCase: true})
	if len(misses) != 1 {
		t.Fatalf("Check(\"QUIK\", IgnoreCase) = %v, want 1 miss", misses)
	}
	m := misses[0]
	if m.Token.Word != "quik" {
		t.Errorf("lowercased token = %q, want \"quik\"", m.Token.Word)
	}
	if len(m.Suggestions) == 0 || m.Suggestions[0] != "quick" {
		t.Errorf("suggestions = %v, want first quick", m.Suggestions)
	}
}