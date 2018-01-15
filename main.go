package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/nsf/termbox-go"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"sort"
	"strings"
	"sync"
	"time"
)

type jsonStock struct {
	Data [][]float64 `json:"d"`
}
type jsonChange struct {
	Base  string             `json:"base"`
	Date  string             `json:"date"`
	Rates map[string]float64 `json:"rates"`
}
type tradeData struct {
	prices []float64
	dates  []float64
	values []float64
}

const (
	optionsValue                  float64           = 1775
	optionsVesting                float64           = 19.6
	coldef                        termbox.Attribute = termbox.ColorDefault
	graphFontSize                                   = 7.0
	reqPriceDayType               string            = "POST"
	reqPriceDayURL                string            = "http://charts.londonstockexchange.com/WebCharts/services/ChartWService.asmx/GetPrices"
	reqPriceDayBody               string            = `{"request":{"SampleTime":"1mm","TimeFrame":"1d","RequestedDataSetType":"ohlc","ChartPriceType":"price","Key":"MAIL.LID","OffSet":-60,"FromDate":null,"ToDate":null,"UseDelay":true,"KeyType":"Topic","KeyType2":"Topic","Language":"en"}}`
	reqPriceDayName               string            = "Today trades"
	reqChangeType                 string            = "GET"
	reqChangeURL                  string            = "https://api.fixer.io/latest?base=USD&symbols=RUB"
	reqChangeBody                 string            = ``
	reqChangeName                 string            = "USD exchange"
	reqPriceWithVolumeMonthlyType string            = "POST"
	reqPriceWithVolumeMonthlyURL  string            = "http://charts.londonstockexchange.com/WebCharts/services/ChartWService.asmx/GetDocsWithVolume"
	reqPriceWithVolumeMonthlyBody string            = `{"request":{"SampleTime":"1d","TimeFrame":"1m","RequestedDataSetType":"documental","ChartPriceType":"price","Key":"MAIL.LID","OffSet":0,"FromDate":null,"ToDate":null,"UseDelay":true,"KeyType":"Topic","KeyType2":"Topic","Docs":[""],"Language":"en"}}`
	reqPriceWithVolumeMonthlyName string            = "Last month trades"
	reqPriceWithVolumeYearlyType  string            = "POST"
	reqPriceWithVolumeYearlyURL   string            = "http://charts.londonstockexchange.com/WebCharts/services/ChartWService.asmx/GetDocsWithVolume"
	reqPriceWithVolumeYearlyBody  string            = `{"request":{"SampleTime":"1w","TimeFrame":"1y","RequestedDataSetType":"documental","ChartPriceType":"price","Key":"MAIL.LID","OffSet":0,"FromDate":null,"ToDate":null,"UseDelay":true,"KeyType":"Topic","KeyType2":"Topic","Docs":[""],"Language":"en"}}`
	reqPriceWithVolumeYearlyName  string            = "Last year trades"
	reqPriceWithVolumeAlltimeType string            = "POST"
	reqPriceWithVolumeAlltimeURL  string            = "http://charts.londonstockexchange.com/WebCharts/services/ChartWService.asmx/GetDocsWithVolume"
	reqPriceWithVolumeAlltimeBody string            = `{"request":{"SampleTime":"1w","TimeFrame":"5y","RequestedDataSetType":"documental","ChartPriceType":"price","Key":"MAIL.LID","OffSet":0,"FromDate":null,"ToDate":null,"UseDelay":true,"KeyType":"Topic","KeyType2":"Topic","Docs":[""],"Language":"en"}}`
	reqPriceWithVolumeAlltimeName string            = "Last 5 years trades"
)

var (
	done          = make(chan string)
	exit          = make(chan string)
	wg            sync.WaitGroup
	waitRequest   sync.WaitGroup
	mu            sync.Mutex
	requestStatus = map[string]map[string]string{}
	sizeX         int
	sizeY         int
	paddingLeft   int
	graphType     int
	dollar        float64
	lastprice     float64
)

func main() {
	f, _ := os.OpenFile("/var/log/self/gdr.log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	defer f.Close()
	log.SetOutput(f)

	termbox.Init()
	termbox.SetOutputMode(termbox.OutputMode(termbox.OutputNormal))
	sizeX, sizeY = termbox.Size()
	defer termbox.Close()

	wg.Add(2)
	go spinner()
	go downloader(true)
	go repeat()

loop:
	for {
		switch ev := termbox.PollEvent(); ev.Type {
		case termbox.EventKey:
			switch ev.Key {
			case termbox.KeySpace:
				if graphType == 0 || graphType == 1 {
					graphType = 2
				} else if graphType == 2 {
					graphType = 3
				} else if graphType == 3 {
					graphType = 4
				} else {
					graphType = 1
				}
				renderGraph()
			default:
				close(done)
				close(exit)
				break loop
			}
		}
	}

	wg.Wait()
}

func repeat() {
	defer wg.Done()
	const (
		timeout = 5 * 60 * time.Second
	)

	for {
		select {
		case <-exit:
			return
		case <-time.After(timeout):
			downloader(false)
		}
	}
}

func spinner() {
	var (
		spin   int
		strBeg = "load "
		xbeg   = sizeX/2 - (len(strBeg)+5)/2
		ybeg   = sizeY / 2
	)
	defer wg.Done()

	for {
		select {
		case <-done:
			return
		case <-time.After(300 * time.Millisecond):
			spin++
			termbox.Clear(coldef, coldef)
			printStatus()
			fmt.Printf("\x1b[%d;%dH%s", ybeg, xbeg, strBeg+strings.Repeat(".", spin))
			termbox.Flush()
			if spin >= 5 {
				spin = 0
			}
		}
	}
}

func downloader(first bool) {
	waitRequest.Add(5)

	go request(reqPriceDayType, reqPriceDayURL, reqPriceDayBody, reqPriceDayName)
	go request(reqChangeType, reqChangeURL, reqChangeBody, reqChangeName)
	go request(reqPriceWithVolumeMonthlyType, reqPriceWithVolumeMonthlyURL, reqPriceWithVolumeMonthlyBody, reqPriceWithVolumeMonthlyName)
	go request(reqPriceWithVolumeYearlyType, reqPriceWithVolumeYearlyURL, reqPriceWithVolumeYearlyBody, reqPriceWithVolumeYearlyName)
	go request(reqPriceWithVolumeAlltimeType, reqPriceWithVolumeAlltimeURL, reqPriceWithVolumeAlltimeBody, reqPriceWithVolumeAlltimeName)

	waitRequest.Wait()

	if first {
		done <- "ok"
	}

	termbox.Clear(coldef, coldef)
	fmt.Printf("\x1b[2J\x1b[%d;%dH", 0, 0)
	paddingLeft = printForecast()

	renderGraph()

	printInfo()
	termbox.Flush()

	return
}

func request(method, url, data, name string) {
	defer waitRequest.Done()

	var (
		resp *http.Response
		err  error
	)
	setStatus(name, url)

	setStatus(name, "send")
	if method == "POST" {
		resp, err = http.Post(url, "application/json", strings.NewReader(data))
	} else {
		resp, err = http.Get(url)
	}

	if err != nil {
		log.Panic(err)
	}

	body, _ := ioutil.ReadAll(resp.Body)
	if resp.StatusCode == 200 {
		setStatus(name, "ok")
		if name == reqChangeName {
			jsonInterface := new(jsonChange)
			err = json.Unmarshal(body, jsonInterface)
			if err != nil {
				log.Printf("JSON error: %s - %s", name, err)
			} else {
				dollar = jsonInterface.Rates["RUB"]
			}
		} else {
			jsonInterface := new(jsonStock)
			err = json.Unmarshal(body, jsonInterface)
			if err != nil {
				log.Printf("JSON error: %s - %s", name, err)
			} else if len(jsonInterface.Data) == 0 {
				log.Printf("NO DATA FOR %s: %+v", name, jsonInterface)
			} else {
				switch name {
				case reqPriceDayName:
					dayRequestCallback(jsonInterface)
				case reqPriceWithVolumeMonthlyName:
					defaultRequestCallback(jsonInterface)
				case reqPriceWithVolumeYearlyName:
					yearRequestCallback(jsonInterface)
				case reqPriceWithVolumeAlltimeName:
					allRequestCallback(jsonInterface)
				}
			}
		}
		setStatus(name, "done")
	} else {
		log.Printf("%s: request %s failed, status %s, response: %s", name, url, resp.Status, body[:])
		setStatus(name, "error")
	}

	return
}

func setStatus(name, status string) {
	mu.Lock()
	if requestStatus[name] == nil {
		requestStatus[name] = map[string]string{
			"name":   name,
			"url":    status,
			"status": "start",
		}
	} else {
		requestStatus[name]["status"] = status
	}
	mu.Unlock()
}
func printStatus() (str string) {
	fmt.Print("\x1b[2J\x1b[0;0H")
	var keys []string
	for k := range requestStatus {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, name := range keys {
		if requestStatus[name]["status"] == "error" {
			fmt.Printf("%s (%s) \x1b[05;31m%s\x1b[0m |\n", requestStatus[name]["url"], name, requestStatus[name]["status"])
		} else {
			fmt.Printf("%s (%s) \x1b[05;32m%s\x1b[0m\n", requestStatus[name]["url"], name, requestStatus[name]["status"])
		}
	}
	return
}

func approximate(max, min float64, arr []float64) (ret []float64) {
	min1, max1 := minmax(arr)

	C := (max - min) / (max1 - min1)
	for _, e := range arr {
		ret = append(ret, min+((e-min)*C))
	}

	return
}

func renderGraph() {
	const (
		status1 = "\u2776 \u2781 \u2782 \u2783 за последний месяц"
		status2 = "\u2780 \u2777 \u2782 \u2783 за последний год"
		status3 = "\u2780 \u2781 \u2778 \u2783 сегодня"
		status4 = "\u2780 \u2781 \u2782 \u2779 за пять лет"
	)
	var (
		image       *bytes.Buffer
		imageStatus string
		imageWidth  int = (sizeX - paddingLeft) * 7
		imageHeight int = (sizeY - 4) * 15
		eraseLength int
	)
	for _, e := range []int{len(status1), len(status2), len(status3), len(status4)} {
		if eraseLength < e {
			eraseLength = e
		}
	}

	if os.Getenv("TERM_PROGRAM") == "iTerm.app" {
		if graphType == 2 {
			image = renderYearGraph(imageWidth, imageHeight)
			imageStatus = status2
		} else if graphType == 3 {
			image = renderDayGraph(imageWidth, imageHeight)
			imageStatus = status3
		} else if graphType == 4 {
			image = renderAlltimeGraph(imageWidth, imageHeight)
			imageStatus = status4
		} else {
			image = renderDefaultGraph(imageWidth, imageHeight)
			imageStatus = status1
		}
		fmt.Printf("\x1b[0;%dH%s", paddingLeft+1, encodeBuffer(image))
		fmt.Printf("\x1b[%d;%dH%s", sizeY-3, int(sizeX/2)-1, strings.Repeat(" ", eraseLength))
		fmt.Printf("\x1b[%d;%dH%s", sizeY-3, int(sizeX/2)-1, imageStatus)
	} else {
		fmt.Printf("\x1b[0;%dH%s", paddingLeft+1, "\tУвы, картинки не картинки в этом терминале")
	}

	return
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

func encodeBuffer(buf *bytes.Buffer) string {
	str := base64.StdEncoding.EncodeToString(buf.Bytes())
	return fmt.Sprintf("\x1b]1337;File=name=none;size=%d;inline=1:%s\a\n", len(str), str)
}

func printForecast() (padding int) {
	const (
		colorDef   = "\x1b[0m"
		colorCol   = "\x1b[48;05;242m"
		colorRed   = "\x1b[48;05;196m"
		colorGreen = "\x1b[48;05;34m"
		start      = 27.0
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
		goodprice = (1.65*1000000)/(dollar*1775) + optionsVesting
	)
	for price := start; price < start+float64(sizeY-4)/2; price = price + step {
		value = optionsValue * (price - optionsVesting)
		rvalue = value * dollar / 1000
		if price >= lastprice*mul && price < lastprice*mul+step {
			color = colorGreen
		} else if price >= goodprice && price < goodprice+step {
			color = colorRed
		} else if even {
			color = colorDef
		} else {
			color = colorCol
		}
		even = !even

		col = fmt.Sprintf("%.2f: % 6s  % 5s", price, ranges(value, ","), ranges(rvalue, ","))
		collength = len(col)
		if collength > padding {
			padding = collength
		}

		fmt.Print(color, col, colorDef, "\n")
	}

	return
}

func printInfo() {
	var (
		gdr            = defaultGdr.prices[len(defaultGdr.prices)-1]
		gdrForecast    = getGdr(append(defaultData.prices, lastprice), append(defaultData.values, defaultData.values[len(defaultData.values)-1]))
		dprice         = gdr * lastprice
		rprice         = dprice * dollar
		rpriceForecast = gdrForecast * lastprice * dollar
		smile          = string(128512)
		smileValue     = lastprice - defaultData.prices[len(defaultData.prices)-1]
		smileValueS    = "+"
	)
	if smileValue < 0 {
		smile = string(128545)
		smileValue = 0 - smileValue
		smileValueS = "-"
	}
	fmt.Printf(
		"\x1b[%d;0H\nСтоимость сейчас: %.2f %s  (%s%.2f) Последнее изменение %s\nGDR: %.2f (прогноз: %.2f => %s рублей)\nОбщая стоимость: %s доллара (%s рублей при курсе %.2f)",
		sizeY-3,
		lastprice,
		smile,
		smileValueS,
		smileValue,
		time.Now().Format("15:04:05"),
		gdr,
		gdrForecast,
		ranges(rpriceForecast, " "),
		ranges(dprice, " "),
		ranges(rprice, " "),
		dollar,
	)
}

func ranges(i float64, divider string) string {
	var out = ""
	for ; i >= 1000.0; i = i / 1000.0 {
		out = fmt.Sprintf("%s%03d", divider, int(i)%1000) + out
	}
	return fmt.Sprintf("%d%s", int(i), out)
}

func getGdr(prices, values []float64) float64 {
	var (
		indexPrices = len(prices) - 1
		indexValues = len(values) - 1
		count       float64
		summ        float64
		i           int
	)
	for ; i < 3; i++ {
		count = count + prices[indexPrices-i]*values[indexValues-i]
		summ = summ + values[indexValues-i]
	}

	return optionsValue - (optionsValue * optionsVesting / (count / summ))
}
