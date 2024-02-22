# Golang client for API building
The purpose of this client is to design and develop clients for any API very fast using generics for request, response models encoding/decoding with supported from the box retry, rate-limit, GZIP/Deflate decoding functionality.

## Installation
> NOTE: Requires at least Go 1.18 since we use generics

To get latest version use:
```
go get github.com/0x9ef/clientx@latest
```

To specify version use:
```
go get github.com/0x9ef/clientx@1.24.4 # version
```

## Usage examples
See the [examples/](https://github.com/0x9ef/clientx/blob/master/examples) folder (CatFacts, PHPNoise APIs)

## Getting Started
The first thing you need to understand: it will be easy :)

The client was developed with consuming needs of modern API development. I have designed and integrated many clients for different APIs and came up with a couple useful things. I was intended to make it easy, understandable even for beginner, and include top of the most necessary functionality.

### Initialize
When you are initializing client, you MUST specify base URL by clientx.WithBaseURL option.

```go
api := clientx.NewAPI(
	clientx.WithBaseURL("https://php-noise.com"),
}
```

### Authorizarion
There is no separate flow for authorization, but it can be done with HTTP headers. Let's talk about Bearer authorization. You have the API Access Token and you have to build HTTP Header: `Authorizarion: Bearer MY_ACCESS_TOKEN`, you could pass it through request options.

```go
// GetOffer accepts offerId and request options that will be applied before request is sent.
func (api *MyAPI) GetOffer(ctx context.Context, offerId string, opts ...clientx.RequestOption) (*Offer, error) {
    return clientx.NewRequestBuilder[struct{}, Offer](api.API).
		Get("/offers/"+offerId, opts...).
		Do(ctx)
}

func main() {
    ... 
    ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	
    resp, err := api.GetOffer(ctx, "off_1234", clientx.WithRequestHeaders(map[string][]string{
        "Authorization":    []string{"Bearer MY_ACCESS_TOKEN"}, 
    }))
}
```

### Options
There is a list of supported options from the box:
* WithRateLimit - enables rate limiting mechanism
* WithRetry - enables backoff retry mechanism

```go
api := New(
	clientx.NewAPI(
		clientx.WithBaseURL("https://php-noise.com"),
		clientx.WithRateLimit(10, 2, time.Minute),
		clientx.WithRetry(10, time.Second*3, time.Minute, clientx.ExponentalBackoff,
			func(resp *http.Response, err error) bool {
				return resp.StatusCode == http.StatusTooManyRequests
			},
		),
	),
)
```

### Rate limiting
Most of the APIs have rate limits. Let's take for example the next limit: 100req/sec, so we want to stay within the limit. The rate limiter functionality supported from the box: wrapper around golang.org/x/time/rate package.

```go
api := New(
	clientx.NewAPI(
		clientx.WithBaseURL("https://php-noise.com"),
		clientx.WithRateLimit(10, 2, time.Minute), // max limit: ^10, burst limit: ^2, interval: ^time.Minute
	),
)
``` 

If the limit is exceeded, all further call will be blocked until we gain enough capacity to perform new requests.

### Retry
Retry functionality can be combined with rate limiting. There are cases when you don't know the rate limiting interval. In this case you can use backoff retry mechanism. You can retry after 1 sec or you can wait for 60 minutes. The 429 (Too Many Requests) status code is an indicator when rate limit is exceeded.

```go
api := New(
	clientx.NewAPI(
		clientx.WithBaseURL("https://php-noise.com"),
		clientx.WithRateLimit(10, 2, time.Minute), 
        // Parameters: max retry attempts, minimal wait time, maximal wait time, retry function (you could provide your own which is suitable for clientx.RetryFunc), trigger function (in our example we consider all 429 statuses as a tigger)
        clientx.WithRetry(10, time.Second*3, time.Minute, clientx.ExponentalBackoff,
			func(resp *http.Response, err error) bool {
				return resp.StatusCode == http.StatusTooManyRequests
			},
		),
	),
)
```

### Request options
You can add custom headers to request or set query parameters, form data, etc... The list of supported request options you can find [here](https://github.com/0x9ef/clientx/blob/master/requestoptions.go).

```go
func (api *MyAPI) GetOffer(ctx context.Context, offerId string, opts ...clientx.RequestOption) (*Offer, error) {
    return clientx.NewRequestBuilder[struct{}, Offer](api.API).
		Get("/offers/"+offerId, opts...).
		Do(ctx)
}

func main() {
    ... 
    ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

    resp, err := api.GetOffer(ctx, "off_1234", clientx.WithRequestHeaders(map[string][]string{
        "Authorization":    []string{"Bearer token_test"}, 
        "X-Correlation-Id": []string{"mdj34fjhgsdb4"},
    }))
}
```

### Query parameters encode
There are two ways to encode query parameters, one can be preferred rather than another one.

```go
type GetOfferParams struct {
    FilterBy string `url:"filter_by"`
}

func (param GetOfferParam) Encode(v url.Values) error {
    v.Set("filter_by", param.FilterBy)
    return nil
}


// Variant based on WithQueryParams (when we want to encode through structure tags) 
func (api *MyAPI) GetOffer(ctx context.Context, offerId string, params GetOfferParams, opts ...clientx.RequestOption) (*Offer, error) {
    return clientx.NewRequestBuilder[struct{}, Offer](api.API).
		Get("/offers/"+offerId, opts...).
        WithQueryParams("url", params).
		Do(ctx)
}

// Variant based on WithEncodableQueryParams when we implement clientx.ParamEncoder interface
func (api *MyAPI) GetOffer(ctx context.Context, offerId string, params GetOfferParams, opts ...clientx.RequestOption) (*Offer, error) {
    return clientx.NewRequestBuilder[struct{}, Offer](api.API).
		Get("/offers/"+offerId, opts...).
        WithEncodableQueryParams(params).
		Do(ctx)
}
```

## Contributing
If you found a bug or have an idea for a new feature, please first discuss it with us by [submitting a new issue](https://github.com/0x9ef/clientx/issues). 