// Copyright (c) 2024 0x9ef. All rights reserved.
// Use of this source code is governed by an MIT license
// that can be found in the LICENSE file.
package clientx

import (
	"bytes"
	"compress/flate"
	"compress/gzip"
	"encoding/json"
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
	case "gzip":
		reader, err = gzip.NewReader(resp.Body)
	case "deflate":
		reader = flate.NewReader(resp.Body)
	default:
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}
		reader, resp.Body = io.NopCloser(bytes.NewBuffer(body)), io.NopCloser(bytes.NewBuffer(body))
	}
	return reader, err
}

func decodeResponse[T any](resp *http.Response, v T) error {
	reader, err := responseReader(resp)
	if err != nil {
		return fmt.Errorf("decode response failed: %w", err)
	}
	defer reader.Close()

	err = json.NewDecoder(reader).Decode(v)
	return err
}
