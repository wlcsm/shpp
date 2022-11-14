package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"

	"codeberg.org/wlcsm/shpp/bufreader"
)

func usage() {
	fmt.Printf(`usage: %s

Reads input through stdin. Args may be passed through environment variables\n`, os.Args[0])
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

var (
	LeftDelimiter  = []byte("%{")
	RightDelimiter = []byte("}%")

	ErrUnclosedDelimiter = errors.New("unclosed delimiter: contains '%{' without a matching '}%'")
)

// Okay theres an annoying bug where if at the end of the buffer we get '%' the
// start of the escape sequence, then we must write it to the end. What I
// propose is that we only read up to the second last character unless it is %
// or }, in which case we check the last one. Otherwise we copy the last one to
// the beginning and read from there
func Run(r io.Reader, w io.Writer) error {
	bufr := bufreader.New(r, w, 4096)
	searchItem := LeftDelimiter

	var stdinPipe io.WriteCloser
	var cmd *exec.Cmd

	for {
		err := bufr.Find(searchItem)
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
