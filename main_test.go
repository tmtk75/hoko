package main

import "testing"

var dirpath Dirpath = "handler"

func Test_invoke_push(t *testing.T) {
	res, err := invoke(dirpath.Path("push"), []byte("hoko"))
	if string(res) != "hello hoko" {
		t.Errorf("hello hoko, but '%s'", res)
	}
	if err != nil {
		t.Errorf("%v", err)
	}
}

func Test_invoke_pushd_no_such_file_or_directory(t *testing.T) {
	res, err := invoke(dirpath.Path("push.d/no_such_file_or_directory"), []byte("hoko"))
	if string(res) != "" {
		t.Errorf("empty, but '%s'", res)
	}
	if err.ExitStatus() != -1 {
		t.Errorf("%v", err)
	}
}

func Test_invoke_pushd_permission_denied(t *testing.T) {
	res, err := invoke(dirpath.Path("push.d/permission_denied"), []byte("hoko"))
	if string(res) != "" {
		t.Errorf("empty, but '%s'", res)
	}
	if err.ExitStatus() != -1 {
		t.Errorf("%v", err)
	}
}

func Test_invoke_event_pushd_exit_1(t *testing.T) {
	res, err := invoke(dirpath.Path("push.d/exit-1"), []byte("hoko"))
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
	if hs[0] != "exit-1" {
		t.Errorf("exit-1, but %v", hs[0])
	}
	if hs[1] != "permission_denied" {
		t.Errorf("permission_denied, but %v", hs[1])
	}
}

func Test_Handlers_no_directory(t *testing.T) {
	hs := Handlers("missing")
	if len(hs) != 0 {
		t.Errorf("0, but %v", len(hs))
	}
}

func Test_Invoke(t *testing.T) {
	rs := Invoke("handler", "push", []byte(""))
	if len(rs) != 3 {
		t.Errorf("3, but %v", len(rs))
	}
	if rs[0].Err != nil {
		t.Errorf("0, but %v", rs[0])
	}
	if rs[1].Err.ExitStatus() != 1 {
		t.Errorf("1, but %v", rs[1])
	}
	if rs[2].Err.ExitStatus() != -1 {
		t.Errorf("-1, but %v", rs[2])
	}
	fail, partial := rs.Failed()
	if !fail {
		t.Errorf("fail: true, but %v", fail)
	}
	if !partial {
		t.Errorf("partial: true, but %v", partial)
	}
}
