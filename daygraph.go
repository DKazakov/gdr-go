package main

import (
	"bytes"
	"fmt"
	"github.com/wcharczuk/go-chart"
	drawing "github.com/wcharczuk/go-chart/drawing"
	util "github.com/wcharczuk/go-chart/util"
)

func renderDayGraph(imageWidth, imageHeight int) (buffer *bytes.Buffer) {
	const (
		graphFontSize = 7.0
	)
	var (
		legendPrice = fmt.Sprintf("price, max: %.2f, min: %.2f", dataMap.dmaxprice, dataMap.dminprice)
		legendNow   = fmt.Sprintf("last closing price %.2f", dataMap.dayprice)
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
		XValues: dataMap.times,
		YValues: dataMap.daily,
	}
	nowSeries = chart.ContinuousSeries{
		Name: legendNow,
		Style: chart.Style{
			Show:        true,
			StrokeColor: drawing.Color{R: 0, G: 0, B: 255, A: 255},
			StrokeWidth: 1.0,
		},
		XValues: dataMap.times,
		YValues: dataMap.dcurrent,
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
				Max: dataMap.dmax,
				Min: dataMap.dmin,
			},
		},
		YAxisSecondary: chart.YAxis{
			Style: chart.Style{
				Show:     true,
				FontSize: graphFontSize,
			},
			Range: &chart.ContinuousRange{
				Max: dataMap.dmax,
				Min: dataMap.dmin,
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
