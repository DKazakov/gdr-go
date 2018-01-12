package main

import (
	"bytes"
	"fmt"
	"github.com/wcharczuk/go-chart"
	drawing "github.com/wcharczuk/go-chart/drawing"
	util "github.com/wcharczuk/go-chart/util"
)

func renderDefaultGraph(imageWidth, imageHeight int) (buffer *bytes.Buffer) {
	const (
		graphFontSize = 7.0
	)
	var (
		legendPrice = fmt.Sprintf("price, max: %.2f, min: %.2f, last: %.2f", dataMap.maxprice, dataMap.minprice, dataMap.monthly[len(dataMap.monthly)-1])
		legendValue = fmt.Sprintf("scaled value, max: %.1fkk, min: %.1fkk", dataMap.maxvalue/1000000, dataMap.minvalue/1000000)
		legendGdr   = fmt.Sprintf("scaled GDR's, max: %.2f, min: %.2f, now: %.2f", dataMap.maxgdr, dataMap.mingdr, dataMap.gdr[len(dataMap.gdr)-1])
		legendNow   = fmt.Sprintf("current price %.2f", dataMap.lastprice)
		priceSeries chart.ContinuousSeries
		gdrSeries   chart.ContinuousSeries
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
		XValues: dataMap.dates,
		YValues: dataMap.monthly,
	}
	gdrSeries = chart.ContinuousSeries{
		Name: legendGdr,
		Style: chart.Style{
			Show:        true,
			StrokeColor: drawing.Color{R: 0, G: 0, B: 0, A: 255},
			StrokeWidth: 1.5,
		},
		XValues: dataMap.gdrdates,
		YValues: dataMap.approximatedgdr,
	}
	valueSeries = chart.ContinuousSeries{
		Name: legendValue,
		Style: chart.Style{
			Show:        true,
			StrokeColor: drawing.Color{R: 0, G: 255, B: 0, A: 255},
			StrokeWidth: 1.5,
		},
		XValues: dataMap.dates,
		YValues: dataMap.approximatedvalues,
	}
	nowSeries = chart.ContinuousSeries{
		Name: legendNow,
		Style: chart.Style{
			Show:        true,
			StrokeColor: drawing.Color{R: 0, G: 0, B: 255, A: 255},
			StrokeWidth: 1.0,
		},
		XValues: dataMap.dates,
		YValues: dataMap.current,
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
				return fmt.Sprintf("%.2d.%.2d", typedDate.Day(), typedDate.Month())
			},
		},
		YAxis: chart.YAxis{
			Style: chart.Style{
				Show:     true,
				FontSize: graphFontSize,
			},
			Range: &chart.ContinuousRange{
				Max: dataMap.max,
				Min: dataMap.min,
			},
		},
		YAxisSecondary: chart.YAxis{
			Style: chart.Style{
				Show:     true,
				FontSize: graphFontSize,
			},
			Range: &chart.ContinuousRange{
				Max: dataMap.max,
				Min: dataMap.min,
			},
		},
		Series: []chart.Series{
			priceSeries,
			gdrSeries,
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
