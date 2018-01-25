package main

import (
	"fmt"
	util "github.com/wcharczuk/go-chart/util"
)

type graphDataLabels struct {
	x, xv, xgdr, waterline string
}
type extremum struct {
	x, xv, xgdr, chart float64
}
type graphData struct {
	name                       string
	x, y, _xv, xv, _xgdr, xgdr []float64
	waterline                  float64
	labels                     *graphDataLabels
	maximum, minimum           *extremum
	valueFormatter             func(interface{}) string
}

func (self *graphData) setValues(y float64, x ...float64) {
	var prev float64
	next := x[0]
	lenx := len(self.x)

	if lenx > 0 {
		prev = self.x[lenx-1]
		if next > prev*2 {
			next = prev
		}
	}

	self.y = append(self.y, y)
	self.x = append(self.x, next)

	if len(x) > 1 {
		self._xv = append(self._xv, x[1])
	}
}
func (self *graphData) setExtremum() {
	var (
		min = new(extremum)
		max = new(extremum)
	)

	min.x, max.x = minmax(self.x)
	if len(self._xv) > 0 {
		min.xv, max.xv = minmax(self._xv)
	}
	if len(self._xgdr) > 0 {
		min.xgdr, max.xgdr = minmax(self._xgdr)
	}
	min.chart, max.chart = minmax([]float64{max.x + 0.5, min.x - 0.5, self.waterline + 0.5, self.waterline - 0.5})
	self.maximum = max
	self.minimum = min

	return
}
func (self *graphData) setGdr(gdr float64) {
	if gdr > 0 {
		self._xgdr = append(self._xgdr, gdr)
	}

	return
}
func (self graphData) getGdr(next ...float64) (gdr float64) {
	var (
		count float64
		summ  float64
		index = len(self._xv)
		i     = index - 3
	)
	if i > 0 {
		for ; i < index; i++ {
			count = count + self.x[i]*self._xv[i]
			summ = summ + self._xv[i]
		}
		if len(next) > 0 {
			count = count + next[0]*next[1]
			summ = summ + next[1]

		}
		gdr = optionsValue - (optionsValue * optionsVesting / (count / summ))
	}

	return gdr
}

func (self *graphData) finalize(waterline float64, formatType string) {
	self.waterline = waterline
	self.setExtremum()
	labels := new(graphDataLabels)

	labels.x = fmt.Sprintf("price, max: %.2f, min: %.2f", self.maximum.x, self.minimum.x)

	if len(self._xv) > 0 {
		C := (self.maximum.x - self.minimum.x) / (self.maximum.xv - self.minimum.xv)
		for _, e := range self._xv {
			self.xv = append(self.xv, self.minimum.x+((e-self.minimum.xv)*C))
		}
		labels.xv = fmt.Sprintf("scaled value, max: %.1fkk, min: %.1fkk", self.maximum.xv, self.minimum.xv)
	}
	if len(self._xgdr) > 0 {
		C := (self.maximum.x - self.minimum.x) / (self.maximum.xgdr - self.minimum.xgdr)
		for _, e := range self._xgdr {
			self.xgdr = append(self.xgdr, self.minimum.x+((e-self.minimum.xgdr)*C))
		}
		labels.xgdr = fmt.Sprintf("scaled GDR's, max: %.2f, min: %.2f", self.maximum.xgdr, self.minimum.xgdr)
	}

	labels.waterline = fmt.Sprintf("current price %.2f", waterline)

	if formatType == "hours" {
		self.valueFormatter = hoursValueFormatter
		labels.waterline = fmt.Sprintf("last closing price %.2f", waterline)
	} else {
		self.valueFormatter = daysValueFormatter
	}
	self.labels = labels

	return
}

func daysValueFormatter(v interface{}) string {
	typed := v.(float64)
	typedDate := util.Time.FromFloat64(typed)
	return fmt.Sprintf("%.2d.%.2d", typedDate.Day(), typedDate.Month())
}
func hoursValueFormatter(v interface{}) string {
	typed := v.(float64)
	typedDate := util.Time.FromFloat64(typed)
	return fmt.Sprintf("%.2d:%.2d", typedDate.Hour(), typedDate.Minute())
}

func minmax(array []float64) (min float64, max float64) {
	min = array[0]
	max = array[0]
	for _, e := range array {
		if max < e {
			max = e
		}
		if min > e {
			min = e
		}
	}
	return
}

type data struct {
	graph                [4]graphData
	gdr, gdrForecast     float64
	lastprice, lastclose float64
	dollar               float64
}

func (self *data) Init() *data {
	self = new(data)

	return self
}

func (self *data) set(name string, data []graphData) {
	switch name {
	case "days":
		data[0].name = "\u2780 \u2777 \u2782 \u2783 за последний месяц"
		self.graph[1] = data[0]
		data[1].name = "\u2780 \u2781 \u2778 \u2783 за последний год"
		self.graph[2] = data[1]
	case "weeks":
		data[0].name = "\u2780 \u2781 \u2782 \u2779 за пять лет"
		self.graph[3] = data[0]
	case "hours":
		data[0].name = "\u2776 \u2781 \u2782 \u2783 сегодня"
		self.graph[0] = data[0]
	case "exchange":
		self.dollar = data[0].x[0]
	default:
		break
	}
}

func (self *data) finalize() int {
	var (
		avg float64
		i   int
	)

	self.lastclose = self.graph[1].x[len(self.graph[1].x)-1]
	self.lastprice = self.graph[0].x[len(self.graph[0].x)-1]

	self.graph[0].finalize(self.lastclose, "hours")
	for i := 1; i < len(self.graph); i++ {
		self.graph[i].finalize(self.lastprice, "days")
	}

	self.gdr = self.graph[1]._xgdr[len(self.graph[1]._xgdr)-1]

	for ; i < len(self.graph[1].xv); i++ {
		avg = avg + self.graph[1].xv[i]
	}
	avg = avg / float64(i)
	self.gdrForecast = self.graph[0].getGdr(self.lastclose, avg)

	return len(self.graph)
}