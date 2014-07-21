package main

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"os"

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
	m.Post("/serf/query/:name", ExecSerf)
	m.Run()
}

func ExecSerf(r render.Render, req *http.Request, params martini.Params) {
	b, err := ioutil.ReadAll(req.Body)
	if err != nil {
		r.Error(400)
		return
	}

	var body WebhookBody
	if err := json.Unmarshal(b, &body); err != nil {
		r.Error(400)
		return
	}
	body.Event = req.Header.Get("x-github-event")
	payload, err := json.Marshal(&body)
	if err != nil {
		r.Error(500)
		return
	}

	var buf bytes.Buffer
	ui := &mcli.BasicUi{Writer: &buf}
	c := make(chan struct{})
	q := command.QueryCommand{c, ui}
	status := q.Run([]string{"-tag", "webhook=.*", "-format", "json", params["name"], string(payload)})

	log.Printf("status: %v\n", status)
	if status == 1 {
		r.Data(500, buf.Bytes())
	} else {
		r.Data(200, buf.Bytes())
	}
}

type WebhookBody struct {
	Event string `json:"event"`
	Ref   string `json:"ref"`
}
