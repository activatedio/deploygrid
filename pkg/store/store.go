package store

import (
	"slices"
	"sync"

	"github.com/go-errors/errors"
	"github.com/rs/zerolog/log"

	"github.com/activatedio/deploygrid/pkg/repository"
)

type StoreData struct {
	entries   map[string]*repository.Resource
	parentMap map[string]map[string]bool
}

func NewStoreData() *StoreData {
	s := &StoreData{}
	s.init()
	return s
}

func (s *StoreData) init() {
	s.entries = map[string]*repository.Resource{}
	s.parentMap = map[string]map[string]bool{}
}

// Entries returns every resource in the snapshot keyed by store name.
func (s *StoreData) Entries() map[string]*repository.Resource {
	return s.entries
}

// Children returns the store names of resources whose Parent is the given
// name, sorted for determinism.
func (s *StoreData) Children(parent string) []string {
	pm := s.parentMap[parent]
	res := make([]string, 0, len(pm))
	for k := range pm {
		res = append(res, k)
	}
	slices.Sort(res)
	return res
}

// Requires locking externally
func (s *StoreData) copy() *StoreData {

	res := NewStoreData()
	for k, v := range s.entries {
		res.entries[k] = v
	}
	for k, v := range s.parentMap {
		res.parentMap[k] = v
	}
	return res
}

// AddAll merges another snapshot into this one.
func (s *StoreData) AddAll(in *StoreData) {
	for k, v := range in.entries {
		s.entries[k] = v
	}
	for k, v := range in.parentMap {
		s.parentMap[k] = v
	}
}

type Store struct {
	err      error
	lock     sync.RWMutex
	data     *StoreData
	snapshot *StoreData
}

func NewStore() *Store {
	s := &Store{}
	s.init()
	return s
}

func (s *Store) init() {
	s.data = NewStoreData()
	s.clearSnapshot()
	s.clearError()
}

func (s *Store) clearSnapshot() {
	s.snapshot = nil
}

func (s *Store) clearError() {
	s.err = nil
}

func (s *Store) getParentMap(key string) map[string]bool {
	if pm, ok := s.data.parentMap[key]; ok {
		return pm
	}
	pm := map[string]bool{}
	s.data.parentMap[key] = pm
	return pm
}

func (s *Store) GetData() (*StoreData, error) {
	s.lock.RLock()
	tmp := s.snapshot
	s.lock.RUnlock()

	if tmp != nil {
		return tmp, s.err
	}

	s.lock.Lock()
	defer s.lock.Unlock()
	tmp = s.snapshot
	// Someone else got the lock and updated it first
	if tmp != nil {
		return tmp, s.err
	}
	s.snapshot = s.data.copy()
	return s.snapshot, s.err
}

func (s *Store) Add(in *repository.Resource) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.addNoLock(in)
	return nil
}

func (s *Store) addNoLock(in *repository.Resource) {

	log.Info().Interface("resource", in).Msg("adding to store")

	s.data.entries[in.Name] = in

	if in.Parent != "" {
		s.getParentMap(in.Parent)[in.Name] = true
	}

	s.clearSnapshot()
	s.clearError()
}

// Parents are set on creation and cannot be modified
func (s *Store) Modify(in *repository.Resource) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	log.Info().Interface("resource", in).Msg("modifying in store")

	if existing, ok := s.data.entries[in.Name]; ok && existing.Parent != in.Parent {
		return errors.New("cannot modify parent")
	}

	s.data.entries[in.Name] = in

	s.clearSnapshot()
	s.clearError()

	return nil
}

func (s *Store) Delete(in *repository.Resource) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	log.Info().Interface("resource", in).Msg("deleting from store")

	if existing, ok := s.data.entries[in.Name]; ok && existing.Parent != in.Parent {
		return errors.New("cannot modify parent")
	}
	delete(s.data.entries, in.Name)

	if in.Parent != "" {
		delete(s.getParentMap(in.Parent), in.Name)
	}

	s.clearSnapshot()
	s.clearError()

	return nil
}

func (s *Store) Replace(in []*repository.Resource) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	log.Info().Interface("resources", in).Msg("replacing store")

	s.init()

	for _, r := range in {
		s.addNoLock(r)
	}

	return nil
}

func (s *Store) Error(err error) {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.errorNoLock(err)
}

func (s *Store) errorNoLock(err error) {
	s.err = err
}
