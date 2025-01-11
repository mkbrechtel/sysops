package web

import (
    "fmt"
    "net/http"
    "patterns.mkbrechtel.dev/docs"
)

func Webserver() {
    mux := http.NewServeMux()
    fileServer := http.FileServer(http.FS(docs.ContentFiles))
    mux.Handle("/", fileServer)

    fmt.Println("Starting server on :4780")
    err := http.ListenAndServe(":4780", mux)
    if err != nil {
        fmt.Printf("Server error: %v\n", err)
    }
}
