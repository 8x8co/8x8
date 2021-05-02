package main

import (
	"bytes"
	"context"
	"errors"
	"html/template"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"

	"github.com/caddyserver/certmagic"
	"github.com/gernest/8x8/pkg/auth"
	"github.com/gernest/8x8/pkg/mw"
	"github.com/gernest/8x8/pkg/xl"
	"github.com/gernest/8x8/templates"
	"github.com/gorilla/mux"
	"github.com/urfave/cli"
	"go.uber.org/zap"
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
	m := mw.New()
	mu := mux.NewRouter()
	mu.HandleFunc("/", func(rw http.ResponseWriter, r *http.Request) {
		err := tpl.ExecuteTemplate(rw, "index.html", map[string]interface{}{})
		if err != nil {
			xl.Error(err, "failed executing index template")
		}
	})
	mu.HandleFunc("/auth/google/login", auth.Login)
	mu.HandleFunc("/auth/google/callback", auth.Callback)

	go func() {
		xl.Info("starting service")
		certmagic.Default.Storage = &certmagic.FileStorage{Path: "/data/8x8/certs"}
		certmagic.Default.Logger = xl.Logger
		certmagic.DefaultACME.Email = "geofreyernest@live.com"
		err := certmagic.HTTPS([]string{host}, m.Then(mu))
		if err != nil {
			xl.Error(err, "exit https server")
		}
		cancel()
	}()
	<-ctx.Done()
	return ctx.Err()
}

func ensure(dir string) error {
	_, err := os.Stat(dir)
	if err != nil {
		if !os.IsNotExist(err) {
			return err
		}
		return os.MkdirAll(dir, 0755)
	}
	return nil
}

func install(ctx *cli.Context) error {
	tpl, err := template.ParseFS(templates.Installation, "*/*")
	if err != nil {
		return err
	}
	w := ctx.String("workdir")
	xl.Info("setting working directory", zap.String("w", w))
	if err := ensure(w); err != nil {
		return err
	}
	d := ctx.String("data")
	xl.Info("setting data directory", zap.String("d", d))
	if err := ensure(w); err != nil {
		return err
	}
	dataDirs := []string{"db", "certs"}
	for _, v := range dataDirs {
		if err := ensure(filepath.Join(d, v)); err != nil {
			return err
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
