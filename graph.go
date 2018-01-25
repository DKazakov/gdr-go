package main

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"github.com/wcharczuk/go-chart"
	drawing "github.com/wcharczuk/go-chart/drawing"
	"strings"
)

type Graph struct {
	pages [4]GraphData
	page  int
}

func (self *Graph) Init(data *Data) *Graph {
	self.pages = data.graph

	return self
}

func (self Graph) getNextPage() int {
	page := self.page + 1
	maxpage := len(self.pages) - 1
	if page > maxpage {
		page = 0
	}

	return page
}
func (self Graph) getPrevPage() int {
	page := self.page - 1
	maxpage := len(self.pages) - 1
	if page < 0 {
		page = maxpage
	}

	return page
}

func (self *Graph) setPage(page int) {
	maxpage := len(self.pages) - 1

	if page < 0 {
		page = 0
	} else if page > maxpage {
		page = maxpage
	}

	self.page = page

	return
}

func (self Graph) print(width, height, left, bottom int) {
	imageWidth := (width - left) * 7
	imageHeight := (height - bottom) * 15

	image := self.render(imageWidth, imageHeight)
	str := base64.StdEncoding.EncodeToString(image.Bytes())
	paginate := self.paginate()

	fmt.Printf("\x1b[%d;%dH\x1b]1337;File=name=none;size=%d;inline=1:%s\a\n", 0, left+1, len(str), str)
	fmt.Printf("\x1b[%d;%dH%s", height-bottom+1, int(width/2)-1, strings.Repeat(" ", len(paginate)))
	fmt.Printf("\x1b[%d;%dH%s", height-bottom+1, int(width/2)-1, paginate)
	return
}

func (self Graph) paginate() string {
	status := []string{
		"\u2776 \u2781 \u2782 \u2783 сегодня",
		"\u2780 \u2777 \u2782 \u2783 за последний месяц",
		"\u2780 \u2781 \u2778 \u2783 за последний год",
		"\u2780 \u2781 \u2782 \u2779 за пять лет",
	}[self.page]

	return status
}

func (self Graph) render(imageWidth, imageHeight int) *bytes.Buffer {
	buffer := bytes.NewBuffer([]byte{})
	source := self.pages[self.page]
	series := []chart.Series{}

	series = append(series, chart.ContinuousSeries{
		Name: source.labels.x,
		Style: chart.Style{
			Show:        true,
			StrokeColor: drawing.Color{R: 255, G: 0, B: 0, A: 255},
			FillColor:   drawing.Color{R: 255, G: 0, B: 0, A: 255},
		},
		XValues: source.y,
		YValues: source.x,
	})
	if len(source.xv) > 0 {
		series = append(series, chart.ContinuousSeries{
			Name: source.labels.xv,
			Style: chart.Style{
				Show:        true,
				StrokeColor: drawing.Color{R: 0, G: 255, B: 0, A: 255},
				StrokeWidth: 1.5,
			},
			XValues: source.y,
			YValues: source.xv,
		})
	}
	if len(source.xgdr) > 0 {
		series = append(series, chart.ContinuousSeries{
			Name: source.labels.xgdr,
			Style: chart.Style{
				Show:        true,
				StrokeColor: drawing.Color{R: 0, G: 0, B: 0, A: 255},
				StrokeWidth: 1.5,
			},
			XValues: source.y,
			YValues: source.xgdr,
		})
	}
	if source.waterline > 0 {
		series = append(series, chart.ContinuousSeries{
			Name: source.labels.waterline,
			Style: chart.Style{
				Show:        true,
				StrokeColor: drawing.Color{R: 0, G: 0, B: 255, A: 255},
				StrokeWidth: 1.0,
			},
			XValues: []float64{source.y[0], source.y[len(source.y)-1]},
			YValues: []float64{source.waterline, source.waterline},
		})
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
				FontSize: 7.0,
			},
			TickPosition:   chart.TickPositionBetweenTicks,
			ValueFormatter: source.valueFormatter,
		},
		YAxis: chart.YAxis{
			Style: chart.Style{
				Show:     true,
				FontSize: 7.0,
			},
			Range: &chart.ContinuousRange{
				Max: source.maximum.chart,
				Min: source.minimum.chart,
			},
		},
		YAxisSecondary: chart.YAxis{
			Style: chart.Style{
				Show:     true,
				FontSize: 7.0,
			},
			Range: &chart.ContinuousRange{
				Max: source.maximum.chart,
				Min: source.minimum.chart,
			},
		},
		Series: series,
	}
	graph.Elements = []chart.Renderable{
		chart.LegendThin(&graph),
	}

	graph.Render(chart.PNG, buffer)

	return buffer
}
