package service_test

import (
	"context"
	"fmt"
	"github.com/activatedio/deploygrid/pkg/apiinfra/util"
	"github.com/activatedio/deploygrid/pkg/apiinfra/zerolog"
	"github.com/activatedio/deploygrid/pkg/config"
	"github.com/activatedio/deploygrid/pkg/repository"
	"github.com/activatedio/deploygrid/pkg/service"
	"github.com/google/uuid"
	"sync"
	"testing"
	"time"
)

func init() {
	zerolog.ConfigureLogging(&config.LoggingConfig{
		Level:   "info",
		DevMode: false,
	})
}

func randomString() string {
	return uuid.New().String()
}

type cycleCounter struct {
	readCount      int64
	writeCount     int64
	readCountLock  sync.Mutex
	writeCountLock sync.Mutex
}

func (c *cycleCounter) incrementReadCount() {
	c.readCountLock.Lock()
	defer c.readCountLock.Unlock()

	c.readCount = c.readCount + 1
}

func (c *cycleCounter) incrementWriteCount() {
	c.writeCountLock.Lock()
	defer c.writeCountLock.Unlock()
	c.writeCount = c.writeCount + 1
}

func TestStore_Concurrency_AddModifyDelete(t *testing.T) {

	c := &cycleCounter{}

	// We make the writer so that each go routine has unique data
	makeWriter := func() func(s *service.Store) {

		parents := []*repository.Resource{
			{
				Name:       randomString(),
				Components: []repository.Component{},
			},
			{
				Name:       randomString(),
				Components: []repository.Component{},
			},
		}

		children := []*repository.Resource{
			{
				Name:       randomString(),
				Parent:     parents[0].Name,
				Components: []repository.Component{},
			},
			{
				Name:       randomString(),
				Parent:     parents[0].Name,
				Components: []repository.Component{},
			},
			{
				Name:       randomString(),
				Parent:     parents[1].Name,
				Components: []repository.Component{},
			},
			{
				Name:       randomString(),
				Parent:     parents[1].Name,
				Components: []repository.Component{},
			},
		}

		return func(s *service.Store) {

			for _, p := range parents {
				util.Check(s.Add(p))
			}
			for _, c := range children {
				util.Check(s.Add(c))
			}
			for _, p := range parents {
				util.Check(s.Modify(p))
			}
			for _, c := range children {
				util.Check(s.Modify(c))
			}
			for _, c := range children {
				util.Check(s.Delete(c))
			}
			for _, p := range parents {
				util.Check(s.Delete(p))
			}

			c.incrementWriteCount()
		}
	}

	reader := func(s *service.Store) {

		d, err := s.GetData()
		util.Check(err)
		if d == nil {
			panic("data is nil")
		}

		c.incrementReadCount()
	}

	ctx, cancel := context.WithCancel(context.Background())

	unit := service.NewStore()

	wg := sync.WaitGroup{}

	for i := 0; i < 100; i++ {

		wg.Add(2)

		writer := makeWriter()

		go func() {
			for {
				select {
				case <-ctx.Done():
					wg.Done()
					return
				default:
					writer(unit)
				}
			}
		}()
		go func() {
			for {
				select {
				case <-ctx.Done():
					wg.Done()
					return
				default:
					reader(unit)
				}
			}
		}()
	}

	time.Sleep(5 * time.Second)
	fmt.Println("cancelling")
	cancel()
	fmt.Println("waiting")
	wg.Wait()
	fmt.Printf("reads: %d, writes %d\n", c.readCount, c.writeCount)

}
