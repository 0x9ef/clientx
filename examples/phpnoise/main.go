package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/0x9ef/clientx"
)

type PHPNoiseAPI struct {
	*clientx.API
	mu             *sync.Mutex
	lastUploadURI  string
	lastUploadSize int
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
	if r.JSON != 0 {
		v.Set("json", "1")
	}
	if r.Base64 != 0 {
		v.Set("base64", "1")
	}
	return nil
}

func (api *PHPNoiseAPI) Generate(ctx context.Context, req GenerateRequest, opts ...clientx.RequestOption) (*Generate, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}

	resp, err := clientx.NewRequestBuilder[GenerateRequest, Generate](api.API).
		Get("/noise.php", opts...).        // make GET to /noise.php and apply request options
		WithStructQueryParams("url", req). // as far as our GenerateParams structure has "query" tag, we can specify this tag to process
		DoWithDecode(ctx)
	if err != nil {
		return nil, err
	}

	api.mu.Lock()
	defer api.mu.Unlock()
	api.lastUploadURI = resp.URI

	return resp, nil
}

func (api *PHPNoiseAPI) GenerateReader(ctx context.Context, req GenerateRequest, opts ...clientx.RequestOption) (io.ReadCloser, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}

	resp, err := clientx.NewRequestBuilder[GenerateRequest, struct{}](api.API).
		Get("/noise.php", opts...).
		WithEncodableQueryParams(req).
		AfterResponse(func(resp *http.Response, _ []byte) error {
			api.mu.Lock()
			defer api.mu.Unlock()
			size, err := strconv.Atoi(resp.Header.Get("Content-Length")) // don't do like that, because Content-Length could be fake
			if err != nil {
				return err
			}
			api.lastUploadSize = size
			return nil
		}).
		Do(ctx)
	if err != nil {
		return nil, err
	}

	return resp.Body, nil
}

func generate(min, max int) int {
	r := rand.New(rand.NewSource(time.Now().Unix()))
	return r.Intn((max - min) + min)
}

func main() {
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

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()

	// Get URI to download and base64 of image
	resp, err := api.Generate(ctx, GenerateRequest{
		R:           generate(0, 255),
		G:           generate(0, 255),
		B:           generate(0, 255),
		Tiles:       10,
		TileSize:    15,
		BorderWidth: 5,
		ColorMode:   ColorModeBrigthness,
		JSON:        1, // Indicates that our response should be in JSON format
		Base64:      1, // Indicates that encoding is base64 (only with JSON option)
	})
	if err != nil {
		panic(err)
	}
	fmt.Println("Current URI:", resp.URI)
	fmt.Println("LastUploaded URI:", api.lastUploadURI)

	// Directly download PNG and write into file
	body, err := api.GenerateReader(ctx, GenerateRequest{
		R:           generate(0, 255),
		G:           generate(0, 255),
		B:           generate(0, 255),
		Tiles:       10,
		TileSize:    15,
		BorderWidth: 5,
		ColorMode:   ColorModeBrigthness,
		// JSON=0, BASE64=0
	})
	if err != nil {
		panic(err)
	}
	fmt.Println("LastUploaded Size:", api.lastUploadSize)
	defer body.Close()

	png, err := os.Create("noise.png")
	if err != nil {
		panic(err)
	}
	if _, err := io.Copy(png, body); err != nil {
		panic(err)
	}
}
