package node

import (
	"sync"
)

type CRDT interface {
	Read() interface{}
	Merge(CRDT)
	Add(string, interface{})
}

type GSet struct {
	sync.Mutex
	Set map[interface{}]struct{}
}

func NewGSet() *GSet {
	return &GSet{Set: make(map[interface{}]struct{})}
}

func (s *GSet) Read() interface{} {
	s.Lock()
	defer s.Unlock()
	values := []interface{}{}
	for value := range s.Set {
		values = append(values, value)
	}
	return values
}

func (s *GSet) Merge(irhs CRDT) {
	s.Lock()
	defer s.Unlock()
	rhs := irhs.(*GSet)
	for value := range rhs.Set {
		s.Set[value] = struct{}{}
	}
}

func (s *GSet) Add(src string, value interface{}) {
	s.Lock()
	defer s.Unlock()
	s.Set[value] = struct{}{}
}

var _ CRDT = &GSet{}

////////////////////////////////////////////////////////////////////////////////

type GCounter struct {
	sync.Mutex
	Counters map[string]int
}

func NewGCounter() *GCounter {
	return &GCounter{Counters: make(map[string]int)}
}

func (c *GCounter) Read() interface{} {
	c.Lock()
	defer c.Unlock()
	sum := 0
	for _, value := range c.Counters {
		sum += value
	}
	return sum
}

func (c *GCounter) Merge(irhs CRDT) {
	c.Lock()
	defer c.Unlock()
	rhsCounters := irhs.(*GCounter).Counters
	for k, v := range c.Counters {
		if rhsCounters[k] > v {
			c.Counters[k] = rhsCounters[k]
		}
	}
	for k, v := range rhsCounters {
		if v > c.Counters[k] {
			c.Counters[k] = v
		}
	}
}

func (c *GCounter) Add(src string, value interface{}) {
	c.Lock()
	defer c.Unlock()
	c.Counters[src] += int(value.(float64))
}

////////////////////////////////////////////////////////////////////////////////

type PNCounter struct {
	sync.Mutex
	Inc *GCounter
	Dec *GCounter
}

func NewPNCounter() *PNCounter {
	return &PNCounter{
		Inc: NewGCounter(),
		Dec: NewGCounter(),
	}
}

func (c *PNCounter) Read() interface{} {
	sum := c.Inc.Read().(int) - c.Dec.Read().(int)
	return sum
}

func (c *PNCounter) Merge(irhs CRDT) {
	rhsCounters := irhs.(*PNCounter)
	c.Inc.Merge(rhsCounters.Inc)
	c.Dec.Merge(rhsCounters.Dec)
}

func (c *PNCounter) Add(src string, value interface{}) {
	v := value.(float64)
	if v > 0 {
		c.Inc.Add(src, v)
	} else {
		c.Dec.Add(src, -v)
	}
}
