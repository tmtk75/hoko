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
		Invoke(event, b)

		r.JSON(204, nil)
	})
	m.Run()
}

type WebhookBody struct {
	Action string `json:"action"`
}

type ExitError struct {
	Err        error
	ExitStatus int
}

func (self *ExitError) Error() string {
	return self.Err.Error()
}

func Invoke(event string, body []byte) ([]byte, *ExitError) {
	return invoke("handler", event, body)
}

func invoke(dirpath, event string, body []byte) ([]byte, *ExitError) {
	path, err := exec.LookPath(fmt.Sprintf("%s/%s", dirpath, event))
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
	if status, ok := cmd.ProcessState.Sys().(syscall.WaitStatus); ok {
		log.Printf("Exit Status: %d", status.ExitStatus())
	}

	return out, nil
}
