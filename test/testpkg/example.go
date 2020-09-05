package pkg

import (
	"io"
	"net/http"
)

// Foo type
// this is Foo type desc
type Foo struct {

	// FooString field docs 1
	//
	// FooString field docs 2-1
	// FooString field docs 2-2
	//
	// FooString field docs 3
	FooString string /* FooA field line comment */ // FooA comment2

	// FooInt field docs
	FooA, FooInt int /* FooB field line comment */ // FooB comment2

	// FooIntArray field docs
	FooIntArray []int /* FooIntArray field line comment */ // FooIntArray comment2

	// Reader field documentation
	Reader io.Reader // Reader line comment

	// Client field documentation
	http.Client // Client field line comment
}

// Status return status
// this is status desc
// this is status desc
func (f Foo) Status(a string) (s string) { // this is Boo.Status method inline comment
	return f.FooString
}

// Get return xxx
// this is status desc
// this is status desc
func (f Foo) Get(client, client2 http.Client) (*http.Response, error) { // this is Boo.Status method inline comment
	return nil, nil
}

// Boo alias Foo type
type Boo = Foo

// Stack test interface type
type Stack interface {
	Create(data map[string]interface{}) error
	Update(data map[string]interface{}) error
	Find(id string) (Foo, error)
	Delete(id string) error
}
