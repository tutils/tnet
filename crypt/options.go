package crypt

type EncoderOptions interface {
}

type EncoderOption func(opts EncoderOptions)

type DecoderOptions interface {
}

type DecoderOption func(opts DecoderOptions)
