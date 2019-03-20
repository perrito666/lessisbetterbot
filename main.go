package main

import (
	"crypto/tls"
	"fmt"
	"log"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/perrito666/lessisbetterbot/skills"

	"github.com/ShiftLeftSecurity/gaum/db/connection"
	"github.com/ShiftLeftSecurity/gaum/db/logging"
	"github.com/ShiftLeftSecurity/gaum/db/postgres"
	irc "github.com/fluffle/goirc/client"
	"github.com/mvdan/xurls"
	"github.com/pkg/errors"
	"gopkg.in/ini.v1"
)

type bot struct {
	logger   *log.Logger
	nickname string
	password string
	channel  string
	storage  connection.DB
}

func (b *bot) connect(connectionString string) error {
	maxConnLifetime := 1 * time.Minute
	logLevel := connection.Error

	connector := postgres.Connector{
		ConnectionString: connectionString,
	}
	db, err := connector.Open(&connection.Information{
		Logger:          logging.NewGoLogger(b.logger),
		LogLevel:        logLevel,
		ConnMaxLifetime: &maxConnLifetime,
	})
	if err != nil {
		return errors.Wrapf(err, "initializing psql backend")
	}
	b.storage = db
	return nil
}

func (b *bot) live() error {
	// Creating a simple IRC client is simple.
	c := irc.SimpleClient(b.nickname)

	// Or, create a config and fiddle with it first:
	cfg := irc.NewConfig(b.nickname)
	cfg.SSL = true
	cfg.SSLConfig = &tls.Config{ServerName: "irc.freenode.net"}
	cfg.Server = "irc.freenode.net:7000"
	cfg.NewNick = func(n string) string { return n + "'" }
	cfg.Pass = b.password
	c = irc.Client(cfg)

	// Add handlers to do things here!
	// e.g. join a channel on connect.
	c.HandleFunc(irc.CONNECTED,
		func(conn *irc.Conn, line *irc.Line) {
			b.logger.Println("connected to freenode")
			conn.Privmsg("nickserv", fmt.Sprintf("identify %s", b.password))
			conn.Join(b.channel)
			b.logger.Printf("joined %q\n", b.channel)
		})
	// And a signal on disconnect
	quit := make(chan bool)
	c.HandleFunc(irc.DISCONNECTED,
		func(conn *irc.Conn, line *irc.Line) {
			b.logger.Println("disconnected from freenode")
			quit <- true
		})

	c.HandleFunc(irc.PRIVMSG, b.handleMsg)

	// Tell client to connect.
	b.logger.Println("will connect")
	if err := c.Connect(); err != nil {
		return errors.Wrap(err, "connecting to freenode")
	}
	b.logger.Println("did not fail to connect (?)")

	// Wait for disconnect
	<-quit
	return nil
}

func hasURL(text string) []*url.URL {
	urls := xurls.Strict().FindAllString(text, -1)
	result := []*url.URL{}
	for _, u := range urls {
		if parsed, err := url.Parse(u); err == nil {
			result = append(result, parsed)
		}
	}
	return result
}

func (b *bot) handleMsg(conn *irc.Conn, line *irc.Line) {
	if strings.ToLower(line.Nick) == strings.ToLower(b.nickname) {
		return
	}
	text := line.Text()
	channel := line.Args[0]
	msgUrls := hasURL(text)
	switch {
	case len(msgUrls) > 0:
		for _, eachURL := range msgUrls {
			u, err := skills.WebPeek(eachURL, b.logger)
			if err != nil {
				conn.Privmsg(channel, fmt.Sprintf("cant fetch title: %v", err))
				break
			}
			conn.Privmsg(channel, fmt.Sprintf("%s: \"%s\"", line.Nick, u))
		}

	}

}

func main() {

	logger := log.New(os.Stdout, "lessisbetterbot: ", log.Ldate|log.Ltime|log.Lshortfile)
	if len(os.Args) < 2 {
		logger.Fatal("this command takes one possitional argument: the path to the config file.")
		os.Exit(1)
	}

	logger.Printf("loading ini file %q\n", os.Args[1])
	cfg, err := ini.Load(os.Args[1])
	if err != nil {
		logger.Fatalf("failed to read config file: %v", err)
		os.Exit(1)
	}

	connectionString := cfg.Section("").Key("pg_connection_string").String()

	libb := bot{
		logger:   logger,
		nickname: cfg.Section("freenode").Key("nickname").String(),
		password: cfg.Section("freenode").Key("password").String(),
		channel:  "#" + cfg.Section("freenode").Key("channel").String(),
	}

	if connectionString != "" {
		err = libb.connect(connectionString)
		if err != nil {
			logger.Fatalf("connecting to the persistence storage: %v", err)
			os.Exit(1)
		}
	}
	logger.Println("going live")
	logger.Println(libb.nickname)
	logger.Println(libb.channel)
	if libb.channel == "#" {
		os.Exit(1)
	}
	fmt.Printf("%v\n", libb.live())

}
