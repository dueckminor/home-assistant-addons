package mqttbridge

import "time"

type Topic struct {
	Name  string
	Value string
	Time  time.Time
	state *Topic
}

var topics = make(map[string]*Topic)
