package main

import "testing"

var dirpath = "handler"

func Test_invoke_push(t *testing.T) {
	res, err := invoke(dirpath, "push", []byte("hoko"))
	if string(res) != "hello hoko" {
		t.Errorf("hello hoko, but '%s'", res)
	}
	if err != nil {
		t.Errorf("%v", err)
	}
}

func Test_invoke_pushd_no_such_file_or_directory(t *testing.T) {
	res, err := invoke(dirpath, "push.d/no_such_file_or_directory", []byte("hoko"))
	if string(res) != "" {
		t.Errorf("empty, but '%s'", res)
	}
	if err.ExitStatus() != -1 {
		t.Errorf("%v", err)
	}
}

func Test_invoke_pushd_permission_denied(t *testing.T) {
	res, err := invoke(dirpath, "push.d/permission_denied", []byte("hoko"))
	if string(res) != "" {
		t.Errorf("empty, but '%s'", res)
	}
	if err.ExitStatus() != -1 {
		t.Errorf("%v", err)
	}
}

func Test_invoke_event_pushd_exit_1(t *testing.T) {
	res, err := invoke(dirpath, "push.d/exit-1", []byte("hoko"))
	if string(res) != "" {
		t.Errorf("hello hoko, but '%s'", res)
	}
	if err == nil {
		t.Errorf("not nil, but %v", err)
	}
	if err.ExitStatus() != 1 {
		t.Errorf("%v", err)
	}
}

func Test_Handlers(t *testing.T) {
	hs := Handlers("push")
	if len(hs) != 2 {
		t.Errorf("2, but %v", len(hs))
	}
}

func Test_Handlers_no_directory(t *testing.T) {
	hs := Handlers("missing")
	if len(hs) != 0 {
		t.Errorf("0, but %v", len(hs))
	}
}
