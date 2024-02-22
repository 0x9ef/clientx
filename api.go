// Copyright (c) 2024 0x9ef. All rights reserved.
// Use of this source code is governed by an MIT license
// that can be found in the LICENSE file.
//
// Package clientx provides functions to build and maintain your own HTTP client.
package clientx

import (
	"net/http"
	"time"

	"golang.org/x/time/rate"
)

// API represents general base API which has to be inherited.
//
//	type DuffelAPI struct {
//	  *clientx.API
//	}
type API struct {
	httpClient *http.Client
	options    *Options
	retry      Retrier
	limiter    Limiter
}

type (
	Option  func(*Options)
	Options struct {
		BaseURL    string
		HttpClient *http.Client
		Headers    http.Header
		// Debug prints responses into os.Stdout.
		Debug bool
		// RateLimitParseFn is a custom function that parses rate limits from HTTP response.
		// For example from X-Ratelimit-Limit, X-Ratelimit-Remaining headers.
		RateLimitParseFn func(*http.Response) (limit int, remaining int, resetAt time.Time, err error)
		RateLimit        *OptionRateLimit
		Retry            *OptionRetry
	}

	OptionRateLimit struct {
		Limit int
		Burst int
		// Per allows configuring limits for different time windows.
		Per time.Duration
	}

	OptionRetry struct {
		MaxAttempts int
		MinWaitTime time.Duration
		MaxWaitTime time.Duration
		// Conditions that will be applied to retry mechanism.
		Conditions []RetryCond
		// Retry function which will be used as main retry logic.
		Fn RetryFunc
	}
)

// NewAPI returns new base API structure with preselected http.DefaultClient
// and options. Applies all options, overwrites HttpClient if such option is presented.
func NewAPI(opts ...Option) *API {
	options := &Options{
		HttpClient: http.DefaultClient,
	}
	for _, opt := range opts {
		opt(options)
	}

	api := &API{
		httpClient: options.HttpClient,
		options:    options,
	}
	if options.Retry != nil {
		api.retry = &backoff{
			minWaitTime: options.Retry.MinWaitTime,
			maxWaitTime: options.Retry.MaxWaitTime,
			maxAttempts: int64(options.Retry.MaxAttempts),
			attempts:    0,
			f:           options.Retry.Fn,
		}
	}
	if options.RateLimit != nil {
		limit := rate.Every(options.RateLimit.Per / time.Duration(options.RateLimit.Limit))
		api.limiter = newAdaptiveBucketLimiter(limit, options.RateLimit.Burst)
	} else {
		api.limiter = newUnlimitedAdaptiveBucketLimiter()
	}

	return api
}

// WithDebug enables debug logging of requests and responses.
// DO NOT USE IN PRODUCTION.
func WithDebug() Option {
	return func(o *Options) {
		o.Debug = true
	}
}

// WithBaseURL sets base URL to perform requests.
func WithBaseURL(url string) Option {
	return func(o *Options) {
		o.BaseURL = url
	}
}

// WithHTTPClient allows you to specify a custom http.Client to use for making requests.
// This is useful if you want to use a custom transport or proxy.
func WithHTTPClient(client *http.Client) Option {
	return func(o *Options) {
		o.HttpClient = client
	}
}

// WithRetry sets custom retrier implementation. Also enables retrying mechanism.
// If f retry function isn't provided ExponentalBackoff algorithm will be used.
func WithRetry(maxAttempts int, minWaitTime, maxWaitTime time.Duration, f RetryFunc, conditions ...RetryCond) Option {
	return func(o *Options) {
		if f == nil {
			f = ExponentalBackoff // uses as default
		}
		o.Retry = &OptionRetry{
			MaxAttempts: maxAttempts,
			MinWaitTime: minWaitTime,
			MaxWaitTime: maxWaitTime,
			Conditions:  conditions,
			Fn:          f,
		}
	}
}

// WithRateLimit sets burst and limit for a ratelimiter.
func WithRateLimit(limit int, burst int, per time.Duration) Option {
	return func(o *Options) {
		o.RateLimit = &OptionRateLimit{
			Limit: limit,
			Burst: burst,
			Per:   per,
		}
	}
}

// WithHeader sets global header. Overwrites values related to key.
func WithHeader(key string, value string) Option {
	return func(o *Options) {
		if len(o.Headers) == 0 {
			o.Headers = make(http.Header)
		}
		o.Headers[key] = []string{value}
	}
}

// WithHeaderSet sets global headers. Overwrites previously defined header set.
func WithHeaderSet(headers map[string][]string) Option {
	return func(o *Options) {
		if len(o.Headers) == 0 {
			o.Headers = make(http.Header)
		}
		o.Headers = headers
	}
}
