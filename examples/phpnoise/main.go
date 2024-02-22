package main

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/0x9ef/clientx"
)

type PHPNoiseAPI struct {
	*clientx.API
}

type (
	GenerateParams struct {
		R           int    `query:"r"`
		G           int    `query:"g"`
		B           int    `query:"b"`
		Tiles       int    `query:"titles"`
		TileSize    int    `query:"tileSize"`
		BorderWidth int    `query:"borderWidth"`
		ColorMode   string `query:"colorMode"`
		JSON        int    `query:"json"`
		Base64      int    `query:"base64"`
	}

	Generate struct {
		Url    string `json:"url"`
		Base64 string `json:"base64"`
	}
)

const (
	ColorModeBrigthness = "brightness"
	ColorModeAround     = "around"
)

// You may use go-validator package to validate through struct fields instead of custom validate function.
func (r *GenerateParams) Validate() error {
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
func (r GenerateParams) Encode(v url.Values) error {
	v.Set("r", strconv.Itoa(r.R))
	v.Set("g", strconv.Itoa(r.G))
	v.Set("b", strconv.Itoa(r.B))
	if r.Tiles != 0 {
		v.Set("tiles", strconv.Itoa(r.Tiles))
	}
	if r.TileSize != 0 {
		v.Set("tileSize", strconv.Itoa(r.TileSize))
	}
	if r.ColorMode != "" {
		v.Set("mode", r.ColorMode)
	}
	return nil
}

func (api *PHPNoiseAPI) Generate(ctx context.Context, param GenerateParams, opts ...clientx.RequestOption) (*Generate, error) {
	return clientx.NewRequestBuilder[GenerateParams, Generate](api.API).
		Get("/noise.php", opts...).
		WithQueryParams("query", param).
		AfterResponse(func(resp *http.Response, noise *Generate) error {
			fmt.Println("Base64", noise.Base64)
			return nil
		}).
		Do(ctx)
}

func generate(min, max int) int {
	rand.Seed(time.Now().Unix())
	return rand.Intn((max - min) + min)
}

func main() {
	api := &PHPNoiseAPI{
		API: clientx.NewAPI(
			clientx.WithBaseURL("https://php-noise.com"),
			clientx.WithRateLimit(10, 2, time.Minute),
			clientx.WithRetry(10, time.Second*3, time.Minute, clientx.ExponentalBackoff,
				func(resp *http.Response, err error) bool {
					return resp.StatusCode == http.StatusTooManyRequests
				},
			),
		),
	}

	for i := 0; i < 10; i++ {
		_, err := api.Generate(context.TODO(), GenerateParams{
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
	}

}
