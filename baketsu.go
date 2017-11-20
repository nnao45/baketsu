package main

import (
	"bufio"
	"bytes"
	"fmt"
	"github.com/google/gopacket"
	"github.com/google/gopacket/pcap"
	"github.com/mattn/go-colorable"
	"gopkg.in/alecthomas/kingpin.v2"
	"io"
	"io/ioutil"
	"math"
	"os"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"
)

var (
	app      = kingpin.New("app", "A baketsu application.")
	interval = app.Flag("interval", "Logging interval").Default("1000ms").Short('i').Duration()
	pipe     = app.Flag("pipe", "Output pipe to os.Stdout").Default("false").Short('p').Bool()
	size     = app.Flag("size", "Baketsu size").Default("100").Short('s').Int64()
	memview  = app.Flag("memview", "Memory viewer").Default("false").Short('v').Bool()
	white    = app.Flag("white", "Non color").Default("false").Short('w').Bool()
	log      = app.Flag("log", "baketsu's result output log file").String()
	upper    = app.Flag("upper", "Info & Count up to threshold(byte)").Default("false").Short('u').Bool()
	lower    = app.Flag("lower", "Info & Count below threshold(byte)").Default("false").Short('l').Bool()
	byt      = app.Flag("byt", "Unit Byte of threshold(byte)").Short('b').Int64()
	kib      = app.Flag("kib", "Unit KiB of threshold(byte)").Short('k').Int64()
	mib      = app.Flag("mib", "Unit MiB of threshold(byte)").Short('m').Int64()
	gib      = app.Flag("gib", "Unit GiB of threshold(byte)").Short('g').Int64()
	tib      = app.Flag("tib", "Unit TiB of threshold(byte)").Short('t').Int64()

	run = app.Command("run", "Running basic mode")

	packet   = app.Command("packet", "Packet capture mode")
	packetF  bool
	device   = packet.Flag("device", "Packet capturing device").String()
	promis   = packet.Flag("promis", "Promiscuous capturing packet").Default("false").Bool()
	filter   = packet.Flag("filter", "Set packet capturing filter").Bool()
	port     = packet.Flag("port", "Packet capturing fliter port").Uint64()
	protocol = packet.Flag("protocol", "Packet capturing fliter protocol").String()
	dsthost  = packet.Flag("dsthost", "Packet capturing fliter dsthost").String()
	srchost  = packet.Flag("srchost", "Packet capturing fliter dsthost").String()
)

const (
	VERSION = "1.0.0"
)

const (
	UNIT_KiBYTE = 1024
	UNIT_MiBYTE = 1048576
	UNIT_GiBYTE = 1073741824
	UNIT_TiBYTE = 1099511627776
)

type ThrOpt struct {
	Byte int64
	KiB  int64
	MiB  int64
	GiB  int64
	TiB  int64
}

func NewthrOpt() *ThrOpt {
	return &ThrOpt{
		Byte: *byt,
		KiB:  *kib * UNIT_KiBYTE,
		MiB:  *mib * UNIT_MiBYTE,
		GiB:  *gib * UNIT_GiBYTE,
		TiB:  *tib * UNIT_TiBYTE,
	}
}

func (t *ThrOpt) IsUse() int64 {
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

const (
	TIME_FORMAT = "15:04:05"
	LOG_FORMAT  = "2006-01-02 15:04:05.000"
)

const (
	COLOR_BLACK_HEADER   = "\x1b[30m"
	COLOR_RED_HEADER     = "\x1b[31m"
	COLOR_GREEN_HEADER   = "\x1b[32m"
	COLOR_YELLOW_HEADER  = "\x1b[33m"
	COLOR_BLUE_HEADER    = "\x1b[34m"
	COLOR_MAGENDA_HEADER = "\x1b[35m"
	COLOR_CYAN_HEADER    = "\x1b[36m"
	COLOR_WHITE_HEADER   = "\x1b[37m"
	COLOR_FOOTER         = "\x1b[0m"
)

type Pallet struct {
	Black   string
	Red     string
	Green   string
	Yellow  string
	Blue    string
	Magenda string
	Cyan    string
	White   string
	Foot    string
}

func NewPallet() *Pallet {
	return &Pallet{
		Black:   COLOR_BLACK_HEADER,
		Red:     COLOR_RED_HEADER,
		Green:   COLOR_GREEN_HEADER,
		Yellow:  COLOR_YELLOW_HEADER,
		Blue:    COLOR_BLUE_HEADER,
		Magenda: COLOR_MAGENDA_HEADER,
		Cyan:    COLOR_CYAN_HEADER,
		White:   COLOR_WHITE_HEADER,
		Foot:    COLOR_FOOTER,
	}

}

type Beaker struct {
	Measure   float64
	Unit      string
	Threshold bool
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

func (w *Water) Scoop(out io.Writer, in io.Reader, baketsu int64) *Water {
	if *pipe {
		out = os.Stdout
	}

	w.Size, _ = io.CopyN(out, in, baketsu)
	return w
}

func pcapture(capCh chan io.Reader, baketsu int64) {
	handle, err := pcap.OpenLive(*device, int32(baketsu), *promis, pcap.BlockForever)
	if err != nil {
		panic(err)
	}
	defer handle.Close()

	if *filter {
		var optAry []string
		if *protocol != "" {
			optAry = append(optAry, *protocol)
		}
		if *port != 0 {
			optAry = append(optAry, fmt.Sprint("port ", strconv.FormatUint(*port, 10)))
		}
		if *srchost != "" {
			optAry = append(optAry, fmt.Sprint("src host ", *srchost))
		}
		if *dsthost != "" {
			optAry = append(optAry, fmt.Sprint("dst host ", *dsthost))
		}
		filstr := strings.Join(optAry, " and ")
		err = handle.SetBPFFilter(filstr)
		if err != nil {
			panic(err)
		}
	}

	packetSource := gopacket.NewPacketSource(handle, handle.LinkType())

	for {
		select {
		case p := <-packetSource.Packets():
			capCh <- bytes.NewReader(p.Data())
		default:
		}
	}
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

func addog(text string, filename string) error {
	var writer *bufio.Writer
	textData := []byte(text)

	writeFile, err := os.OpenFile(filename, os.O_CREATE|os.O_APPEND|os.O_RDWR, 0755)
	writer = bufio.NewWriter(writeFile)
	writer.Write(textData)
	writer.Flush()
	defer writeFile.Close()

	return err
}

func IsIP(ip string) (b bool) {
	if m, _ := regexp.MatchString("^[0-9]{1,3}.[0-9]{1,3}.[0-9]{1,3}.[0-9]{1,3}$", ip); !m {
		return false
	}
	return true
}

func init() {
	kingpin.Version(fmt.Sprint("baketsu's version: ", VERSION))
	switch kingpin.MustParse(app.Parse(os.Args[1:])) {
	case packet.FullCommand():
		packetF = true
	}

	if packetF {
		if *device == "" {
			fmt.Fprintln(os.Stderr, "Sorry, when packet capture mode, must select device.")
			fmt.Fprintln(os.Stderr, "exit 1")
			os.Exit(1)
		}
	}

	if *filter {
		if *protocol != "" && !strings.Contains(*protocol, "tcp") && !strings.Contains(*protocol, "udp") && !strings.Contains(*protocol, "icmp") {
			fmt.Fprintln(os.Stderr, "Sorry, when set packet capture fliter, only support tcp or udp or icmp.")
			fmt.Fprintln(os.Stderr, "exit 1")
			os.Exit(1)
		}
		if *port > 65535 {
			fmt.Fprintln(os.Stderr, "Sorry, when set packet capture fliter, port number 1~65535.")
			fmt.Fprintln(os.Stderr, "exit 1")
			os.Exit(1)
		}
		if *srchost != "" && !IsIP(*srchost) {
			fmt.Fprintln(os.Stderr, "Sorry, when set packet capture fliter, src host is IPv4 format.")
			fmt.Fprintln(os.Stderr, "exit 1")
			os.Exit(1)
		}
		if *dsthost != "" && !IsIP(*dsthost) {
			fmt.Fprintln(os.Stderr, "Sorry, when set packet capture fliter, dst host is IPv4 format.")
			fmt.Fprintln(os.Stderr, "exit 1")
			os.Exit(1)
		}
	}

	if *upper && *lower {
		fmt.Fprintln(os.Stderr, "Sorry, baketshu's threshold option is only one use upper-threshold or lower-threshold.")
		fmt.Fprintln(os.Stderr, "exit 1")
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
			fmt.Fprintln(os.Stderr, "Sorry, baketsu's threshold option is only one use unit.")
			fmt.Fprintln(os.Stderr, "exit 1")
			os.Exit(1)
		}
	}
	if k > 0 {
		fmt.Fprintln(os.Stderr, "Sorry, baketshu's threshold option must used lower or upper with unit option.")
		fmt.Fprintln(os.Stderr, "exit 1")
		os.Exit(1)
	}
}

func main() {
	baketsu := (*size * UNIT_MiBYTE)
	v := new(Vessel)
	mark := ""
	t := new(time.Time)
	start := time.Now()
	tick := time.NewTicker(*interval)
	var m runtime.MemStats
	mode := "[S]"

	p := NewPallet()
	if *white {
		p = new(Pallet)
	}
	var counter int
	thropt := NewthrOpt()

	capCh := make(chan io.Reader)
	if packetF {
		mode = "[P]"
		go pcapture(capCh, baketsu)
	}

	for {
		select {
		default:
			if !packetF {
				water := new(Water)
				water.Scoop(ioutil.Discard, os.Stdin, baketsu)
				v.Lake = v.Lake + water.Size
			}
		case b := <-capCh:
			water := new(Water)
			water.Scoop(ioutil.Discard, b, baketsu)
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
			mark = fmt.Sprintf("%s%s Time: %s%s %sSpd: %.2f %s/s%s %sAll: %.2f %s%s ", p.Green, mode, fmt.Sprint(t.Add(end.Sub(start)).Format(TIME_FORMAT)), p.Foot,
				spdcolor, round(lb.Measure, 2), lb.Unit, p.Foot, p.Magenda, round(sb.Measure, 2), sb.Unit, p.Foot)
			if *upper || *lower {
				mark = mark + fmt.Sprintf("OVER: %d times ", counter)
			}
			if *memview {
				runtime.ReadMemStats(&m)
				mark = mark + fmt.Sprintf("HSys: %d HAlc: %d HIdle: %d HRes: %d", m.HeapSys, m.HeapAlloc, m.HeapIdle, m.HeapReleased)
			}
			if *log != "" {
				err := addog(fmt.Sprintf("%s%s%s%s\n", "[ ", end.Format(LOG_FORMAT), " ] ", mark), *log)
				if err != nil {
					panic(err)
				}
			}
			fmt.Fprintf(colorable.NewColorableStderr(), "\r%s", mark)
			v.Transfer()
		}
	}
}
