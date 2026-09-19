package wordcheck

import (
	"reflect"
	"testing"
)

func TestTokenizeWordsParagraph(t *testing.T) {
	const para = "Hello, world! Don't stop 3.14 now.\nCheck v2.0 code, please.\n"
	got := TokenizeWords(para, TokenizeOpts{})
	want := []Token{
		{Word: "Hello", Start: 0, End: 5, Line: 1},
		{Word: "world", Start: 7, End: 12, Line: 1},
		{Word: "Don't", Start: 14, End: 19, Line: 1},
		{Word: "stop", Start: 20, End: 24, Line: 1},
		{Word: "3.14", Start: 25, End: 29, Line: 1, Num: true},
		{Word: "now", Start: 30, End: 33, Line: 1},
		{Word: "Check", Start: 35, End: 40, Line: 2},
		{Word: "v", Start: 41, End: 42, Line: 2},
		{Word: "2.0", Start: 42, End: 45, Line: 2, Num: true},
		{Word: "code", Start: 46, End: 50, Line: 2},
		{Word: "please", Start: 52, End: 58, Line: 2},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("mismatch:\n got %#v\nwant %#v", got, want)
	}
}

func TestTokenizeWordsIgnoreNumbers(t *testing.T) {
	const para = "Hello, world! Don't stop 3.14 now.\nCheck v2.0 code, please.\n"
	got := TokenizeWords(para, TokenizeOpts{IgnoreNumbers: true})
	for _, tk := range got {
		if tk.Num {
			t.Fatalf("number token not skipped: %#v", tk)
		}
	}
	if len(got) != 9 {
		t.Fatalf("expected 9 word tokens, got %d (%#v)", len(got), got)
	}
}

func TestTokenizeWordsIgnoreCase(t *testing.T) {
	const para = "Hello, world! Don't stop 3.14 now.\nCheck v2.0 code, please.\n"
	got := TokenizeWords(para, TokenizeOpts{IgnoreCase: true})
	for _, tk := range got {
		if tk.Num {
			continue
		}
		for _, c := range tk.Word {
			if c < 'a' || c > 'z' {
				// apostrophes are allowed inside a word
				if c == '\'' {
					continue
				}
				t.Fatalf("non-lowercase word token: %#v", tk)
			}
		}
	}
	if got[0].Word != "hello" || got[2].Word != "don't" || got[6].Word != "check" {
		t.Fatalf("IgnoreCase did not lowercase: %#v", got)
	}
}

func TestTokenizeWordsApostrophes(t *testing.T) {
	const s = "'tis rock'n'roll, y'all. --won't--"
	got := TokenizeWords(s, TokenizeOpts{})
	want := []Token{
		{Word: "tis", Start: 1, End: 4, Line: 1},
		{Word: "rock'n'roll", Start: 5, End: 16, Line: 1},
		{Word: "y'all", Start: 18, End: 23, Line: 1},
		{Word: "won't", Start: 27, End: 32, Line: 1},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("mismatch:\n got %#v\nwant %#v", got, want)
	}
}

func TestTokenizeWordsLines(t *testing.T) {
	const s = "one\n\ntwo three\n"
	got := TokenizeWords(s, TokenizeOpts{})
	want := []Token{
		{Word: "one", Start: 0, End: 3, Line: 1},
		{Word: "two", Start: 5, End: 8, Line: 3},
		{Word: "three", Start: 9, End: 14, Line: 3},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("mismatch:\n got %#v\nwant %#v", got, want)
	}
}

func TestTokenizeWordsDecimalAndComma(t *testing.T) {
	got := TokenizeWords("1,000.5 and 3.14 and 1,000", TokenizeOpts{})
	if len(got) != 5 {
		t.Fatalf("got %d tokens, want 5: %#v", len(got), got)
	}
	want := []Token{
		{Word: "1,000.5", Start: 0, End: 7, Line: 1, Num: true},
		{Word: "and", Start: 8, End: 11, Line: 1},
		{Word: "3.14", Start: 12, End: 16, Line: 1, Num: true},
		{Word: "and", Start: 17, End: 20, Line: 1},
		{Word: "1,000", Start: 21, End: 26, Line: 1, Num: true},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("mismatch:\n got %#v\nwant %#v", got, want)
	}
}

func TestTokenizeWordsUnicodeSkipped(t *testing.T) {
	// A non-ASCII letter is not a-z/A-Z, so each of its bytes is skipped
	// while the ASCII letters before it still tokenize.
	got := TokenizeWords("café", TokenizeOpts{})
	want := []Token{{Word: "caf", Start: 0, End: 3, Line: 1}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("mismatch:\n got %#v\nwant %#v", got, want)
	}
}

func TestTokenizeWordsTrailingApostrophe(t *testing.T) {
	// A trailing apostrophe is punctuation, not part of the word.
	got := TokenizeWords("dogs' bowl", TokenizeOpts{})
	want := []Token{
		{Word: "dogs", Start: 0, End: 4, Line: 1},
		{Word: "bowl", Start: 6, End: 10, Line: 1},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("mismatch:\n got %#v\nwant %#v", got, want)
	}
}

func TestTokenizeWordsNumbersOnlyIgnored(t *testing.T) {
	got := TokenizeWords("123 456.7 8,000", TokenizeOpts{IgnoreNumbers: true})
	if len(got) != 0 {
		t.Fatalf("expected no tokens, got %#v", got)
	}
}

func TestTokenizeWordsEmptyString(t *testing.T) {
	if got := TokenizeWords("", TokenizeOpts{}); len(got) != 0 {
		t.Fatalf("expected no tokens, got %#v", got)
	}
}

func TestTokenizeWordsCRLF(t *testing.T) {
	got := TokenizeWords("a\r\nlonger", TokenizeOpts{})
	want := []Token{
		{Word: "a", Start: 0, End: 1, Line: 1},
		{Word: "longer", Start: 3, End: 9, Line: 2},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("mismatch:\n got %#v\nwant %#v", got, want)
	}
}

func TestTokenizeWordsMixedCasePreserved(t *testing.T) {
	got := TokenizeWords("Hello WoRLD", TokenizeOpts{})
	if got[0].Word != "Hello" || got[1].Word != "WoRLD" {
		t.Fatalf("case not preserved without IgnoreCase: %#v", got)
	}
}

func TestTokenizeWordsStandaloneAndLeadingApostrophe(t *testing.T) {
	got := TokenizeWords("' 'hello '", TokenizeOpts{})
	want := []Token{{Word: "hello", Start: 3, End: 8, Line: 1}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("mismatch:\n got %#v\nwant %#v", got, want)
	}
}

func TestTokenizeWordsApostropheBetweenNumbersNotWord(t *testing.T) {
	// "3'4" is not a word token; the apostrophe has no letters around it
	// in a way that makes a word, so "3" and "4" are separate numbers and
	// the apostrophe is dropped as punctuation.
	got := TokenizeWords("3'4 hello", TokenizeOpts{})
	want := []Token{
		{Word: "3", Start: 0, End: 1, Line: 1, Num: true},
		{Word: "4", Start: 2, End: 3, Line: 1, Num: true},
		{Word: "hello", Start: 4, End: 9, Line: 1},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("mismatch:\n got %#v\nwant %#v", got, want)
	}
}
