package main

import (
	"bytes"
	"fmt"
	"github.com/wcharczuk/go-chart"
	drawing "github.com/wcharczuk/go-chart/drawing"
	util "github.com/wcharczuk/go-chart/util"
)

var (
	yearData *tradeData
)

func renderYearGraph(imageWidth, imageHeight int) (buffer *bytes.Buffer) {
	const (
		graphFontSize = 7.0
	)
	var (
		minprice, maxprice = minmax(yearData.prices)
		min, max           = minmax([]float64{maxprice + 0.5, minprice - 0.5, lastprice + 0.5, lastprice - 0.5})
		minvalue, maxvalue = minmax(yearData.values)
		priceSeries        chart.ContinuousSeries
		valueSeries        chart.ContinuousSeries
		nowSeries          chart.ContinuousSeries
	)

	buffer = bytes.NewBuffer([]byte{})

	priceSeries = chart.ContinuousSeries{
		Name: fmt.Sprintf("price, max: %.2f, min: %.2f, last: %.2f", maxprice, minprice, yearData.prices[len(yearData.prices)-1]),
		Style: chart.Style{
			Show:        true,
			StrokeColor: drawing.Color{R: 255, G: 0, B: 0, A: 255},
			FillColor:   drawing.Color{R: 255, G: 0, B: 0, A: 255},
		},
		XValues: yearData.dates,
		YValues: yearData.prices,
	}
	valueSeries = chart.ContinuousSeries{
		Name: fmt.Sprintf("scaled value, max: %.1fkk, min: %.1fkk", maxvalue/1000000, minvalue/1000000),
		Style: chart.Style{
			Show:        true,
			StrokeColor: drawing.Color{R: 0, G: 255, B: 0, A: 255},
			StrokeWidth: 1.5,
		},
		XValues: yearData.dates,
		YValues: approximate(maxprice, minprice, yearData.values),
	}
	nowSeries = chart.ContinuousSeries{
		Name: fmt.Sprintf("current price %.2f", lastprice),
		Style: chart.Style{
			Show:        true,
			StrokeColor: drawing.Color{R: 0, G: 0, B: 255, A: 255},
			StrokeWidth: 1.0,
		},
		XValues: []float64{yearData.dates[0], yearData.dates[len(yearData.dates)-1]},
		YValues: []float64{lastprice, lastprice},
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
				return fmt.Sprintf("%.2d.%.2d.%.2d", typedDate.Day(), typedDate.Month(), typedDate.Year())
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
			valueSeries,
			nowSeries,
		},
	}
	graph.Elements = []chart.Renderable{
		chart.LegendThin(&graph),
	}

	graph.Render(chart.PNG, buffer)

	return
}

func yearRequestCallback(json *jsonStock) {
	data := new(tradeData)
	for i := 0; i < len(json.Data); i++ {
		date := json.Data[i][0] * 1000000
		data.prices = append(data.prices, json.Data[i][1])
		data.dates = append(data.dates, date)
		data.values = append(data.values, json.Data[i][2])
	}

	yearData = data
}
