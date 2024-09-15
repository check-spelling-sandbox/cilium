// SPDX-License-Identifier: Apache-2.0
// Copyright Authors of Cilium

package monitor

import (
	"log/slog"
	"strings"

	"github.com/prometheus/client_golang/prometheus"

	flowpb "github.com/cilium/cilium/api/v1/flow"
	"github.com/cilium/cilium/pkg/bufuuid"
	"github.com/cilium/cilium/pkg/counter"
	"github.com/cilium/cilium/pkg/hubble/metrics"
	observerTypes "github.com/cilium/cilium/pkg/hubble/observer/types"
	"github.com/cilium/cilium/pkg/lock"
	"github.com/cilium/cilium/pkg/logging"
	"github.com/cilium/cilium/pkg/logging/logfields"
	monitorConsumer "github.com/cilium/cilium/pkg/monitor/agent/consumer"
	nodeTypes "github.com/cilium/cilium/pkg/node/types"
	"github.com/cilium/cilium/pkg/time"
)

// Observer is the receiver of MonitorEvents
type Observer interface {
	GetEventsChannel() chan *observerTypes.MonitorEvent
	GetLogger() *slog.Logger
}

var _ monitorConsumer.MonitorConsumer = (*consumer)(nil)

// consumer is a monitor consumer that sends events to an Observer.
type consumer struct {
	uuider   *bufuuid.Generator
	observer Observer

	lostLock         lock.Mutex
	lostEventCounter *counter.IntervalRangeCounter
	logLimiter       logging.Limiter

	metricLostPerfEvents     prometheus.Counter
	metricLostObserverEvents prometheus.Counter
}

// NewConsumer returns a new consumer that sends events to the provided Observer.
func NewConsumer(observer Observer, lostEventSendInterval time.Duration) *consumer {
	mc := &consumer{
		uuider:           bufuuid.New(),
		observer:         observer,
		lostEventCounter: counter.NewIntervalRangeCounter(lostEventSendInterval),
		logLimiter:       logging.NewLimiter(30*time.Second, 1),

		metricLostPerfEvents: metrics.LostEvents.WithLabelValues(
			strings.ToLower(flowpb.LostEventSource_PERF_EVENT_RING_BUFFER.String())),
		metricLostObserverEvents: metrics.LostEvents.WithLabelValues(
			strings.ToLower(flowpb.LostEventSource_OBSERVER_EVENTS_QUEUE.String())),
	}
	return mc
}

<<<<<<< HEAD
// NotifyAgentEvent implements monitorConsumer.MonitorConsumer.
func (c *consumer) NotifyAgentEvent(typ int, message any) {
	c.sendEvent(func() any {
=======
// sendEventQueueLostEvents tries to send the current value of the lost events
// counter to the observer. If it succeeds to enqueue a notification, it
// resets the counter. Returns a boolean indicating whether the notification
// has been successfully sent.
func (c *consumer) sendNumLostEvents() bool {
	c.lostLock.Lock()
	defer c.lostLock.Unlock()
	// check again, in case multiple
	// routines contended the lock
	if c.numEventsLost == 0 {
		return true
	}

	if c.cachedLostNotification == nil {
		c.cachedLostNotification = c.newEvent(func() interface{} {
			return &observerTypes.LostEvent{
				Source:        observerTypes.LostEventSourceEventsQueue,
				NumLostEvents: c.numEventsLost,
			}
		})
	} else {
		c.cachedLostNotification.Timestamp = time.Now()
		c.cachedLostNotification.Payload.(*observerTypes.LostEvent).NumLostEvents = c.numEventsLost
	}

	select {
	case c.observer.GetEventsChannel() <- c.cachedLostNotification:
		// We now safely reset the counter, as at this point have
		// successfully notified the observer about the amount of events
		// that were lost since the previous LostEvent message. Similarly,
		// we reset the cached notification, so that a new one is created
		// the next time.
		c.numEventsLost = 0
		c.cachedLostNotification = nil
		return true
	default:
		// We do not need to bump the numEventsLost counter here, as we will
		// try to send a new LostEvent notification again during the next
		// invocation of sendEvent
		return false
	}
}

// sendEvent enqueues an event in the observer. If this is not possible, it
// keeps a counter of lost events, which it will regularly try to send to the
// observer as well
func (c *consumer) sendEvent(payloader func() interface{}) {
	if c.numEventsLost > 0 {
		if !c.sendNumLostEvents() {
			// We just failed sending the lost notification, hence it doesn't
			// make sense to try and send the actual event, as we'll most
			// likely fail as well.
			c.countDroppedEvent()
			return
		}
	}

	select {
	case c.observer.GetEventsChannel() <- c.newEvent(payloader):
	default:
		c.countDroppedEvent()
	}
}

func (c *consumer) newEvent(payloader func() interface{}) *observerTypes.MonitorEvent {
	ev := &observerTypes.MonitorEvent{
		Timestamp: time.Now(),
		NodeName:  nodeTypes.GetAbsoluteNodeName(),
		Payload:   payloader(),
	}

	c.uuider.NewInto(&ev.UUID)
	return ev
}

// countDroppedEvent logs that the events channel is full
// and counts how many messages it has lost.
func (c *consumer) countDroppedEvent() {
	c.lostLock.Lock()
	defer c.lostLock.Unlock()
	if c.numEventsLost == 0 && c.logLimiter.Allow() {
		c.observer.GetLogger().WithField("related-metric", "hubble_lost_events_total").
			Warning("hubble events queue is full: dropping messages; consider increasing the queue size (hubble-event-queue-size) or provisioning more CPU")
	}
	c.numEventsLost++
	c.metricLostObserverEvents.Inc()
}

// NotifyAgentEvent implements monitorConsumer.MonitorConsumer
func (c *consumer) NotifyAgentEvent(typ int, message interface{}) {
	c.sendEvent(func() interface{} {
>>>>>>> 2a240854c (spelling: now)
		return &observerTypes.AgentEvent{
			Type:    typ,
			Message: message,
		}
	})
}

// NotifyPerfEvent implements monitorConsumer.MonitorConsumer.
func (c *consumer) NotifyPerfEvent(data []byte, cpu int) {
	c.sendEvent(func() any {
		return &observerTypes.PerfEvent{
			Data: data,
			CPU:  cpu,
		}
	})
}

// NotifyPerfEventLost implements monitorConsumer.MonitorConsumer.
func (c *consumer) NotifyPerfEventLost(numLostEvents uint64, cpu int) {
	c.sendEvent(func() any {
		return &observerTypes.LostEvent{
			Source:        observerTypes.LostEventSourcePerfRingBuffer,
			NumLostEvents: numLostEvents,
			CPU:           cpu,
		}
	})
	c.metricLostPerfEvents.Inc()
}

// sendEvent enqueues an event in the observer. If this is not possible, it
// keeps a counter of lost events, which it will try to send at most once per
// configured interval, and on every call to sendEvent until it succeeds.
func (c *consumer) sendEvent(payloader func() any) {
	c.lostLock.Lock()
	defer c.lostLock.Unlock()

	now := time.Now()
	c.trySendLostEventLocked(now)

	select {
	case c.observer.GetEventsChannel() <- c.newEvent(now, payloader):
	default:
		c.incrementLostEventLocked(now)
	}
}

func (c *consumer) newEvent(ts time.Time, payloader func() any) *observerTypes.MonitorEvent {
	ev := &observerTypes.MonitorEvent{
		Timestamp: ts,
		NodeName:  nodeTypes.GetAbsoluteNodeName(),
		Payload:   payloader(),
	}

	c.uuider.NewInto(&ev.UUID)
	return ev
}

// trySendLostEventLocked tries to send a lost event as needed. If it succeeds, it clears the
// lost event counter, otherwise it does nothing so we keep the existing count. It assumes that
// the caller holds c.lostLock.
func (c *consumer) trySendLostEventLocked(ts time.Time) {
	// check if we should send a lost event
	shouldSend := c.lostEventCounter.IsElapsed(ts)
	if !shouldSend {
		return
	}

	count := c.lostEventCounter.Peek()
	lostEvent := c.newEvent(ts, func() any {
		return &observerTypes.LostEvent{
			Source:        observerTypes.LostEventSourceEventsQueue,
			NumLostEvents: count.Count,
			First:         count.First,
			Last:          count.Last,
		}
	})

	select {
	case c.observer.GetEventsChannel() <- lostEvent:
		// only clear the counter if we successfully sent the lost event
		c.lostEventCounter.Clear()
	default:
	}
}

// incrementLostEventLocked increments the lost event counter. It also logs a warning message if the
// counter was previously empty and the log limiter allows it. It assumes that the caller holds
// c.lostLock.
func (c *consumer) incrementLostEventLocked(ts time.Time) {
	if c.lostEventCounter.Peek().Count == 0 && c.logLimiter.Allow() {
		c.observer.GetLogger().
			Warn(
				"hubble events queue is full: dropping messages; consider increasing the queue size (hubble-event-queue-size) or provisioning more CPU",
				logfields.RelatedMetric, "hubble_lost_events_total",
			)
	}
	c.lostEventCounter.Increment(ts)
	c.metricLostObserverEvents.Inc()
}
