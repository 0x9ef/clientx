// Copyright (c) 2024 0x9ef. All rights reserved.
// Use of this source code is governed by an MIT license
// that can be found in the LICENSE file.
package clientx

import (
	"bytes"
	"compress/flate"
	"compress/gzip"
	"io"
	"net/http"

	"github.com/klauspost/compress/zstd"
	"github.com/ulikunitz/xz"
)

// Empty is an empty payload for request/response decoding.
type Empty struct{}

func responseReader(resp *http.Response) (io.ReadCloser, []byte, error) {
	// Duplicate response body to two readers,
	// the r1 we use to replace resp.Body, and r2 to build flate/gzip readers
	r1, r2, b, err := drainBody(resp.Body)
	if err != nil {
		return nil, nil, err
	}

	var reader io.ReadCloser
	switch resp.Header.Get("Content-Encoding") {
	case "deflate":
		reader = flate.NewReader(r2)
	case "gzip":
		reader, err = gzip.NewReader(r2)
	case "zstd":
		zstdr, err := zstd.NewReader(r2)
		if err != nil {
			return nil, nil, err
		}
		reader = zstdr.IOReadCloser()
	case "xz":
		xzr, err := xz.NewReader(r2)
		if err != nil {
			return nil, nil, err
		}
		reader = io.NopCloser(xzr)
	default:
		reader = r2
	}
	resp.Body = r1

	return reader, b, err
}

func drainBody(r io.ReadCloser) (r1, r2 io.ReadCloser, b []byte, err error) {
	if r == nil || r == http.NoBody {
		// No copying needed. Preserve the magic sentinel meaning of NoBody.
		return http.NoBody, http.NoBody, nil, nil
	}
	var buf bytes.Buffer
	if _, err = buf.ReadFrom(r); err != nil {
		return nil, r, nil, err
	}
	if err = r.Close(); err != nil {
		return nil, r, nil, err
	}
	return io.NopCloser(&buf), io.NopCloser(bytes.NewReader(buf.Bytes())), buf.Bytes(), nil
}

func decodeResponse[T any](enc EncoderDecoder, r io.ReadCloser, dst T) error {
	return enc.Decode(r, dst)
}

type reusableReader struct {
	io.Reader
	readBuf *bytes.Buffer
	backBuf *bytes.Buffer
}

// https://blog.flexicondev.com/read-go-http-request-body-multiple-times
func ReusableReader(r io.Reader) io.ReadCloser {
	readBuf := bytes.Buffer{}
	readBuf.ReadFrom(r) // error handling ignored for brevity
	backBuf := bytes.Buffer{}

	return reusableReader{
		io.TeeReader(&readBuf, &backBuf),
		&readBuf,
		&backBuf,
	}
}

func (r reusableReader) Read(p []byte) (int, error) {
	n, err := r.Reader.Read(p)
	if err == io.EOF {
		r.reset()
	}
	return n, err
}

func (r reusableReader) Close() error {
	return nil
}

func (r reusableReader) reset() {
	io.Copy(r.readBuf, r.backBuf) // nolint: errcheck
}
