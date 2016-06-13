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
	HOKO_VERSION     = "HOKO_VERSION"
	HOKO_PATH        = "HOKO_PATH"
	HOKO_ORIGIN      = "HOKO_ORIGIN"
)

var flags = []cli.Flag{
	cli.StringFlag{Name: "config-file", Value: "./serf.conf", Usage: "Path to serf.conf", EnvVar: "HOKO_CONFIG_FILE"},
	cli.BoolFlag{Name: "debug,d", Usage: "Debug mode not to verify x-hub-signature", EnvVar: "HOKO_DEBUG"},
	cli.BoolFlag{Name: "ignore-deleted", Usage: "Ignore delivers for deleted", EnvVar: "HOKO_IGNORE_DELETED"},
	cli.BoolFlag{Name: "enable-tag", Usage: "Enable query params as tag"},
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
		Flags: []cli.Flag{
			cli.StringFlag{Name: "github-event", Value: "", Usage: "Header value of x-github-event"},
			cli.StringFlag{Name: "github-delivery", Value: "", Usage: "Header value of x-github-delivery"},
		},
		Description: `args: <url> < <request-body>

   e.g)
     SECRET_TOKEN=... \
     http://localhost:9981/serf/query/hoko?webhook=github < reqbody.json
`,
		Action: func(c *cli.Context) {
			if len(c.Args()) < 1 {
				cli.ShowCommandHelp(c, c.Command.Name)
				os.Exit(1)
			}
			opts := GithubWebhookOptions{
				GithubEvent:    c.String("github-event"),
				GithubDelivery: c.String("github-delivery"),
			}
			PostRequest(c.Args()[0], opts)
		},
	},
}

func main() {
	app := cli.NewApp()
	app.Name = "hoko"
	app.Version = "0.4.0dev"
	app.Usage = "A http server for github webhook with serf agent"
	app.Commands = commands

	os.Setenv("PORT", "9981")
	os.Setenv(HOKO_PATH, os.Args[0])
	os.Setenv(HOKO_VERSION, app.Version)

	app.Run(os.Args)
}

func Run(ctx *cli.Context) {
	//log.Printf("SECRET_TOKEN: %v", os.Getenv(ENV_SECRET_TOKEN))
	if !ctx.Bool("d") && len(os.Getenv(ENV_SECRET_TOKEN)) == 0 {
		log.Fatalf("length of %v is zero", ENV_SECRET_TOKEN)
	}
	m := martini.Classic()
	m.Use(render.Renderer())
	m.Post("/serf/:event/:name", func(r render.Render, req *http.Request, params martini.Params, w http.ResponseWriter) {
		ExecSerf(ctx, r, req, params, w)
	})

	//log.Printf("HOKO_PATH: %v", os.Getenv("HOKO_PATH"))
	cwd, _ := os.Getwd()
	log.Printf("version: %v", ctx.App.Version)
	log.Printf("cwd: %v", cwd)
	m.Run()
}

type SerfCmd interface {
	Run(args []string) int
}

func unmarshalGithub(b []byte, ctx *cli.Context, r render.Render, req *http.Request, w http.ResponseWriter) interface{} {
	if !ctx.Bool("d") {
		sign := req.Header.Get("x-hub-signature")
		err := verify(sign, b, r, w)
		if err != nil {
			return nil
		}
	}

	var body WebhookBody
	if err := json.Unmarshal(b, &body); err != nil {
		log.Printf("json.Unmarshal failed: %v", string(b))
		r.Error(400)
		return nil
	}

	body.Event = req.Header.Get("x-github-event")
	body.Delivery = req.Header.Get("x-github-delivery")
	if ctx.Bool("ignore-deleted") && body.Deleted {
		log.Printf("ignore deleted")
		log.Printf("x-github-event: %v", body.Event)
		log.Printf("x-github-delivery: %v", body.Delivery)
		r.Header().Add("content-type", "text/plain")
		r.Data(200, []byte("ignore deleted\n"))
		return nil
	}

	return &body
}

func unmarshalBitbucket(payload []byte, ctx *cli.Context, r render.Render, req *http.Request, w http.ResponseWriter) interface{} {
	if !ctx.Bool("d") {
		token := os.Getenv(ENV_SECRET_TOKEN)
		secret := req.URL.Query().Get("secret")
		if token != secret {
			log.Printf("secret token didn't match")
			r.Error(400)
			return nil
		}
	}

	var body BitbucketWebhookBody
	if err := json.Unmarshal(payload, &body); err != nil {
		log.Printf("json.Unmarshal failed: %v", payload)
		r.Error(400)
		return nil
	}

	if _, ok := body.Push["changes"]; !ok {
		log.Printf("push.changes is missing: %v", body)
		r.Error(400)
		return nil
	}
	if len(body.Push["changes"]) == 0 {
		log.Printf("push.changes is empty: %v", body)
		r.Error(400)
		return nil
	}

	var wb WebhookBody
	ch := body.Push["changes"][0]
	switch ch.New.Type {
	case "branch":
		wb.Ref = "refs/heads/" + ch.New.Name
	case "tag":
		wb.Ref = "refs/tags/" + ch.New.Name
	default:
		log.Printf("unknown type: %v", ch.New.Type)
		r.Error(400)
		return nil
	}
	wb.Repository.Name = body.Repository.Name
	wb.Repository.Owner.Name = body.Repository.Owner.Username
	wb.Pusher.Name = body.Actor.Username
	wb.Event = req.Header.Get("X-Event-Key")
	wb.Created = ch.Created
	wb.Deleted = ch.Truncated
	wb.Delivery = "n/a"
	wb.After = ch.New.Target.Hash
	wb.Before = ch.Old.Target.Hash
	log.Printf("X-Request-UUID: %v", req.Header.Get("X-Request-UUID"))
	log.Printf("X-Hook-UUID: %v", req.Header.Get("X-Hook-UUID"))
	//log.Printf("%v", body)

	if ctx.Bool("ignore-deleted") && ch.Truncated {
		log.Printf("ignore truncated")
		r.Header().Add("content-type", "text/plain")
		r.Data(200, []byte("ignore truncated\n"))
		return nil
	}

	return &wb
}

func ExecSerf(ctx *cli.Context, r render.Render, req *http.Request, params martini.Params, w http.ResponseWriter) {
	b, err := ioutil.ReadAll(req.Body)
	if err != nil {
		log.Printf("ioutil.ReadAll failed: %v", req.Body)
		r.Error(400)
		return
	}

	for k, v := range req.Header {
		log.Printf("%v: %v", k, v)
	}

	var v interface{}
	origin := req.URL.Query().Get("origin")
	log.Printf("origin: %v\n", origin)
	if origin == "bitbucket" {
		os.Setenv(HOKO_ORIGIN, "bitbucket")
		v = unmarshalBitbucket(b, ctx, r, req, w)
	} else {
		os.Setenv(HOKO_ORIGIN, "github")
		v = unmarshalGithub(b, ctx, r, req, w)
	}
	if v == nil {
		return
	}

	payload, err := json.Marshal(v)
	if err != nil {
		log.Printf("json.Marshal failed: %v", v)
		r.Error(500)
		return
	}

	log.Printf("payload-size: %v", len(payload))

	var buf bytes.Buffer
	ui := &mcli.BasicUi{Writer: &buf}
	var cmd SerfCmd
	var args []string

	switch params["event"] {
	case "query":
		c := make(chan struct{})
		cmd = &command.QueryCommand{c, ui}
		args = []string{"-format", "json"}
		if ctx.Bool("enable-tag") {
			args = append(args, buildTagOptions(req.URL.Query())...)
		}
		args = append(args, []string{params["name"], string(payload)}...)
	case "event":
		cmd = &command.EventCommand{ui}
		args = []string{params["name"], string(payload)}
	default:
		log.Printf("[WARN] unknown %v", params["event"])
		r.Error(400)
		return
	}

	log.Printf("args: %v", args)
	status := cmd.Run(args)
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

func buildTagOptions(params map[string][]string) []string {
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
	Repository struct {
		Name  string `json:"name"`
		Owner struct {
			Name string `json:"name"`
		} `json:"owner"`
	} `json:"repository"`
	Event    string `json:"event"`
	Delivery string `json:"delivery"`
	Ref      string `json:"ref"`
	After    string `json:"after"`
	Before   string `json:"before"`
	Created  bool   `json:"created"`
	Deleted  bool   `json:"deleted"`
	//Head_commit map[string]interface{} `json:"head_commit,omitempty"`
	Pusher struct {
		Name string `json:"name"`
	} `json:"pusher,omitempty"`
}

//https://confluence.atlassian.com/bitbucket/manage-webhooks-735643732.html
type BitbucketWebhookBody struct {
	Actor struct {
		Username string `json:"username"`
	} `json:"actor"`
	Repository struct {
		Name  string `json:"name"`
		Owner struct {
			Type     string `json:type`
			Username string `json:username`
		} `json:"owner"`
	} `json:"repository"`
	Push map[string][]struct { // changes
		New struct {
			Name   string `json:"name"`
			Type   string `json:"type"`
			Target struct {
				Hash string `json:"hash"`
			} `json:"target"`
		} `json:"new"`
		Old struct {
			Target struct {
				Hash string `json:"hash"`
			} `json:"target"`
		} `json:"old"`
		Created   bool `json:"created"`
		Truncated bool `json:"truncated"`
	} `json:"push"`
}
