package backtest

import "time"

// TimeSimulator 时间模拟器,用于回测中推进时间
type TimeSimulator struct {
	currentTime time.Time
	endTime     time.Time
	interval    time.Duration
}

// NewTimeSimulator 创建时间模拟器
func NewTimeSimulator(startTime, endTime time.Time, interval time.Duration) *TimeSimulator {
	return &TimeSimulator{
		currentTime: startTime,
		endTime:     endTime,
		interval:    interval,
	}
}

// CurrentTime 获取当前时间
func (ts *TimeSimulator) CurrentTime() time.Time {
	return ts.currentTime
}

// Next 推进到下一个时间点
func (ts *TimeSimulator) Next() bool {
	ts.currentTime = ts.currentTime.Add(ts.interval)
	return ts.currentTime.Before(ts.endTime) || ts.currentTime.Equal(ts.endTime)
}

// HasNext 检查是否还有下一个时间点
func (ts *TimeSimulator) HasNext() bool {
	nextTime := ts.currentTime.Add(ts.interval)
	return nextTime.Before(ts.endTime) || nextTime.Equal(ts.endTime)
}

// Progress 获取回测进度(0.0 - 1.0)
func (ts *TimeSimulator) Progress() float64 {
	total := ts.endTime.Sub(ts.currentTime.Add(-ts.interval)).Seconds()
	elapsed := ts.currentTime.Sub(ts.currentTime.Add(-ts.interval)).Seconds()
	if total <= 0 {
		return 1.0
	}
	progress := elapsed / total
	if progress > 1.0 {
		return 1.0
	}
	return progress
}
