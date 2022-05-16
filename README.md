# FairPrice (Test task)

Fairprice is a golang-package for combining multiple streams of ticker prices into a single price stream.

`NewFairPrice()` – uses upstream PriceStreamSubscriber-instances as data sources, 
merges streams and returns PriceStreamSubscriber-instance. 
```
    fp := fairprice.NewFairPrice(streams...)
    chPrice, chErr := fp.SubscribePriceStream(fairprice.BTCUSDTicker)
```
(An example of use can be seen in `cmd/main.go`)

The function constantly receives updated price data for each source, 
stores the latest values in memory, and recalculates the fair price every minute.

The update time for each source is taken into account. If the ticker price is expired it is not used.
The calculation uses the weighted average formula, in which the weight is defined as the time remaining before the price is deemed to be expired.
(The period is set by the `priceExpirePeriod`constant).
```
    fairPrice = ∑(weight * price) / n
    
    weight = 1 - ∆t/T
```
Thus, older price values have less impact on the result.    

Ideally, the market-volume (and other exchanger-parameters) should also be taken into account to calculate the weight.
```
    weight = ƒ(∆t, volume, ...)
```

