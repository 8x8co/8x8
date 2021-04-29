package main

import (
	"html/template"
	"log"
	"net/http"

	"github.com/gernest/8x8/pkg/auth"
	"github.com/gernest/8x8/templates"
	"github.com/gorilla/mux"
)

//go:generate protoc -I pkg/models/ --go_out=./pkg/models pkg/models/models.proto
func main() {
	tpl, err := template.ParseFS(templates.Files, "*/*.html")
	if err != nil {
		log.Fatal(err)
	}
	mu := mux.NewRouter()
	mu.HandleFunc("/", func(rw http.ResponseWriter, r *http.Request) {
		err := tpl.ExecuteTemplate(rw, "index.html", map[string]interface{}{})
		if err != nil {
			log.Println(err)
		}
	})
	mu.HandleFunc("/auth/google/login", auth.Login)
	mu.HandleFunc("/auth/google/callback", auth.Callback)
	http.ListenAndServe(":8080", mu)
}
