package encoding

import (
	"encoding/json"
	"log"
	"net/http"
	"os"

	"github.com/codegangsta/cli"
	"github.com/martini-contrib/render"
)

func UnmarshalBitbucket(payload []byte, ctx *cli.Context, r render.Render, req *http.Request, w http.ResponseWriter) interface{} {
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
	wb.Delivery = req.Header.Get("X-Hook-UUID")
	wb.After = ch.New.Target.Hash
	wb.Before = ch.Old.Target.Hash
	log.Printf("X-Request-UUID: %v", req.Header.Get("X-Request-UUID"))
	log.Printf("X-Hook-UUID: %v", req.Header.Get("X-Hook-UUID"))
	//log.Printf("%v", body)

	return &wb
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
