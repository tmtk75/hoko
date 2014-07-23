package main

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/hex"
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

func save(r render.Render, name string, b []byte) {
	f, err := os.Create(name)
	defer f.Close()
	if err != nil {
		r.Error(500)
		panic(err)
	}
	n, err := f.Write(b)
	log.Printf("save: %v %v", name, n)
}

func ExecSerf(r render.Render, req *http.Request, params martini.Params) {
	b, err := ioutil.ReadAll(req.Body)
	if err != nil {
		log.Printf("ioutil.ReadAll failed: %v", req.Body)
		r.Error(400)
		return
	}

	save(r, "/tmp/reqbody.json", b)

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

	hubsig := req.Header.Get("x-hub-signature")
	save(r, "/tmp/x-github-event.txt", []byte(hubsig))
	log.Printf("X-Hub-Signature: %v", hubsig)
	//expected := hubsig[4+1 : len(hubsig)]

	secret := os.Getenv("SECRET_TOKEN")
	save(r, "/tmp/secret_token.txt", []byte(secret))

	key, _ := hex.DecodeString(secret)
	mac := hmac.New(sha1.New, key)
	mac.Write(b)
	em := mac.Sum(nil)
	log.Printf("expected-MAC: %v", em)

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
