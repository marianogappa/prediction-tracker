package export

import (
	"bytes"
	"fmt"
	"time"

	"github.com/marianogappa/predictions-tracker/domain"
	chart "github.com/wcharczuk/go-chart/v2"
	"github.com/wcharczuk/go-chart/v2/drawing"
)

func RenderChart(values []domain.CandleValue, rule domain.Rule) ([]byte, error) {
	if len(values) == 0 {
		return nil, fmt.Errorf("no values to chart")
	}

	xVals := make([]float64, len(values))
	yVals := make([]float64, len(values))
	for i, v := range values {
		xVals[i] = float64(v.Timestamp)
		yVals[i] = v.PriceByField(rule.PriceField)
	}

	priceSeries := chart.ContinuousSeries{
		Name: fmt.Sprintf("%s price", rule.PriceField),
		Style: chart.Style{
			StrokeColor: drawing.ColorFromHex("6366f1"),
			StrokeWidth: 2,
		},
		XValues: xVals,
		YValues: yVals,
	}

	targetLine := chart.ContinuousSeries{
		Name: fmt.Sprintf("target: %.2f", rule.Value),
		Style: chart.Style{
			StrokeColor:     drawing.ColorFromHex("f87171"),
			StrokeWidth:     1.5,
			StrokeDashArray: []float64{5, 3},
		},
		XValues: []float64{xVals[0], xVals[len(xVals)-1]},
		YValues: []float64{rule.Value, rule.Value},
	}

	graph := chart.Chart{
		Title:  "Price vs Target",
		Width:  900,
		Height: 400,
		Background: chart.Style{
			Padding: chart.Box{Top: 30, Left: 10, Right: 10, Bottom: 10},
		},
		XAxis: chart.XAxis{
			ValueFormatter: func(v any) string {
				if ts, ok := v.(float64); ok {
					return time.Unix(int64(ts), 0).UTC().Format("01/02 15:04")
				}
				return ""
			},
		},
		YAxis: chart.YAxis{
			ValueFormatter: func(v any) string {
				if f, ok := v.(float64); ok {
					return fmt.Sprintf("%.2f", f)
				}
				return ""
			},
		},
		Series: []chart.Series{priceSeries, targetLine},
	}
	graph.Elements = []chart.Renderable{chart.LegendLeft(&graph)}

	var buf bytes.Buffer
	if err := graph.Render(chart.PNG, &buf); err != nil {
		return nil, fmt.Errorf("rendering chart: %w", err)
	}
	return buf.Bytes(), nil
}
