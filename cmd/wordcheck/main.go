package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/Conedope/wordcheck"
)

const version = "1.0.0"

func main() {
	os.Exit(run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}

func usage(w io.Writer) {
	fmt.Fprintf(w, `wordcheck %s - English spelling checker with an embedded dictionary

Usage:
  wordcheck [options] FILE...
  wordcheck [options]          # read from stdin

Options:
  -replace old=new    replace every occurrence of the word old with new in
                      the given files (in place) and print a replacement count
  -list-dict          print the embedded dictionary (one word per line)
  -max N              with -list-dict, print only the first N words
  -json               report misspellings as a JSON array on stdout (the
                      "words checked" summary then goes to stderr)
  -ignore-case        check case-insensitively (suggestions derived from the
                      lowercased word)
  -ignore-numbers     skip number tokens entirely (numbers are never flagged)
  -no-suggest         do not attach suggestions to reported words
  -version            print version and exit
  -help               print this help and exit

Exit status:
  0  no unrecognized words (or successful -replace / -list-dict)
  1  at least one unrecognized word was found
  2  usage or I/O error
`, version)
}

type jsonMiss struct {
	File        string   `json:"file"`
	Line        int      `json:"line"`
	Column      int      `json:"column"`
	Word        string   `json:"word"`
	Suggestions []string `json:"suggestions,omitempty"`
}

func run(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("wordcheck", flag.ContinueOnError)
	fs.SetOutput(stderr)

	var (
		replArg     = fs.String("replace", "", "old=new replacement to apply in place")
		listDict    = fs.Bool("list-dict", false, "print the embedded dictionary")
		maxN        = fs.Int("max", 0, "with -list-dict, print only the first N words")
		jsonOut     = fs.Bool("json", false, "report misspellings as a JSON array")
		ignoreCase  = fs.Bool("ignore-case", false, "case-insensitive check")
		ignoreNums  = fs.Bool("ignore-numbers", false, "skip number tokens")
		noSuggest   = fs.Bool("no-suggest", false, "do not attach suggestions")
		showVersion = fs.Bool("version", false, "print version and exit")
	)
	fs.Usage = func() { usage(stderr) }

	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			usage(stdout)
			return 0
		}
		return 2
	}

	if *showVersion {
		fmt.Fprintln(stdout, "wordcheck "+version)
		return 0
	}

	opts := wordcheck.TokenizeOpts{IgnoreCase: *ignoreCase, IgnoreNumbers: *ignoreNums}

	if *listDict {
		return listDictionary(stdout, *maxN)
	}

	files := fs.Args()

	if *replArg != "" {
		old, new, ok := strings.Cut(*replArg, "=")
		if !ok || old == "" {
			fmt.Fprintln(stderr, "wordcheck: -replace must be old=new")
			return 2
		}
		if len(files) == 0 {
			fmt.Fprintln(stderr, "wordcheck: -replace requires at least one FILE")
			return 2
		}
		return replaceInFiles(files, old, new, opts, stdout, stderr)
	}

	return checkFiles(files, stdin, opts, *jsonOut, *noSuggest, stdout, stderr)
}

// listDictionary prints the embedded dictionary in alphabetical order,
// truncated to n entries when n > 0.
func listDictionary(w io.Writer, n int) int {
	written := 0
	for _, word := range wordcheck.DictionaryWords() {
		if n > 0 && written >= n {
			break
		}
		fmt.Fprintln(w, word)
		written++
	}
	return 0
}

// lineStarts returns the byte offset at which each 1-based line begins.
func lineStarts(s string) []int {
	starts := []int{0}
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			starts = append(starts, i+1)
		}
	}
	return starts
}

func column(offsets []int, line int, start int) int {
	if line-1 < len(offsets) {
		return start - offsets[line-1] + 1
	}
	return start + 1
}

func checkFiles(files []string, stdin io.Reader, opts wordcheck.TokenizeOpts, jsonOut, noSuggest bool, stdout, stderr io.Writer) int {
	type input struct {
		name  string
		text  string
		total int
	}
	var inputs []input

	if len(files) == 0 {
		data, err := io.ReadAll(stdin)
		if err != nil {
			fmt.Fprintln(stderr, "wordcheck: "+err.Error())
			return 2
		}
		inputs = append(inputs, input{name: "-", text: string(data), total: len(wordcheck.TokenizeWords(string(data), opts))})
	} else {
		for _, f := range files {
			data, err := os.ReadFile(f)
			if err != nil {
				fmt.Fprintln(stderr, "wordcheck: "+err.Error())
				return 2
			}
			inputs = append(inputs, input{name: f, text: string(data), total: len(wordcheck.TokenizeWords(string(data), opts))})
		}
	}

	var misses []jsonMiss
	totalWords := 0
	for _, in := range inputs {
		totalWords += in.total
		offsets := lineStarts(in.text)
		for _, m := range wordcheck.Check(in.text, opts) {
			sugs := m.Suggestions
			if noSuggest {
				sugs = nil
			}
			orig := in.text[m.Token.Start:m.Token.End]
			misses = append(misses, jsonMiss{
				File:        in.name,
				Line:        m.Token.Line,
				Column:      column(offsets, m.Token.Line, m.Token.Start),
				Word:        orig,
				Suggestions: sugs,
			})
		}
	}

	if jsonOut {
		enc := json.NewEncoder(stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(misses); err != nil {
			fmt.Fprintln(stderr, "wordcheck: "+err.Error())
			return 2
		}
		// Keep stdout a pure JSON document; the count goes to stderr.
		fmt.Fprintf(stderr, "%d words checked, %d unrecognized\n", totalWords, len(misses))
	} else {
		for _, m := range misses {
			fmt.Fprintf(stdout, "%s:%d:%d %s", m.File, m.Line, m.Column, m.Word)
			if len(m.Suggestions) > 0 {
				fmt.Fprintf(stdout, " -> %s", strings.Join(m.Suggestions, ", "))
			}
			fmt.Fprintln(stdout)
		}
		fmt.Fprintf(stdout, "%d words checked, %d unrecognized\n", totalWords, len(misses))
	}
	if len(misses) > 0 {
		return 1
	}
	return 0
}

func replaceInFiles(files []string, old, new string, opts wordcheck.TokenizeOpts, stdout, stderr io.Writer) int {
	anyErr := false
	for _, f := range files {
		data, err := os.ReadFile(f)
		if err != nil {
			fmt.Fprintln(stderr, "wordcheck: "+err.Error())
			anyErr = true
			continue
		}
		text := string(data)
		replaced, count := replaceWord(text, old, new, opts)
		if count > 0 {
			if err := os.WriteFile(f, []byte(replaced), 0o644); err != nil {
				fmt.Fprintln(stderr, "wordcheck: "+err.Error())
				anyErr = true
				continue
			}
		}
		fmt.Fprintf(stdout, "%s: %d replacements\n", f, count)
	}
	if anyErr {
		return 2
	}
	return 0
}

// replaceWord swaps every token equal to old for new. Because replacements
// shift offsets, edits are applied from the last token backwards.
func replaceWord(text, old, new string, opts wordcheck.TokenizeOpts) (string, int) {
	tokens := wordcheck.TokenizeWords(text, opts)
	type edit struct{ start, end int }
	var edits []edit
	for _, t := range tokens {
		if t.Num {
			continue
		}
		if t.Word == old {
			edits = append(edits, edit{t.Start, t.End})
		}
	}
	if len(edits) == 0 {
		return text, 0
	}
	var b strings.Builder
	b.Grow(len(text) + len(new)*len(edits))
	prev := 0
	for _, e := range edits {
		b.WriteString(text[prev:e.start])
		b.WriteString(new)
		prev = e.end
	}
	b.WriteString(text[prev:])
	return b.String(), len(edits)
}
