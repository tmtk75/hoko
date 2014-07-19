package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"sort"
	"strings"

	"./cmd"
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
	Err  *cmd.ExitError
}

type WebhookBody struct {
	Action string `json:"action"`
}

var HandlerDirpath = "handler"

func invoke(dirpath, filename string, body []byte) ([]byte, *cmd.ExitError) {
	return cmd.Invoke(dirpath, filename, body)
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
