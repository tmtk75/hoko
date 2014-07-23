package main

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/hex"
	"io/ioutil"
	"os"
	"testing"
)

var body, key, sign []byte

func init() {
	open := func(name string) []byte {
		f, _ := os.Open(name)
		defer f.Close()
		b, _ := ioutil.ReadAll(f)
		return b
	}
	body = open("test/webhook-body.json")
	key = open("test/secret_token.txt")
	sign = open("test/x-hub-signature.txt")
}

func Test_x_hub_signature(t *testing.T) {
	expected, _ := hex.DecodeString(string(sign[4+1 : len(sign)]))

	mac := hmac.New(sha1.New, key)
	mac.Write(body)
	actual := mac.Sum(nil)

	if !hmac.Equal(actual, expected) {
		t.Errorf("%v != %v\n", actual, expected)
	}
}
