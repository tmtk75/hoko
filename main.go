package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"

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
		Invoke(event, b)

		r.JSON(204, nil)
	})
	m.Run()
}

type WebhookBody struct {
	Action string `json:"action"`
}

func Invoke(event string, body []byte) []byte {
	path, err := exec.LookPath(fmt.Sprintf("event/%s", event))
	if err != nil {
		panic(err)
	}
	cmd := exec.Command(path)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		panic(err)
	}
	stdin.Write(body)
	stdin.Close()
	out, err := cmd.Output()
	if err != nil {
		panic(err)
	}
	return out
}
