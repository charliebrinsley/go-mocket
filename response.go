package go_mocket

import (
	"database/sql/driver"
	"fmt"
	"log"
	"reflect"
	"strings"
)

var Catcher *MockCatcher

type MockCatcher struct {
	Mocks                []*FakeResponse
	Logging              bool
	PanicOnEmptyResponse bool // If not response matches - do we need to panic?
}

func (this *MockCatcher) Attach(fr []*FakeResponse) {
	this.Mocks = append(this.Mocks, fr...)
}

// Find suitable response by provided
func (this *MockCatcher) FindResponse(query string, args []driver.NamedValue) *FakeResponse {
	if this.Logging {
		log.Printf("mock_catcher: check query: %s", query)
	}

	for _, resp := range this.Mocks {
		if resp.IsMatch(query, args) {
			resp.MarkAsTriggered()
			return resp
		}
	}

	if this.PanicOnEmptyResponse {
		panic(fmt.Sprintf("No responses matches query %s ", query))
	}

	// Let's have always dummy version of response
	return &FakeResponse{
		Response:   make([]map[string]interface{}, 0),
		Exceptions: &Exceptions{},
	}
}

// Create new FakeResponse and return for chains of attachments
func (this *MockCatcher) NewMock() *FakeResponse {
	fr := &FakeResponse{Exceptions: &Exceptions{}, Response: make([]map[string]interface{}, 0)}
	this.Mocks = append(this.Mocks, fr)
	return fr
}

// Remove all Mocks to start process again
func (this *MockCatcher) Reset() *MockCatcher {
	this.Mocks = make([]*FakeResponse, 0)
	return this
}

// Possible exceptions during query executions
type Exceptions struct {
	HookQueryBadConnection func() bool
	HookExecBadConnection  func() bool
}

// Represents mock of response with holding all required values to return mocked response
type FakeResponse struct {
	Pattern      string                            // SQL query pattern to match with
	Args         []interface{}                     // List args to be matched with
	Response     []map[string]interface{}          // Array of rows to be parsed as result
	Once         bool                              // To trigger only once
	Triggered    bool                              // If it was triggered at least once
	Callback     func(string, []driver.NamedValue) // Callback to execute when response triggered
	RowsAffected int64                             // Defines affected rows count
	LastInsertId int64                             // ID to be returned for INSERT queries
	*Exceptions
}

// Return true either when nothing to compare or deep equal check passed
func (fr *FakeResponse) isArgsMatch(args []driver.NamedValue) bool {
	arguments := make([]interface{}, len(args))
	if len(args) > 0 {
		for index, arg := range args {
			arguments[index] = arg.Value
		}
	}
	return fr.Args == nil || reflect.DeepEqual(fr.Args, arguments)
}

func (fr *FakeResponse) isQueryMatch(query string) bool {
	return fr.Pattern == "" || strings.Contains(query, fr.Pattern)
}

func (fr *FakeResponse) IsMatch(query string, args []driver.NamedValue) bool {
	if fr.Once && fr.Triggered {
		return false
	}
	return fr.isQueryMatch(query) && fr.isArgsMatch(args)
}

func (fr *FakeResponse) MarkAsTriggered() {
	fr.Triggered = true
}

// For chaining init
func (fr *FakeResponse) WithQuery(query string) *FakeResponse {
	fr.Pattern = query
	return fr
}

// Attach Args check for prepared statements
func (fr *FakeResponse) WithArgs(vars ...interface{}) *FakeResponse {
	if len(vars) > 0 {
		fr.Args = make([]interface{}, len(vars))
		for index, v := range vars {
			fr.Args[index] = v
		}
	}
	return fr
}

// Methods to chain and assign some parts of response

func (fr *FakeResponse) WithReply(response []map[string]interface{}) *FakeResponse {
	fr.Response = response
	return fr
}

func (fr *FakeResponse) OneTime() *FakeResponse {
	fr.Once = true
	return fr
}

func (fr *FakeResponse) WithExecException() *FakeResponse {
	fr.Exceptions.HookExecBadConnection = func() bool {
		return true
	}
	return fr
}

func (fr *FakeResponse) WithQueryException() *FakeResponse {
	fr.Exceptions.HookQueryBadConnection = func() bool {
		return true
	}
	return fr
}

func (fr *FakeResponse) WithCallback(f func(string, []driver.NamedValue)) *FakeResponse {
	fr.Callback = f
	return fr
}

func (fr *FakeResponse) WithRowsNum(num int64) *FakeResponse {
	fr.RowsAffected = num
	return fr
}

func (fr *FakeResponse) WithId(id int64) *FakeResponse {
	fr.LastInsertId = id
	return fr
}

func init() {
	Catcher = &MockCatcher{}
}
