package main

import (
	"fmt"
	"os/exec"
	"syscall"
)

type ExitError struct {
	Err    error
	status int
}

func (self *ExitError) Error() string {
	return fmt.Sprintf("%s:%v", self.Err.Error(), self.ExitStatus())
}

func (self *ExitError) ExitStatus() int {
	if exiterr, ok := self.Err.(*exec.ExitError); ok {
		if status, ok := exiterr.Sys().(syscall.WaitStatus); ok {
			return status.ExitStatus()
		}
	}
	return self.status
}

func Exec(path string, body []byte) ([]byte, *ExitError) {
	path, err := exec.LookPath(path)
	if err != nil {
		if _, ok := err.(*exec.Error); ok {
			return nil, &ExitError{err, -1}
		} else {
			panic(err)
		}
	}
	cmd := exec.Command(path)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, &ExitError{err, -2}
	}
	stdin.Write(body)
	stdin.Close()
	out, err := cmd.Output()
	if err != nil {
		return nil, &ExitError{err, -3}
	}

	return out, nil
}
