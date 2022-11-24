package gostrreplacer_test

import (
	"bytes"
	"errors"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/google/go-cmp/cmp"
	"github.com/tomtwinkle/gostrreplacer"
	"golang.org/x/text/transform"
)

func TestCustomTransformer_Transform(t *testing.T) {
	tests := map[string]struct {
		in         string
		matchStr   string
		replaceStr string
		want       string
		wantError  error
	}{
		"no match: only prefix match string": {
			in:         "testtesttesttest",
			matchStr:   "testing",
			replaceStr: "tested",
			want:       "testtesttesttest",
		},
		"match: prefix match string": {
			in:         "testingtesttesttest",
			matchStr:   "testing",
			replaceStr: "tested",
			want:       "testedtesttesttest",
		},
		"match: prefix match string2": {
			in:         "testingtestingtesttest",
			matchStr:   "testing",
			replaceStr: "tested",
			want:       "testedtestedtesttest",
		},
		"match: between match string": {
			in:         "testtestingtesttest",
			matchStr:   "testing",
			replaceStr: "tested",
			want:       "testtestedtesttest",
		},
		"match: between match string2": {
			in:         "testtestingtestingtest",
			matchStr:   "testing",
			replaceStr: "tested",
			want:       "testtestedtestedtest",
		},
		"match: replaceStr > matchStr": {
			in:         "testtestingtestingtest",
			matchStr:   "testing",
			replaceStr: "testinging",
			want:       "testtestingingtestingingtest",
		},
		"match: replaceStr < matchStr too long": {
			in:         strings.Repeat("testtestingtesttestingtest", 1000),
			matchStr:   "testing",
			replaceStr: "tested",
			want:       strings.Repeat("testtestedtesttestedtest", 1000),
		},
		"match: replaceStr > matchStr too long": {
			in:         strings.Repeat("testtestingtesttestingtest", 1000),
			matchStr:   "testing",
			replaceStr: "testinging",
			want:       strings.Repeat("testtestingingtesttestingingtest", 1000),
		},
		"match: multibyte replaceStr < matchStr too long": {
			in:         strings.Repeat("🍣🍺鰤魬🍥🍜👪", 1000),
			matchStr:   "🍥🍜",
			replaceStr: "🐙",
			want:       strings.Repeat("🍣🍺鰤魬🐙👪", 1000),
		},
		"match: multibyte replaceStr > matchStr too long": {
			in:         strings.Repeat("🍣🍺鰤魬🍥🍜👪", 1000),
			matchStr:   "鰤魬",
			replaceStr: "💯💯💯💯",
			want:       strings.Repeat("🍣🍺💯💯💯💯🍥🍜👪", 1000),
		},
	}

	for n, v := range tests {
		name := n
		tt := v

		t.Run(name, func(t *testing.T) {
			var buf bytes.Buffer
			w := transform.NewWriter(&buf, gostrreplacer.NewTransformer(tt.matchStr, tt.replaceStr))
			if _, err := w.Write([]byte(tt.in)); err != nil {
				if tt.wantError != nil && errors.Is(err, tt.wantError) {
					return
				}
				t.Error(err)
			}
			if err := w.Close(); err != nil {
				t.Error(err)
			}
			if len([]rune(tt.want)) != len([]rune(buf.String())) {
				t.Errorf("string length does not match %d=%d\n%s\n%s", len([]rune(tt.want)), len([]rune(buf.String())), tt.want, buf.String())
			}
			if tt.want != buf.String() {
				t.Errorf("string does not match \n%s\n%s", tt.want, buf.String())
			}
		})
	}
}

// Note: go test -fuzz=FuzzTransformer ./...
// nolint: typecheck
func FuzzTransformer(f *testing.F) {
	seeds := [][]byte{
		bytes.Repeat([]byte("一二三四五六七八九十拾壱🍣🍺"), 1000),
		bytes.Repeat([]byte("一二三四🍣五六七八九🍺十拾壱"), 3000),
	}

	for _, b := range seeds {
		f.Add(b)
	}

	f.Fuzz(func(t *testing.T, p []byte) {
		tr := gostrreplacer.NewTransformer("🍣🍺", "🍖🍻")
		for len(p) > 0 {
			if !utf8.Valid(p) {
				t.Skip()
			}
			got, n, err := transform.Bytes(tr, p)
			if err != nil {
				t.Fatal("unexpected error:", err)
			}
			wantStr := strings.ReplaceAll(string(p), "🍣🍺", "🍖🍻")
			if len([]byte(wantStr)) != len(got) {
				t.Errorf("replace byte size not match %d=%d\n%s\n%s", []byte(wantStr), len(got), wantStr, got)
			}
			if diff := cmp.Diff(wantStr, string(got)); diff != "" {
				t.Errorf("string is not match(- +):\n%s", diff)
			}
			p = p[n:]
		}
	})
}
