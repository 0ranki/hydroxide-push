package events

import (
	"log"
	"os"
	"sync"
	"time"

	"github.com/0ranki/hydroxide-push/protonmail"
)

const pollInterval = 10 * time.Second

type Receiver struct {
	c *protonmail.Client

	locker   sync.Mutex
	channels []chan<- *protonmail.Event

	poll chan struct{}
}

func (r *Receiver) receiveEvents() {
	interval := pollInterval
	if os.Getenv("POLL_INTERVAL") {
		var err error
		interval, err = time.ParseDuration(os.Getenv("POLL_INTERVAL") + "s")
		if err != nil {
			log.Printf("failed to parse POLL_INTERVAL: %v\n", err)
			log.Println("falling back to default 10s interval")
		} else {
			log.Printf("poll interval set to %d seconds", int(interval.Seconds()))
		}
	}
	t := time.NewTicker(interval)
	defer t.Stop()

	var last string
	for {
		event, err := r.c.GetEvent(last)
		if err != nil {
			log.Println("cannot receive event:", err)
			select {
			case <-t.C:
			case <-r.poll:
			}
			continue
		}
		last = event.ID

		r.locker.Lock()
		n := len(r.channels)
		for _, ch := range r.channels {
			ch <- event
		}
		r.locker.Unlock()

		if n == 0 {
			break
		}

		select {
		case <-t.C:
		case <-r.poll:
		}
	}
}

func (r *Receiver) Poll() {
	r.poll <- struct{}{}
}

type Manager struct {
	receivers map[string]*Receiver
	locker    sync.Mutex
}

func NewManager() *Manager {
	return &Manager{
		receivers: make(map[string]*Receiver),
	}
}

func (m *Manager) Register(c *protonmail.Client, username string, ch chan<- *protonmail.Event, done <-chan struct{}) *Receiver {
	m.locker.Lock()
	defer m.locker.Unlock()

	r, ok := m.receivers[username]
	if ok {
		r.locker.Lock()
		r.channels = append(r.channels, ch)
		r.locker.Unlock()
	} else {
		r = &Receiver{
			c:        c,
			channels: []chan<- *protonmail.Event{ch},
			poll:     make(chan struct{}),
		}

		go func() {
			r.receiveEvents()

			m.locker.Lock()
			delete(m.receivers, username)
			m.locker.Unlock()
		}()

		m.receivers[username] = r
	}

	if done != nil {
		go func() {
			<-done

			r.locker.Lock()
			for i, c := range r.channels {
				if c == ch {
					r.channels = append(r.channels[:i], r.channels[i+1:]...)
				}
			}
			r.locker.Unlock()

			close(ch)
		}()
	}

	return r
}
