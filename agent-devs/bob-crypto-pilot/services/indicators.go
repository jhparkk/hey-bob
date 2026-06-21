package services

import (
	"log"
	"math"

	"bob-crypto-pilot/db"
)

// CalcAndStoreAllIndicators calculates indicators for every row of daily_prices for a coin
// and stores them back via batch UPDATE in a transaction.
func CalcAndStoreAllIndicators(coin string) error {
	type row struct {
		id    int64
		high  float64
		low   float64
		close float64
	}

	rows, err := db.DB.Query(
		`SELECT id, high, low, close FROM daily_prices WHERE coin=? ORDER BY date ASC`, coin)
	if err != nil {
		return err
	}
	defer rows.Close()

	var data []row
	for rows.Next() {
		var r row
		if err := rows.Scan(&r.id, &r.high, &r.low, &r.close); err != nil {
			return err
		}
		data = append(data, r)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	if len(data) == 0 {
		log.Printf("[indicators] %s: no data", coin)
		return nil
	}

	n := len(data)
	highs := make([]float64, n)
	lows := make([]float64, n)
	closes := make([]float64, n)
	for i, r := range data {
		highs[i] = r.high
		lows[i] = r.low
		closes[i] = r.close
	}

	// Pre-compute EMA9/EMA21 series (incremental, full history)
	ema9Series := calcEMASeries(closes, 9)
	ema21Series := calcEMASeries(closes, 21)

	// Pre-compute MACD / Signal series
	macdSeries, signalSeries := calcMACDSeries(closes)

	// Pre-compute ADX series
	adxSeries := calcADXSeries(highs, lows, closes, 14)

	tx, err := db.DB.Begin()
	if err != nil {
		return err
	}

	stmt, err := tx.Prepare(`UPDATE daily_prices SET
		ma7=?, ma20=?, ma50=?,
		ema9=?, ema21=?,
		rsi14=?,
		macd=?, macd_signal=?,
		bb_upper=?, bb_middle=?, bb_lower=?,
		adx14=?
		WHERE id=?`)
	if err != nil {
		tx.Rollback()
		return err
	}
	defer stmt.Close()

	for i := 0; i < n; i++ {
		cl := closes[:i+1]
		hi := highs[:i+1]
		lo := lows[:i+1]
		m := len(cl)

		var ma7, ma20, ma50 float64
		var rsi14 float64
		var bbU, bbM, bbL float64

		// MA7
		if m >= 7 {
			ma7 = mean(cl[m-7:])
		}
		// MA20 + BB
		if m >= 20 {
			ma20 = mean(cl[m-20:])
			bbM = ma20
			std := stddev(cl[m-20:], ma20)
			bbU = ma20 + 2*std
			bbL = ma20 - 2*std
		}
		// MA50
		if m >= 50 {
			ma50 = mean(cl[m-50:])
		}
		// RSI14
		if m >= 15 {
			rsi14 = calcRSI(cl[m-15:])
		}

		// Use pre-computed series
		ema9 := ema9Series[i]
		ema21 := ema21Series[i]
		macdVal := macdSeries[i]
		signalVal := signalSeries[i]
		adx14 := adxSeries[i]

		_ = hi
		_ = lo

		_, err := stmt.Exec(
			ma7, ma20, ma50,
			ema9, ema21,
			rsi14,
			macdVal, signalVal,
			bbU, bbM, bbL,
			adx14,
			data[i].id,
		)
		if err != nil {
			tx.Rollback()
			return err
		}
	}

	if err := tx.Commit(); err != nil {
		return err
	}
	log.Printf("[indicators] %s: stored indicators for %d rows", coin, n)
	return nil
}

// calcEMASeries returns an EMA series for the given period using the full closes slice.
// ema[i] is the EMA value at index i considering only closes[0..i].
func calcEMASeries(closes []float64, period int) []float64 {
	n := len(closes)
	result := make([]float64, n)
	if n == 0 {
		return result
	}
	k := 2.0 / float64(period+1)
	// Seed: SMA of first `period` elements (or less)
	seed := 0.0
	seedN := period
	if seedN > n {
		seedN = n
	}
	for i := 0; i < seedN; i++ {
		seed += closes[i]
		result[i] = seed / float64(i+1)
	}
	if n <= period {
		return result
	}
	ema := seed / float64(period)
	result[period-1] = ema
	for i := period; i < n; i++ {
		ema = closes[i]*k + ema*(1-k)
		result[i] = ema
	}
	return result
}

// calcMACDSeries returns MACD line and Signal line series for the full closes slice.
func calcMACDSeries(closes []float64) (macdLine []float64, signalLine []float64) {
	n := len(closes)
	macdLine = make([]float64, n)
	signalLine = make([]float64, n)
	if n < 26 {
		return
	}

	ema12 := calcEMASeries(closes, 12)
	ema26 := calcEMASeries(closes, 26)

	raw := make([]float64, n)
	for i := 0; i < n; i++ {
		raw[i] = ema12[i] - ema26[i]
	}
	copy(macdLine, raw)

	// Signal: 9-period EMA of macd line (starting from index 25)
	if n < 35 {
		return
	}
	k9 := 2.0 / float64(9+1)
	// Seed signal with SMA of raw[26..34]
	sigSeed := 0.0
	for i := 26; i < 35; i++ {
		sigSeed += raw[i]
	}
	sig := sigSeed / 9.0
	signalLine[34] = sig
	for i := 35; i < n; i++ {
		sig = raw[i]*k9 + sig*(1-k9)
		signalLine[i] = sig
	}
	return
}

// calcADXSeries returns an ADX series for the full highs/lows/closes slices.
func calcADXSeries(highs, lows, closes []float64, period int) []float64 {
	n := len(closes)
	result := make([]float64, n)
	if n < period*2+1 {
		return result
	}

	plusDMs := make([]float64, n-1)
	minusDMs := make([]float64, n-1)
	trs := make([]float64, n-1)

	for i := 1; i < n; i++ {
		upMove := highs[i] - highs[i-1]
		downMove := lows[i-1] - lows[i]
		if upMove > downMove && upMove > 0 {
			plusDMs[i-1] = upMove
		}
		if downMove > upMove && downMove > 0 {
			minusDMs[i-1] = downMove
		}
		hl := highs[i] - lows[i]
		hpc := math.Abs(highs[i] - closes[i-1])
		lpc := math.Abs(lows[i] - closes[i-1])
		trs[i-1] = math.Max(hl, math.Max(hpc, lpc))
	}

	nt := len(trs)
	if nt < period {
		return result
	}

	// Wilder smoothing (initial sum)
	var smoothTR, smoothPlus, smoothMinus float64
	for i := 0; i < period; i++ {
		smoothTR += trs[i]
		smoothPlus += plusDMs[i]
		smoothMinus += minusDMs[i]
	}

	dxArr := make([]float64, 0, nt-period)
	for i := period; i < nt; i++ {
		smoothTR = smoothTR - smoothTR/float64(period) + trs[i]
		smoothPlus = smoothPlus - smoothPlus/float64(period) + plusDMs[i]
		smoothMinus = smoothMinus - smoothMinus/float64(period) + minusDMs[i]

		var dipVal, dimVal float64
		if smoothTR > 0 {
			dipVal = 100 * smoothPlus / smoothTR
			dimVal = 100 * smoothMinus / smoothTR
		}
		diSum := dipVal + dimVal
		var dx float64
		if diSum > 0 {
			dx = 100 * math.Abs(dipVal-dimVal) / diSum
		}
		dxArr = append(dxArr, dx)
	}

	if len(dxArr) < period {
		return result
	}

	// ADX = EMA of DX
	var adxVal float64
	for i := 0; i < period; i++ {
		adxVal += dxArr[i]
	}
	adxVal /= float64(period)

	// Map adxArr[period-1] → closes index period*2
	startIdx := period * 2 // 1-based TR offset + period smoothing
	if startIdx < n {
		result[startIdx] = adxVal
	}

	for i := period; i < len(dxArr); i++ {
		adxVal = (adxVal*float64(period-1) + dxArr[i]) / float64(period)
		idx := i + period + 1
		if idx < n {
			result[idx] = adxVal
		}
	}

	return result
}

// Indicators holds calculated technical indicators
type Indicators struct {
	MA7, MA20, MA50          float64
	RSI14                    float64
	MACD, MACDSignal         float64
	BBUpper, BBMiddle, BBLower float64
	EMA9, EMA21              float64
	ADX14                    float64
	ATR14, ATR50             float64
	VolumeMA20               float64
	HighestHigh20            float64
}

// DailyIndicators is kept as an alias for backward compatibility
type DailyIndicators = Indicators

// CalcIndicators calculates daily indicators for a coin using daily_prices table
// CalcIndicatorsUpbit는 upbit_daily_prices 테이블 기반으로 지표를 계산한다.
func CalcIndicatorsUpbit(coin string) (*Indicators, error) {
	return calcIndicatorsFromTable(coin, "upbit_daily_prices")
}

func CalcIndicatorsBithumb(coin string) (*Indicators, error) {
	return calcIndicatorsFromTable(coin, "bithumb_daily_prices")
}

func CalcIndicators(coin string) (*Indicators, error) {
	return calcIndicatorsFromTable(coin, "daily_prices")
}

func calcIndicatorsFromTable(coin, table string) (*Indicators, error) {
	// 최근 120봉 조회 (ATR50 = 50개 TR 필요 → 51개 close + 여유)
	rows, err := db.DB.Query(`
		SELECT high, low, close, volume FROM `+table+`
		WHERE coin = ? ORDER BY date DESC LIMIT 120`, coin)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var highs, lows, closes, volumes []float64
	for rows.Next() {
		var h, l, c, v float64
		rows.Scan(&h, &l, &c, &v)
		highs = append(highs, h)
		lows = append(lows, l)
		closes = append(closes, c)
		volumes = append(volumes, v)
	}
	// closes[0] = 최신 (내림차순) → 오름차순으로 reverse
	for i, j := 0, len(closes)-1; i < j; i, j = i+1, j-1 {
		highs[i], highs[j] = highs[j], highs[i]
		lows[i], lows[j] = lows[j], lows[i]
		closes[i], closes[j] = closes[j], closes[i]
		volumes[i], volumes[j] = volumes[j], volumes[i]
	}

	ind := &Indicators{}
	n := len(closes)
	if n == 0 {
		return ind, nil
	}

	// MA7
	if n >= 7 {
		ind.MA7 = mean(closes[n-7:])
	}
	// MA20 + Bollinger Bands
	if n >= 20 {
		ind.MA20 = mean(closes[n-20:])
		ind.BBMiddle = ind.MA20
		std := stddev(closes[n-20:], ind.MA20)
		ind.BBUpper = ind.MA20 + 2*std
		ind.BBLower = ind.MA20 - 2*std
	}
	// MA50
	if n >= 50 {
		ind.MA50 = mean(closes[n-50:])
	}
	// RSI14
	if n >= 15 {
		ind.RSI14 = calcRSI(closes[n-15:])
	}
	// MACD (12, 26, 9) — efficient O(n) incremental EMA
	if n >= 35 {
		k12 := 2.0 / (12.0 + 1)
		k26 := 2.0 / (26.0 + 1)
		k9 := 2.0 / (9.0 + 1)

		// Seed EMA12 and EMA26 with SMA
		ema12 := mean(closes[:12])
		ema26 := mean(closes[:26])

		// Warm up EMA12 through index 25
		for _, v := range closes[12:26] {
			ema12 = v*k12 + ema12*(1-k12)
		}

		// Build MACD line from index 26 onward (O(n))
		macdLine := make([]float64, 0, n-26)
		for _, v := range closes[26:] {
			ema12 = v*k12 + ema12*(1-k12)
			ema26 = v*k26 + ema26*(1-k26)
			macdLine = append(macdLine, ema12-ema26)
		}

		ind.MACD = macdLine[len(macdLine)-1]

		// Signal: 9-period EMA of MACD line
		if len(macdLine) >= 9 {
			signal := mean(macdLine[:9])
			for _, v := range macdLine[9:] {
				signal = v*k9 + signal*(1-k9)
			}
			ind.MACDSignal = signal
		} else {
			ind.MACDSignal = ind.MACD
		}
	}

	// EMA9, EMA21
	if n >= 9 {
		ind.EMA9 = calcEMA(closes, 9)
	}
	if n >= 21 {
		ind.EMA21 = calcEMA(closes, 21)
	}

	// True Range series (length = n-1)
	if n >= 2 {
		trs := calcTR(highs, lows, closes)
		nt := len(trs)

		// ATR14 - 단순 이동평균
		if nt >= 14 {
			ind.ATR14 = mean(trs[nt-14:])
		}
		// ATR50 - 단순 이동평균
		if nt >= 50 {
			ind.ATR50 = mean(trs[nt-50:])
		}
		// ADX14
		if nt >= 28 {
			ind.ADX14 = calcADX(highs, lows, closes, 14)
		}
	}

	// VolumeMA20
	if n >= 20 {
		ind.VolumeMA20 = mean(volumes[n-20:])
	}
	// HighestHigh20
	if n >= 20 {
		ind.HighestHigh20 = maxSlice(highs[n-20:])
	}

	return ind, nil
}

// calcTR computes True Range slice (len = len(closes)-1)
func calcTR(highs, lows, closes []float64) []float64 {
	n := len(closes)
	if n < 2 {
		return nil
	}
	trs := make([]float64, n-1)
	for i := 1; i < n; i++ {
		hl := highs[i] - lows[i]
		hpc := math.Abs(highs[i] - closes[i-1])
		lpc := math.Abs(lows[i] - closes[i-1])
		trs[i-1] = math.Max(hl, math.Max(hpc, lpc))
	}
	return trs
}

// calcADX computes ADX with the given period using EMA smoothing
func calcADX(highs, lows, closes []float64, period int) float64 {
	n := len(closes)
	if n < period*2+1 {
		return 0
	}

	plusDMs := make([]float64, n-1)
	minusDMs := make([]float64, n-1)
	trs := make([]float64, n-1)

	for i := 1; i < n; i++ {
		upMove := highs[i] - highs[i-1]
		downMove := lows[i-1] - lows[i]

		if upMove > downMove && upMove > 0 {
			plusDMs[i-1] = upMove
		}
		if downMove > upMove && downMove > 0 {
			minusDMs[i-1] = downMove
		}

		hl := highs[i] - lows[i]
		hpc := math.Abs(highs[i] - closes[i-1])
		lpc := math.Abs(lows[i] - closes[i-1])
		trs[i-1] = math.Max(hl, math.Max(hpc, lpc))
	}

	atr := calcEMA(trs, period)
	if atr == 0 {
		return 0
	}
	plusDI := 100 * calcEMA(plusDMs, period) / atr
	minusDI := 100 * calcEMA(minusDMs, period) / atr

	diSum := plusDI + minusDI
	if diSum == 0 {
		return 0
	}
	return 100 * math.Abs(plusDI-minusDI) / diSum
}

// maxSlice returns the maximum value in a slice
func maxSlice(data []float64) float64 {
	if len(data) == 0 {
		return 0
	}
	m := data[0]
	for _, v := range data[1:] {
		if v > m {
			m = v
		}
	}
	return m
}

func mean(data []float64) float64 {
	sum := 0.0
	for _, v := range data {
		sum += v
	}
	return sum / float64(len(data))
}

func stddev(data []float64, avg float64) float64 {
	sum := 0.0
	for _, v := range data {
		d := v - avg
		sum += d * d
	}
	return math.Sqrt(sum / float64(len(data)))
}

// calcEMA: period-day EMA using all of closes, seeded with SMA of first period days
func calcEMA(closes []float64, period int) float64 {
	if len(closes) < period {
		return mean(closes)
	}
	k := 2.0 / float64(period+1)
	ema := mean(closes[:period])
	for _, v := range closes[period:] {
		ema = v*k + ema*(1-k)
	}
	return ema
}

// calcRSI: RSI from a slice of closes (expects at least 2 elements)
func calcRSI(closes []float64) float64 {
	if len(closes) < 2 {
		return 50.0
	}
	var gains, losses float64
	for i := 1; i < len(closes); i++ {
		diff := closes[i] - closes[i-1]
		if diff > 0 {
			gains += diff
		} else {
			losses -= diff
		}
	}
	n := float64(len(closes) - 1)
	avgGain := gains / n
	avgLoss := losses / n
	if avgLoss == 0 {
		return 100.0
	}
	rs := avgGain / avgLoss
	return 100 - 100/(1+rs)
}
