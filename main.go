package main

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/codegangsta/cli"
	"github.com/go-martini/martini"
	"github.com/hashicorp/serf/command"
	"github.com/hashicorp/serf/command/agent"
	"github.com/martini-contrib/render"
	mcli "github.com/mitchellh/cli"
)

const (
	ENV_SECRET_TOKEN = "SECRET_TOKEN"
	CONFIG_PATH      = "CONFIG_PATH"
)

var flags = []cli.Flag{
	cli.StringFlag{Name: "config-file", Value: "./serf.conf", Usage: "Path to serf.conf", EnvVar: "HOKO_CONFIG_FILE"},
	cli.BoolFlag{Name: "debug,d", Usage: "Debug mode not to verify x-hub-signature"},
}

var commands = []cli.Command{
	{
		Name:  "run",
		Usage: "Run hoko server with serf agent",
		Flags: flags,
		Action: func(c *cli.Context) {
			configfile := c.String("config-file")
			if len(os.Getenv(CONFIG_PATH)) > 0 {
				configfile = os.Getenv(CONFIG_PATH)
			}
			_, err := os.Stat(configfile)
			if err != nil {
				dir, _ := os.Getwd()
				log.Fatalf("Not found: %v\n", filepath.Join(dir, configfile))
			}

			go Run(c)

			ui := &mcli.BasicUi{Writer: os.Stdout}
			q := agent.Command{Ui: ui, ShutdownCh: make(chan struct{})}
			q.Run([]string{"--config-file", configfile})
		},
	},
	{
		Name:  "server",
		Usage: "Run hoko server alone",
		Flags: flags,
		Action: func(c *cli.Context) {
			Run(c)
		},
	},
	{
		Name:  "post",
		Usage: "Post a request",
		Flags: flags,
		Action: func(c *cli.Context) {
			PostRequest()
		},
	},
}

func main() {
	app := cli.NewApp()
	app.Name = "hoko"
	app.Version = "0.1.1"
	app.Usage = "A http server for github webhook with serf agent"
	app.Commands = commands
	os.Setenv("PORT", "9981")
	app.Run(os.Args)
}

func Run(ctx *cli.Context) {
	//log.Printf("SECRET_TOKEN: %v", os.Getenv(ENV_SECRET_TOKEN))
	if !ctx.Bool("d") && len(os.Getenv(ENV_SECRET_TOKEN)) == 0 {
		log.Fatalf("length of %v is zero", ENV_SECRET_TOKEN)
	}
	m := martini.Classic()
	m.Use(render.Renderer())
	m.Post("/serf/query/:name", func(r render.Render, req *http.Request, params martini.Params, w http.ResponseWriter) {
		ExecSerf(ctx, r, req, params, w)
	})

	m.Run()
}

func ExecSerf(ctx *cli.Context, r render.Render, req *http.Request, params martini.Params, w http.ResponseWriter) {
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

	if !ctx.Bool("d") {
		sign := req.Header.Get("x-hub-signature")
		err := verify(sign, b, r, w)
		if err != nil {
			return
		}
	}

	var buf bytes.Buffer
	ui := &mcli.BasicUi{Writer: &buf}
	c := make(chan struct{})
	q := command.QueryCommand{c, ui}
	args := []string{"-tag", "webhook=.*", "-format", "json"}
	args = append(args, buildArgs(req.URL.Query())...)
	args = append(args, []string{params["name"], string(payload)}...)
	status := q.Run(args)

	if status == 1 {
		log.Printf("status: %v", status)
		r.Data(500, buf.Bytes())
		return
	}

	r.Data(200, buf.Bytes())
}

func verify(sign string, b []byte, r render.Render, w http.ResponseWriter) error {
	log.Printf("x-hub-signature: %v", sign)
	if len(sign) < 5 {
		r.Data(400, []byte(fmt.Sprintf("x-hub-signature is too short: %v", sign)))
		return errors.New("")
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
		return errors.New("")
	}

	return nil
}

func buildArgs(params map[string][]string) []string {
	a := make([]string, len(params)*2)
	i := 0
	for k, v := range params {
		a[i*2] = "-tag"
		a[i*2+1] = fmt.Sprintf("%v=%v", k, v[0])
		i += 1
	}
	return a
}

type WebhookBody struct {
	Event       string                 `json:"event"`
	Ref         string                 `json:"ref"`
	After       string                 `json:"after"`
	Before      string                 `json:"before"`
	Head_commit map[string]interface{} `json:"head_commit,omitempty"`
	Pusher      map[string]interface{} `json:"pusher,omitempty"`
}
