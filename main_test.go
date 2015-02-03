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

func Test_buildArgs(t *testing.T) {
	a := buildTagOptions(map[string][]string{"foo": {"bar"}})
	if !(a[0] == "-tag" && a[1] == "foo=bar") {
		t.Errorf("foo=bar != %v", a[0:2])
	}

	a = buildTagOptions(map[string][]string{"foo": {"bar"}, "fiz": {"biz"}})
	if !(a[2] == "-tag" && a[3] == "fiz=biz") {
		t.Errorf("fiz=biz != %v", a[2:4])
	}
}

func Test_shrink(t *testing.T) {
	c := shrink(
		[]BitbucketCommit{
			BitbucketCommit{Author: "1"},
			BitbucketCommit{Author: "2"},
			BitbucketCommit{Author: "3"},
		})
	if !(len(c) == 2) {
		t.Errorf("%v", len(c))
	}
	if !(c[0].Author == "1") {
		t.Errorf("%v", c[0])
	}
	if !(c[1].Author == "3") {
		t.Errorf("%v", c[1])
	}

	c = shrink(
		[]BitbucketCommit{
			BitbucketCommit{Author: "10"},
			BitbucketCommit{Author: "20"},
		})
	if !(len(c) == 2) {
		t.Errorf("%v", len(c))
	}
	if !(c[0].Author == "10") {
		t.Errorf("%v", c[0])
	}
	if !(c[1].Author == "20") {
		t.Errorf("%v", c[1])
	}

	c = shrink(
		[]BitbucketCommit{
			BitbucketCommit{Author: "300"},
		})
	if !(len(c) == 1) {
		t.Errorf("%v", len(c))
	}
	if !(c[0].Author == "300") {
		t.Errorf("%v", c[0])
	}

	c = shrink([]BitbucketCommit{})
	if !(len(c) == 0) {
		t.Errorf("%v", len(c))
	}
}
