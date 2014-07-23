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
	fbody, _ := os.Open("reqbody.json")
	fsec, _ := os.Open("secret_token.txt")
	fsig, _ := os.Open("x-hub-signature.txt")
	defer fbody.Close()
	defer fsec.Close()
	defer fsig.Close()
	body, _ = ioutil.ReadAll(fbody)
	key, _ = ioutil.ReadAll(fsec)
	sign, _ = ioutil.ReadAll(fsig)
}

func Test_x_hub_signature1(t *testing.T) {
	expected := sign[4+1 : len(sign)]

	mac := hmac.New(sha1.New, key)
	mac.Write(body)
	actualBytes := mac.Sum(nil)
	actualStr := hex.EncodeToString(actualBytes)
	actual := []byte(actualBytes)

	if actualStr != string(expected) {
		t.Errorf("%v != %v\n", actual, expected)
	}
}

func Test_x_hub_signature2(t *testing.T) {
	expected, _ := hex.DecodeString(string(sign[4+1 : len(sign)]))

	mac := hmac.New(sha1.New, key)
	mac.Write(body)
	actualBytes := mac.Sum(nil)

	if !hmac.Equal(actualBytes, expected) {
		t.Errorf("%v != %v\n", actualBytes, expected)
	}
}
