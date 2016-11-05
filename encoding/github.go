package encoding

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/codegangsta/cli"
	"github.com/martini-contrib/render"
)

const (
	ENV_SECRET_TOKEN = "SECRET_TOKEN"
)

func UnmarshalGithub(b []byte, ctx *cli.Context, r render.Render, req *http.Request, w http.ResponseWriter) interface{} {
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
