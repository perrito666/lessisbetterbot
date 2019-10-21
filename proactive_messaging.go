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
func ProactivelySaySomething(privates []string, conn *irc.Conn,
	nick string, db *bolt.DB, logger *log.Logger, tz int) error {
	if len(privates) == 0 {
		logger.Println("no privates to talk to")
		return nil
	}
	newsURLs, err := skills.FetchPoliticalNews(logger)
	if err != nil {
		return err
	}
	found := false
	i := 0
	cotizationGiven := false
	for k, v := range newsURLs {
		var parsed *url.URL
		parsed, err := url.Parse(k)
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
		i++
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
		tzNow := time.Now().Add(time.Duration(tz) * time.Hour)
		nowH, _, _ := tzNow.Clock()
		if nowH < 7 {
			// el cheap-o night mode
			continue
		}
		for _, private := range privates {
			conn.Privmsg(private, fmt.Sprintf("%s: \"%s\"", nick, u))
			conn.Privmsg(private, fmt.Sprintf("(%s)", strings.Replace(v, "\n", " ", -1)))
			// This is argentinian news about dollar, lets also send the
			// exchange rate.
			if !cotizationGiven {
				if strings.Count(strings.ToLower(u), "dólar") > 0 ||
					strings.Count(strings.ToLower(u), "cotización") > 0 ||
					strings.Count(strings.ToLower(u), "default") > 0 {
					usdArs, err := skills.DollArs(skills.USD, db, logger)
					if err != nil {
						logger.Printf("proactive dollars failed: %v", err)
						continue
					}
					conn.Privmsg(private, fmt.Sprintf("%s: %s", nick, usdArs))
				}
				cotizationGiven = true
			}
		}

	}
	return nil
}
