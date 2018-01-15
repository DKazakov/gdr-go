package main

import (
	"bytes"
	"fmt"
	"github.com/wcharczuk/go-chart"
	drawing "github.com/wcharczuk/go-chart/drawing"
	util "github.com/wcharczuk/go-chart/util"
)

var (
	defaultData *tradeData
	defaultGdr  *gdrData
)

func renderDefaultGraph(imageWidth, imageHeight int) (buffer *bytes.Buffer) {
	const (
		graphFontSize = 7.0
	)
	var (
		minprice, maxprice = minmax(defaultData.prices)
		min, max           = minmax([]float64{maxprice + 0.5, minprice - 0.5, lastprice + 0.5, lastprice - 0.5})
		mingdr, maxgdr     = minmax(defaultGdr.gdr)
		minvalue, maxvalue = minmax(defaultData.values)

		priceSeries chart.ContinuousSeries
		gdrSeries   chart.ContinuousSeries
		valueSeries chart.ContinuousSeries
		nowSeries   chart.ContinuousSeries
	)

	buffer = bytes.NewBuffer([]byte{})

	priceSeries = chart.ContinuousSeries{
		Name: fmt.Sprintf("price, max: %.2f, min: %.2f, last: %.2f", maxprice, minprice, defaultData.prices[len(defaultData.prices)-1]),
		Style: chart.Style{
			Show:        true,
			StrokeColor: drawing.Color{R: 255, G: 0, B: 0, A: 255},
			FillColor:   drawing.Color{R: 255, G: 0, B: 0, A: 255},
		},
		XValues: defaultData.dates,
		YValues: defaultData.prices,
	}
	gdrSeries = chart.ContinuousSeries{
		Name: fmt.Sprintf("scaled GDR's, max: %.2f, min: %.2f, now: %.2f", maxgdr, mingdr, defaultGdr.gdr[len(defaultGdr.gdr)-1]),
		Style: chart.Style{
			Show:        true,
			StrokeColor: drawing.Color{R: 0, G: 0, B: 0, A: 255},
			StrokeWidth: 1.5,
		},
		XValues: defaultGdr.dates,
		YValues: approximate(maxprice, minprice, defaultGdr.gdr),
	}
	valueSeries = chart.ContinuousSeries{
		Name: fmt.Sprintf("scaled value, max: %.1fkk, min: %.1fkk", maxvalue/1000000, minvalue/1000000),
		Style: chart.Style{
			Show:        true,
			StrokeColor: drawing.Color{R: 0, G: 255, B: 0, A: 255},
			StrokeWidth: 1.5,
		},
		XValues: defaultData.dates,
		YValues: approximate(maxprice, minprice, defaultData.values),
	}
	nowSeries = chart.ContinuousSeries{
		Name: fmt.Sprintf("current price %.2f", lastprice),
		Style: chart.Style{
			Show:        true,
			StrokeColor: drawing.Color{R: 0, G: 0, B: 255, A: 255},
			StrokeWidth: 1.0,
		},
		XValues: []float64{defaultData.dates[0], defaultData.dates[len(defaultData.dates)-1]},
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
				return fmt.Sprintf("%.2d.%.2d", typedDate.Day(), typedDate.Month())
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

func defaultRequestCallback(json *jsonStock) {
	data := new(tradeData)
	gdr := new(gdrData)
	for i := 0; i < len(json.Data); i++ {
		date := json.Data[i][0] * 1000000
		data.prices = append(data.prices, json.Data[i][1])
		data.dates = append(data.dates, date)
		data.values = append(data.values, json.Data[i][2])
		if i > 1 {
			gdr.gdr = append(gdr.gdr, getGdr(data.prices, data.values))
			gdr.dates = append(gdr.dates, date)
		}
	}

	defaultData = data
	defaultGdr = gdr
}
