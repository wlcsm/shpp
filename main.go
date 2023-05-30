package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
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

const (
	defaultTempFile = "./shpp-cache"
	defaultShebang  = "#!/bin/sh"
)

// TODO what happens when as user presses Ctrl-C? It probably won't clean up the temporary file
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

	if err := Run(stdin, in, args, out, defaultTempFile, defaultShebang); err != nil {
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
func Run(stdin io.Reader, in ByteReader, args []string, w io.Writer, tmpFile, shebang string) error {
	// downside of using a string builder is that it clears its memory every time we reset it.
	f, err := os.OpenFile(tmpFile, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0700)
	if err != nil {
		return fmt.Errorf("creating cache: %w", err)
	}
	defer os.Remove(f.Name())
	defer f.Close()

	shebangWithNL := shebang + "\n"

	for {
		err := search(in, w, LeftDelimiter)
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}

		f.Truncate(0)
		f.WriteString(shebangWithNL)

		err = search(in, f, RightDelimiter)
		if err == io.EOF {
			return ErrUnclosedDelimiter
		}
		if err != nil {
			return err
		}

		cmd := exec.Command(f.Name(), args...)
		cmd.Stdin = stdin
		cmd.Stdout = w
		cmd.Stderr = w

		if err := cmd.Run(); err != nil {
			return fmt.Errorf("running 'sh %v': %w", args, err)
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
