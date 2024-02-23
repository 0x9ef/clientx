package clientx

import (
	"encoding/json"
	"encoding/xml"
	"io"
)

type EncoderDecoder interface {
	Encoder
	Decoder
}

// Encoder is a general interface responsibles for encoding payloads.
type Encoder interface {
	Encode(w io.Writer, v any) error
}

// Decoder is a general interface responsibles for decoding responses.
type Decoder interface {
	Decode(r io.Reader, dst any) error
}

// JSON Encoder/Decoder realization.
var JSONEncoderDecoder = &jsonEncoderDecoder{}

type jsonEncoderDecoder struct{}

func (jsonEncoderDecoder) Encode(w io.Writer, v any) error {
	return json.NewEncoder(w).Encode(v)
}

func (jsonEncoderDecoder) Decode(r io.Reader, dst any) error {
	return json.NewDecoder(r).Decode(dst)
}

// XML Encoder/Decoder realization.
var XMLEncoderDecoder = &xmlEncoderDecoder{}

type xmlEncoderDecoder struct{}

func (xmlEncoderDecoder) Encode(w io.Writer, v any) error {
	return xml.NewEncoder(w).Encode(v)
}

func (xmlEncoderDecoder) Decode(r io.Reader, dst any) error {
	return xml.NewDecoder(r).Decode(dst)
}

// Blank (No Action) Encoder/Decoder realization.
var BlankEncoderDecoder = &blankEncoderDecoder{}

type blankEncoderDecoder struct{}

func (blankEncoderDecoder) Encode(v any) ([]byte, error)      { return nil, nil }
func (blankEncoderDecoder) Decode(r io.Reader, dst any) error { return nil }
