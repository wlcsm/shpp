package bufreader

import (
	"bytes"
	"io"
	"strings"
	"testing"
)

func TestBufReader_Find(t *testing.T) {
	type fields struct {
		in   io.Reader
		out  *bytes.Buffer
		size int
	}
	type args struct {
		delim []byte
	}
	tests := []struct {
		name        string
		fields      fields
		args        args
		expected    error
		expectedOut string
	}{
		{
			name: "Read hello, world",
			fields: fields{
				in:   strings.NewReader("hello"),
				out:  bytes.NewBuffer(nil),
				size: 1,
			},
			args: args{
				delim: []byte("%{"),
			},
			expected:    ErrDelimLargerThanBuffer,
			expectedOut: "",
		},
		{
			name: "Read hello, world",
			fields: fields{
				in:   strings.NewReader("hello"),
				out:  bytes.NewBuffer(nil),
				size: 4096,
			},
			args: args{
				delim: []byte("%{"),
			},
			expected:    io.EOF,
			expectedOut: "hello",
		},
		{
			name: "Read test with a delimiter to see if it stops",
			fields: fields{
				in:   strings.NewReader("hello,%{world"),
				out:  bytes.NewBuffer(nil),
				size: 4096,
			},
			args: args{
				delim: []byte("%{"),
			},
			expected:    nil,
			expectedOut: "hello,",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := New(tt.fields.in, tt.fields.out, tt.fields.size)
			err := b.Find(tt.args.delim)
			if err != tt.expected {
				t.Errorf("BufReader.Find() expected return value='%v' got=%v", tt.expected, err)
			}

			res := tt.fields.out.String()
			if res != tt.expectedOut {
				t.Errorf("BufReader.Find() expected text output='%v' got=%v", tt.expectedOut, res)
			}
		})
	}
}

func TestBufReader_FindMultipleCalls(t *testing.T) {
	type fields struct {
		in   io.Reader
		out  *bytes.Buffer
		size int
	}
	type args struct {
		delim []byte
	}
	tests := []struct {
		name        string
		fields      fields
		args        args
		expectedOut string
	}{
		{
			name: "Read test with a delimiter to see if it reads the whole thing",
			fields: fields{
				in:   strings.NewReader("hello,%{world"),
				out:  bytes.NewBuffer(nil),
				size: 4096,
			},
			args: args{
				delim: []byte("%{"),
			},
			expectedOut: "hello,world",
		},
		{
			// "ab" is first loaded into the buffer, with the first
			// character of the delimiter "bc". The BufReader
			// should then acknowledge this by only writing a and
			// copying "b" to the front of the buffer and loading
			// more information later on
			name: "Delimiter is partially read into buffer, later processed in following round",
			fields: fields{
				in:   strings.NewReader("abcd"),
				out:  bytes.NewBuffer(nil),
				size: 2,
			},
			args: args{
				delim: []byte("bc"),
			},
			expectedOut: "ad",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := New(tt.fields.in, tt.fields.out, tt.fields.size)

			for {
				err := b.Find(tt.args.delim)
				if err == io.EOF {
					break
				}
				if err != nil {
					t.Errorf("BufReader.Find() err=%v", err)
				}
			}

			res := tt.fields.out.String()
			if res != tt.expectedOut {
				t.Errorf("BufReader.Find() expected text output='%v' got=%v", tt.expectedOut, res)

			}
		})
	}
}
