package network

import (
	"time"
)

type Metric struct {
	Timestamp    time.Time
	ClientAddr   string
	Duration     time.Duration
	Hostname     string
	Method       string
	Path         string
	ResponseCode int
}

type MetricCallback func(metric Metric)
