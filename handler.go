package main

import (
    "encoding/json"
    "fmt"
    "io"
    "net/http"

    "github.com/gorilla/mux"
)

func Ping(w http.ResponseWriter, r *http.Request) {
    fmt.Fprintf(w, "pong")
}

func UpdateKey(w http.ResponseWriter, r *http.Request) {
    m := map[string]string{}
    if err := json.NewDecoder(r.Body).Decode(&m); err != nil {
        w.WriteHeader(http.StatusBadRequest)
        return
    }

    for k, v := range m {
        if err := store.Set(k, v); err != nil {
            w.WriteHeader(http.StatusInternalServerError)
            return
        }
    }
}

func GetKey(w http.ResponseWriter, r *http.Request) {
    key := mux.Vars(r)["key"]
    value, err := store.Get(key)
    if err != nil {
        w.WriteHeader(http.StatusInternalServerError)
        return
    }

    b, err := json.Marshal(map[string]string{key: value})
    if err != nil {
        w.WriteHeader(http.StatusInternalServerError)
        return
    }

    io.WriteString(w, string(b))
}

func DeleteKey(w http.ResponseWriter, r *http.Request) {
    key := mux.Vars(r)["key"]
    if err := store.Delete(key); err != nil {
        w.WriteHeader(http.StatusInternalServerError)
        return
    }
}

func Join(w http.ResponseWriter, r *http.Request) {
    payload := struct {
        Addr string `json:"addr"`
    } {}
    if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
        w.WriteHeader(http.StatusBadRequest)
        return
    }

    if err := store.Join(payload.Addr); err != nil {
        w.WriteHeader(http.StatusInternalServerError)
        return
    }
}
