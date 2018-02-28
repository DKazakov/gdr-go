package main

import (
	"fmt"
	"github.com/nsf/termbox-go"
	"log"
	"os"
	"strings"
	"sync"
	"syscall"
	"time"
)

var (
	wg            sync.WaitGroup
	mu            sync.Mutex
	loadingBuffer []string
)

const (
	loadTick       time.Duration = 300 * time.Millisecond
	updateTick     time.Duration = 2 * 60 * time.Second
	optionsValue   float64       = 1775
	optionsVesting float64       = 19.6
)

func loadSpinner(x, y int) *time.Ticker {
	var (
		spin   int
		strBeg = "load "
		xbeg   = x/2 - (len(strBeg)+5)/2
		ybeg   = y / 2
		ticker = time.NewTicker(loadTick)
	)

	go func() {
		for _ = range ticker.C {
			spin++

			termbox.Clear(termbox.ColorDefault, termbox.ColorDefault)

			fmt.Print("\x1b[2J\x1b[0;0H")
			for _, e := range loadingBuffer {
				fmt.Println(e)
			}

			fmt.Printf("\x1b[%d;%dH%s", ybeg, xbeg, strBeg+strings.Repeat(".", spin))
			termbox.Flush()
			if spin >= 5 {
				spin = 0
			}
		}
	}()

	return ticker
}

func load(name string, item *Source, data *Data) {
	defer wg.Done()

	page, err := item.load()
	if err != nil {
		log.Println(name, "loading error", err)
	} else {
		data.set(name, page)
	}

	return
}
func get(name string, item *Source, data *Data) {
	defer wg.Done()

	page, err := item.get()
	if err != nil {
		log.Println(name, "reloading error", err)
	} else {
		data.set(name, page)
	}

	return
}

func main() {
	sources := getSources()
	data := new(Data).Init()

	f, _ := os.OpenFile("/var/log/self/gdr.log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	defer f.Close()
	log.SetOutput(f)
	syscall.Dup2(int(f.Fd()), 2)

	termbox.Init()
	termbox.SetOutputMode(termbox.OutputMode(termbox.OutputNormal))
	sizeX, sizeY := termbox.Size()
	defer termbox.Close()

	loadTicker := loadSpinner(sizeX, sizeY)

	for name, item := range sources {
		wg.Add(1)
		go load(name, item, data)
	}
	wg.Wait()
	time.Sleep(loadTick)
	data.finalize()

	graph := new(Graph).Init(data)
	text := new(Textinfo).Init(data)

	loadTicker.Stop()

	fmt.Println("\x1b[2J")
	left, bottom := text.print(sizeX, sizeY)
	graph.print(sizeX, sizeY, left, bottom)

	updateTicker := time.NewTicker(updateTick)

	go func() {
		for _ = range updateTicker.C {
			data = new(Data).Init()
			for name, item := range sources {
				wg.Add(1)
				go get(name, item, data)
			}
			wg.Wait()
			data.finalize()

			mu.Lock()
			graph.Init(data)
			text.Init(data)
			fmt.Println("\x1b[2J")
			left, bottom = text.print(sizeX, sizeY)
			graph.print(sizeX, sizeY, left, bottom)
			mu.Unlock()
		}
	}()

loop:
	for {
		switch ev := termbox.PollEvent(); ev.Type {
		case termbox.EventKey:
			switch ev.Key {
			case termbox.KeyArrowLeft:
				page := graph.getPrevPage()
				graph.setPage(page)
				graph.print(sizeX, sizeY, left, bottom)
			case termbox.KeyArrowRight:
				page := graph.getNextPage()
				graph.setPage(page)
				graph.print(sizeX, sizeY, left, bottom)
			case termbox.KeySpace:
				//graphType = getNextGraphType(true)
				//renderGraph()
			case termbox.KeyEsc:
				updateTicker.Stop()
				break loop
			case 0:
				switch ev.Ch {
				case 49, 50, 51, 52, 53, 54:
					page := int(ev.Ch) - 49
					graph.setPage(page)
					graph.print(sizeX, sizeY, left, bottom)
				case 113:
					updateTicker.Stop()
					break loop
				default:
					log.Printf("%+v", ev)
				}
			default:
				log.Printf("%+v", ev)
			}
		}
	}
}
