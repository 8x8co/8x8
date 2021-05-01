package main

import (
	"html/template"
	"log"
	"net/http"

	"github.com/gernest/8x8/pkg/auth"
	"github.com/gernest/8x8/templates"
	"github.com/gorilla/mux"
	"golang.org/x/crypto/acme/autocert"
)

const host = "8x8.co.tz"
const host1 = "8x8.co"

//go:generate protoc -I pkg/models/ --go_out=./pkg/models pkg/models/models.proto
//go:generate protoc -I pkg/models/ --go_out=./pkg/models pkg/models/checkers.proto
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
	// redirect all http traffick to https
	go http.ListenAndServe(":80", http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		http.Redirect(rw, r, "https://"+host, http.StatusFound)
	}))
	log.Fatal(http.Serve(autocert.NewListener(host, host1), mu))
}
