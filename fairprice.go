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
)

type TickerPrice struct {
	Ticker Ticker    //
	Time   time.Time //
	Price  string    // decimal value. example: "0", "10", "12.2", "13.2345122"
}

type PriceStreamSubscriber interface {
	SubscribePriceStream(Ticker) (chan TickerPrice, chan error)
}

const (
	timerPeriod    = 5 * time.Second
	minPricePeriod = 2 * timerPeriod
)

type fairPrice struct {
	sources []PriceStreamSubscriber
}

type tickerData struct {
	price TickerPrice  // last price
	err   error        // last error
	mx    sync.RWMutex //
}

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
		time.Sleep(timerPeriod - time.Duration(time.Now().UnixNano())%timerPeriod)
		timer := time.NewTicker(timerPeriod)
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

func calcFairPrice(data []*tickerData) (price string, err error) {
	now := time.Now()
	sum, sumWeight := 0.0, 0.0
	for _, d := range data {
		if tick, err := d.value(); err == nil {
			if price, err := strconv.ParseFloat(tick.Price, 64); err == nil {
				if dt := now.Sub(tick.Time); dt < minPricePeriod {

					weight := 1 - float64(dt)/float64(minPricePeriod)
					// todo: weight should be a function of Volume
					// weight := fn(dt, tick.Volume)

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
