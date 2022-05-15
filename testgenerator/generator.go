package testgenerator

import (
	"encoding/json"
	"fmt"
	"github.com/denisskin/fairprice"
	"io"
	"math/rand"
	"net/http"
	"sync/atomic"
	"time"
)

type priceGenerator struct{}

func New() fairprice.PriceStreamSubscriber {
	return priceGenerator{}
}

func (g priceGenerator) SubscribePriceStream(ticker fairprice.Ticker) (chan fairprice.TickerPrice, chan error) {
	chPrice, chErr := make(chan fairprice.TickerPrice), make(chan error)
	go func() {
		constSpread := (rand.Float64() - 0.5) * 2 * 1e-2 // ±1%
		for {
			if price, ok := randBasePrice.Load().(float64); ok {

				// generate random price: basePrice +spread ±0.1%
				price *= 1 + constSpread + rand.Float64()*0.1e-2

				chPrice <- fairprice.TickerPrice{
					Ticker: ticker,
					Price:  fmt.Sprintf("%.8f", price),
					Time:   time.Now(),
				}
			}
			// random sleep
			time.Sleep(time.Duration(rand.Float64() * float64(3*time.Second)))
		}
	}()
	return chPrice, chErr
}

var randBasePrice atomic.Value

func init() {
	go func() {
		for {
			res, _ := http.Get("https://www.bitstamp.net/api/v2/ticker/btcusd")
			data, _ := io.ReadAll(res.Body)
			res.Body.Close()
			var v struct {
				Ask float64 `json:"ask,string"`
			}
			if err := json.Unmarshal(data, &v); err == nil && v.Ask > 0 {
				randBasePrice.Store(v.Ask)
			}
			time.Sleep(15 * time.Second)
		}
	}()
}
