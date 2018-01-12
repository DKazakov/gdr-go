package main

import (
	"bytes"
	"fmt"
	"github.com/wcharczuk/go-chart"
	drawing "github.com/wcharczuk/go-chart/drawing"
	util "github.com/wcharczuk/go-chart/util"
)

func renderAlltimeGraph(imageWidth, imageHeight int) (buffer *bytes.Buffer) {
	const (
		graphFontSize = 7.0
	)
	var (
		legendPrice = fmt.Sprintf("price, max: %.2f, min: %.2f, last: %.2f", dataMap.atmaxprice, dataMap.atminprice, dataMap.all[len(dataMap.all)-1])
		legendNow   = fmt.Sprintf("current price %.2f", dataMap.lastprice)
		priceSeries chart.ContinuousSeries
		nowSeries   chart.ContinuousSeries
	)

	buffer = bytes.NewBuffer([]byte{})

	priceSeries = chart.ContinuousSeries{
		Name: legendPrice,
		Style: chart.Style{
			Show:        true,
			StrokeColor: drawing.Color{R: 255, G: 0, B: 0, A: 255},
			FillColor:   drawing.Color{R: 255, G: 0, B: 0, A: 255},
		},
		XValues: dataMap.atdates,
		YValues: dataMap.all,
	}
	nowSeries = chart.ContinuousSeries{
		Name: legendNow,
		Style: chart.Style{
			Show:        true,
			StrokeColor: drawing.Color{R: 0, G: 0, B: 255, A: 255},
			StrokeWidth: 1.0,
		},
		XValues: dataMap.atdates,
		YValues: dataMap.atcurrent,
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
				Max: dataMap.atmax,
				Min: dataMap.atmin,
			},
		},
		YAxisSecondary: chart.YAxis{
			Style: chart.Style{
				Show:     true,
				FontSize: graphFontSize,
			},
			Range: &chart.ContinuousRange{
				Max: dataMap.atmax,
				Min: dataMap.atmin,
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
