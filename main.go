package main

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
)

var (
	LeftDelimiter  = []byte("%{")
	RightDelimiter = []byte("}%")

	ErrUnclosedDelimiter = errors.New("unclosed delimiter: contains '%{' without a matching '}%'")
)

func usage() {
	fmt.Println(`usage: ` + os.Args[0] + ` [-h|--help]

Reads input through stdin and pipes anything inside the '%{' '}%' delimiters
into sh and substitutes it back into the text.

Arguments may be passed through environment variables`)
	os.Exit(1)
}

func main() {
	if len(os.Args) == 1 || os.Args[1] == "-h" || os.Args[1] == "--help" {
		usage()
	}

	f, err := os.Open(os.Args[1])
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
	defer f.Close()

	stdin := bufio.NewReader(os.Stdin)
	in := bufio.NewReader(f)
	args := os.Args[2:]
	out := bufio.NewWriter(os.Stdout)

	if err := Run(stdin, in, args, out); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	out.Flush()
}

type ByteReader interface {
	ReadByte() (byte, error)
}

// Analyses input from the reader to find areas encloses in '%{' '}%'
// delimiters. Any text outside these delimiters is directly written to the
// writer. Any text inside the delimiters is first passed to sh via STDIN,
// where the output is then written to the writer.
func Run(stdin io.Reader, in ByteReader, args []string, w io.Writer) error {
	searchItem := LeftDelimiter

	// This is the writer that the main loop will write to. When inside a
	// script block, it will point towards a string buffer, and otherwise,
	// to the output writer.
	out := w

	// the second argument is the script itself, to be fill in later
	allArgs := append([]string{"-c", ""}, args...)

	// downside of using a string builder is that it clears its memory every time we reset it.
	var s strings.Builder

	for {
		err := search(in, out, searchItem)
		if err == io.EOF {
			if bytes.Equal(searchItem, RightDelimiter) {
				return ErrUnclosedDelimiter
			}

			return nil
		}
		if err != nil {
			return err
		}

		if bytes.Equal(searchItem, LeftDelimiter) {
			out = &s

			searchItem = RightDelimiter
		} else {
			allArgs[1] = s.String()
			s.Reset()

			cmd := exec.Command("sh", allArgs...)
			cmd.Stdin = stdin
			cmd.Stdout = w
			cmd.Stderr = w

			if err := cmd.Run(); err != nil {
				return fmt.Errorf("running 'sh %v': %w", allArgs, err)
			}

			out = w
			searchItem = LeftDelimiter
		}
	}
}

// Finds the next instance of the delimiter by continuously reading and writing
// from the buffer.
//
// Note that after successfully finding the delimiter, it will skip it and
// *not* write it later.
//
// If we could avoid writing individual bytes then this could potentially be faster
func search(in ByteReader, out io.Writer, delim []byte) error {
	i := 0
	buf := []byte{0}

	for {
		// There is a possibility that the delim could exist as part of
		// a unicode glyph. I'm ignoring that case for now as it and
		// I'm not sure if the current delims have this problem.
		c, err := in.ReadByte()
		if err != nil {
			if i != 0 {
				out.Write(delim[:i])
			}
			return err
		}

		if c == delim[i] {
			if i == len(delim)-1 {
				return nil
			}

			i++
		} else {
			if i != 0 {
				out.Write(delim[:i])
				i = 0
			}

			buf[0] = c
			out.Write(buf)
		}
	}
}
