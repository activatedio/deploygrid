package repository

import "sync"

type recordingResourceStore struct {
	records []ResourceStoreRecord
	lock    sync.Mutex
}

func (r *recordingResourceStore) Error(err error) {

	r.lock.Lock()
	defer r.lock.Unlock()

	r.records = append(r.records, ResourceStoreRecord{
		EventType: ResourceStoreEventError,
		Error:     err,
	})
}

func (r *recordingResourceStore) GetRecords() []ResourceStoreRecord {
	r.lock.Lock()
	defer r.lock.Unlock()

	return r.records

}

func (r *recordingResourceStore) Add(in *Resource) error {
	r.lock.Lock()
	defer r.lock.Unlock()

	r.records = append(r.records, ResourceStoreRecord{
		EventType: ResourceStoreEventAdd,
		Resource:  in,
	})

	return nil
}

func (r *recordingResourceStore) Modify(in *Resource) error {
	r.lock.Lock()
	defer r.lock.Unlock()

	r.records = append(r.records, ResourceStoreRecord{
		EventType: ResourceStoreEventModify,
		Resource:  in,
	})

	return nil
}

func (r *recordingResourceStore) Delete(in *Resource) error {
	r.lock.Lock()
	defer r.lock.Unlock()

	r.records = append(r.records, ResourceStoreRecord{
		EventType: ResourceStoreEventDelete,
		Resource:  in,
	})

	return nil
}

func (r *recordingResourceStore) Replace(in []*Resource) error {
	r.lock.Lock()
	defer r.lock.Unlock()

	r.records = append(r.records, ResourceStoreRecord{
		EventType:     ResourceStoreEventReplace,
		ResourceArray: in,
	})

	return nil
}

type ResourceStoreEventType struct {
	slug string
}

var (
	ResourceStoreEventAdd     = ResourceStoreEventType{"add"}
	ResourceStoreEventModify  = ResourceStoreEventType{"modify"}
	ResourceStoreEventDelete  = ResourceStoreEventType{"delete"}
	ResourceStoreEventReplace = ResourceStoreEventType{"replace"}
	ResourceStoreEventError   = ResourceStoreEventType{"error"}
)

type ResourceStoreRecord struct {
	EventType     ResourceStoreEventType
	Error         error
	Resource      *Resource
	ResourceArray []*Resource
}

type RecordingResourceStore interface {
	ResourceStore
	GetRecords() []ResourceStoreRecord
}

func NewRecordingResourceStore() RecordingResourceStore {
	return &recordingResourceStore{}
}
