// Copyright (c) 2024 0x9ef. All rights reserved.
// Use of this source code is governed by an MIT license
// that can be found in the LICENSE file.
package clientx

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"
)

type client[Req any, Resp any] struct {
	api           *API
	afterResponse []func(resp *http.Response, data *Resp) error
}

func (c *client[Req, Resp]) do(ctx context.Context, req *RequestBuilder[Req, Resp]) (*http.Response, *Resp, error) {
	// Wait for ratelimits. It is a blocking call.
	if err := c.api.limiter.Wait(ctx); err != nil {
		return nil, nil, err
	}

	resp, err := c.makeRequest(ctx, req)
	if err != nil {
		return nil, nil, err
	}

	var data Resp
	if err := decodeResponse(resp, &data); err != nil {
		return nil, nil, err
	}

	for _, after := range c.afterResponse {
		if err := after(resp, &data); err != nil {
			return nil, nil, fmt.Errorf("after response exec failed: %w", err)
		}
	}

	return resp, &data, nil
}

func (c *client[Req, Resp]) makeRequest(ctx context.Context, req *RequestBuilder[Req, Resp]) (*http.Response, error) {
	u, err := c.buildRequestURL(req.resourcePath)
	if err != nil {
		return nil, err
	}

	r, err := http.NewRequestWithContext(ctx, req.method, u.String(), nil)
	if err != nil {
		return nil, err
	}
	// If method is not GET, try to set payload body
	if req.method != http.MethodGet && req.body != nil {
		r.Body, err = req.encodeRequestPayload()
		if err != nil {
			return nil, err
		}
	}

	// Apply options to request
	for _, o := range req.requestOptions {
		if err := o(r); err != nil {
			return nil, err
		}
	}

	do := func(c *client[Req, Resp], req *http.Request, reuse bool) (*http.Response, error) {
		if reuse && req.Body != nil {
			// In a way when we use retry functionality we have to copy
			// request and pass it to c.httpDoer.Do, but req.Clone() doesn't really deep clone Body
			// and we have to clone body manually as in httputil.DumpRequestOut
			//
			// Issue https://github.com/golang/go/issues/36095
			var b bytes.Buffer
			b.ReadFrom(req.Body)
			req.Body = ioutil.NopCloser(&b)

			cloneReq := req.Clone(ctx)
			cloneReq.Body = ioutil.NopCloser(bytes.NewReader(b.Bytes()))
			req = cloneReq
		}

		resp, err := c.api.httpClient.Do(req)
		if err != nil {
			return nil, err
		}

		if c.api.options.Debug {
			b, err := httputil.DumpResponse(resp, true)
			if err != nil {
				return nil, err
			}
			fmt.Printf("RESPONSE:\n%s\n", string(b))
		}
		return resp, nil
	}
	if c.api.retry == nil {
		// Do single request without using backoff retry mechanism
		return do(c, r, false)
	}

	for {
		resp, err := do(c, r, true)

		var isMatchedCond bool
		for _, cond := range c.api.options.Retry.Conditions {
			if ok := cond(resp, err); ok {
				isMatchedCond = true
				break
			}
		}
		if isMatchedCond {
			// Get next duration interval, sleep and make another request
			// till nextDuration != stopBackoff
			nextDuration := c.api.retry.Next()
			if nextDuration == stopBackoff {
				c.api.retry.Reset()
				return resp, err
			}
			time.Sleep(nextDuration)
			continue
		}

		// Break retries mechanism if conditions weren't matched
		return resp, err
	}
}

func (c *client[Req, Resp]) buildRequestURL(resource string) (*url.URL, error) {
	u, err := url.Parse(c.api.options.BaseURL)
	if err != nil {
		return nil, err
	}
	u.Path = resource
	return u, nil
}
