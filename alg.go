package throtto

import (
	"fmt"
	"log"
	"math"
	"net/http"
	"sync"
	"time"
)

type lweight struct {
	sw, fw float64
	sync.Mutex
}

type lcap struct {
	window, thres, conf float64
	sync.Mutex
}

type lcount struct {
	count, pcount, scount, fcount, dcount int64
	sync.Mutex
}

type lstate struct {
	lwindow float64
	isDrop  bool
	sync.Mutex
}

type lmem struct {
	wdrop []float64
	sync.Mutex
}

type ltask struct {
	tasks  []string
	start  *rstatus
	end    *rstatus
	length int
	gr     int
	sync.Mutex
}

type rstatus struct {
	value string
	next  *rstatus
}

func (t *ltask) push(ts *rstatus) {
	if t.start == nil {
		t.start = ts
		t.end = ts
	} else {
		lp := t.end
		lp.next = ts
		t.end = ts
	}
	t.length++
}

func (t *ltask) pop() {
	t.Lock()
	defer t.Unlock()
	if t.start != nil {
		curr := t.start
		t.start = curr.next
		curr = nil
		t.length--
	}
	t.length = 0
}

type cctrl struct {
	sincr, fdcr, flux float64
}

type limiter struct {
	lweight    *lweight
	lcap       *lcap
	lcount     *lcount
	lstate     *lstate
	lmem       *lmem
	ltask      *ltask
	cctrl      *cctrl
	rejectFunc http.HandlerFunc
	finish     chan bool
	debug      bool
}

func (l *limiter) ptick() {
	tick := time.NewTicker(time.Second)
	go func() {
		for c := range tick.C {
			log.Println("resetting counter", c)
			l.rcounter()
		}
	}()
}

func (l *limiter) pschedule() {
	for {
		select {
		case <-l.finish:
			return
		default:
			l.ltask.Lock()
			if len(l.ltask.tasks) > 0 {
				stat, ntask := l.ltask.tasks[0], l.ltask.tasks[1:]
				l.ltask.tasks = ntask
				l.add(stat)
				l.wupdate(stat)
				l.balance(stat)
			}
			l.ltask.Unlock()
		}
	}
}

func (l *limiter) calc(stat string) {
	l.add(stat)
	l.wupdate(stat)
	l.balance(stat)
}

func (l *limiter) allow() bool {
	l.lcount.Lock()
	l.lcap.Lock()
	defer l.lcount.Unlock()
	defer l.lcap.Unlock()
	return l.lcount.count < int64(math.Ceil(l.lcap.window))
}

func (l *limiter) next(code int) error {
	l.ltask.Lock()
	defer l.ltask.Unlock()
	l.ltask.push(&rstatus{value: getStatus(code)})
	if l.ltask.length == 1 && l.ltask.gr == 0 {
		l.ltask.gr++
		go l.process()
	}
	return nil
}

func (l *limiter) process() {
	if l.ltask.start != nil {
		curr := l.ltask.start
		for curr != nil {
			l.calc(curr.value)
			curr = curr.next
			l.ltask.start = curr
			l.ltask.pop()
		}
	}
	l.ltask.gr--
}

func (l *limiter) add(status string) {
	l.lcount.Lock()
	defer l.lcount.Unlock()
	if status == success {
		l.lcount.scount++
	} else if status == failure {
		l.lcount.fcount++
	} else {
		l.lcount.dcount++
	}
}

func (l *limiter) tcount(status string) int64 {
	l.lcount.Lock()
	defer l.lcount.Unlock()
	if status == success {
		return l.lcount.scount
	} else if status == failure {
		return l.lcount.fcount
	} else {
		return l.lcount.dcount
	}
}

func (l *limiter) rcounter() {
	l.lcount.Lock()
	defer l.lcount.Unlock()
	l.lcount.pcount = (l.lcount.pcount + l.lcount.count) / 2
	l.lcount.count = 0
	l.lcount.dcount = 0
	l.lcount.scount = 0
	l.lcount.fcount = 0
}

func (l *limiter) wupdate(status string) {
	l.lcap.Lock()
	defer l.lcap.Unlock()
	l.lweight.Lock()
	defer l.lweight.Unlock()
	l.lstate.Lock()
	defer l.lstate.Unlock()
	var nw float64
	var isDrop bool
	switch status {
	case failure:
		nw = l.remedy()
		isDrop = true
	case success:
		if l.lstate.isDrop {
			l.lweight.sw = l.cctrl.flux
			l.lweight.fw = 1 - l.cctrl.flux
			l.lstate.isDrop = false
		}
		if l.lcap.window < float64(l.lcap.conf) {
			nw = l.quick()
		} else if l.lcap.window < l.lcap.thres {
			nw = l.slow()
		} else {
			nw = l.congavd()
		}
	}
	if l.lcap.window < 1.0 {
		nw = 1.0
	}
	l.lstate.lwindow = l.lcap.window
	l.lcap.window = nw
	if isDrop {
		if l.lcap.window < l.lstate.lwindow && !l.lstate.isDrop {
			l.lmem.Lock()
			defer l.lmem.Unlock()
			l.lcount.Lock()
			defer l.lcount.Unlock()
			l.lmem.wdrop = append(l.lmem.wdrop, float64(l.lcount.pcount))
			var total float64
			for _, v := range l.lmem.wdrop {
				total += v
			}
			l.lcap.thres = total / float64(len(l.lmem.wdrop))
			l.debugln(fmt.Sprint("new threshold", l.lcap.thres))
			l.lstate.isDrop = true
		}
	}
}

func (l *limiter) balance(nstat string) {
	l.lweight.Lock()
	defer l.lweight.Unlock()
	if nstat == success {
		if l.lweight.sw < l.cctrl.flux || l.lweight.sw+l.cctrl.flux > 1.0 {
			return
		}
		l.lweight.sw += l.cctrl.flux
		l.lweight.fw -= l.cctrl.flux
		return
	}

	if nstat == failure {
		if l.lweight.fw < l.cctrl.flux || l.lweight.fw+l.cctrl.flux > 1.0 {
			return
		}
		l.lweight.fw += l.cctrl.flux
		l.lweight.sw -= l.cctrl.flux
		return
	}
}

func (l *limiter) quick() float64 {
	return l.lcap.window + l.cctrl.sincr
}

func (l *limiter) slow() float64 {
	return l.lcap.window + (l.cctrl.sincr * l.lweight.sw)
}

func (l *limiter) congavd() float64 {
	return l.lcap.window + (l.cctrl.sincr*l.lweight.sw)/l.lcap.window
}

func (l *limiter) remedy() float64 {
	return (l.lcap.window * l.lweight.fw) / l.cctrl.fdcr
}

func (l *limiter) debugln(str string) {
	if l.debug {
		log.Println(str)
	}
}
