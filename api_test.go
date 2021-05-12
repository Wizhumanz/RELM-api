package main

import (
	// "fmt"
	"os"
	"reflect"
	"testing"
)

type testPair struct {
	Inputs   []interface{}
	Expected interface{}
}

var testData []testPair

func setup() {
	testData = []testPair{
		{
			Inputs: []interface{}{
				[]Listing{
					{KEY: "A"},
					{KEY: "B"},
					{KEY: "C"},
				},
				Listing{KEY: "C"},
			},
			Expected: []Listing{
				{KEY: "A"},
				{KEY: "B"},
			},
		},
		{
			Inputs: []interface{}{
				[]Listing{
					{KEY: "D"},
					{KEY: "E"},
					{KEY: "F"},
				},
				Listing{KEY: "D"},
			},
			Expected: []Listing{
				{KEY: "E"},
				{KEY: "F"},
			},
		},
	}
}

func shutdown() {
	//shutdown
}

func TestMain(m *testing.M) {
	setup()
	code := m.Run()
	shutdown()
	os.Exit(code)
}

func TestDeleteElement(t *testing.T) {
	for _, pair := range testData {
		arg1 := pair.Inputs[0].([]Listing)
		arg2 := pair.Inputs[1].(Listing)
		expected := pair.Expected.([]Listing)
		res := deleteElement(arg1, arg2)
		if !reflect.DeepEqual(res, expected) {
			t.Errorf("Expected deleteElement(%s, %s) to equal %s, actual result = %s", arg1, arg2, expected, res)
		}
	}
}
