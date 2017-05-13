// Copyright 2016 Qiang Xue. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package config

import (
	"testing"
)

func TestLoadIniFile(t *testing.T) {
	// YAML is tested differently because it loads a hash as map[interface{}]interface{}
	c := New()
	err := c.Load("testdata/c1.ini", "testdata/c2.ini")
	if err != nil {
		t.Error(err)
	}
	tests := []struct {
		name     string
		expected interface{}
	}{
		{"A1", "a1"},
		{"A2", 3},
		{"A3", true},
		{"A4", 2.13},
		{"A5", "a5"},
		{"A6.B1", "b1"},
		{"A6.B2.C1", "c1"},
		{"A6.B2.C2", "c2"},
	}
	for _, test := range tests {
		if c.Get(test.name) != test.expected {
			t.Errorf(`Get(%q) = %v, expected %v`, test.name, c.Get(test.name), test.expected)
		}
	}
}

