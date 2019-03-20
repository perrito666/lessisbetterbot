package skills

import (
	"log"
	"net/url"

	"github.com/badoux/goscraper"
	"github.com/pkg/errors"
)

// WebPeek checks a URL and returns a suitable text representation for it.
func WebPeek(siteURL *url.URL, logger *log.Logger) (string, error) {

	s, err := goscraper.Scrape(siteURL.String(), 5)
	if err != nil {
		return "", errors.Wrapf(err, "scrapping %q URL", siteURL.String())
	}
	if len(s.Preview.Title) > 0 {
		return s.Preview.Title, nil
	}
	if len(s.Preview.Description) > 0 {
		// TODO shorten this string
		return s.Preview.Description, nil
	}
	return "", nil
}
