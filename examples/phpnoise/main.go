package main

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"net/http"
	"net/url"
	"strconv"
	"sync"
	"time"

	"github.com/0x9ef/clientx"
)

type PHPNoiseAPI struct {
	*clientx.API
	mu            *sync.Mutex
	lastUploadURI string
}

func New(api *clientx.API) *PHPNoiseAPI {
	return &PHPNoiseAPI{
		API: api,
		mu:  new(sync.Mutex),
	}
}

type (
	GenerateRequest struct {
		R           int       `url:"r"`
		G           int       `url:"g"`
		B           int       `url:"b"`
		Tiles       int       `url:"titles"`
		TileSize    int       `url:"tileSize"`
		BorderWidth int       `url:"borderWidth"`
		ColorMode   ColorMode `url:"colorMode"`
		JSON        int       `url:"json"`
		Base64      int       `url:"base64"`
	}

	ColorMode string

	Generate struct {
		URI    string `json:"uri"`
		Base64 string `json:"base64"`
	}
)

const (
	ColorModeBrigthness ColorMode = "brightness"
	ColorModeAround     ColorMode = "around"
)

func (mode ColorMode) String() string {
	return string(mode)
}

// You may use go-validator package to validate through struct fields instead of custom validate function.
func (r *GenerateRequest) Validate() error {
	if r.R > 255 {
		return errors.New("R is exceeded >255")
	}
	if r.G > 255 {
		return errors.New("G is exceeded >255")
	}
	if r.B > 255 {
		return errors.New("B is exceeded >255")
	}
	if r.Tiles < 1 || r.Tiles > 50 {
		return errors.New("invalid tiles number (1-50)")
	}
	if r.TileSize < 1 || r.TileSize > 20 {
		return errors.New("invalid tile size (1-20)")
	}
	if r.BorderWidth > 15 {
		return errors.New("invalid BorderWidth (0-15)")
	}
	if r.ColorMode != ColorModeBrigthness && r.ColorMode != ColorModeAround {
		return errors.New("invalid ColorMode, supported: brightness, around")
	}
	return nil
}

// Uses when calling WithEncodableQueryParams.
func (r GenerateRequest) Encode(v url.Values) error {
	v.Set("r", strconv.Itoa(r.R))
	v.Set("g", strconv.Itoa(r.G))
	v.Set("b", strconv.Itoa(r.B))
	v.Set("borderWidth", strconv.Itoa(r.BorderWidth))
	if r.Tiles != 0 {
		v.Set("tiles", strconv.Itoa(r.Tiles))
	}
	if r.TileSize != 0 {
		v.Set("tileSize", strconv.Itoa(r.TileSize))
	}
	if r.ColorMode != "" {
		v.Set("mode", r.ColorMode.String())
	}
	return nil
}

func (api *PHPNoiseAPI) Generate(ctx context.Context, req GenerateRequest, opts ...clientx.RequestOption) (*Generate, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}

	return clientx.NewRequestBuilder[GenerateRequest, Generate](api.API).
		Get("/noise.php", opts...).
		WithQueryParams("url", req).
		AfterResponse(func(resp *http.Response, model *Generate) error {
			api.mu.Lock()
			defer api.mu.Unlock()
			api.lastUploadURI = model.URI
			return nil
		}).
		Do(ctx)
}

func generate(min, max int) int {
	rand.Seed(time.Now().Unix())
	return rand.Intn((max - min) + min)
}

func main() {
	api := New(
		clientx.NewAPI(
			clientx.WithBaseURL("https://php-noise.com"),
			clientx.WithHeader("Authorization", "Bearer MY_ACCESS_TOKEN"),
			clientx.WithRateLimit(10, 2, time.Minute),
			clientx.WithRetry(10, time.Second*3, time.Minute, clientx.ExponentalBackoff,
				func(resp *http.Response, err error) bool {
					return resp.StatusCode == http.StatusTooManyRequests
				},
			),
		),
	)

	resp, err := api.Generate(context.TODO(), GenerateRequest{
		R:           generate(0, 255),
		G:           generate(0, 255),
		B:           generate(0, 255),
		Tiles:       10,
		TileSize:    15,
		BorderWidth: 5,
		ColorMode:   ColorModeBrigthness,
		JSON:        1,
		Base64:      1,
	})
	if err != nil {
		panic(err)
	}
	fmt.Println("Current URI:", resp.URI)
	fmt.Println("LastUploaded URI:", api.lastUploadURI)
}
