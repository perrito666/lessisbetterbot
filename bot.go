package main

import (
	"crypto/tls"
	"fmt"
	"log"
	"net/url"
	"regexp"
	"strings"

	irc "github.com/fluffle/goirc/client"
	"github.com/mvdan/xurls"
	"github.com/perrito666/lessisbetterbot/skills"
	"github.com/pkg/errors"
)

// bot is the basic irc bot actor.
type bot struct {
	logger *log.Logger
	cfg    *Config
}

// live tarts the bot loop.
func (b *bot) live() error {
	// Or, create a config and fiddle with it first:
	cfg := irc.NewConfig(b.cfg.NickName, b.cfg.NickName, b.cfg.Ident)
	cfg.SSL = true
	cfg.SSLConfig = &tls.Config{ServerName: "irc.freenode.net"}
	cfg.Server = "irc.freenode.net:7000"
	cfg.NewNick = func(n string) string { return n + "'" }
	cfg.Pass = b.cfg.Password
	c := irc.Client(cfg)

	// Add handlers to do things here!
	// e.g. join a channel on connect.
	c.HandleFunc(irc.CONNECTED,
		func(conn *irc.Conn, line *irc.Line) {
			b.logger.Println("connected to freenode")
			if b.cfg.NickservCmd != "" {
				conn.Privmsg("nickserv", fmt.Sprintf("identify %s", b.cfg.NickservCmd))
			}
			for _, channel := range b.cfg.Channels {
				conn.Join("#" + channel)
				b.logger.Printf("joined #%s\n", channel)
			}
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

// hasURL returns all urls found in a string, necessary for the webpeek url
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

var billeteRegex = regexp.MustCompile(`c[oó]mo est[aá] el billete.*\?`)
var billetinioRegex = regexp.MustCompile(`c[oó]mo est[aá] el billetiño.*\?`)

// handleMsg will try to find a skill that suits the message and handle it.
func (b *bot) handleMsg(conn *irc.Conn, line *irc.Line) {
	if strings.ToLower(line.Nick) == strings.ToLower(b.cfg.NickName) {
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
	case billeteRegex.MatchString(strings.ToLower(text)):
		usdArs, err := skills.DollArs(skills.USD)
		if err != nil {
			b.logger.Printf("dollars failed: %v", err)
			break
		}
		conn.Privmsg(channel, fmt.Sprintf("%s: %s", line.Nick, usdArs))

	case billetinioRegex.MatchString(strings.ToLower(text)):
		usdArs, err := skills.DollArs(skills.REAL)
		if err != nil {
			b.logger.Printf("reais failed: %v", err)
			break
		}
		conn.Privmsg(channel, fmt.Sprintf("%s: %s", line.Nick, usdArs))

	}

}
