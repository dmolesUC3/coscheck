package streaming

import (
	"errors"
	"fmt"
	"io"
	"net/url"

	"code.cloudfoundry.org/bytefmt"
)

const DefaultRangeSize = int64(5 * bytefmt.MEGABYTE)

func NextRange(currentTotal int64, maxRangeSize int64, contentLength int64) (start, end int64, size int) {
	start = currentTotal
	end = currentTotal + maxRangeSize
	if end > contentLength {
		end = contentLength - 1
	}
	size = int((end + 1) - currentTotal)
	return start, end, size
}

// ReadExactly reads exactly the number of bytes to fill the specified buffer,
// otherwise returning an error.
func ReadExactly(in io.Reader, buffer []byte) (err error) {
	bytesRead, err := io.ReadFull(in, buffer)
	if err == nil {
		expected := len(buffer)
		if bytesRead != expected {
			err = fmt.Errorf("expected to read %d bytes, got %d", expected, bytesRead)
		}
	}
	return
}

// WriteExactly writes exactly the number of bytes found in the specified buffer,
// otherwise returning an error.
func WriteExactly(out io.Writer, data []byte) (err error) {
	bytesWritten, err := out.Write(data)
	if err == nil {
		expected := len(data)
		if bytesWritten != expected {
			err = fmt.Errorf("expected to write %d bytes, got %d", expected, bytesWritten)
		}
	}
	return
}

// ValidAbsURL parses the specified URL string, returning an error if the
// URL cannot be parsed, or is not absolute (i.e., does not have a scheme)
func ValidAbsURL(urlStr string) (*url.URL, error) { // TODO: add an error hint
	u, err := url.Parse(urlStr)
	if err != nil {
		return u, err
	}
	if !u.IsAbs() {
		msg := fmt.Sprintf("URL '%v' must have a scheme", urlStr)
		return nil, errors.New(msg)
	}
	return u, nil
}

