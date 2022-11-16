package main

import (
	"bufio"
	"bytes"
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

func main() {
	if len(os.Args) > 1 && (os.Args[1] == "-h" || os.Args[1] == "--help") {
		usage()
	}

	if err := Run(os.Stdin, os.Stdout); err != nil {
		fmt.Printf(err.Error())
		os.Exit(1)
	}
}

// Analyses input from the reader to find areas encloses in '%{' '}%'
// delimiters. Any text outside these delimiters is directly written to the
// writer. Any text inside the delimiters is first passed to sh via STDIN,
// where the output is then written to the writer.
func Run(r io.Reader, w io.Writer) error {
	searchItem := LeftDelimiter

	bufr := bufio.NewReader(r)
	bufw := bufio.NewWriter(w)
	defer bufw.Flush()

	out := bufw

	var stdinPipe io.WriteCloser
	var cmd *exec.Cmd

	for {
		err := search(bufr, out, searchItem)
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
			cmd = exec.Command("sh")
			cmd.Stdout = bufw
			cmd.Stderr = bufw

			stdinPipe, err = cmd.StdinPipe()
			if err != nil {
				return err
			}
			out = bufio.NewWriter(stdinPipe)

			if err = cmd.Start(); err != nil {
				return err
			}

			searchItem = RightDelimiter
		} else {
			out.Flush()
			stdinPipe.Close()

			if err := cmd.Wait(); err != nil {
				return err
			}

			out = bufw
			searchItem = LeftDelimiter
		}
	}
}

// Finds the next instance of the delimiter by continuously reading and writing
// from the buffer.
//
// Note that after successfully finding the delimiter, it will skip it and
// *not* write it later.
func search(in *bufio.Reader, out *bufio.Writer, delim []byte) error {
	i := 0

	for {
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

			out.WriteByte(c)
		}
	}
}
