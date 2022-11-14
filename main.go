package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"

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

var (
	TempFile = filepath.Join(os.TempDir(), "shpp.tmp")
)

// Okay theres an annoying bug where if at the end of the buffer we get '%' the
// start of the escape sequence, then we must write it to the end. What I
// propose is that we only read up to the second last character unless it is %
// or }, in which case we check the last one. Otherwise we copy the last one to
// the beginning and read from there
func Run(r io.Reader, w io.Writer) error {
	bufr := bufreader.New(r, w, 4096)
	searchItem := LeftDelimiter

	var f *os.File

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
			f.Close()

			// probably not the best way to do this
			os.Args[0] = TempFile
			cmd := exec.Command("sh", os.Args...)
			cmd.Stdout = w
			cmd.Stderr = w

			// ignore errors, they will appear in the text
			_ = cmd.Run()

			bufr.SetOutput(w)
		} else {
			searchItem = RightDelimiter

			f, err = os.OpenFile(TempFile, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0755)
			if err != nil {
				return err
			}

			bufr.SetOutput(f)
		}
	}
}
