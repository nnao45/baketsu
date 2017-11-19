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
	size =		kingpin.Flag("size", "Baketsu size").Default("100").Short('s').Int64()
	memview =	kingpin.Flag("memview", "Memory viewer").Default("false").Short('v').Bool()
	white =		kingpin.Flag("white", "Non color").Default("false").Short('w').Bool()
	upper =		kingpin.Flag("upper", "Info & Count up to threshold(byte)").Short('u').Bool()
	lower =		kingpin.Flag("lower", "Info & Count below threshold(byte)").Short('l').Bool()
	byt =		kingpin.Flag("byt", "Unit Byte of threshold(byte)").Short('b').Int64()
	kib =		kingpin.Flag("kib", "Unit KiB of threshold(byte)").Short('k').Int64()
	mib =		kingpin.Flag("mib", "Unit MiB of threshold(byte)").Short('m').Int64()
	gib =		kingpin.Flag("gib", "Unit GiB of threshold(byte)").Short('g').Int64()
	tib =		kingpin.Flag("tib", "Unit TiB of threshold(byte)").Short('t').Int64()
)

const (
	UNIT_KiBYTE = 1024
	UNIT_MiBYTE = 1048576
	UNIT_GiBYTE = 1073741824
	UNIT_TiBYTE = 1099511627776
)

type ThrOpt struct {
        Byte    int64
        KiB     int64
        MiB     int64
        GiB     int64
        TiB     int64
}

func NewthrOpt() *ThrOpt{
        return &ThrOpt{
        Byte:   *byt,
        KiB:     *kib * UNIT_KiBYTE,
        MiB:     *mib * UNIT_MiBYTE,
        GiB:     *gib * UNIT_GiBYTE,
        TiB:     *tib * UNIT_TiBYTE,
        }
}

func (t *ThrOpt) IsUse() int64{
        var i int64
        if t.Byte != 0 {
                i = t.Byte
        } else if t.KiB != 0 {
                i = t.KiB
        } else if t.MiB != 0 {
                i = t.MiB
        } else if t.GiB != 0 {
		i = t.GiB
	} else if t.TiB != 0 {
		i = t.TiB
	}
	return i
}

const	TIME_FORMAT = "15:04:05"

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
	Measure		float64
	Unit		string
	Threshold	bool
}

func (b *Beaker) truncByte(i int64, t *ThrOpt, IsLake bool) *Beaker {
	if i < UNIT_KiBYTE {
		b.Measure = float64(i)
		b.Unit = "Byte"
	} else if i < UNIT_MiBYTE {
		b.Measure = float64(i) / float64(UNIT_KiBYTE)
		b.Unit = "KiB"
	} else if i < UNIT_GiBYTE {
		b.Measure = float64(i) / float64(UNIT_MiBYTE)
		b.Unit = "MiB"
	} else if i < UNIT_TiBYTE {
		b.Measure = float64(i) / float64(UNIT_GiBYTE)
		b.Unit = "GiB"
	} else if i >= UNIT_TiBYTE {
		b.Measure = float64(i) / float64(UNIT_TiBYTE)
		b.Unit = "TiB"
	}

	if IsLake {
		if *upper {
			if i > t.IsUse() {
				b.Threshold = true
			}
		}
		if *lower {
			if i < t.IsUse() {
				b.Threshold = true
			}
		}
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
	if *upper && *lower {
		fmt.Println("Sorry, baketshu's threshold option is only one use upper-threshold or lowwer-threshold.")
		fmt.Println("exit 1")
		os.Exit(1)
	}
	check := []int64{*byt, *kib, *mib, *gib, *tib}
	var i int
	var k int
	for _, c := range check {
		if !*upper && !*lower {
			if c != 0 {
			k++
			}
		}
		if *upper || *lower {
			if c != 0 {
			i++
			}
		}
	}
	if *upper || *lower {
		if i != 1 {
			fmt.Println("Sorry, baketshu's threshold option is only one use unit.")
			fmt.Println("exit 1")
			os.Exit(1)
		}
	}
	if k > 0 {
		fmt.Println("Sorry, baketshu's threshold option must used lower or upper with unit option.")
		fmt.Println("exit 1")
		os.Exit(1)
	}
}

func main() {
	baketsu := (*size * UNIT_MiBYTE)
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
	var counter int

	thropt := NewthrOpt()

	for {
		select {
		default:
			water := new(Water)
			water.Scoop(baketsu)
			v.Lake = v.Lake + water.Size
		case <-tick.C:
			lb, sb := new(Beaker), new(Beaker)
			lb.truncByte(v.Lake, thropt, true)
			spdcolor := p.Cyan
			if lb.Threshold {
				spdcolor = p.Red
				counter++
			}
			sb.truncByte(v.Sea, thropt, false)
			fmt.Fprintf(colorable.NewColorableStderr(), "\r%s", strings.Repeat(" ", len(mark)))
			end := time.Now()
			mark = fmt.Sprintf("%sTime: %s%s %sSpd: %.2f %s/s%s %sAll: %.2f %s%s ", p.Green, fmt.Sprint(t.Add(end.Sub(start)).Format(TIME_FORMAT)), p.Foot(),
						 spdcolor, round(lb.Measure, 2), lb.Unit, p.Foot(), p.Magenda, round(sb.Measure, 2), sb.Unit, p.Foot())
			if *upper || *lower {
				mark = mark + fmt.Sprintf("OVER: %d times", counter)
			}
			if *memview {
				runtime.ReadMemStats(&m)
				mark = mark + fmt.Sprintf("HSys: %d HAlc: %d HIdle: %d HRes: %d", m.HeapSys, m.HeapAlloc, m.HeapIdle, m.HeapReleased)
			}
			fmt.Fprintf(colorable.NewColorableStderr(), "\r%s", mark)
			v.Transfer()
		}
	}
}
