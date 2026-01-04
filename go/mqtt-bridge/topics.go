package mqttbridge

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/dueckminor/home-assistant-addons/go/services/mqtt"
)

type Topic struct {
	Name  string
	Value string
	Time  time.Time
}

type TopicRef interface {
	Get() Topic
	AddHandler(handler func(topic Topic))
}

type topicRef struct {
	Name     string
	topic    *Topic
	onChange []func(topic Topic)
}

func (t *topicRef) Get() Topic {
	return *t.topic
}
func (t *topicRef) AddHandler(handler func(topic Topic)) {
	t.onChange = append(t.onChange, handler)
}

var (
	topics   = make(map[string]*topicRef)
	topicsMu sync.RWMutex
	handlers []func(topic Topic)
)

func AddHandler(handlerFunc func(topic Topic)) {
	handlers = append(handlers, handlerFunc)
}

func GetTopics() []Topic {
	topicsMu.RLock()
	defer topicsMu.RUnlock()
	result := make([]Topic, 0, len(topics))
	for _, topicRef := range topics {
		if topicRef.topic != nil {
			result = append(result, *topicRef.topic)
		}
	}
	return result
}

func Listen(ctx context.Context, mqttConn mqtt.Conn) {
	newTopics := make(chan Topic, 100)
	changedTopics := make(chan topicRef, 100)

	addOneTopic := func(topic Topic) {
		// this function is called with topicsMu locked
		name := topic.Name
		fmt.Println("Adding topic:", name)
		topicR, exists := topics[name]
		if !exists {
			topicR = &topicRef{topic: &topic}
			topics[name] = topicR
		} else {
			topicR.topic = &topic
		}

		if strings.HasPrefix(name, "homeassistant/sensor") && strings.HasSuffix(name, "/config") {
			createOrUpdateMeasurement(topicR, func(name string) TopicRef {
				if existingTopic, ok := topics[name]; ok {
					return existingTopic
				}
				newTopic := &topicRef{Name: name}
				topics[name] = newTopic
				return newTopic
			})
		}

		if len(topicR.onChange) > 0 || len(handlers) > 0 {
			changedTopics <- *topicR
		}
	}

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case topic := <-newTopics:
				// Process this topic and any additional ones in the channel
				func() {
					topicsMu.Lock()
					defer topicsMu.Unlock()
					addOneTopic(topic)
					for {
						select {
						case <-ctx.Done():
							return
						case topic := <-newTopics:
							addOneTopic(topic)
						default:
							// No more topics in channel, exit inner loop
							return
						}
					}
				}()
			}
		}
	}()

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case topicR := <-changedTopics:
				// Call onChange handlers
				for _, handler := range topicR.onChange {
					handler(*topicR.topic)
				}
				for _, handler := range handlers {
					handler(*topicR.topic)
				}
			}
		}
	}()

	mqttConn.SubscribeCtx(ctx, "#", func(topic string, payload string) {
		newTopics <- Topic{
			Name:  topic,
			Value: payload,
			Time:  time.Now(),
		}
	})
}
