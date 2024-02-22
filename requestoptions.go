// Copyright (c) 2024 0x9ef. All rights reserved.
// Use of this source code is governed by an MIT license
// that can be found in the LICENSE file.
package clientx

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/gorilla/schema"
)

type RequestOption func(req *http.Request) error

// WithRequestParams encodes params automatically by accesing fields with custom tag.
func WithRequestParams[T any](tag string, params ...T) RequestOption {
	return func(req *http.Request) error {
		q := req.URL.Query()
		enc := schema.NewEncoder()
		enc.SetAliasTag(tag)

		for _, param := range params {
			if err := enc.Encode(param, q); err != nil {
				return fmt.Errorf("failed to encode query params: %w", err)
			}
		}
		req.URL.RawQuery = q.Encode()
		return nil
	}
}

// WithRequestEncodableParams encodes params by implementing ParamEncoder[T] interface,
// calls Encode(url.Values) functional to set query params.
func WithRequestEncodableParams[T any](params ...ParamEncoder[T]) RequestOption {
	return func(req *http.Request) error {
		q := req.URL.Query()
		for _, param := range params {
			if err := param.Encode(q); err != nil {
				return fmt.Errorf("failed to encode query params: %w", err)
			}
		}
		req.URL.RawQuery = q.Encode()
		return nil
	}
}

func WithRequestForm(form url.Values) RequestOption {
	return func(req *http.Request) error {
		req.Body = io.NopCloser(strings.NewReader(form.Encode()))
		return nil
	}
}

func WithRequestHeaders(headers map[string][]string) RequestOption {
	return func(req *http.Request) error {
		req.Header = headers
		return nil
	}
}
