package fairprice

import (
	"errors"
	"fmt"
	"strconv"
	"sync"
	"time"
)

type Ticker string

const (
	BTCUSDTicker Ticker = "BTC_USD"

	TickPeriod = 60 * time.Second

	//
	priceExpirePeriod = 2 * TickPeriod
)

type TickerPrice struct {
	Ticker Ticker    //
	Time   time.Time //
	Price  string    // decimal value. example: "0", "10", "12.2", "13.2345122"
}

type PriceStreamSubscriber interface {
	SubscribePriceStream(Ticker) (chan TickerPrice, chan error)
}

type fairPrice struct {
	sources []PriceStreamSubscriber
}

type tickerData struct {
	price TickerPrice  // last price
	err   error        // last error
	mx    sync.RWMutex //
}

// NewFairPrice uses upstream PriceStreamSubscriber-instances as data sources,
// merges streams and returns PriceStreamSubscriber-instance.
func NewFairPrice(sources ...PriceStreamSubscriber) PriceStreamSubscriber {
	return &fairPrice{sources}
}

func (f *fairPrice) SubscribePriceStream(ticker Ticker) (chan TickerPrice, chan error) {

	// subscribe on ticker-prices for each source
	data := make([]*tickerData, len(f.sources))
	for i, src := range f.sources {
		d := &tickerData{}
		data[i] = d
		go d.subscribe(src, ticker)
	}

	chPrice, chErr := make(chan TickerPrice), make(chan error)
	go func() {
		// align the time
		time.Sleep(TickPeriod - time.Duration(time.Now().UnixNano())%TickPeriod)

		timer := time.NewTicker(TickPeriod)
		defer timer.Stop()
		for range timer.C {
			if price, err := calcFairPrice(data); err == nil {
				chPrice <- TickerPrice{
					Ticker: ticker,
					Time:   time.Now(),
					Price:  price,
				}
			} else { // on error close channel
				chErr <- err
				close(chPrice)
				close(chErr)
				return
			}
		}
	}()
	return chPrice, chErr
}

// calcFairPrice calculates the fair price as a weighted average of the latest ticker data.
// 		fairPrice = ∑(weight * price) / n
//
// The weight is defined as the time remaining before the price is deemed to be obsolete.
// 		weight = 1-∆t/T
//
func calcFairPrice(data []*tickerData) (price string, err error) {
	now := time.Now()
	sum, sumWeight := 0.0, 0.0
	for _, d := range data {
		if tick, err := d.value(); err == nil {
			if price, err := strconv.ParseFloat(tick.Price, 64); err == nil {
				if dt := now.Sub(tick.Time); dt < priceExpirePeriod {

					weight := 1 - float64(dt)/float64(priceExpirePeriod)
					// todo: weight should be a function of Volume
					// weight := ƒ(dt, tick.Volume)

					sum += price * weight
					sumWeight += weight
				}
			}
		}
	}
	if sumWeight == 0 {
		return "", errors.New("no valid data")
	}
	return fmt.Sprintf("%.8f", sum/sumWeight), nil
}

func (t *tickerData) subscribe(s PriceStreamSubscriber, ticker Ticker) {
	for {
		prices, errs := s.SubscribePriceStream(ticker)
		for {
			select {
			case price, ok := <-prices:
				if !ok {
					break
				}
				t.setPrice(price)

			case err := <-errs:
				t.setError(err)
				break // on error stop listening and re-subscribe
			}
		}
	}
}

func (t *tickerData) value() (TickerPrice, error) {
	t.mx.RLock()
	defer t.mx.RUnlock()
	return t.price, t.err
}

func (t *tickerData) setError(err error) {
	t.mx.Lock()
	defer t.mx.Unlock()
	t.err = err
}

func (t *tickerData) setPrice(price TickerPrice) {
	t.mx.Lock()
	defer t.mx.Unlock()
	t.price = price
}
