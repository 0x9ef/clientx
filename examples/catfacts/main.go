package main

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/0x9ef/clientx"
)

type CatFactAPI struct {
	*clientx.API
}

type Fact struct {
	Fact   string `json:"fact"`
	Length int    `json:"length"`
}

func (api *CatFactAPI) GetFact(ctx context.Context, opts ...clientx.RequestOption) (*Fact, error) {
	return clientx.NewRequestBuilder[struct{}, Fact](api.API).
		Get("/fact", opts...).
		AfterResponse(func(resp *http.Response, fact *Fact) error {
			fmt.Println("Done", fact.Fact, fact.Length)
			return nil
		}).
		Do(ctx)
}

func main() {
	api := &CatFactAPI{
		API: clientx.NewAPI(
			clientx.WithBaseURL("https://catfact.ninja"),
			clientx.WithRateLimit(100, 100, time.Minute),
			clientx.WithRetry(10, time.Second*3, time.Minute, clientx.ExponentalBackoff,
				func(resp *http.Response, err error) bool {
					return resp.StatusCode == http.StatusTooManyRequests
				},
			),
		),
	}

	var wg sync.WaitGroup
	for i := 0; i < 110; i++ {
		wg.Add(1)
		go func(wg *sync.WaitGroup) {
			defer wg.Done()
			_, err := api.GetFact(context.TODO())
			if err != nil {
				panic(err)
			}
		}(&wg)
	}
	wg.Wait()
}
