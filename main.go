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

const (
	WorkingDirectory = "/opt/8x8"
	DataDirectory    = "/data/8x8"
	SystemDUnitFile  = "/etc/systemd/system/8x8.service"
	SystemDEnvFile   = "/etc/8x8/8x8.env"
)

var DefaultEmail = os.Getenv("8x8_DEFAULT_EMAIL")

//go:generate protoc -I pkg/models/ --go_out=./pkg/models pkg/models/models.proto
//go:generate protoc -I pkg/models/ --go_out=./pkg/models pkg/models/checkers.proto

func main() {
	a := cli.NewApp()
	a.Name = "8x8 realtime checkers game"
	a.Commands = cli.Commands{
		{
			Name:   "install",
			Usage:  "installs systemd unit files and sets up 8x8",
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
		certmagic.DefaultACME.Email = DefaultEmail
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
	xl.Info("setting working directory", zap.String("w", WorkingDirectory))
	if err := ensure(WorkingDirectory); err != nil {
		return err
	}
	xl.Info("setting data directory", zap.String("d", DataDirectory))
	if err := ensure(DataDirectory); err != nil {
		return err
	}
	dataDirs := []string{"db", "certs"}
	for _, v := range dataDirs {
		if err := ensure(filepath.Join(DataDirectory, v)); err != nil {
			return err
		}
	}
	xl.Info("Setting up systemd")
	var buf bytes.Buffer
	err = tpl.ExecuteTemplate(&buf, "8x8.service", map[string]interface{}{
		"WorkingDirectory": WorkingDirectory,
		"Data":             DataDirectory,
	})
	if err != nil {
		return err
	}
	xl.Info("writing systemd service file", zap.String("path", SystemDUnitFile))
	err = ioutil.WriteFile(SystemDUnitFile, buf.Bytes(), 0600)
	if err != nil {
		return err
	}

	env := filepath.Dir(SystemDEnvFile)
	xl.Info("creating env variable directory", zap.String("path", env))
	if err := ensure(env); err != nil {
		return err
	}
	_, err = os.Stat(SystemDEnvFile)
	if os.IsNotExist(err) {
		buf.Reset()
		err = tpl.ExecuteTemplate(&buf, "8x8.env", map[string]interface{}{
			"GOOGLE_CLIENT_ID":     os.Getenv("GOOGLE_CLIENT_ID"),
			"GOOGLE_CLIENT_SECRET": os.Getenv("GOOGLE_CLIENT_SECRET"),
		})
		if err != nil {
			return err
		}

		xl.Info("writing systemd service env file", zap.String("path", SystemDEnvFile))
		err = ioutil.WriteFile(SystemDEnvFile, buf.Bytes(), 0600)
		if err != nil {
			return err
		}
	}
	xl.Info("systemctl  enable 8x8.service # to start at boot")
	xl.Info("systemctl  start 8x8.service # to star the service")
	return nil
}
