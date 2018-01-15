package main

import (
	"bytes"
	"fmt"
	"github.com/wcharczuk/go-chart"
	drawing "github.com/wcharczuk/go-chart/drawing"
	util "github.com/wcharczuk/go-chart/util"
	"log"
)

var (
	allData *tradeData
)

func renderAlltimeGraph(imageWidth, imageHeight int) (buffer *bytes.Buffer) {
	var (
		minprice, maxprice = minmax(allData.prices)
		min, max           = minmax([]float64{maxprice + 0.5, minprice - 0.5, lastprice + 0.5, lastprice - 0.5})
		priceSeries        chart.ContinuousSeries
		nowSeries          chart.ContinuousSeries
	)

	buffer = bytes.NewBuffer([]byte{})

	priceSeries = chart.ContinuousSeries{
		Name: fmt.Sprintf("price, max: %.2f, min: %.2f, last: %.2f", maxprice, minprice, allData.prices[len(allData.prices)-1]),
		Style: chart.Style{
			Show:        true,
			StrokeColor: drawing.Color{R: 255, G: 0, B: 0, A: 255},
			FillColor:   drawing.Color{R: 255, G: 0, B: 0, A: 255},
		},
		XValues: allData.dates,
		YValues: allData.prices,
	}
	nowSeries = chart.ContinuousSeries{
		Name: fmt.Sprintf("current price %.2f", lastprice),
		Style: chart.Style{
			Show:        true,
			StrokeColor: drawing.Color{R: 0, G: 0, B: 255, A: 255},
			StrokeWidth: 1.0,
		},
		XValues: []float64{allData.dates[0], allData.dates[len(allData.dates)-1]},
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
				return fmt.Sprintf("%.2d.%.2d", typedDate.Month(), typedDate.Year())
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

func allRequestCallback(json *jsonStock) {
	data := new(tradeData)

	for i := 0; i < len(json.Data); i++ {
		date := json.Data[i][0] * 1000000

		if i > 0 {
			if json.Data[i][1] < json.Data[i-1][1]*2 {
				data.prices = append(data.prices, json.Data[i][1])
			} else {
				data.prices = append(data.prices, json.Data[i-1][1])
				log.Printf("Skip wrong price: %+v", json.Data[i])
			}
		} else {
			data.prices = append(data.prices, json.Data[i][1])
		}
		data.dates = append(data.dates, date)
	}

	allData = data
}
