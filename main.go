package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"

	"codeberg.org/wlcsm/shpp/pipebuf"
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
	bufr := pipebuf.New(r, w, 4096)
	searchItem := LeftDelimiter

	var stdinPipe io.WriteCloser
	var cmd *exec.Cmd

	for {
		err := bufr.ProcessUntil(searchItem)
		if err == io.EOF {
			if bytes.Equal(searchItem, RightDelimiter) {
				return ErrUnclosedDelimiter
			}
			return nil
		}
		if err != nil {
			return err
		}

		if bytes.Equal(searchItem, RightDelimiter) {
			searchItem = LeftDelimiter
			stdinPipe.Close()

			if err := cmd.Wait(); err != nil {
				return err
			}

			bufr.Out = w
		} else {
			cmd = exec.Command("sh")
			cmd.Stdout = w
			cmd.Stderr = w

			stdinPipe, err = cmd.StdinPipe()
			if err != nil {
				return err
			}
			bufr.Out = stdinPipe

			if err = cmd.Start(); err != nil {
				return err
			}

			searchItem = RightDelimiter
		}
	}
}
