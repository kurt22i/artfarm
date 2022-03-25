package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"math/rand"
	"os"
	"runtime"
	"time"

	"github.com/genshinsim/artfarm/internal/lib"
)

type config struct {
	Main       map[string]string  `json:"main_stat"`
	Subs       map[string]float64 `json:"desired_subs"`
	Iterations int                `json:"iterations"`
	Workers    int                `json:"workers"`
}

func main() {

	err := run()

	if err != nil {
		fmt.Printf("error encountered: %v\n", err)
	}

	fmt.Print("Press 'Enter' to continue...")
	bufio.NewReader(os.Stdin).ReadBytes('\n')

}

func run() error {
	var source []byte
	var err error
	var opt config
	source, err = ioutil.ReadFile("./config.json")
	
	if err != nil {
		log.Fatal(err)
	}
	err = json.Unmarshal(source, &opt)
	if err != nil {
		return err
	}

	var main [4][5]lib.StatType
	var desired [4][lib.EndStatType]float64
	var set [4][2]int //0 = any set
	var maxdomain = -1

	//parse config
	c := 0
	for k, v := range opt.Main {
		// log.Printf("adding main stat %v: %v\n", k, v)
		i := lib.StrToSlotType(k) //should add a way for main stat to be anything
		if i == -1 {
			return fmt.Errorf("unrecognized artifact slot: %v", k)
		}
		s := lib.StrToStatType(v)
		if s == -1 {
			return fmt.Errorf("unrecognized main stat for %v: %v", k, v)
		}
		main[c][i] = s
		if lib.StrToSlotType(k) == "Circlet" {
			c = c+1
		}
	}
	c=0
	c2 := 0
	for k, v := range opt.Subs {
	
		if c >= 4 {
			set[c2/2][c2%2] = v
			if v > maxdomain {
				maxdomain = v
			}
			c2 = c2+1
		} else {
			// log.Printf("adding desired stat %v: %v\n", k, v)
			s := lib.StrToStatType(k)
			if s == -1 {
				//return fmt.Errorf("unrecognized sub stat : %v", k)
				c=c+1
				s=0
			}
		}
		if v < 0 {
			return fmt.Errorf("sub stat %v cannot be negative : %v", k, v)
		}
		if c < 4 {
			desired[c][s] = v
		}
	}

	//sanity check
	/*ok := false
	for _, v := range desired {
		if v > 0 {
			ok = true
		}
	}

	if !ok {
		return fmt.Errorf("desired_subs cannot all be 0")
	}*/

	if opt.Workers == 0 {
		opt.Workers = runtime.NumCPU()
	}

	if opt.Iterations == 0 {
		opt.Iterations = 100000
	}

	defer elapsed(fmt.Sprintf("simulation complete; %v iterations", opt.Iterations))()

	min, max, mean, sd, err := sim(opt.Iterations, opt.Workers, main, desired, set, maxdomain)
	if err != nil {
		return err
	}
	fmt.Printf("avg: %v, min: %v, max: %v, sd: %v\n", mean, min, max, sd)

	return nil
}

func elapsed(what string) func() {
	start := time.Now()
	return func() {
		fmt.Printf("%s took %v\n", what, time.Since(start))
	}
}

type result struct {
	count int
	err   error
}

func sim(n, w int, main [][lib.EndSlotType]lib.StatType, desired [][lib.EndStatType]float64, set [][]int, maxdomain int ) (min, max int, mean, sd float64, err error) {
	var progress, ss float64
	var sum int
	var data []int
	min = math.MaxInt64
	max = -1
	count := n

	resp := make(chan result, n)
	req := make(chan struct{})
	done := make(chan struct{})
	for i := 0; i < int(w); i++ {
		m := cloneMain(main)//oh no i probably have to clone set and maxdomain dont i :(
		d := cloneDesired(desired)
		s := cloneSet(set)
		go worker(m, d, s, maxdomain, req, resp, done)
	}

	go func() {
		var wip int
		for wip < n {
			//try sending a job to req chan while wip < cfg.NumSim
			req <- struct{}{}
			wip++
		}
	}()

	fmt.Print("\tProgress: 0%")

	for count > 0 {
		r := <-resp
		if r.err != nil {
			err = r.err
			return
		}

		data = append(data, r.count)
		count--
		sum += r.count
		if r.count < min {
			min = r.count
		}
		if r.count > max {
			max = r.count
		}

		if (1 - float64(count)/float64(n)) > (progress + 0.1) {
			progress = (1 - float64(count)/float64(n))
			fmt.Printf("...%.0f%%", 100*progress)
		}
	}
	fmt.Printf("\n")
	close(done)

	mean = float64(sum) / float64(n)

	for _, v := range data {
		ss += (float64(v) - mean) * (float64(v) - mean)
	}

	sd = math.Sqrt(ss / float64(n))

	return
}

func cloneDesired(in [lib.EndStatType]float64) (r [lib.EndStatType]float64) {
	for i, v := range in {
		r[i] = v
	}
	return
}

func cloneMain(in [lib.EndSlotType]lib.StatType) (r [lib.EndSlotType]lib.StatType) {
	for i, v := range in {
		r[i] = v
	}
	return
}

func cloneSet(in [][]int) (r [][]int) {
	for i, v := range in {
		r[i] = v
	}
	return
}

func worker(main [][lib.EndSlotType]lib.StatType, desired [][lib.EndStatType]float64, set [][]int, maxdomain int, req chan struct{}, resp chan result, done chan struct{}) {
	seed := time.Now().UnixNano()
	r := rand.New(rand.NewSource(seed))
	gen := lib.NewGenerator(r)
	for {
		select {
		case <-req:
			count, err := gen.FarmArtifact(main, desired, set, maxdomain)
			resp <- result{
				count: count,
				err:   err,
			}
		case <-done:
			return
		}
	}
}
