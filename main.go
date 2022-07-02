package main

import (
	"bufio"
	"compress/gzip"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"sync"
)

// read is a single read of sequencing
type read struct {
	ID       string
	sequence string
	quality  string
}

// pair is a pair of reads from a paired-end experiment
type pair struct {
	begin  bool
	r1, r2 read
}

// seRead is equivalent to pair, but for single-end experiment
type seRead struct {
	begin bool
	r1    read
}

type fqBufferedWriter struct {
	f    *os.File
	gz   *gzip.Writer
	buff *bufio.Writer
}

// reader open FASTQ file and parse its content as read type
func reader(path string, out chan<- read) {
	file, err := os.Open(path)
	if err != nil {
		log.Fatalln(err)
	}
	defer file.Close()
	gr, err := gzip.NewReader(file)
	if err != nil {
		log.Fatalln(err)
	}
	defer gr.Close()
	scanner := bufio.NewScanner(gr)
	const (
		id   = 0
		seq  = 1
		qual = 3
	)

	counter := 0
	var rName, rSeq, rQual string
	for scanner.Scan() {
		switch counter % 4 {
		case id:
			rName = scanner.Text()
		case seq:
			rSeq = scanner.Text()
		case qual:
			rQual = scanner.Text()
			r := read{rName, rSeq, rQual}
			out <- r
		}
		counter++
	}
	close(out)
}

// pairSplitter splits the read sequence - and quality - at the nth position. It works on paired-end reads
func pairSplitter(in1 <-chan read, in2 <-chan read, out chan<- pair, n int) {
	for v := range in1 {
		v2 := <-in2

		if (len(v.sequence) <= n) || (len(v2.sequence) <= n) {
			log.Print("Discarded: " + v.ID + " and " + v2.ID)
			continue
		}
		firstRead1 := read{ID: v.ID, sequence: v.sequence[0:n], quality: v.quality[0:n]}
		firstRead2 := read{ID: v2.ID, sequence: v2.sequence[0:n], quality: v2.quality[0:n]}
		out <- pair{true, firstRead1, firstRead2}

		sl1 := len(v.sequence)
		sl2 := len(v2.sequence)
		lastRead1 := read{ID: v.ID, sequence: v.sequence[n:sl1], quality: v.quality[n:sl1]}
		lastRead2 := read{ID: v2.ID, sequence: v2.sequence[n:sl2], quality: v2.quality[n:sl2]}
		out <- pair{false, lastRead1, lastRead2}

	}
	close(out)
}

// pairWriter writes the output files for paired-end experiment. It creates 4 output files.
func pairWriter(in <-chan pair, beginR1, beginR2, EndR1, EndR2 io.Writer) {
	var wg sync.WaitGroup
	for p := range in {

		wg.Add(2)
		if p.begin {
			go writeRead(p.r1, beginR1, &wg)
			go writeRead(p.r2, beginR2, &wg)
		} else {
			go writeRead(p.r1, EndR1, &wg)
			go writeRead(p.r2, EndR2, &wg)
		}
		wg.Wait()
	}
}

// singleSplitter is equivalent to pairSplitter, but for single-end experiment
func singleSplitter(in1 <-chan read, out chan<- seRead, n int) {
	for v := range in1 {

		if len(v.sequence) <= n {
			log.Print("Discarded: " + v.ID)
			continue
		}
		firstRead1 := read{ID: v.ID, sequence: v.sequence[0:n], quality: v.quality[0:n]}
		out <- seRead{true, firstRead1}

		sl1 := len(v.sequence)
		lastRead1 := read{ID: v.ID, sequence: v.sequence[n:sl1], quality: v.quality[n:sl1]}
		out <- seRead{false, lastRead1}

	}
	close(out)
}

// singleWriter is equivalent to pairWriter, but for single-end. It creates 2 output files.
func singleWriter(in <-chan seRead, beginR1, EndR1 io.Writer) {
	var wg sync.WaitGroup
	for p := range in {

		wg.Add(1)
		if p.begin {
			go writeRead(p.r1, beginR1, &wg)
		} else {
			go writeRead(p.r1, EndR1, &wg)
		}
		wg.Wait()
	}
}

// writeRead uses the provided io.Writer to output the read, in FASTQ format.
func writeRead(r read, w io.Writer, wg *sync.WaitGroup) {
	_, err := w.Write([]byte(fmt.Sprintf("%s\n%s\n+\n%s\n", r.ID, r.sequence, r.quality)))
	if err != nil {
		log.Fatalln(err)
	}
	wg.Done()
}

func main() {
	// CLI setup
	flag.Usage = func() {
		w := flag.CommandLine.Output() // may be os.Stderr - but not necessarily

		fmt.Fprintf(w, "Usage of %s: \n", os.Args[0])

		flag.PrintDefaults()

		fmt.Fprintf(w, "\nExample: fq-splitPair -r1 example/test_r1.fq.gz -r2 example/test_r2.fq.gz -n 10 \n")

	}
	r1Path := flag.String("r1", "", "Path for your R1 FASTQ file")
	r2Path := flag.String("r2", "", "Path for your R2 FASTQ file")
	sePath := flag.String("se", "", "Path for your single end FASTQ file")
	n := flag.Int("n", 0, "Position to splitPair your read. Ex: n=3, seq=AAATTTTT would give AAA and TTTTT.")
	out := flag.String("out", "test-1", "Output basename for the first bases file")
	flag.Parse()

	// Input validations
	if *n == 0 {
		log.Fatalln("Need to provide n greater than 0. Ex: 35")
	}
	if *r1Path == "" && *r2Path == "" && *sePath == "" {
		log.Fatalln("Need to provide a path for you FASTQ file. It can be paired-end experiment (-r1 and -r2) or single-end (-se)")
	}
	if (*r1Path != "" || *r2Path != "") && *sePath != "" {
		log.Fatalln("You can only use paired-end (by providing -r1 and -r2) ot single-end (-se). Not both.")
	}

	if *r1Path != "" || *r2Path != "" {
		handlePaired(r1Path, r2Path, n, out)
	} else if *sePath != "" {
		handleSingle(r1Path, n, out)
	}

}

// handlePaired splits reads from paired-end experiments
func handlePaired(r1Path *string, r2Path *string, n *int, out *string) {
	// Initialize writers
	b1 := newFqBufferedWriter(*out + "_begin_R1.fq.gz")
	b2 := newFqBufferedWriter(*out + "_begin_R2.fq.gz")
	b3 := newFqBufferedWriter(*out + "_end_R1.fq.gz")
	b4 := newFqBufferedWriter(*out + "_end_R2.fq.gz")

	splitPair(r1Path, r2Path, n, b1.buff, b2.buff, b3.buff, b4.buff)

	b1.buff.Flush()
	b2.buff.Flush()
	b3.buff.Flush()
	b4.buff.Flush()
}

// handlePairedReads splits reads from single-end experiments
func handleSingle(r1Path *string, n *int, out *string) {
	// Initialize writers
	b1 := newFqBufferedWriter(*out + "_begin_SE.fq.gz")
	b2 := newFqBufferedWriter(*out + "_end_SE.fq.gz")

	splitSingle(r1Path, n, b1.buff, b2.buff)

	b1.buff.Flush()
	b2.buff.Flush()

}

func newFqBufferedWriter(beginNameR1 string) *fqBufferedWriter {
	f1, err := os.OpenFile(beginNameR1, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		log.Fatalln(err)
	}
	gz1 := gzip.NewWriter(f1)
	b1 := bufio.NewWriter(gz1)
	return &fqBufferedWriter{buff: b1, gz: gz1, f: f1}
}

func splitPair(r1Path, r2Path *string, n *int, beginR1, beginR2, endR1, EndR2 io.Writer) {
	// Running
	r1 := make(chan read)
	r2 := make(chan read)
	pairs := make(chan pair)

	go reader(*r1Path, r1)
	go reader(*r2Path, r2)
	go pairSplitter(r1, r2, pairs, *n)

	pairWriter(pairs, beginR1, beginR2, endR1, EndR2)
}

func splitSingle(r1Path *string, n *int, beginR1, endR1 io.Writer) {
	// Running
	r1 := make(chan read)
	reads := make(chan seRead)

	go reader(*r1Path, r1)
	go singleSplitter(r1, reads, *n)

	singleWriter(reads, beginR1, endR1)
}
