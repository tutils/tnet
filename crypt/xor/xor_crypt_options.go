package xor

import (
	"math/rand"

	"github.com/tutils/tnet/crypt"
)

type xorEncoderOptions struct {
	sourceNewer RandomSourceNewer
}

func newXorEncoderOptions(opts ...crypt.EncoderOption) *xorEncoderOptions {
	var opt xorEncoderOptions
	for _, o := range opts {
		o(&opt)
	}
	if opt.sourceNewer == nil {
		opt.sourceNewer = rand.NewSource
	}
	return &opt
}

type RandomSourceNewer func(int64) rand.Source

func WithEncoderRandomSourceNewer(newer RandomSourceNewer) crypt.EncoderOption {
	return func(opts crypt.EncoderOptions) {
		if o, ok := opts.(*xorEncoderOptions); ok {
			o.sourceNewer = newer
		}
	}
}

type xorDecoderOptions struct {
	sourceNewer RandomSourceNewer
}

func newXorDecoderOptions(opts ...crypt.DecoderOption) *xorDecoderOptions {
	var opt xorDecoderOptions
	for _, o := range opts {
		o(&opt)
	}
	if opt.sourceNewer == nil {
		opt.sourceNewer = rand.NewSource
	}
	return &opt
}

func WithDecoderRandomSourceNewer(newer RandomSourceNewer) crypt.DecoderOption {
	return func(opts crypt.DecoderOptions) {
		if o, ok := opts.(*xorDecoderOptions); ok {
			o.sourceNewer = newer
		}
	}
}
