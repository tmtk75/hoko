package main

import "testing"

func TestInvoke(t *testing.T) {
	Invoke("push", []byte("{}"))
}
