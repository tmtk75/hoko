package main

import (
	"testing"
)

func TestBuildTagOptions(t *testing.T) {
	a := buildTagOptions(map[string][]string{"foo": {"bar"}})
	if !(a[0] == "-tag" && a[1] == "foo=bar") {
		t.Errorf("foo=bar != %v", a[0:2])
	}

	a = buildTagOptions(map[string][]string{"foo": {"bar"}, "fiz": {"biz"}})
	if !(a[2] == "-tag" && a[3] == "fiz=biz") {
		t.Errorf("fiz=biz != %v", a[2:4])
	}
}
