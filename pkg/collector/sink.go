// Package collector watches one Kubernetes cluster and pushes what it sees to
// a deploygrid server as Observations.
package collector

import (
	"sync"

	"github.com/activatedio/deploygrid/pkg/repository"
	"github.com/activatedio/deploygrid/pkg/store"
)

// kindSink receives reflector events for one resource kind. It mirrors the
// resources locally (so a full snapshot can be re-sent at any time) and
// queues deltas for the pusher.
type kindSink struct {
	kind   string
	mirror *store.Store
	lock   sync.Mutex
	// pending deltas since the last push; nil values mark removals
	upserts map[string]*repository.Resource
	removes map[string]*repository.Resource
	// full requests that the next push sends a snapshot instead of deltas
	full bool
	// errors are the latest watch problems; errorsDirty marks ones not yet
	// pushed so a push happens even without resource changes
	errors      []string
	errorsDirty bool
	notify      func()
}

func newKindSink(kind string, notify func()) *kindSink {
	return &kindSink{
		kind:    kind,
		mirror:  store.NewStore(),
		upserts: map[string]*repository.Resource{},
		removes: map[string]*repository.Resource{},
		full:    true,
		notify:  notify,
	}
}

func (s *kindSink) Add(in *repository.Resource) error {
	s.lock.Lock()
	_ = s.mirror.Add(in)
	delete(s.removes, in.Name)
	s.upserts[in.Name] = in
	s.lock.Unlock()
	s.notify()
	return nil
}

func (s *kindSink) Modify(in *repository.Resource) error {
	s.lock.Lock()
	// Add handles parent bookkeeping for resources the mirror has not seen.
	_ = s.mirror.Add(in)
	delete(s.removes, in.Name)
	s.upserts[in.Name] = in
	s.lock.Unlock()
	s.notify()
	return nil
}

func (s *kindSink) Delete(in *repository.Resource) error {
	s.lock.Lock()
	_ = s.mirror.Delete(in)
	delete(s.upserts, in.Name)
	s.removes[in.Name] = in
	s.lock.Unlock()
	s.notify()
	return nil
}

func (s *kindSink) Replace(in []*repository.Resource) error {
	s.lock.Lock()
	_ = s.mirror.Replace(in)
	s.upserts = map[string]*repository.Resource{}
	s.removes = map[string]*repository.Resource{}
	s.full = true
	s.errors = nil
	s.errorsDirty = true
	s.lock.Unlock()
	s.notify()
	return nil
}

func (s *kindSink) Error(err error) {
	s.lock.Lock()
	s.errors = append(s.errors, s.kind+": "+err.Error())
	if len(s.errors) > 5 {
		s.errors = s.errors[len(s.errors)-5:]
	}
	s.errorsDirty = true
	s.lock.Unlock()
	s.notify()
}

// RequestFull makes the next push a snapshot.
func (s *kindSink) RequestFull() {
	s.lock.Lock()
	s.full = true
	s.lock.Unlock()
}

// batch is what one push for this kind contains.
type batch struct {
	kind    string
	full    bool
	upserts []*repository.Resource
	removes []*repository.Resource
	errors  []string
}

// take drains pending changes. When a full snapshot is due it returns every
// mirrored resource. The second result is false when nothing is pending.
func (s *kindSink) take() (batch, bool) {
	s.lock.Lock()
	defer s.lock.Unlock()

	b := batch{kind: s.kind, errors: append([]string(nil), s.errors...)}
	if s.full {
		data, _ := s.mirror.GetData()
		for _, r := range data.Entries() {
			b.upserts = append(b.upserts, r)
		}
		b.full = true
		s.full = false
		s.errorsDirty = false
		s.upserts = map[string]*repository.Resource{}
		s.removes = map[string]*repository.Resource{}
		return b, true
	}
	if len(s.upserts) == 0 && len(s.removes) == 0 && !s.errorsDirty {
		return b, false
	}
	s.errorsDirty = false
	for _, r := range s.upserts {
		b.upserts = append(b.upserts, r)
	}
	for _, r := range s.removes {
		b.removes = append(b.removes, r)
	}
	s.upserts = map[string]*repository.Resource{}
	s.removes = map[string]*repository.Resource{}
	return b, true
}

// restore puts a batch back after a failed push so nothing is lost; the
// next push becomes a snapshot, which is simpler than replaying order.
func (s *kindSink) restore() {
	s.lock.Lock()
	s.full = true
	s.lock.Unlock()
}
