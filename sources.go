package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
)

type JsonStock struct {
	Data  [][]float64        `json:"d"`
	Base  string             `json:"base"`
	Date  string             `json:"date"`
	Rates map[string]float64 `json:"rates"`
}

type Source struct {
	url      string
	method   string
	postdata string
	status   string
	index    int
	process  func(*JsonStock) []GraphData
}

func InitSource(options ...string) (self *Source) {
	self = new(Source)

	if options[0] == "GET" {
		self.method = options[0]
		self.url = options[1]
	} else {
		self.method = "POST"
		self.postdata = options[0]
		self.url = options[1]
	}

	return self
}

func (self *Source) load() (data []GraphData, err error) {
	self.setStatus("load", 33)
	data, err = self.get()
	if err != nil {
		self.setStatus("error", 31)
	} else {
		self.setStatus("done", 32)
	}

	return data, err
}

func (self *Source) get() (data []GraphData, err error) {
	var (
		resp *http.Response
	)

	if self.postdata == "" {
		resp, err = http.Get(self.url)
	} else {
		resp, err = http.Post(self.url, "application/json", strings.NewReader(self.postdata))
	}
	if err != nil {
		log.Println("http request error:", err)
	} else {
		body, _ := ioutil.ReadAll(resp.Body)
		if resp.StatusCode == 200 {
			jsonInterface := new(JsonStock)
			err = json.Unmarshal(body, jsonInterface)
			if err != nil {
				log.Printf("JSON error: %s - %s", self.url, err)
			} else {
				if self.process != nil {
					data = self.process(jsonInterface)
				}
			}
		} else {
			err = errors.New(fmt.Sprintf("non-200 response for %s, data: %s", self.url, self.postdata))
			log.Printf("request %s failed, status %s, response: %s", self.url, resp.Status, body[:])
		}
	}

	return
}
func (self *Source) setStatus(name string, color ...int) {
	var statusColor int
	if name == "error" {
		self.status = fmt.Sprintf("\x1b[05;31m%s\x1b[0m", name)
	} else {
		if len(color) == 0 {
			statusColor = 32
		} else {
			statusColor = color[0]
		}

		self.status = fmt.Sprintf("\x1b[05;%dm%s\x1b[0m", statusColor, name)
	}
	statusString := fmt.Sprintf("%s: %s", self.url, self.status)

	if self.index == 0 {
		mu.Lock()
		loadingBuffer = append(loadingBuffer, statusString)
		self.index = len(loadingBuffer)
		mu.Unlock()
	}

	loadingBuffer[self.index-1] = statusString

	return
}

func wrapper(data ...GraphData) (ret []GraphData) {
	return data
}

func daysCallback(jsonInterface *JsonStock) []GraphData {
	var (
		month     = new(GraphData)
		year      = new(GraphData)
		lastMonth = len(jsonInterface.Data) - 31
	)

	for i, e := range jsonInterface.Data {
		date := e[0] * 1000000
		year.setValues(date, e[1], e[6])
		if i > lastMonth {
			month.setValues(date, e[1], e[6])
			gdr := year.getGdr()
			month.setGdr(gdr)
		}
	}

	return wrapper(*month, *year)
}
func weeksCallback(jsonInterface *JsonStock) []GraphData {
	var (
		fiveyears = new(GraphData)
	)

	for _, e := range jsonInterface.Data {
		fiveyears.setValues(e[0]*1000000, e[1])
	}

	return wrapper(*fiveyears)
}
func hoursCallback(jsonInterface *JsonStock) []GraphData {
	var (
		today = new(GraphData)
	)

	for _, e := range jsonInterface.Data {
		today.setValues(e[0]*1000000, e[1], e[6])
	}

	return wrapper(*today)
}
func exchangeCallback(jsonInterface *JsonStock) []GraphData {
	dollar := new(GraphData)
	dollar.setValues(0, float64(jsonInterface.Rates["RUB"])/float64(jsonInterface.Rates["USD"]))

	return wrapper(*dollar)
}

func getSources() map[string]*Source {
	days := InitSource(makeStockData("1d", "1y"))
	days.process = daysCallback

	weeks := InitSource(makeStockData("1d", "5y"))
	weeks.process = weeksCallback

	hours := InitSource(makeStockData("1mm", "1d"))
	hours.process = hoursCallback

	exchange := InitSource("GET", "http://data.fixer.io/latest?symbols=RUB,USD&access_key=2c9d0b143d653c87830759e564b07708")
	exchange.process = exchangeCallback

	source := map[string]*Source{
		"days":     days,
		"weeks":    weeks,
		"hours":    hours,
		"exchange": exchange,
	}

	return source
}

func makeStockData(st, tf string) (data, url string) {
	return fmt.Sprintf(`{"request":{"SampleTime":"%s","TimeFrame":"%s","RequestedDataSetType":"ohlc","ChartPriceType":"price","Key":"MAIL.LID","OffSet":-60,"FromDate":null,"ToDate":null,"UseDelay":true,"KeyType":"Topic","KeyType2":"Topic","Language":"en"}}`, st, tf), "http://charts.londonstockexchange.com/WebCharts/services/ChartWService.asmx/GetPricesWithVolume"
}
