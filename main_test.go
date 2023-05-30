package main

import (
	"bytes"
	"errors"
	"strings"
	"testing"
)

func TestRun(t *testing.T) {
	for name, test := range map[string]struct {
		stdin  string
		in     string
		args   []string
		expect string
	}{
		"Simple hello world": {
			in:     "hello, world",
			expect: "hello, world",
		},
		"Closed delimiter": {
			in:     "hello, %{ printf world }%",
			expect: "hello, world",
		},
		"Use stdin": {
			stdin:  "world",
			in:     "hello, %{ cat - }%",
			expect: "hello, world",
		},
		"Use arguments": {
			in:     "hello, %{ printf $1 }%",
			args:   []string{"world"},
			expect: "hello, world",
		},
		"stdin with arguments": {
			stdin:  "world",
			in:     `hello, %{ printf "world $1" }%`,
			args:   []string{"again!"},
			expect: "hello, world again!",
		},
	} {
		t.Run(name, func(t *testing.T) {
			stdin := strings.NewReader(test.stdin)
			in := strings.NewReader(test.in)
			out := &bytes.Buffer{}

			if err := Run(stdin, in, test.args, out, defaultTempFile, defaultShebang); err != nil {
				t.Error(err)
			}

			if got := out.String(); got != test.expect {
				t.Errorf("got output:\n%s\nwanted:\n%s\n", got, test.expect)
			}
		})
	}
}

func TestUnclosedDelimter(t *testing.T) {
	var args []string
	out := &bytes.Buffer{}
	in := strings.NewReader("hello, %{ cat")

	if err := Run(nil, in, args, out, defaultTempFile, defaultShebang); !errors.Is(err, ErrUnclosedDelimiter) {
		t.Error("expect error due to unclosed delimeter, instead got", err)
	}
}
