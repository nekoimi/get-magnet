package aria2

import (
	"sync"
	"time"
)

type ActiveRepo struct {
	mmu       *sync.RWMutex
	magnetMap map[string]time.Time
}

func newActiveRepo() *ActiveRepo {
	return &ActiveRepo{
		mmu:       &sync.RWMutex{},
		magnetMap: make(map[string]time.Time),
	}
}

func (ar *ActiveRepo) put(gid string) {
	ar.mmu.Lock()
	defer ar.mmu.Unlock()
	ar.magnetMap[gid] = time.Now()
}

func (ar *ActiveRepo) del(gid string) {
	ar.mmu.Lock()
	defer ar.mmu.Unlock()
	delete(ar.magnetMap, gid)
}

func (ar *ActiveRepo) isEmpty() bool {
	ar.mmu.Lock()
	defer ar.mmu.Unlock()
	return len(ar.magnetMap) > 0
}

func (ar *ActiveRepo) each(callback func(gid string, createdAt time.Time)) {
	ar.mmu.RLock()
	defer ar.mmu.RUnlock()
	for gid, t := range ar.magnetMap {
		callback(gid, t)
	}
}
