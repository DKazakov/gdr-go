package main

import (
	"fmt"
	"time"
)

type Textinfo struct {
	gdr, gdrForecast     float64
	lastprice, lastclose float64
	lastupdate           string
	dollar               float64
	up                   bool
}

func (self *Textinfo) Init(data *Data) *Textinfo {
	self.gdr = data.gdr
	self.gdrForecast = data.gdrForecast
	self.lastprice = data.lastprice
	self.lastclose = data.lastclose
	self.dollar = data.dollar
	self.lastupdate = fmt.Sprintf("%.2d:%.2d:%.2d", data.lastupdate.Hour(), data.lastupdate.Minute(), data.lastupdate.Second())
	self.up = data.lastprice >= data.lastclose

	return self
}

func (self Textinfo) forecast(height int) (padding int) {
	const (
		colorDef   = "\x1b[0m"
		colorCol   = "\x1b[48;05;242m"
		colorRed   = "\x1b[48;05;196m"
		colorGreen = "\x1b[48;05;34m"
		step       = 0.5
		mul        = 0.993
	)
	var (
		color     string
		even      = true
		value     float64
		rvalue    float64
		col       string
		collength int
		goodprice = (1.65*1000000)/(self.dollar*1775) + optionsVesting
		start, _  = minmax([]float64{float64(int(self.lastprice - 3)), float64(int(goodprice - 3))})
	)
	for price := start; price < start+float64(height-4)/2; price = price + step {
		value = optionsValue * (price - optionsVesting)
		rvalue = value * self.dollar / 1000
		if price >= self.lastprice*mul && price < self.lastprice*mul+step {
			color = colorGreen
		} else if price >= goodprice && price < goodprice+step {
			color = colorRed
		} else if even {
			color = colorDef
		} else {
			color = colorCol
		}
		even = !even

		col = fmt.Sprintf("%.2f: % 6s  % 5s", price, self._ranges(value, ","), self._ranges(rvalue, ","))
		collength = len(col)
		if collength > padding {
			padding = collength
		}

		fmt.Print(color, col, colorDef, "\n")
	}
	return
}
func (self Textinfo) info(height int) int {
	const (
		smilegood  = string(128512)
		smilebad   = string(128545)
		infoHeight = 3
	)
	var (
		smile          string
		dprice         = self.gdr * self.lastprice
		rprice         = dprice * self.dollar
		rpriceForecast = self.gdrForecast * self.lastprice * self.dollar
	)
	if self.up {
		smile = fmt.Sprintf("%s  (%s%.2f)", smilegood, "+", self.lastprice-self.lastclose)
	} else {
		smile = fmt.Sprintf("%s  (%.2f)", smilebad, self.lastprice-self.lastclose)
	}

	fmt.Printf(
		"\x1b[%d;0H\nСтоимость сейчас: %.2f %s Последнее обновление %s, последняя попытка %s\nGDR: %.2f (прогноз: %.2f => %s рублей)\nОбщая стоимость: %s доллара (%s рублей при курсе %.2f)",
		height-infoHeight,
		self.lastprice,
		smile,
		self.lastupdate,
		time.Now().Format("15:04:05"),
		self.gdr,
		self.gdrForecast,
		self._ranges(rpriceForecast, " "),
		self._ranges(dprice, " "),
		self._ranges(rprice, " "),
		self.dollar,
	)
	return infoHeight
}
func (self Textinfo) _ranges(i float64, divider string) string {
	var out = ""
	for ; i >= 1000.0; i = i / 1000.0 {
		out = fmt.Sprintf("%s%03d", divider, int(i)%1000) + out
	}
	return fmt.Sprintf("%d%s", int(i), out)
}

func (self Textinfo) print(width, height int) (paddingLeft, paddingBottom int) {
	fmt.Printf("\x1b[0;0H")
	paddingLeft = self.forecast(height)
	paddingBottom = self.info(height)
	return paddingLeft, 4
}
