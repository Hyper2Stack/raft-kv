package main

import (
    "bytes"
    "encoding/json"
    "flag"
    "fmt"
    "log"
    "net/http"
    "os"
)

var httpAddr string
var raftAddr string
var joinAddr string

func init() {
    flag.StringVar(&httpAddr, "httpaddr", ":8000", "http bind address")
    flag.StringVar(&raftAddr, "raftaddr", ":9000", "raft bind address")
    flag.StringVar(&joinAddr, "join", "", "cluster master node, can be omitted")
    flag.Usage = func() {
        fmt.Fprintf(os.Stderr, "Usage: %s [options] <raft-data-dir> \n", os.Args[0])
        flag.PrintDefaults()
    }
}

func join(joinAddr, raftAddr string) error {
    b, err := json.Marshal(map[string]string{"addr": raftAddr})
    if err != nil {
        return err
    }

    url := fmt.Sprintf("http://%s/join", joinAddr)
    res, err := http.Post(url, "application-type/json", bytes.NewReader(b))
    if err != nil {
        return err
    }

    if res.StatusCode/100 != 2 {
        return fmt.Errorf("status code is %d", res.StatusCode)
    }

    return nil
}

func main() {
    flag.Parse()

    if flag.NArg() != 1 {
        flag.Usage()
        os.Exit(1)
    }

    raftDir := flag.Arg(0)
    singleton := joinAddr == ""
    os.MkdirAll(raftDir, 0700)

    if err := store.Open(raftDir, raftAddr, singleton); err != nil {
        log.Fatalf("Failed to open store, %v\n", err)
    }

    if !singleton {
        if err := join(joinAddr, raftAddr); err != nil {
            log.Fatalf("Failed to join node %s, %v\n", joinAddr, err)
        }
    }

    log.Printf("Starting http server on %s ...\n", httpAddr)
    http.ListenAndServe(fmt.Sprintf("%s", httpAddr), NewRouter())
}
