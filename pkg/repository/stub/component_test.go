package stub_test

import (
	"context"
	"github.com/activatedio/deploygrid/pkg/config"
	"github.com/activatedio/deploygrid/pkg/repository"
	"github.com/activatedio/deploygrid/pkg/repository/stub"
	"github.com/stretchr/testify/assert"
	"slices"
	"testing"
)

func TestComponentRepository_List(t *testing.T) {

	a := assert.New(t)

	type s struct {
		arrange func() ([]repository.Component, repository.ComponentCriteria)
		assert  func(got []repository.Component, err error)
	}

	fullList := []repository.Component{
		{
			Namespace:  "n1",
			ApiVersion: "a1",
			Kind:       "k1",
			Name:       "name1",
		},
		{
			Namespace:  "n1",
			ApiVersion: "a2",
			Kind:       "k2",
			Name:       "name2",
		},
		{
			Namespace:  "n2",
			ApiVersion: "a1",
			Kind:       "k1",
			Name:       "name1",
		},
		{
			Namespace:  "n2",
			ApiVersion: "a2",
			Kind:       "k2",
			Name:       "name2",
		},
	}

	cases := map[string]s{
		"empty": {
			arrange: func() ([]repository.Component, repository.ComponentCriteria) {
				return nil, repository.ComponentCriteria{}
			},
			assert: func(got []repository.Component, err error) {
				a.Nil(got)
				a.Nil(err)
			},
		},
		"two matches no namespace": {
			arrange: func() ([]repository.Component, repository.ComponentCriteria) {
				return fullList, repository.ComponentCriteria{
					ApiVersion: "a1",
					Kind:       "k1",
				}
			},
			assert: func(got []repository.Component, err error) {
				a.Nil(err)
				a.Equal([]repository.Component{
					fullList[0],
					fullList[2],
				}, got)
			},
		},
		"two matches namespace": {
			arrange: func() ([]repository.Component, repository.ComponentCriteria) {
				return fullList, repository.ComponentCriteria{
					Namespace:  "n1",
					ApiVersion: "a1",
					Kind:       "k1",
				}
			},
			assert: func(got []repository.Component, err error) {
				a.Nil(err)
				a.Equal([]repository.Component{
					fullList[0],
				}, got)
			},
		},
	}

	for k, v := range cases {
		t.Run(k, func(t *testing.T) {

			bs, c := v.arrange()
			unit := stub.NewComponentRepository(bs)

			v.assert(unit.List(context.TODO(), c))
		})
	}
}

func TestComponentRepositoryClusterAwareAccessor_ClusterNames(t *testing.T) {

	a := assert.New(t)

	type s struct {
		arrange func() []byte
		assert  func(got []string)
	}

	cases := map[string]s{
		"empty": {
			arrange: func() []byte {
				return []byte(`---`)
			},
			assert: func(got []string) {
				a.Nil(got)
			},
		},
		"two empty": {
			arrange: func() []byte {
				return []byte(`---
one: []
two: []
`)
			},
			assert: func(got []string) {
				slices.Sort(got)
				a.Equal([]string{"one", "two"}, got)
			},
		},
	}

	for k, v := range cases {
		t.Run(k, func(t *testing.T) {

			bs := v.arrange()

			unit := stub.NewComponentRepositoryClusterAwareAccessor(stub.ComponentRepositoryClusterAwareAccessorParams{
				RepositoryStubConfig: &config.RepositoryStubConfig{
					StaticDataBytes: bs,
				},
			})

			v.assert(unit.ClusterNames(context.TODO()))
		})
	}
}

func TestComponentRepositoryClusterAwareAccessor_Get(t *testing.T) {

	a := assert.New(t)

	type s struct {
		arrange func() (string, []byte)
		assert  func(got repository.ComponentRepository, err error)
	}

	cases := map[string]s{
		"empty": {
			arrange: func() (string, []byte) {
				return "stub", []byte(`---`)
			},
			assert: func(got repository.ComponentRepository, err error) {
				a.Nil(got)
				a.EqualError(err, "cluster not found: stub")
			},
		},
		"two empty": {
			arrange: func() (string, []byte) {
				return "one", []byte(`---
one: []
two: []
`)
			},
			assert: func(got repository.ComponentRepository, err error) {
				a.Nil(err)
				l, err := got.List(context.TODO(), repository.ComponentCriteria{})
				a.Len(l, 0)
				a.Nil(err)
			},
		},
		"two full": {
			arrange: func() (string, []byte) {
				return "one", []byte(`---
one: 
  - {}
  - {}
two: []
`)
			},
			assert: func(got repository.ComponentRepository, err error) {
				a.Nil(err)
				l, err := got.List(context.TODO(), repository.ComponentCriteria{})
				a.Len(l, 2)
				a.Nil(err)
			},
		},
	}

	for k, v := range cases {
		t.Run(k, func(t *testing.T) {

			name, bs := v.arrange()

			unit := stub.NewComponentRepositoryClusterAwareAccessor(stub.ComponentRepositoryClusterAwareAccessorParams{
				RepositoryStubConfig: &config.RepositoryStubConfig{
					StaticDataBytes: bs,
				},
			})

			v.assert(unit.Get(context.TODO(), name))
		})
	}
}
