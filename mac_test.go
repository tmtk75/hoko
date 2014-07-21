package main

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/hex"
	"io/ioutil"
	"os"
	"testing"
)

func Test_hmac(t *testing.T) {
	// sha1=19c5b5cd530fa784e911125b7fa17c09f18f1db0
	xhubsig := "19c5b5cd530fa784e911125b7fa17c09f18f1db0"
	expected, _ := hex.DecodeString(xhubsig)

	//
	f, _ := os.Open("payload")
	b, _ := ioutil.ReadAll(f)
	secretToken := "d9f8cf3b877081d0ed9f8904eb9981f70be3254c"
	key, _ := hex.DecodeString(secretToken)
	//key := []byte(secretToken)

	mac := hmac.New(sha1.New, key)
	mac.Write(b)
	actual := mac.Sum(nil)

	if !hmac.Equal(expected, actual) {
		t.Errorf("\nexpected-MAC: %v\nactual-MAC: %v\n", expected, actual)
	}
}
