agent:
	serf agent -config-file serf-config.json 

tags:
	serf tags -set webhook=push

hup:
	kill -1 `ps axu | egrep 'serf agent' | egrep -v 'egrep serf agent' | awk '{print $$2}'`

sample:
	curl -v -XPOST -H"x-github-event: push" \
	  localhost:3000/serf/query/github -d '{"action":"push"}'

sample-sh:
	curl -v -XPOST -H"x-github-event: push" \
	  localhost:3000/ -d '{"action":"push"}'

#
# ansible
#
.py:
	virtualenv .py

#.py/bin/ansible:
#	. .py/bin/activate
#	pip install .py
