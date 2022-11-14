package main

import (
	"bytes"
	"errors"
	"strings"
	"testing"
)

func TestRun(t *testing.T) {
	tests := []struct {
		name    string
		in      string
		wantW   string
		wantErr error
	}{
		{
			name:    "Simple hello world",
			in:      `hello, world`,
			wantW:   "hello, world",
			wantErr: nil,
		},
		{
			name:    "Unclosed delimiter test",
			in:      `hello, %{world`,
			wantW:   "",
			wantErr: ErrUnclosedDelimiter,
		},
		{
			name:    "Closed delimiter test",
			in:      `hello, %{printf world}%`,
			wantW:   "hello, world",
			wantErr: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := strings.NewReader(tt.in)
			w := &bytes.Buffer{}
			if err := Run(r, w); err != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("Run() error = %v, wantErr %v", err, tt.wantErr)
				}
				return
			}
			if gotW := w.String(); gotW != tt.wantW {
				t.Errorf("Run() = %v, want %v", gotW, tt.wantW)
			}
		})
	}
}
