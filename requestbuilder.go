// Copyright (c) 2024 0x9ef. All rights reserved.
// Use of this source code is governed by an MIT license
// that can be found in the LICENSE file.
package clientx

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"net/url"
)

type ParamEncoder[T any] interface {
	Encode(v url.Values) error
}

func NormalizeParams[T ParamEncoder[T]](params []T) []ParamEncoder[T] {
	p := make([]ParamEncoder[T], len(params))
	for i, param := range params {
		p[i] = ParamEncoder[T](param)
	}
	return p
}

type RequestBuilder[Req any, Resp any] struct {
	client         *client[Req, Resp]
	method         string
	resourcePath   string
	requestOptions []RequestOption
	body           *Req
	errDecodeFn    func(*http.Response) (bool, error)
}

func (rb *RequestBuilder[Req, Resp]) encodeRequestPayload(enc EncoderDecoder) (io.ReadCloser, error) {
	payload := &bytes.Buffer{}
	if err := enc.Encode(payload, rb.body); err != nil {
		return nil, err
	}
	return io.NopCloser(payload), nil
}

// NewRequestBuilder creates a new request builder from API for designated Req, Resp.
// Default method is GET. If you want to specify method, you should call corresponding Get/Post/Put/Patch/Delete methods.
func NewRequestBuilder[Req any, Resp any](api *API) *RequestBuilder[Req, Resp] {
	return &RequestBuilder[Req, Resp]{
		client: &client[Req, Resp]{api: api},
		method: http.MethodGet,
	}
}

// AfterResponse adds to a chain function that will be executed after response is obtained.
// Note! The second argument (decoded) in f function is only available when using DoWithDecode method to perform request.
func (rb *RequestBuilder[Req, Resp]) AfterResponse(f func(resp *http.Response, body []byte) error) *RequestBuilder[Req, Resp] {
	rb.client.afterResponse = append(rb.client.afterResponse, f)
	return rb
}

// WithForm sets the form data for the request.
func (rb *RequestBuilder[Req, Resp]) WithForm(obj url.Values) *RequestBuilder[Req, Resp] {
	rb.requestOptions = append(rb.requestOptions, WithRequestForm(obj))
	return rb
}

// WithStructQueryParams sets URL query parameters from structure by accesing field with provided tag alias.
func (rb *RequestBuilder[Req, Resp]) WithStructQueryParams(tag string, params ...Req) *RequestBuilder[Req, Resp] {
	rb.requestOptions = append(rb.requestOptions, WithRequestQueryParams(tag, params...))
	return rb
}

// WithEncodableQueryParams sets URL query parameters from structure which implements ParamEncoder interface.
func (rb *RequestBuilder[Req, Resp]) WithEncodableQueryParams(params ...ParamEncoder[Req]) *RequestBuilder[Req, Resp] {
	rb.requestOptions = append(rb.requestOptions, WithRequestQueryEncodableParams(params...))
	return rb
}

// WithErrorDecode sets custom error decoding function. Will be executed immediately after request is performed.
func (rb *RequestBuilder[Req, Resp]) WithErrorDecode(f func(resp *http.Response) (bool, error)) *RequestBuilder[Req, Resp] {
	rb.errDecodeFn = f
	return rb
}

// Get builds GET request with no body specified.
// Appends request options (includes request options that were specified at NewRequestBuilder).
func (rb *RequestBuilder[Req, Resp]) Get(path string, opts ...RequestOption) *RequestBuilder[Req, Resp] {
	rb.method = http.MethodGet
	rb.resourcePath = path
	rb.requestOptions = append(rb.requestOptions, opts...)
	return rb
}

// Post builds POST request with specified body. Appends request options.
func (rb *RequestBuilder[Req, Resp]) Post(path string, body *Req, opts ...RequestOption) *RequestBuilder[Req, Resp] {
	rb.method = http.MethodPost
	rb.resourcePath = path
	rb.body = body
	rb.requestOptions = append(rb.requestOptions, opts...)
	return rb
}

// Patch builds PATCH request with specified body (if any). Appends request options.
func (rb *RequestBuilder[Req, Resp]) Patch(path string, body *Req, opts ...RequestOption) *RequestBuilder[Req, Resp] {
	rb.method = http.MethodPatch
	rb.resourcePath = path
	rb.requestOptions = append(rb.requestOptions, opts...)
	rb.body = body
	return rb
}

// Put builds PUT request with specified body (if any). Appends request options.
func (rb *RequestBuilder[Req, Resp]) Put(path string, body *Req, opts ...RequestOption) *RequestBuilder[Req, Resp] {
	rb.method = http.MethodPut
	rb.resourcePath = path
	rb.requestOptions = append(rb.requestOptions, opts...)
	rb.body = body
	return rb
}

// Delete builds DELETE request with specified body (if any). Appends request options.
func (rb *RequestBuilder[Req, Resp]) Delete(path string, body *Req, opts ...RequestOption) *RequestBuilder[Req, Resp] {
	rb.method = http.MethodDelete
	rb.resourcePath = path
	rb.requestOptions = append(rb.requestOptions, opts...)
	rb.body = body
	return rb
}

// Head builds HEAD request. Appends request options.
func (rb *RequestBuilder[Req, Resp]) Head(path string, opts ...RequestOption) *RequestBuilder[Req, Resp] {
	rb.method = http.MethodHead
	rb.resourcePath = path
	rb.requestOptions = append(rb.requestOptions, opts...)
	return rb
}

// Trace builds TRACE request. Appends request options.
func (rb *RequestBuilder[Req, Resp]) Trace(path string, opts ...RequestOption) *RequestBuilder[Req, Resp] {
	rb.method = http.MethodTrace
	rb.resourcePath = path
	rb.requestOptions = append(rb.requestOptions, opts...)
	return rb
}

// Options builds OPTIONS request. Appends request options.
func (rb *RequestBuilder[Req, Resp]) Options(path string, opts ...RequestOption) *RequestBuilder[Req, Resp] {
	rb.method = http.MethodOptions
	rb.resourcePath = path
	rb.requestOptions = append(rb.requestOptions, opts...)
	return rb
}

// Connect builds CONNECT request. Appends request options.
func (rb *RequestBuilder[Req, Resp]) Connect(path string, opts ...RequestOption) *RequestBuilder[Req, Resp] {
	rb.method = http.MethodConnect
	rb.resourcePath = path
	rb.requestOptions = append(rb.requestOptions, opts...)
	return rb
}

// Do executes request and returns *http.Response. Returns error if any.
func (rb *RequestBuilder[Req, Resp]) Do(ctx context.Context) (*http.Response, error) {
	resp, _, err := rb.client.do(ctx, rb, false, JSONEncoderDecoder)
	return resp, err
}

// DoWithDecode executes request and decodes response into Resp object. Returns error if any.
func (rb *RequestBuilder[Req, Resp]) DoWithDecode(ctx context.Context, enc ...EncoderDecoder) (*Resp, error) {
	if len(enc) == 0 {
		enc = append(enc, JSONEncoderDecoder) // JSON by default
	} else if len(enc) > 1 {
		return nil, errors.New("enc length should be 0 or 1")
	}
	_, decoded, err := rb.client.do(ctx, rb, true, enc[0])
	return decoded, err
}
