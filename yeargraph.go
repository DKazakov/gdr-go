package main

import (
	"bytes"
	"fmt"
	"github.com/wcharczuk/go-chart"
	drawing "github.com/wcharczuk/go-chart/drawing"
	util "github.com/wcharczuk/go-chart/util"
)

func renderYearGraph(imageWidth, imageHeight int) (buffer *bytes.Buffer) {
	const (
		graphFontSize = 7.0
	)
	var (
		legendPrice = fmt.Sprintf("price, max: %.2f, min: %.2f, last: %.2f", dataMap.ymaxprice, dataMap.yminprice, dataMap.yearly[len(dataMap.yearly)-1])
		legendValue = fmt.Sprintf("scaled value, max: %.1fkk, min: %.1fkk", dataMap.ymaxvalue/1000000, dataMap.yminvalue/1000000)
		legendNow   = fmt.Sprintf("current price %.2f", dataMap.lastprice)
		priceSeries chart.ContinuousSeries
		valueSeries chart.ContinuousSeries
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
		XValues: dataMap.ydates,
		YValues: dataMap.yearly,
	}
	valueSeries = chart.ContinuousSeries{
		Name: legendValue,
		Style: chart.Style{
			Show:        true,
			StrokeColor: drawing.Color{R: 0, G: 255, B: 0, A: 255},
			StrokeWidth: 1.5,
		},
		XValues: dataMap.ydates,
		YValues: dataMap.yapproximatedvalues,
	}
	nowSeries = chart.ContinuousSeries{
		Name: legendNow,
		Style: chart.Style{
			Show:        true,
			StrokeColor: drawing.Color{R: 0, G: 0, B: 255, A: 255},
			StrokeWidth: 1.0,
		},
		XValues: dataMap.ydates,
		YValues: dataMap.ycurrent,
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
				Max: dataMap.ymax,
				Min: dataMap.ymin,
			},
		},
		YAxisSecondary: chart.YAxis{
			Style: chart.Style{
				Show:     true,
				FontSize: graphFontSize,
			},
			Range: &chart.ContinuousRange{
				Max: dataMap.ymax,
				Min: dataMap.ymin,
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
