package main

import (
	"bytes"
	"fmt"
	"github.com/wcharczuk/go-chart"
	drawing "github.com/wcharczuk/go-chart/drawing"
	util "github.com/wcharczuk/go-chart/util"
)

var (
	dayData *tradeData
)

func renderDayGraph(imageWidth, imageHeight int) (buffer *bytes.Buffer) {
	const (
		graphFontSize = 7.0
	)
	var (
		minprice, maxprice = minmax(dayData.prices)
		closingprice       = defaultData.prices[len(defaultData.prices)-1]
		min, max           = minmax([]float64{maxprice + 0.5, minprice - 0.5, closingprice + 0.5, closingprice - 0.5})
		priceSeries        chart.ContinuousSeries
		nowSeries          chart.ContinuousSeries
	)

	buffer = bytes.NewBuffer([]byte{})

	priceSeries = chart.ContinuousSeries{
		Name: fmt.Sprintf("price, max: %.2f, min: %.2f", maxprice, minprice),
		Style: chart.Style{
			Show:        true,
			StrokeColor: drawing.Color{R: 255, G: 0, B: 0, A: 255},
			FillColor:   drawing.Color{R: 255, G: 0, B: 0, A: 255},
		},
		XValues: dayData.dates,
		YValues: dayData.prices,
	}
	nowSeries = chart.ContinuousSeries{
		Name: fmt.Sprintf("last closing price %.2f", defaultData),
		Style: chart.Style{
			Show:        true,
			StrokeColor: drawing.Color{R: 0, G: 0, B: 255, A: 255},
			StrokeWidth: 1.0,
		},
		XValues: []float64{dayData.dates[0], dayData.dates[len(dayData.dates)-1]},
		YValues: []float64{closingprice, closingprice},
	}

	graph := chart.Chart{
		Width:  imageWidth,
		Height: imageHeight,
		Background: chart.Style{
			Padding: chart.Box{
				Top:    20,
				Left:   0,
				Right:  0,
				Bottom: 0,
			},
		},
		XAxis: chart.XAxis{
			Style: chart.Style{
				Show:     true,
				FontSize: graphFontSize,
			},
			TickPosition: chart.TickPositionBetweenTicks,
			ValueFormatter: func(v interface{}) string {
				typed := v.(float64)
				typedDate := util.Time.FromFloat64(typed)
				return fmt.Sprintf("%.2d:%.2d", typedDate.Hour(), typedDate.Minute())
			},
		},
		YAxis: chart.YAxis{
			Style: chart.Style{
				Show:     true,
				FontSize: graphFontSize,
			},
			Range: &chart.ContinuousRange{
				Max: max,
				Min: min,
			},
		},
		YAxisSecondary: chart.YAxis{
			Style: chart.Style{
				Show:     true,
				FontSize: graphFontSize,
			},
			Range: &chart.ContinuousRange{
				Max: max,
				Min: min,
			},
		},
		Series: []chart.Series{
			priceSeries,
			nowSeries,
		},
	}
	graph.Elements = []chart.Renderable{
		chart.LegendThin(&graph),
	}

	graph.Render(chart.PNG, buffer)

	return
}

func dayRequestCallback(json *jsonStock) {
	data := new(tradeData)

	for _, now := range json.Data {
		data.prices = append(data.prices, now[1])

		time := now[0] * 1000000
		data.dates = append(data.dates, time)
		lastprice = now[1]
	}

	dayData = data
}
