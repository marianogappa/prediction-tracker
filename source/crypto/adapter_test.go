package crypto

import (
	"testing"

	"github.com/marianogappa/crypto-candles/candles/common"
)

func TestParseAsset(t *testing.T) {
	tests := []struct {
		input   string
		want    common.MarketSource
		wantErr bool
	}{
		{
			input: "BINANCE:BTC/USDT",
			want: common.MarketSource{
				Type: common.COIN, Provider: "BINANCE",
				BaseAsset: "BTC", QuoteAsset: "USDT",
			},
		},
		{
			input: "coinbase:eth/usd",
			want: common.MarketSource{
				Type: common.COIN, Provider: "COINBASE",
				BaseAsset: "ETH", QuoteAsset: "USD",
			},
		},
		{input: "INVALID", wantErr: true},
		{input: "BINANCE:BTCUSDT", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := ParseAsset(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("got %+v, want %+v", got, tt.want)
			}
		})
	}
}
