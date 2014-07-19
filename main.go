package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"sort"
	"strings"
	"syscall"

	"github.com/codegangsta/cli"
	"github.com/go-martini/martini"
	"github.com/martini-contrib/render"
)

var logger = log.New(os.Stdout, "", log.LstdFlags)

func main() {
	app := cli.NewApp()
	app.Name = "hoko"
	app.Version = "0.0.0"
	app.Commands = []cli.Command{
		{
			Name: "run",
			Action: func(c *cli.Context) {
				Run()
			},
		},
	}
	app.Run(os.Args)
}

func Run() {
	m := martini.Classic()
	m.Use(render.Renderer())
	m.Post("/", func(r render.Render, req *http.Request) {
		b, err := ioutil.ReadAll(req.Body)
		if err != nil {
			r.Error(400)
			return
		}

		//var body WebhookBody
		var body map[string]interface{}
		if err := json.Unmarshal(b, &body); err != nil {
			r.Error(400)
			return
		}

		logger.Printf("%s\n", req.Header)
		logger.Printf("%s\n", body)

		event := strings.ToLower(req.Header.Get("x-github-event"))
		Invoke(HandlerDirpath, event, b)

		r.JSON(204, nil)
	})
	m.Run()
}

func Invoke(dirpath, event string, body []byte) Results {
	hs := Handlers(event)
	rs := make([]*Result, len(hs)+1)
	// primary
	a, e := invoke(dirpath, event, body)
	rs[0] = &Result{a, e}
	// others
	for i, h := range hs {
		a, e := invoke(fmt.Sprintf("%s/%s.d", dirpath, event), h, body)
		rs[i+1] = &Result{a, e}
	}
	return rs
}

type Results []*Result

func (self *Results) Failed() (fail bool, partial bool) {
	fail, partial = false, false
	for _, e := range *self {
		if e.Err != nil {
			fail = true
		} else {
			partial = true
		}
	}
	return
}

type Result struct {
	Body []byte
	Err  *ExitError
}

type WebhookBody struct {
	Action string `json:"action"`
}

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

var HandlerDirpath = "handler"

func invoke(dirpath, filename string, body []byte) ([]byte, *ExitError) {
	path, err := exec.LookPath(fmt.Sprintf("%s/%s", dirpath, filename))
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

func Handlers(event string) []string {
	files, _ := ioutil.ReadDir(fmt.Sprintf("%s/%s.d", HandlerDirpath, event))
	paths := make(sort.StringSlice, len(files))
	for i, e := range files {
		paths[i] = e.Name()
	}
	paths.Sort()
	return paths
}
