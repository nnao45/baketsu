package main

import (
	"bufio"
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

const BUF_SIZE = UNIT_MBYTE * 500 // 500Mbytes

func truncByte(i int) (f float64, s string) {
	if i < UNIT_KBYTE {
		f = float64(i)
		s = "Byte"
	} else if i < UNIT_MBYTE {
		f = float64(i) / float64(UNIT_KBYTE)
		s = "KB"
	} else if i < UNIT_GBYTE {
		f = float64(i) / float64(UNIT_MBYTE)
		s = "MB"
	} else if i < UNIT_TBYTE {
		f = float64(i) / float64(UNIT_GBYTE)
		s = "GB"
	} else if i >= UNIT_TBYTE {
		f = float64(i) / float64(UNIT_TBYTE)
		s = "TB"
	}
	return
}

func round(f float64, places int) float64 {
	shift := math.Pow(10, float64(places))
	return math.Floor(f*shift+.5) / shift
}

func main() {
	var (
		lake  int
		water int
		sea   int
		err   error
		mark  string
	)

	stop := make(chan struct{}, 0)
	restart := make(chan struct{}, 0)

	baketsu := make([]byte, BUF_SIZE)

	go func() {
		for {
			select {
			case <-stop:
				<-restart
			default:
				water, err = os.Stdin.Read(baketsu)
				if err != nil {
					lake = lake + len(bufio.NewScanner(os.Stdin).Bytes())
					ef, es := truncByte(lake)
					fmt.Printf("ALL: %.2f %s\n", round(ef, 2), es)
					os.Exit(0)
				}
				lake = lake + water
				bytes.NewBuffer(baketsu).Reset()
			}
		}
	}()
	for {
		time.Sleep(1000 * time.Millisecond)
		stop <- struct{}{}
		f, s := truncByte(lake)
		af, as := truncByte(sea)
		fmt.Printf("\r%s", strings.Repeat(" ", len(mark)))
		mark = fmt.Sprintf("SPD: %.2f %s/s ALL: %.2f %s", round(f, 2), s, round(af, 2), as)
		fmt.Printf("\r%s", mark)
		sea = sea + lake
		lake = 0
		restart <- struct{}{}
	}
}
