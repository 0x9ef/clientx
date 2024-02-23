// Copyright (c) 2024 0x9ef. All rights reserved.
// Use of this source code is governed by an MIT license
// that can be found in the LICENSE file.
package clientx

import (
	"bytes"
	"compress/flate"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
)

// Empty is an empty payload for request/response decoding.
type Empty struct{}

func responseReader(resp *http.Response) (io.ReadCloser, error) {
	var err error
	var reader io.ReadCloser
	switch resp.Header.Get("Content-Encoding") {
	case "deflate":
		reader = io.NopCloser(flate.NewReader(resp.Body))
	case "gzip":
		reader, err = gzip.NewReader(resp.Body)
		if err != nil {
			return nil, err
		}
		reader = io.NopCloser(reader)
	default:
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}
		reader, resp.Body = io.NopCloser(bytes.NewBuffer(body)), io.NopCloser(bytes.NewBuffer(body))
	}
	return reader, err
}

func decodeResponse[T any](enc EncoderDecoder, resp *http.Response, dst T) error {
	reader, err := responseReader(resp)
	if err != nil {
		return fmt.Errorf("decode response failed: %w", err)
	}
	defer reader.Close()

	err = enc.Decode(reader, dst)
	return err
}
