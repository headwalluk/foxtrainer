package wizard

import (
	"bufio"
	"errors"
	"io"
)

// ErrInputEnded means the input ran out before the wizard finished; nothing is saved.
var ErrInputEnded = errors.New("input ended before the questions were answered")

// lineReader hands out at most one line per Read and records when the input runs out.
//
// huh creates a fresh scanner for every form; without this, the first form's scanner would
// buffer answers meant for later forms, and an empty input would silently accept every default.
type lineReader struct {
	source *bufio.Reader
	ended  bool
}

// newLineReader wraps input.
func newLineReader(input io.Reader) *lineReader {
	return &lineReader{source: bufio.NewReader(input)}
}

// Read copies bytes up to and including the next newline.
func (reader *lineReader) Read(buffer []byte) (int, error) {
	count := 0

	var readError error

	for count < len(buffer) {
		character, byteError := reader.source.ReadByte()
		if byteError != nil {
			readError = byteError

			if errors.Is(byteError, io.EOF) {
				reader.ended = true
			}

			break
		}

		buffer[count] = character
		count++

		if character == '\n' {
			break
		}
	}

	if count > 0 && errors.Is(readError, io.EOF) {
		readError = nil
	}

	return count, readError
}
