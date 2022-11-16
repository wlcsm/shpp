package main

import (
	"bytes"
	"errors"
	"strings"
	"testing"
)

func TestRun(t *testing.T) {
	type testInput struct {
		Stdin string
		In    string
		Args  []string
	}

	tests := []struct {
		name    string
		cfg     testInput
		wantW   string
		wantErr error
	}{
		{
			name: "Simple hello world",
			cfg: testInput{
				In: "hello, world",
			},
			wantW:   "hello, world",
			wantErr: nil,
		},
		{
			name: "Unclosed delimiter",
			cfg: testInput{
				In: "hello, %{ cat",
			},
			wantW:   "",
			wantErr: ErrUnclosedDelimiter,
		},
		{
			name: "Closed delimiter",
			cfg: testInput{
				In: "hello, %{ printf world }%",
			},
			wantW:   "hello, world",
			wantErr: nil,
		},
		{
			name: "Use stdin",
			cfg: testInput{
				Stdin: "world",
				In:    "hello, %{ printf world }%",
			},
			wantW:   "hello, world",
			wantErr: nil,
		},
		{
			name: "Use arguments",
			cfg: testInput{
				In:   "hello, %{ printf $0 }%",
				Args: []string{"world"},
			},
			wantW:   "hello, world",
			wantErr: nil,
		},
		{
			name: "Stdin with arguments",
			cfg: testInput{
				Stdin: "world",
				In:    `hello, %{ printf "world $0" }%`,
				Args:  []string{"again!"},
			},
			wantW:   "hello, world again!",
			wantErr: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out := &bytes.Buffer{}
			cfg := Config{
				Stdin: strings.NewReader(tt.cfg.Stdin),
				In:    strings.NewReader(tt.cfg.In),
				Args:  tt.cfg.Args,
				Out:   out,
			}

			if err := Run(cfg); err != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("Run() error = %v, wantErr %v", err, tt.wantErr)
				}
				return
			}
			if gotW := out.String(); gotW != tt.wantW {
				t.Errorf("Run() = %v, want %v", gotW, tt.wantW)
			}
		})
	}
}
