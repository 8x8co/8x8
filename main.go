package main

import (
	"context"
	"errors"
	"html/template"
	"log"
	"net/http"
	"os"

	"github.com/gernest/8x8/pkg/auth"
	"github.com/gernest/8x8/pkg/xl"
	"github.com/gernest/8x8/templates"
	"github.com/gorilla/mux"
	"github.com/urfave/cli"
	"go.uber.org/zap"
	"golang.org/x/crypto/acme/autocert"
)

const host = "8x8.co.tz"

//go:generate protoc -I pkg/models/ --go_out=./pkg/models pkg/models/models.proto
//go:generate protoc -I pkg/models/ --go_out=./pkg/models pkg/models/checkers.proto

func main() {
	a := cli.NewApp()
	a.Name = "8x8 realtime checkers game"
	a.Action = run
	if err := a.Run(os.Args); err != nil {
		if !errors.Is(err, context.Canceled) {
			xl.Error(err, "failed to run the app")
		}
		os.Exit(1)
	}
}

func run(_ *cli.Context) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
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
	svr := &http.Server{
		Addr: ":80",
		Handler: http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
			http.Redirect(rw, r, "https://"+host, http.StatusFound)
		}),
	}
	defer svr.Shutdown(context.Background())

	go func() {
		xl.Info("starting http service", zap.String("addr", svr.Addr))
		err := svr.ListenAndServe()
		if err != nil {
			xl.Error(err, "exit http server")
		}
		cancel()
	}()
	stls := &http.Server{
		Handler: mu,
	}
	go func() {
		ls := autocert.NewListener(host)
		xl.Info("starting http service", zap.String("addr", ls.Addr().String()))
		err := stls.Serve(ls)
		if err != nil {
			xl.Error(err, "exit https server")
		}
		cancel()
	}()
	<-ctx.Done()
	return ctx.Err()
}
