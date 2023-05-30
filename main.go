package main

import (
	"bufio"
	"errors"
	"flag"
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

var (
	help    = flag.Bool("h", false, "help message")
	tmpFile = flag.String("t", "./shpp-cache", "temporary file for storing scripts for execution")
	shebang = flag.String("p", "/bin/sh", "program used to run scripts")
)

func usage() {
	fmt.Fprintf(flag.CommandLine.Output(), "usage %s [flags] [file]:\n", os.Args[0])
	flag.PrintDefaults()
	flag.CommandLine.Output().Write([]byte(`
Funnels all text inside '%{' '}%' delimiters into a file, executes it and
writes the stdout and stderr back into the original text.

	$ cat index.template
	<ul>
	%{
	while read line ; do
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

Use the -p flag to change the program used to execute the script blocks.

	$ cat python_test.md
	This is %{print('python syntax')}%
	$ ./shpp -p /usr/bin/python3 < python_test.md
	This is python syntax
`))
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
	flag.Parse()
	args := flag.Args()

	if *help || len(args) > 1 {
		usage()
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

		// If it doesn't has data, then exec.Command will hang waiting
		// for it. So just keep it nil if nothing is there.
		if stdinHasData() {
			stdin = in
		}
		in = bufio.NewReader(f)
	}

	out := bufio.NewWriter(os.Stdout)
	defer out.Flush()

	defer os.Remove(*tmpFile)

	return Process(stdin, in, args, out, *tmpFile, *shebang)
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
		return fmt.Errorf("creating cache: %w", err)
	}
	defer f.Close()

	shebang := "#!" + program + "\n"

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
		f.WriteString(shebang)

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
