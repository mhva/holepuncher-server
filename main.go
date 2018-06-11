package main

import (
	"encoding/hex"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli"
)

const helpTemplate = `NAME:
   {{.Name}}{{if .Usage}} - {{.Usage}}{{end}}

USAGE:
   {{if .UsageText}}{{.UsageText}}{{else}}{{.HelpName}} {{if .VisibleFlags}}[options]{{end}}{{if .Commands}} command [command options]{{end}} {{if .ArgsUsage}}{{.ArgsUsage}}{{else}}[arguments...]{{end}}{{end}}{{if .Version}}{{if not .HideVersion}}

VERSION:
   {{.Version}}{{end}}{{end}}{{if .Description}}

DESCRIPTION:
   {{.Description}}{{end}}{{if len .Authors}}

AUTHOR{{with $length := len .Authors}}{{if ne 1 $length}}S{{end}}{{end}}:
   {{range $index, $author := .Authors}}{{if $index}}
   {{end}}{{$author}}{{end}}{{end}}{{if .VisibleCommands}}

OPTIONS:
   {{range $index, $option := .VisibleFlags}}{{if $index}}
   {{end}}{{$option}}{{end}}{{end}}{{if .Copyright}}

COPYRIGHT:
   {{.Copyright}}{{end}}
`

func parseKey(keyName string, value string, fallback []byte) ([]byte, error) {
	if len(value) > 0 {
		key, err := hex.DecodeString(value)
		if err != nil {
			log.WithField("cause", err).Error("Couldn't parse %s", keyName)
			return nil, err
		}
		return key, nil
	} else if len(fallback) > 0 {
		return fallback[:], nil
	}

	msg := fmt.Sprintf("%s is empty or missing", strings.ToUpper(keyName[0:1])+keyName[1:])
	log.Error(msg)
	return nil, errors.New(msg)
}

func startServer(c *cli.Context) error {
	log.SetFormatter(
		&log.TextFormatter{
			FullTimestamp: true,
		},
	)

	if c.Bool("verbose") {
		log.SetLevel(log.DebugLevel)
	} else {
		log.SetLevel(log.InfoLevel)
	}

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(45 * time.Second))

	hostKey, err := parseKey("server key", c.String("server-key"), embeddedHostKey[:])
	if err != nil {
		return err
	}
	peerKey, err := parseKey("peer key", c.String("peer-key"), embeddedPeerKey[:])
	if err != nil {
		return err
	}

	protobufAPI := newProtobufAPIServer(hostKey, peerKey)
	r.Mount("/proto", protobufAPI.Routes())

	log.WithField("address", c.String("listen")).Info("Starting holepuncher server")
	err = http.ListenAndServe(c.String("listen"), r)
	if err != nil {
		log.WithField("cause", err).Error("Couldn't start server")
		return err
	}
	return nil
}

func main() {
	app := cli.NewApp()
	app.Name = "holepuncher-server"
	app.Usage = "server that punches holes"
	app.UsageText = "holepuncher-server [options]"
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  "listen, l",
			Usage: "listen `address`",
			Value: "localhost:9000",
		},
		cli.StringFlag{
			Name:  "server-key, s",
			Usage: "pre-shared server `key`",
		},
		cli.StringFlag{
			Name:  "peer-key, p",
			Usage: "pre-shared peer `key`",
		},
		cli.BoolFlag{
			Name:  "verbose, v",
			Usage: "verbose mode",
		},
	}
	app.CustomAppHelpTemplate = helpTemplate
	app.HideVersion = true
	app.Action = startServer

	err := app.Run(os.Args)
	if err != nil {
		os.Exit(1)
	}
}
