# raft-kv

raft-kv is a sample use of the [Hashicorp Raft implementation](https://github.com/hashicorp/raft).

## Setup single node

```bash
mkdir -p /tmp/gopath
export GOPATH=/tmp/gopath
go get github.com/hyper2stack/raft-kv
```

Run single node:
```bash
$GOPATH/bin/raft-kv -httpaddr :8000 -raftaddr :9000 /tmp/raft0
```

Set a key and read its value:
```bash
curl -XPOST localhost:8000/keys -d '{"foo": "bar"}'
curl -XGET  localhost:8000/keys/foo
```

## Setup cluster

Let's bring up 2 more nodes, so we have a 3-node cluster. That way we can tolerate the failure of 1 node:

```bash
$GOPATH/bin/raft-kv -httpaddr :8001 -raftaddr :9001 -join :8000 /tmp/raft1
$GOPATH/bin/raft-kv -httpaddr :8002 -raftaddr :9002 -join :8000 /tmp/raft2
```

This tells each new node to join the existing node. Once joined, each node now knows about the key:
```bash
curl -XGET localhost:8000/keys/foo
curl -XGET localhost:8001/keys/foo
curl -XGET localhost:8002/keys/foo
```
