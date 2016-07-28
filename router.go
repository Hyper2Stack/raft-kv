package main

import (
    "net/http"

    "github.com/gorilla/mux"
)

type Route struct {
    Method      string
    Pattern     string
    HandlerFunc http.HandlerFunc
}

var routers = []Route{
    Route{"GET",    "/ping",       Ping     },
    Route{"POST",   "/keys",       UpdateKey},
    Route{"GET",    "/keys/{key}", GetKey   },
    Route{"DELETE", "/keys/{key}", DeleteKey},
    Route{"POST",   "/join",       Join     },
}

func NewRouter() *mux.Router {
    router := mux.NewRouter()
    for _, r := range routers {
        router.Methods(r.Method).Path(r.Pattern).HandlerFunc(r.HandlerFunc)
    }

    return router
}
