package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/codegangsta/cli"
	"github.com/go-martini/martini"
	"github.com/hashicorp/serf/command"
	"github.com/martini-contrib/render"
	mcli "github.com/mitchellh/cli"
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
	m.Post("/", ExecHandlers)
	m.Post("/serf/:name", ExecSerf)
	m.Run()
}

func ExecSerf(r render.Render, req *http.Request, params martini.Params) {
	b, err := ioutil.ReadAll(req.Body)
	if err != nil {
		r.Error(400)
		return
	}

	ui := &mcli.BasicUi{Writer: os.Stdout}
	c := make(chan struct{})
	q := command.QueryCommand{c, ui}
	q.Run([]string{params["name"], string(b)})

	r.JSON(204, nil)
}

func ExecHandlers(r render.Render, req *http.Request) {
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
	rs := Invoke(HandlerDirpath, event, b)

	if b, p := rs.Failed(); b {
		if p {
			r.Data(207, []byte(rs.Response("plain/text")))
		} else {
			r.JSON(500, nil)
		}
	} else {
		r.JSON(204, nil)
	}
}

func (rs *Results) Response(mimetype string) string {
	switch mimetype {
	case "plain/text":
		a := ""
		for _, r := range *rs {
			if r.Err != nil {
				a += "500\t"
			} else {
				a += "200\t"
			}
			a += r.Path + "\n"
		}
		return a
	}
	log.Printf("Unsupported: %v", mimetype)
	return ""
}

type Dirpath string

func (d Dirpath) Path(child ...string) string {
	a := make([]string, len(child)+1)
	a[0] = string(d)
	for i, e := range child {
		a[i+1] = e
	}
	return filepath.Join(a...)
}

func Invoke(dirpath Dirpath, event string, body []byte) Results {
	hs := Handlers(event)
	rs := make([]*Result, len(hs)+1)
	// primary
	p := dirpath.Path(event)
	a, e := invoke(p, body)
	rs[0] = &Result{a, e, p}
	// others
	for i, h := range hs {
		p := dirpath.Path(fmt.Sprintf("%s.d", event), h)
		a, e := invoke(p, body)
		rs[i+1] = &Result{a, e, p}
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
	Path string
}

type WebhookBody struct {
	Action string `json:"action"`
}

var HandlerDirpath Dirpath = "handler"

func invoke(path string, body []byte) ([]byte, *ExitError) {
	return Exec(path, body)
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
