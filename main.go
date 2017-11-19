package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"math"
	"os"
	"strings"
	"time"
	"runtime"
	"github.com/mattn/go-colorable"
	"gopkg.in/alecthomas/kingpin.v2"
)

var (
        size = kingpin.Flag("size", "Baketsu size").Default("100").Short('s').Int64()
	memview = kingpin.Flag("memview", "Memory viewer").Default("false").Short('m').Bool()
	white = kingpin.Flag("white", "Non color").Default("false").Short('w').Bool()
)

const (
	UNIT_KBYTE = 1024
	UNIT_MBYTE = 1048576
	UNIT_GBYTE = 1073741824
	UNIT_TBYTE = 1099511627776
	TIME_FORMAT = "15:04:05"
)

const (
	COLOR_BLACK_HEADER = "\x1b[30m"
	COLOR_RED_HEADER = "\x1b[31m"
	COLOR_GREEN_HEADER = "\x1b[32m"
	COLOR_YELLOW_HEADER = "\x1b[33m"
	COLOR_BLUE_HEADER = "\x1b[34m"
	COLOR_MAGENDA_HEADER = "\x1b[35m"
	COLOR_CYAN_HEADER = "\x1b[36m"
	COLOR_WHITE_HEADER = "\x1b[37m"
	COLOR_FOOTER = "\x1b[0m"
)

type Pallet struct {
	Black	string
	Red	string
	Green	string
	Yellow	string
	Blue	string
	Magenda	string
	Cyan	string
	White	string
}

func NewPallet() *Pallet {
	return &Pallet{
		Black:		COLOR_BLACK_HEADER,
                Red:		COLOR_RED_HEADER,
                Green:		COLOR_GREEN_HEADER,
                Yellow:		COLOR_YELLOW_HEADER,
                Blue:		COLOR_BLUE_HEADER,
                Magenda:	COLOR_MAGENDA_HEADER,
                Cyan:		COLOR_CYAN_HEADER,
                White:		COLOR_WHITE_HEADER,
	}

}

func (p *Pallet) Foot() (footer string){
		return COLOR_FOOTER
}

type Beaker struct {
	Measure float64
	Unit    string
}

func (b *Beaker) truncByte(i int64) *Beaker {
	if i < UNIT_KBYTE {
		b.Measure = float64(i)
		b.Unit = "Byte"
	} else if i < UNIT_MBYTE {
		b.Measure = float64(i) / float64(UNIT_KBYTE)
		b.Unit = "KB"
	} else if i < UNIT_GBYTE {
		b.Measure = float64(i) / float64(UNIT_MBYTE)
		b.Unit = "MB"
	} else if i < UNIT_TBYTE {
		b.Measure = float64(i) / float64(UNIT_GBYTE)
		b.Unit = "GB"
	} else if i >= UNIT_TBYTE {
		b.Measure = float64(i) / float64(UNIT_TBYTE)
		b.Unit = "TB"
	}
	return b
}

type Water struct {
	Size int64
}

func (w *Water) Scoop(baketsu int64) *Water {
	w.Size, _ = io.CopyN(ioutil.Discard, os.Stdin, baketsu)
	return w
}

type Vessel struct {
	Lake int64
	Sea  int64
}

func (v *Vessel) Transfer() *Vessel {
	v.Sea = v.Sea + v.Lake
	v.Lake = 0
	return v
}

func round(f float64, places int) float64 {
	shift := math.Pow(10, float64(places))
	return math.Floor(f*shift+.5) / shift
}

func init() {
	kingpin.Parse()
}

func main() {
	baketsu := (*size * UNIT_MBYTE)
	v := new(Vessel)
	mark := ""
	t := new(time.Time)
	start := time.Now()
	tick := time.NewTicker(time.Millisecond * 1000)
	var m runtime.MemStats

	p := NewPallet()
	if *white {
		p = new(Pallet)
	}

	for {
		select {
		default:
			water := new(Water)
			water.Scoop(baketsu)
			v.Lake = v.Lake + water.Size
		case <-tick.C:
			lb, sb := new(Beaker), new(Beaker)
			lb.truncByte(v.Lake)
			sb.truncByte(v.Sea)
			fmt.Fprintf(colorable.NewColorableStderr(), "\r%s", strings.Repeat(" ", len(mark)))
			end := time.Now()
			mark = fmt.Sprintf("%sTime: %s%s %sSpd: %.2f %s/s%s %sAll: %.2f %s%s ", p.Green, fmt.Sprint(t.Add(end.Sub(start)).Format(TIME_FORMAT)), p.Foot(),
						 p.Cyan, round(lb.Measure, 2), lb.Unit, p.Foot(), p.Magenda, round(sb.Measure, 2), sb.Unit, p.Foot())
			if *memview {
				runtime.ReadMemStats(&m)
				mark = mark + fmt.Sprintf("HSys: %d HAlc: %d HIdle: %d HRes: %d", m.HeapSys, m.HeapAlloc, m.HeapIdle, m.HeapReleased)
			}
			fmt.Fprintf(colorable.NewColorableStderr(), "\r%s", mark)
			v.Transfer()
		}
	}
}
