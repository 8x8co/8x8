package main

import (
	"bytes"
	"context"
	"errors"
	"html/template"
	"io/ioutil"
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
	a.Commands = cli.Commands{
		{
			Name:  "install",
			Usage: "installs systemd unit files and sets up 8x8",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "workdir,w",
					Usage: "working directory",
					Value: "/opt/8x8",
				},
				cli.StringFlag{
					Name:  "data,d",
					Usage: "database directory",
					Value: "/data/8x8",
				},
			},
			Action: install,
		},
	}
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
		return err
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

func install(ctx *cli.Context) error {
	tpl, err := template.ParseFS(templates.Installation, "*/*")
	if err != nil {
		return err
	}
	w := ctx.String("workdir")
	xl.Info("setting working directory", zap.String("w", w))
	_, err = os.Stat(w)
	if err != nil {
		if !os.IsNotExist(err) {
			return err
		}
		if err := os.MkdirAll(w, 0755); err != nil {
			xl.Error(err, "failed to create working direcory")
			return context.Canceled
		}
	}
	d := ctx.String("data")
	xl.Info("setting data directory", zap.String("d", d))
	_, err = os.Stat(d)
	if err != nil {
		if !os.IsNotExist(err) {
			return err
		}
		if err := os.MkdirAll(w, 0755); err != nil {
			xl.Error(err, "failed to create data direcory")
			return context.Canceled
		}
	}
	xl.Info("Setting up systemd")
	var buf bytes.Buffer
	err = tpl.ExecuteTemplate(&buf, "8x8.service", map[string]interface{}{
		"WorkingDirectory": w,
		"Data":             d,
	})
	if err != nil {
		return err
	}
	path := "/etc/systemd/system/8x8.service"
	xl.Info("writing systemd service file", zap.String("path", path))
	err = ioutil.WriteFile(path, buf.Bytes(), 0600)
	if err != nil {
		return err
	}
	xl.Info("systemctl  enable 8x8.service # to start at boot")
	xl.Info("systemctl  start 8x8.service # to star the service")
	return nil
}
