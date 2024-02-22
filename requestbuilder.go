// Copyright (c) 2024 0x9ef. All rights reserved.
// Use of this source code is governed by an MIT license
// that can be found in the LICENSE file.
package clientx

import (
	"bytes"
	"context"
	"encoding/json"
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
}

func (rb *RequestBuilder[Req, Resp]) encodeRequestPayload() (io.ReadCloser, error) {
	payload := &bytes.Buffer{}
	if err := json.NewEncoder(payload).Encode(rb.body); err != nil {
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

// WithForm sets the form data for the request.
func (r *RequestBuilder[Req, Resp]) WithForm(obj url.Values) *RequestBuilder[Req, Resp] {
	r.requestOptions = append(r.requestOptions, WithRequestForm(obj))
	return r
}

// WithQueryParams sets URL query parameters from structure by accesing field with provided tag alias.
func (r *RequestBuilder[Req, Resp]) WithQueryParams(tag string, params ...Req) *RequestBuilder[Req, Resp] {
	r.requestOptions = append(r.requestOptions, WithRequestParams(tag, params...))
	return r
}

// WithEncodableQueryParams sets URL query parameters from structure which implements ParamEncoder interface.
func (r *RequestBuilder[Req, Resp]) WithEncodableQueryParams(params ...ParamEncoder[Req]) *RequestBuilder[Req, Resp] {
	r.requestOptions = append(r.requestOptions, WithRequestEncodableParams(params...))
	return r
}

// AfterResponse adds to a chain function that will be executed after response is obtained.
func (r *RequestBuilder[Req, Resp]) AfterResponse(f func(resp *http.Response, decoded *Resp) error) *RequestBuilder[Req, Resp] {
	r.client.afterResponse = append(r.client.afterResponse, f)
	return r
}

// Get builds GET request with no body specified.
// Appends request options (includes request options that were specified at NewRequestBuilder).
func (r *RequestBuilder[Req, Resp]) Get(path string, opts ...RequestOption) *RequestBuilder[Req, Resp] {
	r.method = http.MethodGet
	r.resourcePath = path
	r.requestOptions = append(r.requestOptions, opts...)
	return r
}

// Post builds POST request with specified body. Appends request options.
func (r *RequestBuilder[Req, Resp]) Post(path string, body *Req, opts ...RequestOption) *RequestBuilder[Req, Resp] {
	r.method = http.MethodPost
	r.resourcePath = path
	r.body = body
	r.requestOptions = append(r.requestOptions, opts...)
	return r
}

// Patch builds PATCH request with specified body (if any). Appends request options.
func (r *RequestBuilder[Req, Resp]) Patch(path string, body *Req, opts ...RequestOption) *RequestBuilder[Req, Resp] {
	r.method = http.MethodPatch
	r.resourcePath = path
	r.requestOptions = append(r.requestOptions, opts...)
	r.body = body
	return r
}

// Put builds PUT request with specified body (if any). Appends request options.
func (r *RequestBuilder[Req, Resp]) Put(path string, body *Req, opts ...RequestOption) *RequestBuilder[Req, Resp] {
	r.method = http.MethodPut
	r.resourcePath = path
	r.requestOptions = append(r.requestOptions, opts...)
	r.body = body
	return r
}

// DELETE builds DELETE request with specified body (if any). Appends request options.
func (r *RequestBuilder[Req, Resp]) Delete(path string, body *Req, opts ...RequestOption) *RequestBuilder[Req, Resp] {
	r.method = http.MethodDelete
	r.resourcePath = path
	r.requestOptions = append(r.requestOptions, opts...)
	r.body = body
	return r
}

// Do executes request and decodes response into Resp object, returns error if any.
func (r *RequestBuilder[Req, Resp]) Do(ctx context.Context) (*Resp, error) {
	_, data, err := r.client.do(ctx, r)
	if err != nil {
		return nil, err
	}

	return data, nil
}
