package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

var (
	binOnce sync.Once
	binPath string
	binErr  error
)

// buildBinary compiles the CLI once per test run and returns its path.
// The binary lives in a scratch directory that outlives any single test so
// every test can exec it.
func buildBinary(t *testing.T) string {
	t.Helper()
	binOnce.Do(func() {
		dir, err := os.MkdirTemp("", "wordcheck-bin")
		if err != nil {
			binErr = err
			return
		}
		binPath = filepath.Join(dir, "wordcheck")
		cmd := exec.Command("go", "build", "-o", binPath, ".")
		cmd.Env = append(os.Environ(), "CGO_ENABLED=0")
		out, buildErr := cmd.CombinedOutput()
		if buildErr != nil {
			binErr = buildErr
			t.Logf("build output:\n%s", out)
		}
	})
	if binErr != nil {
		t.Fatalf("failed to build CLI: %v", binErr)
	}
	return binPath
}

// runCLI runs the built binary with args and optional stdin, returning
// stdout, stderr and the exit code.
func runCLI(t *testing.T, stdin string, args ...string) (string, string, int) {
	t.Helper()
	cmd := exec.Command(buildBinary(t), args...)
	cmd.Env = append(os.Environ(), "CGO_ENABLED=0")
	if stdin != "" {
		cmd.Stdin = strings.NewReader(stdin)
	}
	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	code := 0
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			code = ee.ExitCode()
		} else {
			t.Fatalf("running CLI: %v", err)
		}
	}
	return stdout.String(), stderr.String(), code
}

func writeFixture(t *testing.T, content string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "sample.txt")
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

const misspellText = "the quik brown dog jumped over the lazzy wall\n"

func TestCLICheckMisspellings(t *testing.T) {
	p := writeFixture(t, misspellText)
	stdout, _, code := runCLI(t, "", p)
	if code != 1 {
		t.Fatalf("exit code = %d, want 1", code)
	}
	name := filepath.Base(p)
	line := name + ":1:5 quik -> quick"
	if !strings.Contains(stdout, line) {
		t.Errorf("stdout missing %q\nstdout:\n%s", line, stdout)
	}
	if !strings.Contains(stdout, "lazzy -> lazy") {
		t.Errorf("stdout missing lazzy suggestion\nstdout:\n%s", stdout)
	}
	if !strings.Contains(stdout, "9 words checked, 2 unrecognized") {
		t.Errorf("stdout missing summary line\nstdout:\n%s", stdout)
	}
}

func TestCLICheckCleanFile(t *testing.T) {
	p := writeFixture(t, "the quick brown dog jumped over the lazy wall\n")
	stdout, _, code := runCLI(t, "", p)
	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
	if !strings.Contains(stdout, "9 words checked, 0 unrecognized") {
		t.Errorf("stdout missing clean summary\nstdout:\n%s", stdout)
	}
}

func TestCLIJSONShape(t *testing.T) {
	p := writeFixture(t, misspellText)
	stdout, stderr, code := runCLI(t, "", "-json", p)
	if code != 1 {
		t.Fatalf("exit code = %d, want 1", code)
	}
	var out []struct {
		File        string   `json:"file"`
		Line        int      `json:"line"`
		Column      int      `json:"column"`
		Word        string   `json:"word"`
		Suggestions []string `json:"suggestions,omitempty"`
	}
	if err := json.Unmarshal([]byte(stdout), &out); err != nil {
		t.Fatalf("stdout is not valid JSON: %v\nstdout:\n%s", err, stdout)
	}
	if len(out) != 2 {
		t.Fatalf("JSON array has %d entries, want 2: %+v", len(out), out)
	}
	if out[0].File != p || out[0].Word != "quik" || out[0].Line != 1 || out[0].Column != 5 {
		t.Errorf("first JSON entry = %+v, want file=%s word=quik line=1 column=5", out[0], p)
	}
	if out[1].Word != "lazzy" || out[1].Column != 36 {
		t.Errorf("second JSON entry = %+v, want word=lazzy column=36", out[1])
	}
	if len(out[0].Suggestions) == 0 || out[0].Suggestions[0] != "quick" {
		t.Errorf("first entry suggestions = %v, want quick first", out[0].Suggestions)
	}
	if !strings.Contains(stderr, "9 words checked, 2 unrecognized") {
		t.Errorf("summary should be on stderr in JSON mode, got %q", stderr)
	}
}

func TestCLIJSONCleanFile(t *testing.T) {
	p := writeFixture(t, "hello world\n")
	stdout, _, code := runCLI(t, "", "-json", p)
	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
	var out []map[string]any
	if err := json.Unmarshal([]byte(stdout), &out); err != nil {
		t.Fatalf("stdout not valid JSON: %v\n%s", err, stdout)
	}
	if len(out) != 0 {
		t.Fatalf("expected empty JSON array, got %+v", out)
	}
}

func TestCLIJSONNoSuggest(t *testing.T) {
	p := writeFixture(t, misspellText)
	stdout, _, code := runCLI(t, "", "-json", "-no-suggest", p)
	if code != 1 {
		t.Fatalf("exit code = %d, want 1", code)
	}
	var out []map[string]any
	if err := json.Unmarshal([]byte(stdout), &out); err != nil {
		t.Fatalf("stdout not valid JSON: %v", err)
	}
	if _, ok := out[0]["suggestions"]; ok {
		t.Errorf("suggestions key present despite -no-suggest: %+v", out[0])
	}
}

func TestCLIReplace(t *testing.T) {
	p := writeFixture(t, "hello world hello\nhello again\n")
	stdout, stderr, code := runCLI(t, "", "-replace", "hello=hi", p)
	if code != 0 {
		t.Fatalf("exit code = %d, want 0; stderr: %s", code, stderr)
	}
	data, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	if got := string(data); got != "hi world hi\nhi again\n" {
		t.Errorf("file after -replace = %q, want %q", got, "hi world hi\nhi again\n")
	}
	if !strings.Contains(stdout, "3 replacements") {
		t.Errorf("stdout = %q, want replacement count", stdout)
	}
}

func TestCLIReplaceSubwordOnly(t *testing.T) {
	// "new" must be replaced token-wise, not inside "renewal".
	p := writeFixture(t, "renewal new\n")
	stdout, _, code := runCLI(t, "", "-replace", "new=old", p)
	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
	data, _ := os.ReadFile(p)
	if got := string(data); got != "renewal old\n" {
		t.Errorf("file after -replace = %q, want %q", got, "renewal old\n")
	}
	if !strings.Contains(stdout, "1 replacements") {
		t.Errorf("stdout = %q, want 1 replacement", stdout)
	}
}

func TestCLIReplaceInvalid(t *testing.T) {
	p := writeFixture(t, "hello\n")
	_, stderr, code := runCLI(t, "", "-replace", "novalue", p)
	if code != 2 {
		t.Fatalf("exit code = %d, want 2", code)
	}
	if !strings.Contains(stderr, "-replace must be old=new") {
		t.Errorf("stderr = %q, want usage error message", stderr)
	}
}

func TestCLIReplaceNoFile(t *testing.T) {
	_, stderr, code := runCLI(t, "", "-replace", "hello=hi")
	if code != 2 {
		t.Fatalf("exit code = %d, want 2", code)
	}
	if !strings.Contains(stderr, "-replace requires at least one FILE") {
		t.Errorf("stderr = %q, want error message", stderr)
	}
}

func TestCLIIgnoreNumbers(t *testing.T) {
	content := "hello 12345\n"
	p := writeFixture(t, content)

	stdout, _, code := runCLI(t, "", p)
	if code != 0 {
		t.Fatalf("default exit code = %d, want 0", code)
	}
	if !strings.Contains(stdout, "2 words checked, 0 unrecognized") {
		t.Errorf("default stdout missing number token count: %q", stdout)
	}

	stdout, _, code = runCLI(t, "", "-ignore-numbers", p)
	if code != 0 {
		t.Fatalf("-ignore-numbers exit code = %d, want 0", code)
	}
	if !strings.Contains(stdout, "1 words checked, 0 unrecognized") {
		t.Errorf("-ignore-numbers stdout = %q, want 1 word", stdout)
	}
}

func TestCLIListDict(t *testing.T) {
	stdout, _, code := runCLI(t, "", "-list-dict")
	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
	n := len(strings.Split(strings.TrimRight(stdout, "\n"), "\n"))
	if n < 2000 {
		t.Errorf("-list-dict printed %d words, want > 2000", n)
	}
	if !strings.HasPrefix(stdout, "a\nabout") {
		t.Errorf("-list-dict should start alphabetically, got %q", stdout[:20])
	}
	if !strings.Contains(stdout, "spelling\n") {
		t.Errorf("-list-dict output missing \"spelling\"")
	}
}

func TestCLIListDictMax(t *testing.T) {
	stdout, _, code := runCLI(t, "", "-list-dict", "-max", "5")
	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
	if n := len(strings.Split(strings.TrimRight(stdout, "\n"), "\n")); n != 5 {
		t.Errorf("-max 5 printed %d words, want 5", n)
	}
}

func TestCLIMissingFile(t *testing.T) {
	p := filepath.Join(t.TempDir(), "does-not-exist.txt")
	_, stderr, code := runCLI(t, "", p)
	if code != 2 {
		t.Fatalf("exit code = %d, want 2", code)
	}
	if !strings.Contains(stderr, "does-not-exist.txt") {
		t.Errorf("stderr = %q, want missing file name", stderr)
	}
}

func TestCLIStdin(t *testing.T) {
	stdout, _, code := runCLI(t, "the quik brown dog\n")
	if code != 1 {
		t.Fatalf("exit code = %d, want 1", code)
	}
	if !strings.Contains(stdout, "-:1:5 quik -> quick") {
		t.Errorf("stdin output = %q, want -:1:5 quik -> quick", stdout)
	}
}

func TestCLIVersion(t *testing.T) {
	stdout, _, code := runCLI(t, "", "-version")
	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
	if strings.TrimSpace(stdout) != "wordcheck "+version {
		t.Errorf("stdout = %q, want %q", stdout, "wordcheck "+version)
	}
}

func TestCLIHelp(t *testing.T) {
	stdout, _, code := runCLI(t, "", "-help")
	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
	for _, s := range []string{"Usage:", "-replace", "-list-dict", "-json", "-ignore-case", "-ignore-numbers", "-no-suggest", "-version"} {
		if !strings.Contains(stdout, s) {
			t.Errorf("-help output missing %q\n%s", s, stdout)
		}
	}
}

func TestCLINoSuggest(t *testing.T) {
	p := writeFixture(t, misspellText)
	stdout, _, code := runCLI(t, "", "-no-suggest", p)
	if code != 1 {
		t.Fatalf("exit code = %d, want 1", code)
	}
	if strings.Contains(stdout, "->") {
		t.Errorf("-no-suggest output still contains suggestions:\n%s", stdout)
	}
	if !strings.Contains(stdout, ":1:5 quik\n") {
		t.Errorf("output missing bare miss line:\n%s", stdout)
	}
}

func TestCLIIgnoreCase(t *testing.T) {
	p := writeFixture(t, "The Quick Brown Dog\n")
	_, _, code := runCLI(t, "", p)
	if code != 1 {
		t.Fatalf("case-sensitive exit code = %d, want 1", code)
	}
	_, _, code = runCLI(t, "", "-ignore-case", p)
	if code != 0 {
		t.Fatalf("-ignore-case exit code = %d, want 0", code)
	}
}