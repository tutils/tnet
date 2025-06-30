package cmd

import (
	"encoding/base64"
	"encoding/gob"
	"os"
	"strings"

	"github.com/tutils/tnet/crypt"
	"github.com/tutils/tnet/crypt/xor"
)

var (
	xorCrypt = xor.NewCrypt(33280939)
)

func encodeCmdline() (string, error) {
	w1 := &strings.Builder{}
	w2 := base64.NewEncoder(base64.RawStdEncoding, w1)
	defer w2.Close()
	w3 := xorCrypt.NewEncoder(w2, xor.WithEncoderRandomSourceNewer(crypt.NewLCGSource))
	// w4 := zlib.NewWriter(w3)
	// defer w4.Close()
	w4 := w3
	w5 := gob.NewEncoder(w4)
	if err := w5.Encode(os.Args[1:]); err != nil {
		return "", err
	}
	// w4.Close()
	w2.Close()
	return w1.String(), nil
}

func decodeCmdline(s string) error {
	r1 := strings.NewReader(s)
	r2 := base64.NewDecoder(base64.RawStdEncoding, r1)
	r3 := xorCrypt.NewDecoder(r2, xor.WithDecoderRandomSourceNewer(crypt.NewLCGSource))
	// r4, err := zlib.NewReader(r3)
	// if err != nil {
	// 	return err
	// }
	// defer r4.Close()
	r4 := r3
	r5 := gob.NewDecoder(r4)
	var args []string
	if err := r5.Decode(&args); err != nil {
		return err
	}
	if len(os.Args) >= 1 {
		os.Args = append(os.Args[1:], args...)
	}
	return nil
}
