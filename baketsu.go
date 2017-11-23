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
	"unicode/utf8"
)

var (
	app      = kingpin.New("baketsu", "A baketsu application.")
	interval = app.Flag("interval", "Logging interval").Default("1000ms").Short('i').Duration()
	pipe     = app.Flag("pipe", "Output pipe to os.Stdout").Default("false").Short('p').Bool()
	size     = app.Flag("size", "Baketsu size").Default("100").Short('s').Int64()
	memview  = app.Flag("memview", "Memory viewer").Default("false").Short('v').Bool()
	white    = app.Flag("white", "Non color").Default("false").Short('w').Bool()
	log      = app.Flag("log", "baketsu's result output log file").String()
	upper    = app.Flag("upper", "Info & Count up to threshold(byte)").Default("false").Short('u').Bool()
	lower    = app.Flag("lower", "Info & Count below threshold(byte)").Default("false").Short('l').Bool()
	byt      = app.Flag("byt", "Unit Byte of threshold(byte)").Int64()
	kib      = app.Flag("kib", "Unit KiB of threshold(byte)").Int64()
	mib      = app.Flag("mib", "Unit MiB of threshold(byte)").Int64()
	gib      = app.Flag("gib", "Unit GiB of threshold(byte)").Int64()
	tib      = app.Flag("tib", "Unit TiB of threshold(byte)").Int64()

	run = app.Command("run", "Running basic mode")

	scan  = app.Command("scan", "Receive string stream with word scanner")
	scanF bool
	word  = scan.Flag("word", "Count match word when scanning").String()
	wordR []rune
	cha   = scan.Flag("cha", "Unit Char of threshold(rune)").Int64()
	hun   = scan.Flag("hun", "Unit Hundred of threshold(rune)").Int64()
	mil   = scan.Flag("mil", "Unit Million of threshold(rune)").Int64()
	bil   = scan.Flag("bil", "Unit Billion of threshold(rune)").Int64()

	packet   = app.Command("packet", "Packet capture mode")
	packetF  bool
	device   = packet.Flag("device", "Packet capturing device").Required().String()
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
	UNIT_HUNDRE = 100
	UNIT_MILLI  = 1000000
	UNIT_BILLI  = 1000000000
)

const (
	WORD_BUFFER = 512
)

type ThrOpt struct {
	Byte int64
	KiB  int64
	MiB  int64
	GiB  int64
	TiB  int64
	CHA  int64
	HUN  int64
	MIL  int64
	BIL  int64
}

func NewThrOpt() *ThrOpt {
	return &ThrOpt{
		Byte: *byt,
		KiB:  *kib * UNIT_KiBYTE,
		MiB:  *mib * UNIT_MiBYTE,
		GiB:  *gib * UNIT_GiBYTE,
		TiB:  *tib * UNIT_TiBYTE,
		CHA:  *cha,
		HUN:  *hun * UNIT_HUNDRE,
		MIL:  *mil * UNIT_MILLI,
		BIL:  *bil * UNIT_BILLI,
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
	} else if t.CHA != 0 {
		i = t.CHA
	} else if t.HUN != 0 {
		i = t.HUN
	} else if t.MIL != 0 {
		i = t.MIL
	} else if t.BIL != 0 {
		i = t.BIL
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
	COLOR_PLAIN_HEADER   = "\x1b[0m"
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
	Plain   string
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
		Plain:   COLOR_PLAIN_HEADER,
	}

}

type DrawOut struct {
	Time  string
	Speed string
	All   string
	Foot  string
}

func NewDrawOut(p *Pallet) *DrawOut {
	return &DrawOut{
		Time:  p.Green,
		Speed: p.Cyan,
		All:   p.Magenda,
		Foot:  p.Plain,
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

func (b *Beaker) truncWord(i int64, t *ThrOpt, IsLake bool) *Beaker {

	if i < UNIT_HUNDRE {
		b.Measure = float64(i)
		b.Unit = "Char"
	} else if i < UNIT_MILLI {
		b.Measure = float64(i) / float64(UNIT_HUNDRE)
		b.Unit = "Hundred Char"
	} else if i < UNIT_BILLI {
		b.Measure = float64(i) / float64(UNIT_MILLI)
		b.Unit = "Million Char"
	} else if i >= UNIT_BILLI {
		b.Measure = float64(i) / float64(UNIT_BILLI)
		b.Unit = "Billion Char"
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
	Size  int64
	SizeS int
	Match int
	Count int
}

func (w *Water) Scoop(out io.Writer, in io.Reader, baketsu int64) *Water {
	if *pipe {
		out = os.Stdout
	}
	var str string
	var i int
	var k int

	if !scanF {
		w.Size, _ = io.CopyN(out, in, baketsu)
	} else {
		scanner := bufio.NewScanner(in)
		for scanner.Scan() {
			t := scanner.Text()
			if *word != "" {
				for _, run := range t {
					if run == wordR[k] {
						if k < len(wordR)-1 {
							k++
						} else if k == len(wordR)-1 {
							w.Match++
						}
					} else {
						k = 0
					}
				}
			}
			str = str + t
			i++
			if i == WORD_BUFFER {
				break
			}
		}
		w.Count = utf8.RuneCountInString(str)
	}

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
	Lake   int64
	Sea    int64
	Bucket int
}

func (v *Vessel) Transfer() *Vessel {
	v.Sea = v.Sea + v.Lake
	v.Lake = 0
	return v
}

type Result struct {
	Var       []interface{}
	Fixed     string
	Log       string
	Thres     string
	MemStats  string
	Matchstat string
	Colorable bool
}

func NewResult() *Result {
	v := make([]interface{}, 0, 10)
	return &Result{
		Var:       v,
		Fixed:     "",
		Log:       "",
		Thres:     "",
		MemStats:  "",
		Matchstat: "",
		Colorable: true,
	}
}

type Format struct {
	Basic      string
	BasicColor string
	Char       string
	CharColor  string
}

func NewFormat() *Format {
	return &Format{
		Basic:      "%s Time: %s Spd: %.2f %s/s All: %.2f %s ",
		BasicColor: "%s%s Time: %s %sSpd: %.2f %s/s %sAll: %.2f %s%s ",
		Char:       "%s Time: %s Spd: %v %s/s All: %v %s ",
		CharColor:  "%s%s Time: %s %sSpd: %v %s/s %sAll: %v %s%s ",
	}
}

func (r *Result) Fix(d *DrawOut, f *Format) (l, s string) {
	ary := make([]interface{}, 0, 10)
	if r.Colorable {
		for i, v := range r.Var {
			if i == 0 {
				ary = append(ary, d.Time)
			} else if i == 2 {
				ary = append(ary, d.Speed)
			} else if i == 4 {
				ary = append(ary, d.All)
			}
			ary = append(ary, v)
		}
		ary = append(ary, d.Foot)
	}

	var basicformat string
	var colorformat string

	if !scanF {
		basicformat = f.Basic
		colorformat = f.BasicColor
	} else {
		basicformat = f.Char
		colorformat = f.CharColor
	}

	if *log != "" {
		l = fmt.Sprintf(basicformat,
			r.Var[0], r.Var[1], r.Var[2], r.Var[3], r.Var[4], r.Var[5])
	}
	if r.Colorable {
		s = fmt.Sprintf(colorformat,
			ary[0], ary[1], ary[2], ary[3], ary[4], ary[5], ary[6], ary[7], ary[8], ary[9])
	} else {
		s = fmt.Sprintf(basicformat,
			r.Var[0], r.Var[1], r.Var[2], r.Var[3], r.Var[4], r.Var[5])
	}
	return
}

func (r *Result) SumF() string {
	return r.Fixed + r.Thres + r.Matchstat + r.MemStats
}

func (r *Result) SumL() string {
	return r.Log + r.Thres + r.Matchstat + r.MemStats
}

type Base struct {
	Baketsu  int64
	Vessel   *Vessel
	Time     *time.Time
	Result   *Result
	Start    time.Time
	Ticker   *time.Ticker
	Pallet   *Pallet
	Format   *Format
	ThrOpt   *ThrOpt
	MemStats *runtime.MemStats
	Counter  int
	Mode     string
	CapCh    chan io.Reader
}

func NewBase() *Base {
	capCh := make(chan io.Reader)
	return &Base{
		Baketsu:  (*size * UNIT_MiBYTE),
		Vessel:   new(Vessel),
		Time:     new(time.Time),
		Result:   NewResult(),
		Start:    time.Now(),
		Ticker:   time.NewTicker(*interval),
		Pallet:   NewPallet(),
		Format:   NewFormat(),
		ThrOpt:   NewThrOpt(),
		MemStats: new(runtime.MemStats),
		Counter:  0,
		Mode:     "[B]",
		CapCh:    capCh,
	}
}

func (b *Base) SumBasicLake(w *Water) *Base {
	b.Vessel.Lake = b.Vessel.Lake + w.Size
	return b
}

func (b *Base) SumCharLake(w *Water) *Base {
	b.Vessel.Lake = b.Vessel.Lake + int64(w.Count)
	return b
}

func (b *Base) SumCharBucket(w *Water) *Base {
	b.Vessel.Bucket = b.Vessel.Bucket + w.Match
	return b
}

func init() {
	app.HelpFlag.Short('h')
	app.Version(fmt.Sprint("baketsu's version: ", VERSION))
	switch kingpin.MustParse(app.Parse(os.Args[1:])) {
	case packet.FullCommand():
		packetF = true
	case scan.FullCommand():
		scanF = true
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
	} else {
		if *protocol != "" || *port != 0 || *srchost != "" || *dsthost != "" {
			fmt.Fprintln(os.Stderr, "Sorry, this option only using when set packet capture fliter.")
			fmt.Fprintln(os.Stderr, "exit 1")
			os.Exit(1)
		}
	}

	if *word != "" {
		wordR = []rune(*word)
	}

	if *upper && *lower {
		fmt.Fprintln(os.Stderr, "Sorry, baketshu's threshold option is only one use upper-threshold or lower-threshold.")
		fmt.Fprintln(os.Stderr, "exit 1")
		os.Exit(1)
	}
	var check []int64
	if !scanF {
		check = []int64{*byt, *kib, *mib, *gib, *tib}
	} else {
		check = []int64{*cha, *hun, *mil, *bil}
	}
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
	b := NewBase()
	if packetF {
		b.Mode = "[P]"
		go pcapture(b.CapCh, b.Baketsu)
	}
	if scanF {
		b.Mode = "[S]"
	}

	for {
		select {
		default:
			if !packetF {
				water := new(Water)
				water.Scoop(ioutil.Discard, os.Stdin, b.Baketsu)
				if scanF {
					b.SumCharLake(water)
					b.SumCharBucket(water)
				} else {
					b.SumBasicLake(water)
				}
			}
		case p := <-b.CapCh:
			water := new(Water)
			water.Scoop(ioutil.Discard, p, b.Baketsu)
			b.SumBasicLake(water)
		case <-b.Ticker.C:
			fmt.Fprintf(colorable.NewColorableStderr(), "\r%s", strings.Repeat(" ", len(b.Result.SumF())))
			b.Result = NewResult()
			lb, sb := new(Beaker), new(Beaker)
			if !scanF {
				lb.truncByte(b.Vessel.Lake, b.ThrOpt, true)
			} else {
				lb.truncWord(b.Vessel.Lake, b.ThrOpt, true)
			}
			d := NewDrawOut(b.Pallet)
			if lb.Threshold {
				d.Speed = b.Pallet.Red
				b.Counter++
			}
			if *white {
				b.Result.Colorable = false
			}
			if !scanF {
				sb.truncByte(b.Vessel.Sea, b.ThrOpt, false)
			} else {
				sb.truncWord(b.Vessel.Sea, b.ThrOpt, false)
			}
			end := time.Now()
			b.Result.Var = []interface{}{b.Mode, fmt.Sprint(b.Time.Add(end.Sub(b.Start)).Format(TIME_FORMAT)),
				round(lb.Measure, 2), lb.Unit, round(sb.Measure, 2), sb.Unit}
			b.Result.Log, b.Result.Fixed = b.Result.Fix(d, b.Format)
			if *upper || *lower {
				b.Result.Thres = fmt.Sprintf("Over: %d times ", b.Counter)
			}
			if *memview {
				runtime.ReadMemStats(b.MemStats)
				b.Result.MemStats = fmt.Sprintf("HSys: %d HAlc: %d HIdle: %d HRes: %d", b.MemStats.HeapSys, b.MemStats.HeapAlloc, b.MemStats.HeapIdle, b.MemStats.HeapReleased)
			}
			if *word != "" {
				b.Result.Matchstat = fmt.Sprintf("Match: %d Word ", b.Vessel.Bucket)
			}
			if *log != "" {
				err := addog(fmt.Sprintf("%s%s%s%s\n", "[ ", end.Format(LOG_FORMAT), " ] ", b.Result.SumL()), *log)
				if err != nil {
					panic(err)
				}
			}
			fmt.Fprintf(colorable.NewColorableStderr(), "\r%s", (b.Result.SumF()))
			b.Vessel.Transfer()
		}
	}
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
