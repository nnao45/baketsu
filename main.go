package main

import (
	"io"
	"io/ioutil"
	"bytes"
	"fmt"
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

const BUF_SIZE = UNIT_MBYTE * 10 // 10Mbytes

type Beaker struct {
        Measure float64
        Unit    string
}

func (b *Beaker)truncByte(i int64) *Beaker{
	if i < UNIT_KBYTE {
		b.Measure = float64(i)
		b.Unit = "Byte"
	} else if i < UNIT_MBYTE {
		b.Measure = float64(i) / float64(UNIT_KBYTE)
		b.Unit  = "KB"
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

func (w *Water) Scoop(/*baketsu []byte*/) *Water {
	//w.Size, w.Free = os.Stdin.Read(baketsu)
	w.Size,_ = io.CopyN(ioutil.Discard, os.Stdin, BUF_SIZE)
	return w
}

type Vessel struct {
	Lake int64
	Sea  int64
}

func (v *Vessel) Transfer() *Vessel{
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

	stop := make(chan struct{}, 0)
	received := make(chan struct{}, 0)
	restart := make(chan struct{}, 0)

	baketsu := make([]byte, BUF_SIZE)

	go func() {
		for {
			select {
			case <-stop:
				received <- struct{}{}
				<-restart
			default:
				water := new(Water)
				water.Scoop(/*baketsu*/)
				v.Lake = v.Lake + water.Size
				bytes.NewBuffer(baketsu).Reset()
			}
		}
	}()

	for {
		time.Sleep(1000 * time.Millisecond)
		stop <- struct{}{}
		<-received
		lb, sb := new(Beaker), new(Beaker)
		lb.truncByte(v.Lake)
		sb.truncByte(v.Sea)
		fmt.Printf("\r%s", strings.Repeat(" ", len(mark)))
		mark = fmt.Sprintf("SPD: %.2f %s/s ALL: %.2f %s", round(lb.Measure, 2), lb.Unit, round(sb.Measure, 2), sb.Unit)
		fmt.Printf("\r%s", mark)
		v.Transfer()
		restart <- struct{}{}
	}
}
