package domain

type CandleValue struct {
	Timestamp int64   `json:"timestamp"`
	Open      float64 `json:"open"`
	High      float64 `json:"high"`
	Low       float64 `json:"low"`
	Close     float64 `json:"close"`
	Source    string  `json:"source"`
}

func (v CandleValue) PriceByField(field string) float64 {
	switch field {
	case "open":
		return v.Open
	case "high":
		return v.High
	case "low":
		return v.Low
	case "close":
		return v.Close
	default:
		return v.Close
	}
}
