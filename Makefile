#
.DEFAULT_GOAL := help

run:
	SECRET_TOKEN=`cat test/secret_token.txt` \
		go run main.go client.go run --ignore-deleted

tags:
	serf tags -set webhook=push

secret:
	@cat test/secret_token.txt

gh-sample:
	curl -v -XPOST \
	  -H"x-hub-signature: `cat test/x-hub-signature.txt`" \
	  localhost:9981/serf/query/hoko \
	  -d @test/webhook-body.json 

bb-sample:
	curl -v -XPOST \
	  -H"content-type: application/x-www-form-urlencoded" \
	  "localhost:9981/serf/event/bitbucket?origin=bitbucket&secret=`cat test/secret_token.txt`" \
	  -d @test/bitbucket-webhook-body

bb-wh-branch:
	curl -v -XPOST \
	  -H"X-Request-UUID: 43ac8346-2f1f-450d-9dcb-d2e9c85e04b4" \
	  -H"X-Hook-UUID: c47c0ee9-b46f-462f-9e80-1f7d8135e199" \
	  -H"X-Event-Key: repo:push" \
	  -H"content-type: application/json" \
	  "localhost:9981/serf/event/bitbucket?origin=bitbucket&secret=`cat test/secret_token.txt`" \
	  -d @test/webhook-foobar-relaese-0.9.json

bb-wh-tag:
	curl -v -XPOST \
	  -H"X-Request-UUID: 43ac8346-2f1f-450d-9dcb-d2e9c85e04b4" \
	  -H"X-Hook-UUID: c47c0ee9-b46f-462f-9e80-1f7d8135e199" \
	  -H"X-Event-Key: repo:push" \
	  -H"content-type: application/json" \
	  "localhost:9981/serf/event/bitbucket?origin=bitbucket&secret=`cat test/secret_token.txt`" \
	  -d @test/webhook-foobar-v0.9.0.json

hup:
	kill -1 `ps axu | egrep 'serf agent' | egrep -v 'egrep serf agent' | awk '{print $$2}'`

post:
	echo '{"event":"custom"}' | \
	  SECRET_TOKEN=`cat test/secret_token.txt` go run \
	  main.go client.go post

query:
	curl -v -XPOST localhost:9981/serf/query/hoko -d '{"ref":"fizbiz"}'

event:
	curl -v -XPOST localhost:9981/serf/event/webhook -d '{"ref":"foobar"}'

#
#
#
VERSION := $(shell git describe --tags)
build:
	for os in linux darwin; do \
	  GOOS=$$os GOARCH=amd64 go build \
	    -o pkg/hoko_$${os}_amd64 \
	    -ldflags "-X main.Version=$(VERSION)"; \
	done

shasum:
	shasum -a 256 pkg/dist/hoko_linux_amd64.zip

install: main.go client.go
	go install -ldflags "-X main.Version=$(VERSION)"

hoko: main.go client.go
	go build

release: hoko
	cp -f pkg/dist/hoko_linux_amd64.zip pkg/dist/hoko-$(VERSION)_linux_amd64.zip 
	ghr -u tmtk75 v$(VERSION) pkg/dist/hoko-$(VERSION)_linux_amd64.zip

zip: pkg/dist/hoko_linux_amd64.zip pkg/dist/hoko_darwin_amd64.zip

pkg/dist/hoko_linux_amd64.zip: pkg/dist/hoko_linux_amd64
	cd pkg/dist; mv hoko_linux_amd64 hoko; zip hoko_linux_amd64.zip hoko

pkg/dist/hoko_darwin_amd64.zip: pkg/dist/hoko_darwin_amd64
	cd pkg/dist; mv hoko_darwin_amd64 hoko; zip hoko_darwin_amd64.zip hoko

clean:
	rm -f hoko

distclean: clean
	rm -rf pkg

.PHONY: help
help:
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) \
	| awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-16s\033[0m %s\n", $$1, $$2}'
