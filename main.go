package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/nsf/termbox-go"
	"github.com/wcharczuk/go-chart"
	drawing "github.com/wcharczuk/go-chart/drawing"
	util "github.com/wcharczuk/go-chart/util"
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
	reqPriceDayName               string            = "now"
	reqChangeType                 string            = "GET"
	reqChangeURL                  string            = "https://api.fixer.io/latest?base=USD&symbols=RUB"
	reqChangeBody                 string            = ``
	reqChangeName                 string            = "change"
	reqPriceWithVolumeMonthlyType string            = "POST"
	reqPriceWithVolumeMonthlyURL  string            = "http://charts.londonstockexchange.com/WebCharts/services/ChartWService.asmx/GetDocsWithVolume"
	reqPriceWithVolumeMonthlyBody string            = `{"request":{"SampleTime":"1d","TimeFrame":"1m","RequestedDataSetType":"documental","ChartPriceType":"price","Key":"MAIL.LID","OffSet":0,"FromDate":null,"ToDate":null,"UseDelay":true,"KeyType":"Topic","KeyType2":"Topic","Docs":[""],"Language":"en"}}`
	reqPriceWithVolumeMonthlyName string            = "pricesandvolume"
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
	dates              []float64
	monthly            []float64
	daily              []float64
	current            []float64
	gdr                []float64
	approximatedgdr    []float64
	gdrdates           []float64
	values             []float64
	approximatedvalues []float64
	dollar             float64
	min                float64
	max                float64
	lastprice          float64
	minprice           float64
	maxprice           float64
	minvalue           float64
	maxvalue           float64
	mingdr             float64
	maxgdr             float64
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

	switch ev := termbox.PollEvent(); ev.Type {
	case termbox.EventKey:
		close(done)
		close(exit)
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
	waitRequest.Add(3)

	go request(reqPriceDayType, reqPriceDayURL, reqPriceDayBody, reqPriceDayName)
	go request(reqChangeType, reqChangeURL, reqChangeBody, reqChangeName)
	go request(reqPriceWithVolumeMonthlyType, reqPriceWithVolumeMonthlyURL, reqPriceWithVolumeMonthlyBody, reqPriceWithVolumeMonthlyName)

	waitRequest.Wait()
	if first {
		done <- "ok"
	}
	if dataMap.lastprice != dataMap.daily[len(dataMap.daily)-1] {
		postprocessData()

		termbox.Clear(coldef, coldef)
		fmt.Printf("\x1b[2J\x1b[%d;%dH", 0, 0)
		paddingLeft := printForecast()

		graphWidth := (sizeX - paddingLeft) * 7
		graphHeight := (sizeY - 4) * 15

		if os.Getenv("TERM_PROGRAM") == "iTerm.app" {
			image := renderGraph(graphWidth, graphHeight)
			fmt.Printf("\x1b[0;%dH%s", paddingLeft+1, encodeBuffer(image))
		} else {
			fmt.Printf("\x1b[0;%dH%s", paddingLeft+1, "\tУвы, картинки не картинки в этом терминале")
		}
		printInfo()
		termbox.Flush()

	} else {
		fmt.Printf("\x1b[%d;%dH\x1b[05;32mОбновлено: %s\x1b[0m", sizeY-3, 0, time.Now().Format("15:04:05"))
	}

	dataMap.dates = []float64{}
	dataMap.monthly = []float64{}
	dataMap.daily = []float64{}
	dataMap.current = []float64{}
	dataMap.gdr = []float64{}
	dataMap.approximatedgdr = []float64{}
	dataMap.gdrdates = []float64{}
	dataMap.values = []float64{}
	dataMap.approximatedvalues = []float64{}

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
	} else {
		// bad idea, паника портит терминал
		log.Printf("%s: request %s failed, status %s, response: %s", name, url, resp.Status, body[:])
		setStatus(name, "error")
	}

	if name == reqChangeName {
		jsonInterface := new(jsonChange)
		err = json.Unmarshal(body, jsonInterface)
		if err != nil {
			log.Panicln(name, err)
		}
		dataMap.dollar = jsonInterface.Rates["RUB"]
	} else {
		jsonInterface := new(jsonStock)
		err = json.Unmarshal(body, jsonInterface)
		if err != nil {
			log.Panicln(name, err, string(body[:]))
		}
		switch name {
		case reqPriceDayName:
			for _, now := range jsonInterface.Data {
				dataMap.daily = append(dataMap.daily, now[1])
			}
		case reqPriceWithVolumeMonthlyName:
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
		}
	}
	setStatus(name, "done")

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
			fmt.Printf("%s \x1b[05;31m%s\x1b[0m |\n", requestStatus[name]["url"], requestStatus[name]["status"])
		} else {
			fmt.Printf("%s \x1b[05;32m%s\x1b[0m\n", requestStatus[name]["url"], requestStatus[name]["status"])
		}
	}
	return
}

func postprocessData() {
	dataMap.lastprice = dataMap.daily[len(dataMap.daily)-1]

	for i := 0; i < len(dataMap.monthly); i++ {
		dataMap.current = append(dataMap.current, dataMap.lastprice)
	}
	var (
		minprice, maxprice = minmax(dataMap.monthly)
		minvalue, maxvalue = minmax(dataMap.values)
		mingdr, maxgdr     = minmax(dataMap.gdr)
	)

	valuesKoefficient := (maxprice - minprice) / (maxvalue - minvalue)
	for _, e := range dataMap.values {
		dataMap.approximatedvalues = append(dataMap.approximatedvalues, minprice+((e-minvalue)*valuesKoefficient))
	}

	gdrKoefficient := (maxprice - minprice) / (maxgdr - mingdr)
	for _, e := range dataMap.gdr {
		dataMap.approximatedgdr = append(dataMap.approximatedgdr, minprice+((e-mingdr)*gdrKoefficient))
	}
	dataMap.min, dataMap.max = minmax([]float64{maxprice + 0.5, minprice - 0.5, dataMap.current[0] + 0.5, dataMap.current[0] - 0.5})
	dataMap.minprice, dataMap.maxprice, dataMap.minvalue, dataMap.maxvalue, dataMap.mingdr, dataMap.maxgdr = minprice, maxprice, minvalue, maxvalue, mingdr, maxgdr

	return
}
func renderGraph(imageWidth, imageHeight int) (buffer *bytes.Buffer) {
	const (
		graphFontSize = 7.0
	)
	var (
		legendPrice = fmt.Sprintf("price, max: %.2f, min: %.2f, last: %.2f", dataMap.maxprice, dataMap.minprice, dataMap.monthly[len(dataMap.monthly)-1])
		legendValue = fmt.Sprintf("scaled value, max: %.1fkk, min: %.1fkk", dataMap.maxvalue/1000000, dataMap.minvalue/1000000)
		legendGdr   = fmt.Sprintf("scaled GDR's, max: %.2f, min: %.2f, now: %.2f", dataMap.maxgdr, dataMap.mingdr, dataMap.gdr[len(dataMap.gdr)-1])
		legendNow   = fmt.Sprintf("current price %.2f", dataMap.lastprice)
	)
	buffer = bytes.NewBuffer([]byte{})

	priceSeries := chart.ContinuousSeries{
		Name: legendPrice,
		Style: chart.Style{
			Show:        true,
			StrokeColor: drawing.Color{R: 255, G: 0, B: 0, A: 255},
			FillColor:   drawing.Color{R: 255, G: 0, B: 0, A: 255},
		},
		XValues: dataMap.dates,
		YValues: dataMap.monthly,
	}
	gdrSeries := chart.ContinuousSeries{
		Name: legendGdr,
		Style: chart.Style{
			Show:        true,
			StrokeColor: drawing.Color{R: 0, G: 0, B: 0, A: 255},
			StrokeWidth: 1.5,
		},
		XValues: dataMap.gdrdates,
		YValues: dataMap.approximatedgdr,
	}
	valueSeries := chart.ContinuousSeries{
		Name: legendValue,
		Style: chart.Style{
			Show:        true,
			StrokeColor: drawing.Color{R: 0, G: 255, B: 0, A: 255},
			StrokeWidth: 1.5,
		},
		XValues: dataMap.dates,
		YValues: dataMap.approximatedvalues,
	}
	nowSeries := chart.ContinuousSeries{
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
	)
	var (
		color     string
		even      = true
		step      = 0.5
		mul       = 0.98
		goodprice = (1.65*1000000)/(dataMap.dollar*1775) + optionsVesting
		value     float64
		rvalue    float64
		col       string
		collength int
		start     = 26.0
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

	return fmt.Sprintf("%.0f%s", i, out)
}

func getGdr(prices, values []float64) float64 {
	var (
		index = len(prices) - 1
		count float64
		summ  float64
		i     int
	)
	for ; i < 3; i++ {
		count = count + prices[index-i]*values[index-i]
		summ = summ + values[index-i]
	}

	return optionsValue - (optionsValue * optionsVesting / (count / summ))
}
