package jsonpreprocess

import (
	"testing"
)

func TestTrimComment(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"empty", ``, ``},
		{"spaces", " \t\n", " \t\n"},
		{"text", `[1, 2]`, `[1, 2]`},
		{"text with string", `{"foo": 1}`, `{"foo": 1}`},
		{"text with line comment ",
			`[1, 2] // this is a line comment`,
			`[1, 2] `},
		{"text with block comment ",
			"[1, 2, /* this is\na block comment */ 3]",
			"[1, 2,  3]"},
		{"text with string and comment",
			`{"url": "http://example.com"} // this is a line comment`,
			`{"url": "http://example.com"} `},
	}
	for _, test := range tests {
		actual, err := TrimComment(test.input)
		if err != nil {
			t.Fatal(err)
		}
		if actual != test.expected {
			t.Errorf("%s: got\n\t%+q\nexpected\n\t%q", test.name, actual, test.expected)
		}
	}
}
