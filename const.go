package throtto

const (
	//DefFlux default flux number (weight transfer rate)
	DefFlux float64 = 0.00001
	//DefMaxIncr default maximum window increment
	DefMaxIncr float64 = 5.0
	//DefMaxDecr default maximum window decrement
	DefMaxDecr float64 = 2.0
	//DefCapConf default max ability for handler to serve request
	DefCapConf int64 = 100
	//MaxTask max task length
	MaxTask int = 1024
)

const (
	success = "SUCCESS"
	failure = "FAIL"
	exceed  = "PASS"
)
