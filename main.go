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
		Args:  os.Args[1:],
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

	out := bufw

	var cmd *exec.Cmd

	var f *os.File
	defer func() {
		if f != nil {
			os.Remove(f.Name())
		}
	}()

	// arguments to shell program. Need to put the file name before the
	// other arguments
	args := make([]string, len(c.Args)+1)
	copy(args[1:], c.Args)

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
			f, err = openFile()
			if err != nil {
				return err
			}

			out = bufio.NewWriter(f)

			searchItem = RightDelimiter
		} else {
			out.Flush()
			f.Close()

			args[0] = f.Name()
			cmd = exec.Command("sh", args...)

			// I think it inherits the parents stdin
			cmd.Stdin = os.Stdin
			cmd.Stdout = bufw
			cmd.Stderr = bufw

			if err := cmd.Run(); err != nil {
				return fmt.Errorf("running %v: %w", args, err)
			}

			out = bufw
			searchItem = LeftDelimiter
		}
	}
}

// global variables are useful
var tmpFileName string

func openFile() (*os.File, error) {
	if len(tmpFileName) == 0 {
		f, err := os.CreateTemp(".", "tmpFile*")
		if err != nil {
			return nil, err
		}

		tmpFileName = f.Name()
		return f, nil
	}

	// opens the for writing and ignore any existing contents
	return os.OpenFile(tmpFileName, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
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
