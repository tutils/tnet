package xor

import (
	"bytes"
	"io/ioutil"
	"testing"
)

func TestNewCrypt(t *testing.T) {
	buf := &bytes.Buffer{}
	c := NewCrypt(544141)
	en := c.NewEncoder(buf)
	de := c.NewDecoder(buf)

	en.Write([]byte("abcdefg"))
	t.Log(buf.Bytes())

	bs, _ := ioutil.ReadAll(de)
	t.Log(string(bs))
}
