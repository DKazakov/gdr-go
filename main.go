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

const (
	optionsValue                  float64           = 1775
	optionsVesting                float64           = 19.6
	coldef                        termbox.Attribute = termbox.ColorDefault
	reqPriceDayType               string            = "POST"
	reqPriceDayURL                string            = "http://charts.londonstockexchange.com/WebCharts/services/ChartWService.asmx/GetPrices"
	reqPriceDayBody               string            = `{"request":{"SampleTime":"1mm","TimeFrame":"1d","RequestedDataSetType":"ohlc","ChartPriceType":"price","Key":"MAIL.LID","OffSet":-60,"FromDate":null,"ToDate":null,"UseDelay":true,"KeyType":"Topic","KeyType2":"Topic","Language":"en"}}`
	reqPriceDayName               string            = "Now trades"
	reqChangeType                 string            = "GET"
	reqChangeURL                  string            = "https://api.fixer.io/latest?base=USD&symbols=RUB"
	reqChangeBody                 string            = ``
	reqChangeName                 string            = "USD exchange"
	reqPriceWithVolumeMonthlyType string            = "POST"
	reqPriceWithVolumeMonthlyURL  string            = "http://charts.londonstockexchange.com/WebCharts/services/ChartWService.asmx/GetDocsWithVolume"
	reqPriceWithVolumeMonthlyBody string            = `{"request":{"SampleTime":"1d","TimeFrame":"1m","RequestedDataSetType":"documental","ChartPriceType":"price","Key":"MAIL.LID","OffSet":0,"FromDate":null,"ToDate":null,"UseDelay":true,"KeyType":"Topic","KeyType2":"Topic","Docs":[""],"Language":"en"}}`
	reqPriceWithVolumeMonthlyName string            = "Monthly trades"
	reqPriceWithVolumeYearlyType  string            = "POST"
	reqPriceWithVolumeYearlyURL   string            = "http://charts.londonstockexchange.com/WebCharts/services/ChartWService.asmx/GetDocsWithVolume"
	reqPriceWithVolumeYearlyBody  string            = `{"request":{"SampleTime":"1w","TimeFrame":"1y","RequestedDataSetType":"documental","ChartPriceType":"price","Key":"MAIL.LID","OffSet":0,"FromDate":null,"ToDate":null,"UseDelay":true,"KeyType":"Topic","KeyType2":"Topic","Docs":[""],"Language":"en"}}`
	reqPriceWithVolumeYearlyName  string            = "Yearly trades"
	reqPriceWithVolumeAlltimeType string            = "POST"
	reqPriceWithVolumeAlltimeURL  string            = "http://charts.londonstockexchange.com/WebCharts/services/ChartWService.asmx/GetDocsWithVolume"
	reqPriceWithVolumeAlltimeBody string            = `{"request":{"SampleTime":"1w","TimeFrame":"10y","RequestedDataSetType":"documental","ChartPriceType":"price","Key":"MAIL.LID","OffSet":0,"FromDate":null,"ToDate":null,"UseDelay":true,"KeyType":"Topic","KeyType2":"Topic","Docs":[""],"Language":"en"}}`
	reqPriceWithVolumeAlltimeName string            = "All time trades"
)

type jsonStock struct {
	Data [][]float64 `json:"d"`
}
type jsonChange struct {
	Base  string             `json:"base"`
	Date  string             `json:"date"`
	Rates map[string]float64 `json:"rates"`
}
type graphData struct {
	dates               []float64
	ydates              []float64
	atdates             []float64
	times               []float64
	monthly             []float64
	yearly              []float64
	daily               []float64
	all                 []float64
	current             []float64
	ycurrent            []float64
	dcurrent            []float64
	atcurrent           []float64
	gdr                 []float64
	approximatedgdr     []float64
	yapproximatedgdr    []float64
	gdrdates            []float64
	values              []float64
	yvalues             []float64
	approximatedvalues  []float64
	yapproximatedvalues []float64
	dollar              float64
	min                 float64
	ymin                float64
	dmin                float64
	atmin               float64
	max                 float64
	ymax                float64
	dmax                float64
	atmax               float64
	lastprice           float64
	dayprice            float64
	minprice            float64
	yminprice           float64
	dminprice           float64
	atminprice          float64
	maxprice            float64
	ymaxprice           float64
	dmaxprice           float64
	atmaxprice          float64
	minvalue            float64
	yminvalue           float64
	maxvalue            float64
	ymaxvalue           float64
	mingdr              float64
	maxgdr              float64
}

var (
	done          = make(chan string)
	exit          = make(chan string)
	wg            sync.WaitGroup
	waitRequest   sync.WaitGroup
	mu            sync.Mutex
	requestStatus = map[string]map[string]string{}
	dataMap       = new(graphData)
	sizeX         int
	sizeY         int
	paddingLeft   int
	graphType     int
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
	if len(dataMap.daily) == 0 {
		fmt.Printf("\x1b[%d;%dH\x1b[05;196mОшибка!!!!: %s\x1b[0m", sizeY-3, 0, time.Now().Format("15:04:05"))
	} else if dataMap.lastprice != dataMap.daily[len(dataMap.daily)-1] {
		postprocessData()

		termbox.Clear(coldef, coldef)
		fmt.Printf("\x1b[2J\x1b[%d;%dH", 0, 0)
		paddingLeft = printForecast()

		renderGraph()

		printInfo()
		termbox.Flush()

	} else {
		fmt.Printf("\x1b[%d;%dH\x1b[05;32mОбновлено: %s\x1b[0m", sizeY-3, 0, time.Now().Format("15:04:05"))
	}

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
				dataMap.dollar = jsonInterface.Rates["RUB"]
			}
		} else {
			jsonInterface := new(jsonStock)
			err = json.Unmarshal(body, jsonInterface)
			if err != nil {
				log.Printf("JSON error: %s - %s", name, err)
			} else {
				switch name {
				case reqPriceDayName:
					dataMap.times = []float64{}
					dataMap.daily = []float64{}
					for _, now := range jsonInterface.Data {
						dataMap.daily = append(dataMap.daily, now[1])

						time := now[0] * 1000000
						dataMap.times = append(dataMap.times, time)
					}
				case reqPriceWithVolumeMonthlyName:
					dataMap.dates = []float64{}
					dataMap.monthly = []float64{}
					dataMap.values = []float64{}
					dataMap.gdr = []float64{}
					dataMap.gdrdates = []float64{}
					for i := 0; i < len(jsonInterface.Data); i++ {
						date := jsonInterface.Data[i][0] * 1000000
						dataMap.monthly = append(dataMap.monthly, jsonInterface.Data[i][1])
						dataMap.dates = append(dataMap.dates, date)
						dataMap.values = append(dataMap.values, jsonInterface.Data[i][2])
						if i > 1 {
							dataMap.gdr = append(dataMap.gdr, getGdr(dataMap.monthly, dataMap.values))
							dataMap.gdrdates = append(dataMap.gdrdates, date)
						}
					}
				case reqPriceWithVolumeYearlyName:
					dataMap.ydates = []float64{}
					dataMap.yearly = []float64{}
					dataMap.yvalues = []float64{}
					for i := 0; i < len(jsonInterface.Data); i++ {
						date := jsonInterface.Data[i][0] * 1000000
						dataMap.yearly = append(dataMap.yearly, jsonInterface.Data[i][1])
						dataMap.ydates = append(dataMap.ydates, date)
						dataMap.yvalues = append(dataMap.yvalues, jsonInterface.Data[i][2])
					}
				case reqPriceWithVolumeAlltimeName:
					dataMap.atdates = []float64{}
					dataMap.all = []float64{}
					for i := 0; i < len(jsonInterface.Data); i++ {
						date := jsonInterface.Data[i][0] * 1000000

						if i > 0 {
							if jsonInterface.Data[i][1] < jsonInterface.Data[i-1][1]*2 {
								dataMap.all = append(dataMap.all, jsonInterface.Data[i][1])
							} else {
								dataMap.all = append(dataMap.all, jsonInterface.Data[i-1][1])
								log.Printf("Skip wrong price: %+v", jsonInterface.Data[i])
							}
						} else {
							dataMap.all = append(dataMap.all, jsonInterface.Data[i][1])
						}
						dataMap.atdates = append(dataMap.atdates, date)
					}
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

func postprocessData() {
	dataMap.lastprice = dataMap.daily[len(dataMap.daily)-1]
	dataMap.dayprice = dataMap.monthly[len(dataMap.monthly)-1]

	dataMap.current = []float64{}
	for i := 0; i < len(dataMap.monthly); i++ {
		dataMap.current = append(dataMap.current, dataMap.lastprice)
	}
	dataMap.ycurrent = []float64{}
	for i := 0; i < len(dataMap.yearly); i++ {
		dataMap.ycurrent = append(dataMap.ycurrent, dataMap.lastprice)
	}
	dataMap.dcurrent = []float64{}
	for i := 0; i < len(dataMap.daily); i++ {
		dataMap.dcurrent = append(dataMap.dcurrent, dataMap.dayprice)
	}
	dataMap.atcurrent = []float64{}
	for i := 0; i < len(dataMap.all); i++ {
		dataMap.atcurrent = append(dataMap.atcurrent, dataMap.lastprice)
	}
	var (
		minprice, maxprice     = minmax(dataMap.monthly)
		yminprice, ymaxprice   = minmax(dataMap.yearly)
		dminprice, dmaxprice   = minmax(dataMap.daily)
		atminprice, atmaxprice = minmax(dataMap.all)
		minvalue, maxvalue     = minmax(dataMap.values)
		yminvalue, ymaxvalue   = minmax(dataMap.yvalues)
		mingdr, maxgdr         = minmax(dataMap.gdr)
	)

	valuesKoefficient := (maxprice - minprice) / (maxvalue - minvalue)
	dataMap.approximatedvalues = []float64{}
	for _, e := range dataMap.values {
		dataMap.approximatedvalues = append(dataMap.approximatedvalues, minprice+((e-minvalue)*valuesKoefficient))
	}

	yvaluesKoefficient := (ymaxprice - yminprice) / (ymaxvalue - yminvalue)
	dataMap.yapproximatedvalues = []float64{}
	for _, e := range dataMap.yvalues {
		dataMap.yapproximatedvalues = append(dataMap.yapproximatedvalues, yminprice+((e-yminvalue)*yvaluesKoefficient))
	}

	gdrKoefficient := (maxprice - minprice) / (maxgdr - mingdr)
	dataMap.approximatedgdr = []float64{}
	for _, e := range dataMap.gdr {
		dataMap.approximatedgdr = append(dataMap.approximatedgdr, minprice+((e-mingdr)*gdrKoefficient))
	}

	dataMap.min, dataMap.max = minmax([]float64{maxprice + 0.5, minprice - 0.5, dataMap.current[0] + 0.5, dataMap.current[0] - 0.5})
	dataMap.ymin, dataMap.ymax = minmax([]float64{ymaxprice + 0.5, yminprice - 0.5, dataMap.current[0] + 0.5, dataMap.current[0] - 0.5})
	dataMap.dmin, dataMap.dmax = minmax([]float64{dmaxprice + 0.5, dminprice - 0.5, dataMap.current[0] + 0.5, dataMap.current[0] - 0.5})
	dataMap.atmin, dataMap.atmax = minmax([]float64{atmaxprice + 0.5, atminprice - 0.5, dataMap.current[0] + 0.5, dataMap.current[0] - 0.5})

	dataMap.minprice, dataMap.maxprice, dataMap.minvalue, dataMap.maxvalue, dataMap.mingdr, dataMap.maxgdr = minprice, maxprice, minvalue, maxvalue, mingdr, maxgdr
	dataMap.yminprice, dataMap.ymaxprice, dataMap.yminvalue, dataMap.ymaxvalue = yminprice, ymaxprice, yminvalue, ymaxvalue
	dataMap.dminprice, dataMap.dmaxprice = dminprice, dmaxprice
	dataMap.atminprice, dataMap.atmaxprice = atminprice, atmaxprice

	return
}
func renderGraph() {
	const (
		status1 = "\u2776 \u2781 \u2782 \u2783 за последний месяц"
		status2 = "\u2780 \u2777 \u2782 \u2783 за последний год"
		status3 = "\u2780 \u2781 \u2778 \u2783 сегодня"
		status4 = "\u2780 \u2781 \u2782 \u2779 за всё время"
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
		goodprice = (1.65*1000000)/(dataMap.dollar*1775) + optionsVesting
	)
	for price := start; price < start+float64(sizeY-4)/2; price = price + step {
		value = optionsValue * (price - optionsVesting)
		rvalue = value * dataMap.dollar / 1000
		if price >= dataMap.lastprice*mul && price < dataMap.lastprice*mul+step {
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
		gdr            = dataMap.gdr[len(dataMap.gdr)-1]
		gdrForecast    = getGdr(append(dataMap.monthly, dataMap.lastprice), append(dataMap.values, dataMap.values[len(dataMap.values)-1]))
		dprice         = gdr * dataMap.lastprice
		rprice         = dprice * dataMap.dollar
		rpriceForecast = gdrForecast * dataMap.lastprice * dataMap.dollar
		smile          = string(128512)
		smileValue     = dataMap.lastprice - dataMap.monthly[len(dataMap.monthly)-1]
		smileValueS    = "+"
	)
	if dataMap.lastprice < dataMap.monthly[len(dataMap.monthly)-1] {
		smile = string(128545)
		smileValue = 0 - smileValue
		smileValueS = "-"
	}
	fmt.Printf(
		"\x1b[%d;0H\nСтоимость сейчас: %.2f %s  (%s%.2f) Последнее изменение %s\nGDR: %.2f (прогноз: %.2f => %s рублей)\nОбщая стоимость: %s доллара (%s рублей при курсе %.2f)",
		sizeY-3,
		dataMap.lastprice,
		smile,
		smileValueS,
		smileValue,
		time.Now().Format("15:04:05"),
		gdr,
		gdrForecast,
		ranges(rpriceForecast, " "),
		ranges(dprice, " "),
		ranges(rprice, " "),
		dataMap.dollar,
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
