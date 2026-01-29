package main

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/0x9ef/clientx"
)

type CatFactAPI struct {
	*clientx.API
}

func New(api *clientx.API) *CatFactAPI {
	return &CatFactAPI{
		API: api,
	}
}

type (
	EmptyRequest struct{}
	Fact         struct {
		Fact   string `json:"fact"`
		Length int    `json:"length"`
	}
)

func (api *CatFactAPI) GetFact(ctx context.Context, opts ...clientx.RequestOption) (*Fact, error) {
	return clientx.NewRequestBuilder[EmptyRequest, Fact](api.API).
		Get("/fact", opts...).
		DoWithDecode(ctx)
}

func main() {
	api := New(
		clientx.NewAPI(
			clientx.WithBaseURL("https://catfact.ninja"),
			clientx.WithRateLimit(100, 100, time.Minute),
			clientx.WithRetry(10, time.Second*3, time.Minute, clientx.ExponentalBackoff,
				func(resp *http.Response, err error) bool {
					return resp.StatusCode == http.StatusTooManyRequests
				},
			),
		),
	)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	resp, err := api.GetFact(ctx)
	if err != nil {
		panic(err)
	}

	fmt.Printf("Fact (len=%d): %s\n", resp.Length, resp.Fact)
}
