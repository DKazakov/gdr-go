package main

import (
	"fmt"
	util "github.com/wcharczuk/go-chart/util"
)

type GraphDataLabels struct {
	x, xv, xgdr, waterline string
}
type Extremum struct {
	x, xv, xgdr, chart float64
}
type GraphData struct {
	name                       string
	x, y, _xv, xv, _xgdr, xgdr []float64
	waterline                  float64
	labels                     *GraphDataLabels
	maximum, minimum           *Extremum
	valueFormatter             func(interface{}) string
}

func (self *GraphData) setValues(y float64, x ...float64) {
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
func (self *GraphData) setExtremum() {
	var (
		min = new(Extremum)
		max = new(Extremum)
	)

	min.x, max.x = minmax(self.x)
	if len(self._xv) > 0 {
		min.xv, max.xv = minmax(self._xv)
	}
	if len(self._xgdr) > 0 {
		min.xgdr, max.xgdr = minmax(self._xgdr)
	}
	delta := (max.x - min.x) * 0.01
	min.chart, max.chart = minmax([]float64{max.x + delta, min.x - delta, self.waterline + 0.5, self.waterline - 0.5})
	self.maximum = max
	self.minimum = min

	return
}
func (self *GraphData) setGdr(gdr float64) {
	if gdr > 0 {
		self._xgdr = append(self._xgdr, gdr)
	}

	return
}
func (self GraphData) getGdr(next ...float64) (gdr float64) {
	var (
		count float64
		summ  float64
		index = len(self._xv)
		i     = index - 3
	)
	if i > 0 {
		if len(next) > 0 {
			count = count + next[0]*next[1]
			summ = summ + next[1]
			i = i + 1
		}
		for ; i < index; i++ {
			count = count + self.x[i]*self._xv[i]
			summ = summ + self._xv[i]
		}
		gdr = optionsValue - (optionsValue * optionsVesting / (count / summ))
	}

	return gdr
}

func (self *GraphData) finalize(waterline float64, formatType string) {
	self.waterline = waterline
	self.setExtremum()
	labels := new(GraphDataLabels)

	if len(self.x) > 0 {
		labels.x = fmt.Sprintf("price, max: %.2f, min: %.2f, last: %.2f", self.maximum.x, self.minimum.x, self.x[len(self.x)-1])
	}

	if len(self._xv) > 0 {
		C := (self.maximum.x - self.minimum.x) / (self.maximum.xv - self.minimum.xv)
		for _, e := range self._xv {
			self.xv = append(self.xv, self.minimum.x+((e-self.minimum.xv)*C))
		}
		labels.xv = fmt.Sprintf("scaled value, max: %.3fkk, min: %.3fkk", self.maximum.xv/1000000, self.minimum.xv/1000000)
	}
	if len(self._xgdr) > 0 {
		C := (self.maximum.x - self.minimum.x) / (self.maximum.xgdr - self.minimum.xgdr)
		for _, e := range self._xgdr {
			self.xgdr = append(self.xgdr, self.minimum.x+((e-self.minimum.xgdr)*C))
		}
		labels.xgdr = fmt.Sprintf("scaled GDR's, max: %.2f, min: %.2f", self.maximum.xgdr, self.minimum.xgdr)
	}

	labels.waterline = fmt.Sprintf("current price %.2f", waterline)

	self.valueFormatter = daysValueFormatter
	if formatType == "hours" {
		self.valueFormatter = hoursValueFormatter
		labels.waterline = fmt.Sprintf("last closing price %.2f", waterline)
	} else if formatType == "months" {
		self.valueFormatter = monthsValueFormatter
	}
	self.labels = labels

	return
}

func monthsValueFormatter(v interface{}) string {
	typed := v.(float64)
	typedDate := util.Time.FromFloat64(typed)
	return fmt.Sprintf("%.2d.%.2d", typedDate.Month(), typedDate.Year())
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
	if len(array) > 0 {
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
	}

	return
}

type Data struct {
	graph                [4]GraphData
	gdr, gdrForecast     float64
	lastprice, lastclose float64
	dollar               float64
}

func (self *Data) Init() *Data {
	self = new(Data)

	return self
}

func (self *Data) set(name string, data []GraphData) {
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

func (self *Data) finalize() int {
	var (
		avg float64
		i   int
	)

	if len(self.graph[1].x) > 0 {
		self.lastclose = self.graph[1].x[len(self.graph[1].x)-1]
	} else {
		self.lastclose = 0
	}
	if len(self.graph[0].x) > 0 {
		self.lastprice = self.graph[0].x[len(self.graph[0].x)-1]
	} else {
		self.lastprice = self.lastclose
	}

	self.graph[0].finalize(self.lastclose, "hours")
	self.graph[1].finalize(self.lastprice, "days")
	self.graph[2].finalize(self.lastprice, "days")
	self.graph[3].finalize(self.lastprice, "months")

	self.gdr = self.graph[1]._xgdr[len(self.graph[1]._xgdr)-1]

	for ; i < len(self.graph[1].xv); i++ {
		avg = avg + self.graph[1]._xv[i]
	}
	avg = avg / float64(i)
	self.gdrForecast = self.graph[1].getGdr(self.lastprice, avg)

	return len(self.graph)
}
