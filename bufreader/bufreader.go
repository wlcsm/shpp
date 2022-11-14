package bufreader

import (
	"bytes"
	"errors"
	"fmt"
	"io"
)

var ErrDelimLargerThanBuffer = errors.New("delimiter cannot be larger than buffer")

// Single allocation reader.
// The idea is that we will keep reading and writing to out, until we encounter
// the delimiter.
// Not safe for concurrent use
type BufReader struct {
	In  io.Reader
	Out io.Writer

	// index of first non-read byte
	i int
	// index of first invalid byte
	end int

	buf []byte
}

func New(in io.Reader, out io.Writer, size int) *BufReader {
	return &BufReader{
		In:  in,
		Out: out,

		i:   0,
		end: 0,
		buf: make([]byte, size),
	}
}

func (b *BufReader) Empty() bool {
	return b.i == b.end
}

func (b *BufReader) Full() bool {
	return b.end == len(b.buf)
}

// Loads the buffer again, won't necessarily fill it completely
func (b *BufReader) Refill() (int, error) {
	if b.Full() {
		return 0, nil
	}

	// if we have just exhausted the buffer, then reset the 
	// internal pointers both to the beginning.
	if b.i == b.end {
		b.i, b.end = 0, 0
	}

	n, err := b.In.Read(b.buf[b.end:])
	if err != nil {
		return n, err
	}

	b.end += n

	return n, nil
}

// Writes the remaining contents of the buffer
func (b *BufReader) Flush(n int) (int, error) {
	availableBytes := b.end - b.i
	if n > availableBytes {
		return 0, fmt.Errorf("buffer contains %d bytes, cannot flush '%d' bytes", availableBytes, n)
	}

	valid_buf := b.buf[b.i : b.i+n]

	b.i += n
	if b.i == b.end {
		b.i, b.end = 0, 0
	}

	return b.Out.Write(valid_buf)
}

// Writes the remaining contents of the buffer
func (b *BufReader) FlushAll() (int, error) {
	return b.Flush(b.end - b.i)
}

// Finds the next instance of the delimiter by continuously reading and writing
// from the buffer
func (b *BufReader) Find(delim []byte) (err error) {
	if len(delim) > len(b.buf) {
		return ErrDelimLargerThanBuffer
	}

	for {
		if b.Empty() {
			_, err := b.Refill()
			if err != nil {
				return err
			}
		}

		i := bytes.Index(b.buf[b.i:b.end], delim)
		if i != -1 {
			// don't flush the prefix of the delimiter
			_, err := b.Flush(i)
			if err != nil {
				return err
			}

			b.i += len(delim)
			return nil
		}

		// Check if part of the delimiter is a contained at the end of
		// the buffer. In which case we need to read more of the buffer
		// to obtain the rest of it
		i = prefixAsSuffix(b.buf, delim)
		if i == -1 {
			_, err := b.FlushAll()
			if err != nil {
				return err
			}
			continue
		}

		// don't flush the prefix of the delimiter
		_, err := b.Flush(i - b.i)
		if err != nil {
			return err
		}

		// copy the part of the delimiter back to the beginning of the
		// buffer
		copy(b.buf, b.buf[b.i:b.end])
		b.i, b.end = 0, b.end-b.i

		_, err = b.Refill()
		if err != nil {
			return err
		}
	}
}

// Checks if a prefix of s is a suffix of buf. If so it returns the index where
// the largest such prefix starts, otherwise returns -1 if not found.
// So if buf = "abcde", s = "def",
// then it will return 3 since the prefix "de" of s is a suffix of buf.
//
// This is useful in stream processing to indicate that a string may exist in
// the stream, but it hasn't yet been read into the buffer yet
func prefixAsSuffix(buf []byte, s []byte) int {
	for i := len(s) - 1; i > 0; i-- {
		if bytes.HasSuffix(buf, s[:i]) {
			return len(buf) - i
		}
	}
	return -1
}
