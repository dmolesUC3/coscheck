package objects

import (
	"crypto/md5"
	"crypto/sha256"
	"errors"
	"fmt"
	"hash"
	"io"
	"log"
	"net/url"
	"time"

	"github.com/dmolesUC3/cos/internal/logging"
	"github.com/dmolesUC3/cos/internal/streaming"
)

// The Object type represents the location of an object in cloud storage.
type Object interface {
	Protocol() string
	Endpoint() *url.URL
	Bucket() *string
	Key() *string
	ContentLength() (int64, error)
	ReadRange(startInclusive, endInclusive int64, buffer []byte) (int64, error)
	StreamUp(body io.Reader, length int64) (err error)
	Delete() (err error)
	Logger() logging.Logger
	Reset()
}

func ProtocolUriStr(obj Object) string {
	return fmt.Sprintf("%v://%v/%v", obj.Protocol(), logging.PrettyStrP(obj.Bucket()), logging.PrettyStrP(obj.Key()))
}

func Download(obj Object, rangeSize int64, out io.Writer) (int64, error) {
	// this will 404 if the object doesn't exist
	contentLength, err := obj.ContentLength()
	if err != nil {
		return 0, err
	}
	logger := obj.Logger()
	progress := logging.ReportProgress(contentLength, logger, time.Second)
	defer close(progress)

	var totalRead int64
	for ; totalRead < contentLength; {
		start, end, size := streaming.NextRange(totalRead, rangeSize, contentLength)
		buffer := make([]byte, size)
		bytesRead, err := obj.ReadRange(start, end, buffer)
		if err != nil {
			return totalRead, err
		}
		err = streaming.WriteExactly(out, buffer)
		if err != nil {
			return totalRead, err
		}
		totalRead += bytesRead
		progress <- totalRead
	}
	return totalRead, nil
}

// CalcDigest calculates the digest of the object using the specified algorithm
// (md5 or sha256), using ranged downloads of the specified size.
func CalcDigest(obj Object, downloadRangeSize int64, algorithm string) ([] byte, error) {
	h := newHash(algorithm)
	_, err := Download(obj, downloadRangeSize, h)
	if err != nil {
		return nil, err
	}
	digest := h.Sum(nil)
	return digest, nil
}

// ValidAbsURL parses the specified URL string, returning an error if the
// URL cannot be parsed, or is not absolute (i.e., does not have a scheme)
func ValidAbsURL(urlStr string) (*url.URL, error) {
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

// newHash returns a new hash of the specified algorithm ("sha256" or "md5")
func newHash(algorithm string) hash.Hash {
	if algorithm == "sha256" {
		return sha256.New()
	} else if algorithm == "md5" {
		return md5.New()
	}
	log.Fatalf("unsupported digest algorithm: '%v'\n", algorithm)
	return nil
}
