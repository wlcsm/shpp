package main

import (
	"bytes"
	"errors"
	"os/exec"
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
			in:     "hello, %{ printf $0 }%",
			args:   []string{"world"},
			expect: "hello, world",
		},
	} {
		t.Run(name, func(t *testing.T) {
			stdin := strings.NewReader(test.stdin)
			in := strings.NewReader(test.in)
			out := &bytes.Buffer{}

			command := "/bin/sh"
			execArgs := append([]string{"-c", ""}, test.args...)

			exe := func(arg string) error {
				execArgs[1] = arg

				cmd := exec.Command(command, execArgs...)
				cmd.Stdin = stdin
				cmd.Stdout = out
				cmd.Stderr = out

				return cmd.Run()
			}

			if err := ExecCodeBlocks(in, out, exe); err != nil {
				t.Error(err)
			}

			if got := out.String(); got != test.expect {
				t.Errorf("got output:\n%s\nwanted:\n%s\n", got, test.expect)
			}
		})
	}
}

func TestUnclosedDelimter(t *testing.T) {
	out := &bytes.Buffer{}
	in := strings.NewReader("hello, %{ cat")

	if err := ExecCodeBlocks(in, out, nil); !errors.Is(err, ErrUnclosedDelimiter) {
		t.Error("expect error due to unclosed delimeter, instead got", err)
	}
}
