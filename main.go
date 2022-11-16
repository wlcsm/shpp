package main

import (
	"bufio"
	"strings"
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
	if len(os.Args) == 1 || os.Args[1] == "-h" || os.Args[1] == "--help" {
		usage()
	}

	f, err := os.Open(os.Args[1])
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
	defer f.Close()

	cfg := Config{
		Stdin: os.Stdin,
		In:    f,
		Args:  os.Args[2:],
		Out:   os.Stdout,
	}

	if err := Run(cfg); err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
}

type Config struct {
	Stdin io.Reader
	In    io.Reader
	Args  []string

	Out io.Writer
}

// Analyses input from the reader to find areas encloses in '%{' '}%'
// delimiters. Any text outside these delimiters is directly written to the
// writer. Any text inside the delimiters is first passed to sh via STDIN,
// where the output is then written to the writer.
func Run(c Config) error {
	searchItem := LeftDelimiter

	bufr := bufio.NewReader(c.In)
	bufw := bufio.NewWriter(c.Out)

	var out io.Writer = bufw

	// the second argument is the script itself, to be fill in later
	args := append([]string{"-c", ""}, c.Args...)

	// downside of using a string builder is that it clears its memory every time we reset it.
	var s strings.Builder

	for {
		err := search(bufr, out, searchItem)
		if err == io.EOF {
			if bytes.Equal(searchItem, RightDelimiter) {
				return ErrUnclosedDelimiter
			}

			bufw.Flush()
			return nil
		}
		if err != nil {
			return err
		}

		if bytes.Equal(searchItem, LeftDelimiter) {
			out = &s

			searchItem = RightDelimiter
		} else {
			args[1] = s.String()
			s.Reset()

			cmd := exec.Command("sh", args...)

			cmd.Stdin = os.Stdin
			cmd.Stdout = bufw
			cmd.Stderr = bufw

			if err := cmd.Run(); err != nil {
				return fmt.Errorf("running sh %v: %w", args, err)
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
//
// If we could avoid writing individual bytes then this could potentially be faster
func search(in *bufio.Reader, out io.Writer, delim []byte) error {
	i := 0
	buf := []byte{0}

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

			buf[0] = c
			out.Write(buf)
		}
	}
}
