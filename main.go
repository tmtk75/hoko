package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/codegangsta/cli"
	"github.com/go-martini/martini"
	"github.com/hashicorp/serf/cmd/serf/command"
	"github.com/hashicorp/serf/cmd/serf/command/agent"
	"github.com/martini-contrib/render"

	mcli "github.com/mitchellh/cli"

	"github.com/tmtk75/hoko/encoding"
)

const (
	CONFIG_PATH  = "CONFIG_PATH"
	HOKO_VERSION = "HOKO_VERSION"
	HOKO_PATH    = "HOKO_PATH"
	HOKO_ORIGIN  = "HOKO_ORIGIN"
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

var Version = ""

func main() {
	app := cli.NewApp()
	app.Name = "hoko"
	app.Version = Version
	app.Usage = "A http server for github webhook with serf agent"
	app.Commands = commands

	os.Setenv("PORT", "9981")
	os.Setenv(HOKO_PATH, os.Args[0])
	os.Setenv(HOKO_VERSION, app.Version)

	app.Run(os.Args)
}

func Run(ctx *cli.Context) {
	//log.Printf("SECRET_TOKEN: %v", os.Getenv(ENV_SECRET_TOKEN))
	if !ctx.Bool("d") && len(os.Getenv(encoding.ENV_SECRET_TOKEN)) == 0 {
		log.Fatalf("length of %v is zero", encoding.ENV_SECRET_TOKEN)
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
		v = encoding.UnmarshalBitbucket(b, ctx, r, req, w)
	} else {
		os.Setenv(HOKO_ORIGIN, "github")
		v = encoding.UnmarshalGithub(b, ctx, r, req, w)
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
