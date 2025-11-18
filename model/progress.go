package model

import (
	"slices"
	"sync"
	"time"
)

type progress struct {
	mu        sync.Mutex
	vIDByEpID map[string]map[string]bool
	rIDByvID  map[string]map[string]bool
	progByrID map[string]ProgressItem
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

func (p *progress) addRendition(vID, rID, desc string, total time.Duration) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.rIDByvID == nil {
		p.rIDByvID = map[string]map[string]bool{}
	}
	if p.rIDByvID[vID] == nil {
		p.rIDByvID[vID] = map[string]bool{}
	}
	p.rIDByvID[vID][rID] = true

	if p.progByrID == nil {
		p.progByrID = map[string]ProgressItem{}
	}
	p.progByrID[rID] = ProgressItem{
		CreatedAt: time.Now(),
		Desc:      desc,
		Total:     total,
	}
}

func (p *progress) updateRendition(rID string, progress time.Duration) {
	p.mu.Lock()
	defer p.mu.Unlock()
	pi, ok := p.progByrID[rID]
	if !ok {
		return
	}
	pi.Value = progress
	p.progByrID[rID] = pi
}

func (p *progress) clearRendition(vID, rID string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	delete(p.progByrID, rID)
	delete(p.rIDByvID[vID], rID)
}

func (p *progress) getByEpisodeID(epID string) []ProgressItem {
	p.mu.Lock()
	defer p.mu.Unlock()
	var a []ProgressItem
	for vID := range p.vIDByEpID[epID] {
		for rID := range p.rIDByvID[vID] {
			a = append(a, p.progByrID[rID])
		}
	}
	slices.SortFunc(a, func(a, b ProgressItem) int {
		return a.CreatedAt.Compare(b.CreatedAt)
	})
	return a
}

type ProgressItem struct {
	CreatedAt time.Time
	Desc      string
	Total     time.Duration
	Value     time.Duration
}

// Progress returns Value/Total, or 0 if Total == 0.
func (pi *ProgressItem) Progress() float64 {
	if pi.Total == 0 {
		return 0
	}
	return float64(pi.Value) / float64(pi.Total)
}
