package main

import (
	"fmt"
	"log"
	"net/url"
	"strings"
	"time"

	irc "github.com/fluffle/goirc/client"
	"github.com/perrito666/lessisbetterbot/skills"
	bolt "go.etcd.io/bbolt"
)

// ProactivelySaySomething will send messages to irc channels
func ProactivelySaySomething(channels []string, conn *irc.Conn,
	nick string, db *bolt.DB, logger *log.Logger, tz int) error {
	if len(channels) == 0 {
		logger.Println("no channels to talk to")
		return nil
	}
	newsURLs, err := skills.FetchPoliticalNews(logger)
	if err != nil {
		return err
	}
	found := false
	for i := range newsURLs {
		var parsed *url.URL
		parsed, err := url.Parse(newsURLs[i])
		if err != nil {
			logger.Printf("failed to parse fetched url: %v", err)
			// log this error
			continue
		}

		u, err := webFromCache(parsed, nick, tz, db)
		if err != nil {
			errMsg := strings.Split(fmt.Sprintf("%v", err), "\n")[0]
			if len(errMsg) > 100 {
				errMsg = errMsg[:100] + " (truncated)"
			}
			logger.Printf("failed to fetch url from cache: %v", errMsg)
			continue
		}

		if u != "" {
			// this url was already published
			continue
		}
		// this code does not make me proud but the country is in flames
		// I would make this a python oneliner if I could :p
		if i > 0 && found {
			// let us not spam the channel if there are many news
			<-time.After(5 * time.Second)
		}
		found = true
		u, err = webFromCacheOrHit(parsed, nick, tz, db)
		if err != nil {
			errMsg := strings.Split(fmt.Sprintf("%v", err), "\n")[0]
			if len(errMsg) > 100 {
				errMsg = errMsg[:100] + " (truncated)"
			}
			logger.Printf("failed to fetch url from cache: %v", errMsg)
			continue
		}
		for _, channel := range channels {
			conn.Privmsg("#"+channel, fmt.Sprintf("%s: \"%s\"", nick, u))
			// This is argentinian news about dollar, lets also send the
			// exchange rate.
			if strings.Count(strings.ToLower(u), "dólar") > 0 ||
				strings.Count(strings.ToLower(u), "cotiz") > 0 ||
				strings.Count(strings.ToLower(u), "default") > 0 {
				usdArs, err := skills.DollArs(skills.USD, db, logger)
				if err != nil {
					logger.Printf("proactive dollars failed: %v", err)
					continue
				}
				conn.Privmsg("#"+channel, fmt.Sprintf("%s: %s", nick, usdArs))
			}
		}

	}
	return nil
}
