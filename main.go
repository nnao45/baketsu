package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"math"
	"os"
	"strings"
	"time"
)

const (
	UNIT_KBYTE = 1024
	UNIT_MBYTE = 1048576
	UNIT_GBYTE = 1073741824
	UNIT_TBYTE = 1099511627776
)

const BUF_SIZE = UNIT_MBYTE * 100 // 100Mbytes

const TIME_FORMAT = "15:04:05"

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
	Free error
}

func (w *Water) Scoop() *Water {
	w.Size, _ = io.CopyN(ioutil.Discard, os.Stdin, BUF_SIZE)
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

func main() {
	v := new(Vessel)
	mark := ""

	t := new(time.Time)
	start := time.Now()
	tick := time.NewTicker(time.Millisecond * 1000)

	for {
		select {
		default:
			water := new(Water)
			water.Scoop()
			v.Lake = v.Lake + water.Size
		case <-tick.C:
			lb, sb := new(Beaker), new(Beaker)
			lb.truncByte(v.Lake)
			sb.truncByte(v.Sea)
			fmt.Printf("\r%s", strings.Repeat(" ", len(mark)))
			end := time.Now()
			mark = fmt.Sprintf("%s SPD: %.2f %s/s ALL: %.2f %s", fmt.Sprint(t.Add(end.Sub(start)).Format(TIME_FORMAT)), round(lb.Measure, 2), lb.Unit, round(sb.Measure, 2), sb.Unit)
			fmt.Printf("\r%s", mark)
			v.Transfer()
		}
	}
}
