package main

import (
	"fmt"
	"github.com/denisskin/fairprice"
	"github.com/denisskin/fairprice/testgenerator"
	"os"
)

func main() {

	//----- init random streams
	streams := make([]fairprice.PriceStreamSubscriber, 100)
	for i := range streams {
		streams[i] = testgenerator.New()
	}

	//---- init fair-price stream
	fp := fairprice.NewFairPrice(streams...)
	chPrice, chErr := fp.SubscribePriceStream(fairprice.BTCUSDTicker)
	for {
		select {
		case p := <-chPrice:
			fmt.Printf("%s\t%s\t%s\n", p.Ticker, p.Time.Format("2006-01-02 15:04:05"), p.Price)

		case err := <-chErr:
			fmt.Fprintln(os.Stderr, err)
		}
	}
}
