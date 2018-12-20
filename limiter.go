package throtto

import (
	"net/http"
	"sync"
)

var lmt *limiter
var once sync.Once

//New create request limitter instance
func New(cfg *Config) RequestLimitter {
	once.Do(func() {
		if cfg == nil {
			cfg = defConf()
		}
		lmt = &limiter{
			cctrl: &cctrl{
				flux:  cfg.Flux,
				sincr: cfg.MaxIncementRate,
				fdcr:  cfg.MaxDecrementRate,
			},
			lcap: &lcap{
				thres:  float64(2 * cfg.CapConfidence),
				window: float64(cfg.CapConfidence),
				conf:   float64(cfg.CapConfidence),
			},
			lmem: &lmem{
				wdrop: []float64{float64(cfg.CapConfidence * 2)},
			},
			lstate: new(lstate),
			ltask: &ltask{
				tasks: make([]string, 0),
			},
			lweight: &lweight{
				sw: cfg.Flux,
				fw: 1.0 - cfg.Flux,
			},
			lcount:     new(lcount),
			rejectFunc: cfg.RejectFunc,
			finish:     make(chan bool, 1),
			debug:      cfg.Debug,
		}
	})
	return lmt
}

func defConf() *Config {
	return &Config{
		Flux:             DefFlux,
		MaxIncementRate:  DefMaxIncr,
		MaxDecrementRate: DefMaxDecr,
		CapConfidence:    DefCapConf,
	}
}

//RequestLimitter contract
//members:
//	ProtectOverRequest bind the handler with request limitter
type RequestLimitter interface {
	ProtectOverRequest(http.Handler) http.Handler
	Stop()
}

//Config as request limitter configuration
type Config struct {
	Flux             float64
	MaxIncementRate  float64
	MaxDecrementRate float64
	CapConfidence    int64
	RejectFunc       http.HandlerFunc
	Debug            bool
}

//ProtectOverRequest protect handler from exceeding request
func (l *limiter) ProtectOverRequest(next http.Handler) http.Handler {
	go l.ptick()
	return limitHandler(l, next)
}

func (l *limiter) Stop() {
	l.finish <- true
}
