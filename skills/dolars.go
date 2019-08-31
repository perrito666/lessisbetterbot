package skills

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/pkg/errors"
	"github.com/shopspring/decimal"
	bolt "go.etcd.io/bbolt"
)

const (
	// USD holds the string to look for when parsing for USD currency
	USD = "Dolar U.S.A"
	// REAL holds the string to look for when parsing for Real currency
	REAL = "Real *"
)

// DollArs returns Argentinian peso to USD exchange rate according to Argentina national bank
func DollArs(currency string, db *bolt.DB, logger *log.Logger) (string, error) {
	if currency == "" {
		currency = USD
	}
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
		if innerS.HasClass("tit") && innerS.Text() == currency {
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

	sellDiff, buyDiff, err := tryAndStoreCurrency(currency, buy, sell, db)
	if err != nil {
		logger.Printf("ERROR: storing currency exchange: %v", err)
	}
	sellString := ""
	if sellDiff.IsNegative() {
		sellString = fmt.Sprintf("▼ %s", sellDiff.String())
	} else if sellDiff.Equal(decimal.Zero) {
		sellString = fmt.Sprintf("=")
	} else {
		sellString = fmt.Sprintf("▲ %s", sellDiff.String())
	}

	buyString := ""
	if buyDiff.IsNegative() {
		buyString = fmt.Sprintf("▼ %s", buyDiff.String())
	} else if buyDiff.Equal(decimal.Zero) {
		buyString = fmt.Sprintf("=")
	} else {
		buyString = fmt.Sprintf("▲ %s", buyDiff.String())
	}
	return fmt.Sprintf("(Nacion) Compra: %q (%s), Venta: %q (%s)", buy, buyString, sell, sellString), nil
}

type currencyRecord struct {
	Sell decimal.Decimal `json:"sell"`
	Buy  decimal.Decimal `json:"buy"`
}

func tryAndStoreCurrency(currency string, buy, sell []byte, db *bolt.DB) (decimal.Decimal, decimal.Decimal, error) {
	nSell := strings.Replace(string(sell), ",", ".", -1)
	numericSell, err := decimal.NewFromString(nSell)
	sellDiff := decimal.NewFromFloat(0.0)
	buyDiff := decimal.NewFromFloat(0.0)
	if err != nil {
		return sellDiff, buyDiff, errors.Wrapf(err, "sell %q is not a valid currency value", nSell)
	}
	nBuy := strings.Replace(string(buy), ",", ".", -1)
	numericBuy, err := decimal.NewFromString(nBuy)
	if err != nil {
		return sellDiff, buyDiff, errors.Wrapf(err, "buy %q is not a valid currency value", nBuy)
	}
	newRecord := currencyRecord{
		Sell: numericSell,
		Buy:  numericBuy,
	}

	db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(currency))
		key, value := b.Cursor().Last()
		if key != nil && value != nil {
			record := &currencyRecord{}
			err := json.Unmarshal(value, record)
			if err != nil {
				return nil
			}
			sellDiff = numericSell.Sub(record.Sell)
			buyDiff = numericBuy.Sub(record.Buy)
		}
		return nil
	})
	db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(currency))
		key := []byte(time.Now().UTC().Format(time.RFC3339))
		data, err := json.Marshal(&newRecord)
		if err != nil {
			return errors.Wrapf(err, "marshaling record for %s", currency)
		}
		return b.Put(key, data)
	})

	return sellDiff, buyDiff, nil
}
