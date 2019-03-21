package skills

import (
	"fmt"
	"net/http"

	"github.com/PuerkitoBio/goquery"
	"github.com/pkg/errors"
)

// DollArs returns Argentinian peso to USD exchange rate according to Argentina national bank
func DollArs() (string, error) {
	res, err := http.Get("http://www.bna.com.ar/Personas")
	if err != nil {
		return "", errors.Wrap(err, "getting bna website")
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		return "", errors.Errorf("status code error: %d %s", res.StatusCode, res.Status)
	}

	// Load the HTML document
	doc, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		return "", errors.Wrap(err, "reading site body")
	}
	var buy, sell []byte
	var dollar bool
	extractUSD := func(i int, innerS *goquery.Selection) {
		if innerS.HasClass("tit") && innerS.Text() == "Dolar U.S.A" {
			dollar = true
			return
		}
		if dollar && i == 1 {
			buyText := innerS.Text()
			buy = []byte(buyText)
		}
		if dollar && i == 2 {
			sellText := innerS.Text()
			sell = []byte(sellText)
			dollar = false
		}
	}
	// Find the review items
	doc.Find("#billetes tr").Each(func(i int, s *goquery.Selection) {
		s.Find("td").Each(extractUSD)
	})

	return fmt.Sprintf("(Nacion) Compra: %q, Venta: %q", buy, sell), nil
}
