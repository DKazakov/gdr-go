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

type jsonStock struct {
	Data  [][]float64        `json:"d"`
	Base  string             `json:"base"`
	Date  string             `json:"date"`
	Rates map[string]float64 `json:"rates"`
}

type source struct {
	url      string
	method   string
	postdata string
	status   string
	index    int
	process  func(*jsonStock) []graphData
}

func InitSource(options ...string) (self *source) {
	const defaulturl = "http://charts.londonstockexchange.com/WebCharts/services/ChartWService.asmx/GetDocsWithVolume"
	self = new(source)
	optlen := len(options)

	if optlen == 0 {
		self.method = "POST"
		self.url = defaulturl
		self.postdata = `{"request":{"SampleTime":"1d","TimeFrame":"1y","RequestedDataSetType":"documental","ChartPriceType":"price","Key":"MAIL.LID","OffSet":0,"FromDate":null,"ToDate":null,"UseDelay":true,"KeyType":"Topic","KeyType2":"Topic","Docs":[""],"Language":"en"}}`
	} else if optlen == 1 {
		self.method = "POST"
		self.url = defaulturl
		self.postdata = options[0]
	} else {
		if options[0] == "GET" {
			self.method = options[0]
			self.url = options[1]
		} else {
			self.method = "POST"
			self.postdata = options[0]
			self.url = options[1]
		}
	}

	return self
}

func (self *source) load() (data []graphData, err error) {
	self.setStatus("load")
	data, err = self.get()
	if err != nil {
		self.setStatus("error")
	} else {
		self.setStatus("done")
	}

	return data, err
}

func (self *source) get() (data []graphData, err error) {
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
			jsonInterface := new(jsonStock)
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
func (self *source) setStatus(name string) {
	if name == "error" {
		self.status = fmt.Sprintf("\x1b[05;31m%s\x1b[0m", name)
	} else {
		self.status = fmt.Sprintf("\x1b[05;32m%s\x1b[0m", name)
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

func wrapper(data ...graphData) (ret []graphData) {
	return data
}

func daysCallback(jsonInterface *jsonStock) []graphData {
	var (
		month     = new(graphData)
		year      = new(graphData)
		lastMonth = len(jsonInterface.Data) - 31
	)

	for i, e := range jsonInterface.Data {
		date := e[0] * 1000000
		year.setValues(date, e[1], e[2])
		if i > lastMonth {
			month.setValues(date, e[1], e[2])
			gdr := year.getGdr()
			month.setGdr(gdr)
		}
	}

	return wrapper(*month, *year)
}
func weeksCallback(jsonInterface *jsonStock) []graphData {
	var (
		fiveyears = new(graphData)
	)

	for _, e := range jsonInterface.Data {
		fiveyears.setValues(e[0]*1000000, e[1])
	}

	return wrapper(*fiveyears)
}
func hoursCallback(jsonInterface *jsonStock) []graphData {
	var (
		today = new(graphData)
	)

	for _, e := range jsonInterface.Data {
		today.setValues(e[0]*1000000, e[1])
	}

	return wrapper(*today)
}
func exchangeCallback(jsonInterface *jsonStock) []graphData {
	dollar := new(graphData)
	dollar.setValues(0, jsonInterface.Rates["RUB"])

	return wrapper(*dollar)
}

func getSources() map[string]*source {
	// daily	nil				- nil			post-default(postdata)-default(url)
	// weekly	string(DATA)	- nil			post-postdata-default(url)
	// hourly	string(DATA)	- string(URL)	post-postdata-url
	// change	GET				- string(URL)	get-url
	days := InitSource()
	days.process = daysCallback

	weeks := InitSource(`{"request":{"SampleTime":"1w","TimeFrame":"5y","RequestedDataSetType":"documental","ChartPriceType":"price","Key":"MAIL.LID","OffSet":0,"FromDate":null,"ToDate":null,"UseDelay":true,"KeyType":"Topic","KeyType2":"Topic","Docs":[""],"Language":"en"}}`)
	weeks.process = weeksCallback

	hours := InitSource(`{"request":{"SampleTime":"1mm","TimeFrame":"1d","RequestedDataSetType":"ohlc","ChartPriceType":"price","Key":"MAIL.LID","OffSet":-60,"FromDate":null,"ToDate":null,"UseDelay":true,"KeyType":"Topic","KeyType2":"Topic","Language":"en"}}`, "http://charts.londonstockexchange.com/WebCharts/services/ChartWService.asmx/GetPrices")
	hours.process = hoursCallback

	exchange := InitSource("GET", "https://api.fixer.io/latest?base=USD&symbols=RUB")
	exchange.process = exchangeCallback

	source := map[string]*source{
		"days":     days,
		"weeks":    weeks,
		"hours":    hours,
		"exchange": exchange,
	}

	return source
}
