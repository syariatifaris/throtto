package throtto

//GetCounts gets the current running counter request
func GetCounts() (success, fail, drop, request int64) {
	if lmt == nil {
		return
	}
	if lmt.lcount == nil {
		return
	}
	lmt.lcount.Lock()
	defer lmt.lcount.Unlock()
	success = lmt.lcount.scount
	fail = lmt.lcount.fcount
	drop = lmt.lcount.dcount
	request = lmt.lcount.count
	return
}

//GetCaps gets window and threshold capacity
func GetCaps() (win float64, thres float64) {
	if lmt == nil {
		return
	}
	if lmt.lcap == nil {
		return
	}
	lmt.lcap.Lock()
	defer lmt.lcap.Unlock()
	win = lmt.lcap.window
	thres = lmt.lcap.thres
	return
}

//GetWeights gets the current lb weight
func GetWeights() (success, fail float64) {
	if lmt == nil {
		return
	}
	if lmt.lweight == nil {
		return
	}
	lmt.lweight.Lock()
	defer lmt.lweight.Unlock()
	success = lmt.lweight.sw
	fail = lmt.lweight.fw
	return
}
