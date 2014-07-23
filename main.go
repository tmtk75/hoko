package main

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	"github.com/codegangsta/cli"
	"github.com/go-martini/martini"
	"github.com/hashicorp/serf/command"
	"github.com/hashicorp/serf/command/agent"
	"github.com/martini-contrib/render"
	mcli "github.com/mitchellh/cli"
)

const ENV_SECRET_TOKEN = "SECRET_TOKEN"

var ctx *cli.Context

func main() {
	app := cli.NewApp()
	app.Name = "hoko"
	app.Version = "0.0.0"
	app.Commands = []cli.Command{
		{
			Name:  "run",
			Usage: "Run hoko server",
			Flags: []cli.Flag{
				cli.BoolFlag{"debug,d", "debug mode not to verify x-hub-signature"},
			},
			Action: func(c *cli.Context) {
				ctx = c
				Run()
			},
		},
		{
			Name:  "agent",
			Usage: "Run serf agent with a config file",
			Action: func(c *cli.Context) {
				if len(c.Args()) == 0 {
					cli.ShowCommandHelp(c, "agent")
					os.Exit(1)
				}
				_, err := os.Stat(c.Args()[0])
				if err != nil {
					log.Fatalf("Not found: %v\n", c.Args()[0])
				}

				ui := &mcli.BasicUi{Writer: os.Stdout}
				q := agent.Command{Ui: ui, ShutdownCh: make(chan struct{})}
				q.Run([]string{"--config-file", c.Args()[0]})
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

func ExecSerf(r render.Render, req *http.Request, params martini.Params, w http.ResponseWriter) {
	b, err := ioutil.ReadAll(req.Body)
	if err != nil {
		log.Printf("ioutil.ReadAll failed: %v", req.Body)
		r.Error(400)
		return
	}

	var body WebhookBody
	if err := json.Unmarshal(b, &body); err != nil {
		log.Printf("json.Unmarshal failed: %v", b)
		r.Error(400)
		return
	}

	body.Event = req.Header.Get("x-github-event")
	payload, err := json.Marshal(&body)
	if err != nil {
		log.Printf("json.Marshal failed: %v", body)
		r.Error(500)
		return
	}

	// verify x-hub-signature
	sign := req.Header.Get("x-hub-signature")
	if !ctx.Bool("d") {
		log.Printf("x-hub-signature: %v", sign)
		//log.Printf("SECRET_TOKEN: %v", os.Getenv(ENV_SECRET_TOKEN))
		if len(os.Getenv(ENV_SECRET_TOKEN)) == 0 {
			log.Printf("length of %v is zero", ENV_SECRET_TOKEN)
			r.Data(500, []byte("cannot verify because of missing secret token"))
			return
		}
		if len(sign) < 5 {
			r.Data(400, []byte(fmt.Sprintf("x-hub-signature is too short: %v", sign)))
			return
		}
		expected, _ := hex.DecodeString(string(sign[4+1 : len(sign)])) // 4+1 is to skip `sha1=`
		expected = []byte(expected)

		mac := hmac.New(sha1.New, []byte(os.Getenv(ENV_SECRET_TOKEN)))
		mac.Write(b)
		actual := mac.Sum(nil)

		if !hmac.Equal(actual, expected) {
			log.Printf("%v != %v", actual, expected)
			w.Header().Set("content-type", "text/plain")
			r.Data(400, []byte("x-hub-signature not verified"))
			return
		}
	}

	var buf bytes.Buffer
	ui := &mcli.BasicUi{Writer: &buf}
	c := make(chan struct{})
	q := command.QueryCommand{c, ui}
	status := q.Run([]string{"-tag", "webhook=.*", "-format", "json", params["name"], string(payload)})

	if status == 1 {
		log.Printf("status: %v", status)
		r.Data(500, buf.Bytes())
	} else {
		r.Data(200, buf.Bytes())
	}
}

type WebhookBody struct {
	Event string `json:"event"`
	Ref   string `json:"ref"`
}
