package main

import (
	"bufio"
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

const defaultCommand = "/bin/sh"

func usage() {
	os.Stdout.WriteString(`usage: shpp [file] [args ... ]

Executes all text inside '%{' '}%' delimiters with the following command and
inserts the stdout and stderr back into the original text.

	<prog> -c <text> [<args> ... ]


The <prog> is /bin/sh by default and can be changed with the SHPP_COMMAND
environment variable.

	$ cat template.html
	<p>%{echo "Hello, world}%</p>

	$ shpp template.html
	<p>Hello, world</p>

shpp will read from stdin if no arguments are supplied. Otherwise it will read
from the input file provided as the first argument.

	$ shpp < template.html
	<p>Hello, world</p>

When the code inside the delimiters is executed, it will have access to
STDIN, environment variables, and positional arguments of the parent process.

Passing via stdin

	$ cat template.html
	<p>%{ cat }%</p>

	$ echo 'Hello World' | shpp template.yaml
	<p>Hello, world</p>

Passing via environment variables

	$ echo '%{echo $MSG}%' | MSG='Hello, world' ./shpp
	Hello, world

Passing via positional arguments

	$ cat template.html
	<p>%{ echo $1, $2 }%</p>

	$ echo 'Hello World' | shpp template.yaml 'Hello' 'world'
	<p>Hello, world</p>

The "-" character in the place of the template file name signifies the template
should be read from STDIN. This is useful when needed to provide positional
arguments.

	$ echo '%{printf $1 $2}%' | shpp - 'Hello,' 'world'
	Hello, world

Environment variables:

	SHPP_COMMAND  The command used to execute the codeblock (default: /bin/sh)
`)
	os.Exit(1)
}

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

	command := defaultCommand
	if s := os.Getenv("SHPP_COMMAND"); len(s) != 0 {
		command = s
	}

	// If the a file is given, then read from it and pass stdin.
	// Otherwise just read from stdin.
	var stdin io.Reader
	in := bufio.NewReader(os.Stdin)

	if len(args) > 0 {
		if args[0] != "-" {
			f, err := os.Open(args[0])
			if err != nil {
				return err
			}
			defer f.Close()

			// If stdin doesn't have data, exec.Command will hang
			if stdinHasData() {
				stdin = in
			}
			in = bufio.NewReader(f)
		}

		// Skip the input file, args is now just the list of arguments
		// that we will be provided to the code blocks.
		args = args[1:]
	}

	out := bufio.NewWriter(os.Stdout)
	defer out.Flush()

	execArgs := append([]string{"-c", ""}, args...)

	// Executes the codeblock by running the command
	//    <cmd> -c <codeblock> [ <args> ... ]
	exe := func(arg string) error {
		execArgs[1] = arg

		cmd := exec.Command(command, execArgs...)
		cmd.Stdin = stdin
		cmd.Stdout = out
		cmd.Stderr = out

		return cmd.Run()
	}

	return ExecCodeBlocks(in, out, exe)
}

func stdinHasData() bool {
	stat, _ := os.Stdin.Stat()
	return (stat.Mode() & os.ModeCharDevice) == 0
}

type byteReader interface {
	ReadByte() (byte, error)
}

// Executes codeblocks in the input and writes everything else.
func ExecCodeBlocks(in byteReader, w io.Writer, exe func(string) error) error {
	var buf strings.Builder

	for {
		err := search(in, w, LeftDelimiter)
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}

		err = search(in, &buf, RightDelimiter)
		if err == io.EOF {
			return ErrUnclosedDelimiter
		}
		if err != nil {
			return err
		}

		if err := exe(buf.String()); err != nil {
			return fmt.Errorf("executing code block: %w", err)
		}

		buf.Reset()
	}
}

// Finds the next instance of the delimiter by continuously reading and writing
// from the buffer. It will *not* write the delimiter.
func search(in byteReader, out io.Writer, delim []byte) error {
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
