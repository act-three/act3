package model

import (
	"slices"
	"sync"
	"time"
)

type progress struct {
	mu          sync.Mutex
	vIDByEpID   map[string]map[string]bool
	progByVidID map[string]ProgressItem
}

func (p *progress) addEpisodeVideo(epID, vID string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.vIDByEpID == nil {
		p.vIDByEpID = map[string]map[string]bool{}
	}
	if p.vIDByEpID[epID] == nil {
		p.vIDByEpID[epID] = map[string]bool{}
	}
	p.vIDByEpID[epID][vID] = true
}

func (p *progress) addVideo(vID, desc string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.progByVidID == nil {
		p.progByVidID = map[string]ProgressItem{}
	}
	p.progByVidID[vID] = ProgressItem{
		CreatedAt: time.Now(),
		Desc:      desc,
	}
}

func (p *progress) updateVideo(vID string, value float64) {
	p.mu.Lock()
	defer p.mu.Unlock()
	pi, ok := p.progByVidID[vID]
	if !ok {
		return
	}
	pi.value = value
	p.progByVidID[vID] = pi
}

func (p *progress) clearVideo(vID string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	delete(p.progByVidID, vID)
}

func (p *progress) getByEpisodeID(epID string) []ProgressItem {
	p.mu.Lock()
	defer p.mu.Unlock()
	var a []ProgressItem
	for vID := range p.vIDByEpID[epID] {
		a = append(a, p.progByVidID[vID])
	}
	slices.SortFunc(a, func(a, b ProgressItem) int {
		return a.CreatedAt.Compare(b.CreatedAt)
	})
	return a
}

type ProgressItem struct {
	CreatedAt time.Time
	Desc      string
	value     float64
}

func (pi *ProgressItem) Progress() float64 {
	return pi.value
}
