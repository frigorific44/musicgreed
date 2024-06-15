package concurrency

import "sync"

type RWInt struct {
	value int
	mutex sync.RWMutex
}

func (rwi *RWInt) Get() int {
	rwi.mutex.RLock()
	defer rwi.mutex.RUnlock()
	i := rwi.value
	return i
}

func (rwi *RWInt) Set(i int) {
	rwi.mutex.Lock()
	defer rwi.mutex.Unlock()
	rwi.value = i
}
