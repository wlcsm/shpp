package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
)

var (
	LeftDelimiter  = []byte("%{")
	RightDelimiter = []byte("}%")

	ErrUnclosedDelimiter = errors.New("unclosed delimiter: contains '%{' without a matching '}%'")
)

const (
	defaultTmpFile = "./shpp-cache"
	defaultProgram = "/bin/sh"
)

func usage() {
	os.Stdout.WriteString(`usage: shpp [file]

Funnels all text inside '%{' '}%' delimiters into a file, executes it and
writes the stdout and stderr back into the original text.

Environment variables:

	SHPP_PROGRAM  The shebang used to execute the code blocks (default: %s)

	SHPP_TMPFILE  The temporary file used to execute the code blocks (default: %s)

Examples:

	$ cat index.template
	<ul>
	%{
	while read line; do
	   echo '<li>'$line'</li>'
	done
	}%
	</ul>

	$ seq 5 | ./shpp index.template
	<ul>
	<li>1</li>
	<li>2</li>
	<li>3</li>
	<li>4</li>
	<li>5</li>

	</ul>
`)
	os.Exit(1)
}

// TODO what happens when as user presses Ctrl-C? It probably won't clean up the temporary file
func main() {
	if err := run(); err != nil {
		fmt.Fprint(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	args := os.Args[1:]
	if len(args) > 0 && (args[0] == "-h" || args[0] == "--help") {
		usage()
	}

	shebang := defaultProgram
	if s := os.Getenv("SHPP_PROGRAM"); len(s) != 0 {
		shebang = s
	}

	tmpFile := defaultTmpFile
	if t := os.Getenv("SHPP_TMPFILE"); len(t) != 0 {
		tmpFile = t
	}

	// If the a file is given, then read from it and pass stdin.
	// Otherwise just read from stdin.
	var stdin io.Reader
	in := bufio.NewReader(os.Stdin)

	if len(args) == 1 {
		f, err := os.Open(args[0])
		if err != nil {
			return err
		}
		defer f.Close()

		// If it doesn't have data, then exec.Command will hang
		if stdinHasData() {
			stdin = in
		}
		in = bufio.NewReader(f)
	}

	out := bufio.NewWriter(os.Stdout)
	defer out.Flush()

	defer os.Remove(tmpFile)

	return Process(stdin, in, args, out, tmpFile, shebang)
}

func stdinHasData() bool {
	stat, _ := os.Stdin.Stat()
	return (stat.Mode() & os.ModeCharDevice) == 0
}

type ByteReader interface {
	ReadByte() (byte, error)
}

// Analyses input from the reader to find areas encloses in '%{' '}%'
// delimiters. Any text outside these delimiters is directly written to the
// writer. Any text inside the delimiters is written to a file, and executed
// with the given program before being written to the writer.
func Process(stdin io.Reader, in ByteReader, args []string, w io.Writer, tmpFile, program string) error {
	f, err := os.OpenFile(tmpFile, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0700)
	if err != nil {
		return fmt.Errorf("opening temporary file '%s': %w", tmpFile, err)
	}
	defer f.Close()

	shebang := "#!" + program

	// if there was a problem os.OpenFile would have caught it
	workingFile, _ := filepath.Abs(tmpFile)

	for {
		err := search(in, w, LeftDelimiter)
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}

		f.Truncate(0)
		f.Seek(0, 0)
		f.WriteString(shebang + "\n")

		err = search(in, f, RightDelimiter)
		if err == io.EOF {
			return ErrUnclosedDelimiter
		}
		if err != nil {
			return err
		}

		cmd := exec.Command(workingFile, args...)
		cmd.Stdin = stdin
		cmd.Stdout = w
		cmd.Stderr = w

		if err := cmd.Run(); err != nil {
			return fmt.Errorf("running '%s %v': %w", f.Name(), args, err)
		}
	}
}

// Finds the next instance of the delimiter by continuously reading and writing
// from the buffer. It will *not* write the delimiter.
func search(in ByteReader, out io.Writer, delim []byte) error {
	i := 0
	buf := []byte{0}

	for {
		// There is a possibility that the delim could exist as part of
		// a unicode glyph. I'm ignoring that case for now.
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
