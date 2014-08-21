# hoko

A HTTP server for github webhook with serf agent, which provides HTTP interface to serf agent for query. The main purpose is to propagate webhook events to serf cluster. Any hosts can receive webhook events anywhere if the hosts join the serf cluster.

<img src='http://tmtk75.github.com.s3.amazonaws.com/hoko/demo.gif'/>


## Getting Started

Download a configuration file, <https://raw.githubusercontent.com/tmtk75/hoko/master/serf.conf> and save as `serf.conf`.

Download a file, <https://raw.githubusercontent.com/tmtk75/hoko/master/handler/echo.sh> and save as `./handler/echo.sh`.

Download a binary, unzip it.

* [Linux amd64 v0.1.0](https://github.com/tmtk75/hoko/releases/download/v0.1.0/hoko_linux_amd64.zip)
* [Darwin amd64 v0.1.0](https://github.com/tmtk75/hoko/releases/download/v0.1.0/hoko_darwin_amd64.zip)

Then the expected layout is:

```
./-- hoko
 |-- serf.conf
 `-- handler
      `-- echo.sh
```

OK, Run it. It starts listening at 3000.

```
$ ./hoko run -d
```

You can see some outputs in the terminal hoko runs like this, which is mainly serf agent's log.

```
==> Starting Serf agent...
==> Starting Serf agent RPC...
...
    2014/07/27 10:18:04 [INFO] agent: Received event: member-join
```

In another terminal, execute curl like this and you'll get a response in JSON.

```
$ curl localhost:3000/serf/query/hoko -d '{}'
{
  "Acks": [
    "hostname"
  ],
  "Responses":
    "hostname": "\n------- ./handler/echo.sh\nSERF_EVENT: query\nSERF_QUERY_NAME: hoko\nSERF_SELF_NAME: hostname\nSERF_TAG_WEBHOOK: .*\npayload: \"event\":\"\",\"ref\":\"\",\"after\":\"\",\"before\":\"\"}\n\n\n"
  }
}
```

OK, hoko runs perfectly.


## How it works at serf level

hoko invokes query command when it receives POST request.

```
curl -XPOST localhost:3000/serf/query/hoko -d '{}'
```

For example, this request makes hoko to invoke the next command.
	
```
$ serf query -tag webhook=.* hoko "{}"
```

In order to handle this query event, it needs a setting about tag and serf event-handler option like

```
-tag webhook=<something> -event-handler=query:hoko=<command>
```

The previous response is built by `handler/echo.sh` at default via serf query. A [serf configuration file](http://www.serfdom.io/docs/agent/options.html), `serf.conf` defines it to handle events.

```
{
  "tags": {
    "webhook": ".*"
  },
  "event_handlers": [
    "query:hoko=./handler/echo.sh"
  ]
}
```

hoko responds stdout from serf agent as HTTP response.

Next diagram briefly describes typical flow of hoko.

```
 [curl]      [hoko]     
   |            |
   |            |------>> [serf agent]  (This agent is launched by hoko)
   |    POST    |              |
   |----------->|              |
   |            |  validate    |
   |            |-----,        |
   |            |     |        |
   |            |<----'        |
   |            |  query hoko  |
   |            |-----,        |
   |            |     |------->|
   |            |     |        |  exec echo.sh
   |            |     |        |-----, 
   |            |     |        |     |
   |            |     |        |<----'         [other agents]
   |            |     |        | propagate           |
   |            |     |        |----------- - - - - >|
   |            |     |        |                     |
   |            |     |        | responses
   |            |     |        |<----------
   |            |     |<-------'
   |            |<----'
   |  response  |
   |<-----------|
   |            |
```


# Github Webhook Configration

Configure your webhook rerfering [Webhooks](https://developer.github.com/webhooks/).

For example,

* Payload URL: http://54.92.95.64:3000/serf/query/hoko
* Content type: `application/json`
* Secret: Your secret key

Regarding the last part of path `hoko`, it's handled as a name of serf query. It needs serf agent option `-event-handler=query:hoko=ls` to receive the event. See [event-handlers.html](http://www.serfdom.io/docs/agent/event-handlers.html) of serf document.

Regarding `Secret`, initially, please see <https://developer.github.com/webhooks/securing/>. You had better to set a secret value in both github and your host to receive webhook.

Then you run hoko without `-d` option like,

```
$ SECRET_TOKEN=$(cat secret.txt) ./hoko run
```

`-d` option in Getting Started is debug option to suppress verification `x-hub-signature` header.

If secret token doesn't match, hoko responds HTTP status 400.


# A use-case, receiving webhook events at a different host

Here is an overview for this section.

```
       github
         |
         v
       :3000
     hoko-master                           hoko-slave
     54.92.127.130                         54.92.118.98
                     :7946 <----> :7946
       serf agent                            another
       in hoko                               serf agent
                     hoko[webhook] event
                    -------------------->    query:hoko="cat >> payloads"
```

Let's say you have two hosts like

```
hoko-master    54.92.127.130
hoko-slave     54.92.118.98
```

Open a few ports, 22, 3000, 7946


In 54.92.127.130, run hoko with a secret token which `secret.txt has`.

```
$ SECRET_TOKEN=$(cat ./secrettxt) ./hoko_linux_amd64 run
```

In 54.92.118.98, Run a serf agent joining 54.92.127.130.

```
$ ./serf agent \
    -tag webhook=foobar \
    -event-handler=query:hoko="cat >> payloads" \
    -join 54.92.127.130
```

In the setting of webhook, Set Payload URL: `http://54.92.127.130:3000/serf/query/hoko`

OK, probably you are ready, just in case, type next in 54.92.118.98:

```
$ ./serf members
hoko-master  54.92.127.130:7946  alive  webhook=.*
hoko-slave   54.92.118.98:7946   alive  webhook=foobar
```

Let's invoke a webhook event! Create a remote branch and delete it.

```
$ git push origin master:foobar
$ git push origin :foobar
```

In 54.92.118.98, see `payloads` with cat.

```
$ cat payloads
{"event":"push","ref":"refs/heads/foobar","after":"0000000000000000000000000000000000000000","before":"7d422ef2df2059b996566f51f2532c5b50cb3905","pusher":{"email":"whoever...@gmail.com","name":"tmtk75"}}
{"event":"delete","ref":"foobar","after":"","before":""}
```

It means the webhook event was propagated and the serf agent received it.

## Contribution

1. Fork it (<http://github.com/tmtk75/hoko>)
1. Create your feature branch (git checkout -b my-new-feature)
1. Commit your changes (git commit -am 'Add some feature')
1. Push to the branch (git push origin my-new-feature)
1. Create new Pull Request

## Credits

serf: <https://github.com/hashicorp/serf>

## License

[MIT License](http://opensource.org/licenses/MIT)

