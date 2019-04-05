package main

import (
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"time"

	irc "github.com/fluffle/goirc/client"
	"github.com/mvdan/xurls"
	"github.com/perrito666/lessisbetterbot/skills"
	"github.com/pkg/errors"
	bolt "go.etcd.io/bbolt"
)

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
	lowerText := strings.ToLower(text)

	switch {
	case len(msgUrls) > 0: // WebPeek
		for _, eachURL := range msgUrls {
			u, err := webFromCacheOrHit(eachURL, line.Nick, b.cfg.TimeZone, b.db)
			if err != nil {
				errMsg := strings.Split(fmt.Sprintf("%v", err), "\n")[0]
				if len(errMsg) > 100 {
					errMsg = errMsg[:100] + " (truncated)"
				}
				conn.Privmsg(channel, fmt.Sprintf("cant fetch title: %v", errMsg))
				break
			}
			conn.Privmsg(channel, fmt.Sprintf("%s: \"%s\"", line.Nick, u))
		}
	case billeteRegex.MatchString(lowerText): //dollars
		usdArs, err := skills.DollArs(skills.USD, b.db, b.logger)
		if err != nil {
			b.logger.Printf("dollars failed: %v", err)
			break
		}
		conn.Privmsg(channel, fmt.Sprintf("%s: %s", line.Nick, usdArs))

	case billetinioRegex.MatchString(lowerText): //reais
		usdArs, err := skills.DollArs(skills.REAL, b.db, b.logger)
		if err != nil {
			b.logger.Printf("reais failed: %v", err)
			break
		}
		conn.Privmsg(channel, fmt.Sprintf("%s: %s", line.Nick, usdArs))

	case strings.Index(lowerText, "como viene el brexit?") > -1: // brexit
		brexit, err := skills.BrexitStatus()
		if err != nil {
			b.logger.Printf("brexit failed (lol): %v", err)
			break
		}
		conn.Privmsg(channel, fmt.Sprintf("%s: Current brexit status is %s", line.Nick, brexit))
	}

}

// Process URLs in the message text ------------------------------------------------
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

type webHit struct {
	Nick     string
	PostTime string
	Title    string
	Num      int64
}

// check if we have the passed in web in store or needs fetching.
func webFromCacheOrHit(eachURL *url.URL, nick string, tz int, db *bolt.DB) (string, error) {
	cacheHit := &webHit{}
	err := db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("webpeek"))
		v := b.Get([]byte(eachURL.String()))
		if len(v) == 0 {
			return nil
		}
		return json.Unmarshal(v, cacheHit)
	})
	if err != nil {
		return "", errors.Wrap(err, "cant fetch title")
	}
	if len(cacheHit.PostTime) != 0 {
		pt, err := time.Parse(time.RFC3339, cacheHit.PostTime)
		if err != nil {
			return "", errors.Wrap(err, "db returned invalid time")
		}
		pt = pt.Add(time.Duration(tz) * time.Hour)
		printableTime := pt.Format("02/01/06 03:04")

		return fmt.Sprintf("[%d] (by %s on %s) %s", cacheHit.Num, cacheHit.Nick, printableTime, cacheHit.Title), nil
	}
	u, err := skills.WebPeek(eachURL)
	if err != nil {
		return "", errors.Wrap(err, "cant fetch title")
	}
	err = db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("webpeek"))

		cacheHit.Nick = nick
		cacheHit.PostTime = time.Now().UTC().Format(time.RFC3339)
		cacheHit.Title = u

		id, _ := b.NextSequence()
		cacheHit.Num = int64(id)

		buf, err := json.Marshal(cacheHit)
		if err != nil {
			return err
		}

		return b.Put([]byte(eachURL.String()), buf)
	})
	if err != nil {
		return "", errors.Wrap(err, "cant store title")
	}
	return fmt.Sprintf("[%d] %s", cacheHit.Num, u), nil

}
