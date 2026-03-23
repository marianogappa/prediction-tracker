package crypto

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/marianogappa/crypto-candles/candles"
	"github.com/marianogappa/crypto-candles/candles/common"
	"github.com/marianogappa/predictions-tracker/domain"
)

type Adapter struct {
	market candles.Market
}

func NewAdapter() *Adapter {
	return &Adapter{
		market: candles.NewMarket(),
	}
}

// ParseAsset parses "BINANCE:BTC/USDT" into a MarketSource.
func ParseAsset(asset string) (common.MarketSource, error) {
	parts := strings.SplitN(asset, ":", 2)
	if len(parts) != 2 {
		return common.MarketSource{}, fmt.Errorf("invalid asset format %q: expected EXCHANGE:BASE/QUOTE", asset)
	}
	provider := strings.ToUpper(parts[0])
	pair := strings.SplitN(parts[1], "/", 2)
	if len(pair) != 2 {
		return common.MarketSource{}, fmt.Errorf("invalid pair format %q: expected BASE/QUOTE", parts[1])
	}
	return common.MarketSource{
		Type:       common.COIN,
		Provider:   provider,
		BaseAsset:  strings.ToUpper(pair[0]),
		QuoteAsset: strings.ToUpper(pair[1]),
	}, nil
}

func (a *Adapter) FetchCandles(_ context.Context, asset string, from time.Time, interval time.Duration) ([]domain.CandleValue, error) {
	ms, err := ParseAsset(asset)
	if err != nil {
		return nil, err
	}

	iter, err := a.market.Iterator(ms, from, interval)
	if err != nil {
		return nil, fmt.Errorf("creating iterator for %s: %w", asset, err)
	}

	var vals []domain.CandleValue
	for {
		candlestick, err := iter.Next()
		if err != nil {
			if errors.Is(err, common.ErrNoNewTicksYet) || errors.Is(err, common.ErrOutOfCandlesticks) {
				break
			}
			if errors.Is(err, common.ErrExchangeReturnedNoTicks) {
				break
			}
			return vals, fmt.Errorf("fetching candle: %w", err)
		}
		vals = append(vals, domain.CandleValue{
			Timestamp: int64(candlestick.Timestamp),
			Open:      float64(candlestick.OpenPrice),
			High:      float64(candlestick.HighestPrice),
			Low:       float64(candlestick.LowestPrice),
			Close:     float64(candlestick.ClosePrice),
			Source:    ms.Provider,
		})
	}
	return vals, nil
}
