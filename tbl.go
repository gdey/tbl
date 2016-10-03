// Copyright 2016 Gautam Dey. All rights reserved.
// Use of this source code is governed by FreeBDS License (2-clause Simplified BSD.)
// that can be found in the LICENSE file.

package tbl

import (
	"flag"
	"fmt"
	"math/rand"
	"reflect"
	"runtime"
	"strconv"
	"strings"
)

var runorder = flag.String("tblTest.RunOrder", "", "List of comma seperated index of the test cases to run.")

// Test holds the testcases.
type Test struct {
	cases []reflect.Value
	vType reflect.Type
	// InOrder defines weather to run the test case in the order defined or randomly.
	// This option is overridden by the tblTest.RunOrder command line flag.
	InOrder bool
}

func panicF(format string, vals ...interface{}) {
	pc, _, _, ok := runtime.Caller(2)
	details := runtime.FuncForPC(pc)
	var callSite string
	if ok && details != nil {
		callSite = fmt.Sprintf("Called from %v: ", details.Name())
	}
	panic(fmt.Sprintf(callSite+format, vals...))
}

func runOrder() (idx []int, ok bool) {

	if runorder == nil || *runorder == "" {
		return []int{}, false
	}
	for _, s := range strings.Split(*runorder, ",") {
		// Only care about the good values.
		if i, err := strconv.ParseInt(s, 10, 64); err == nil {
			idx = append(idx, int(i))
		}
	}
	return idx, len(idx) > 0
}

// Cases takes a list of test cases to use for the table driven tests.
//   The test cases can be any type, as long as they are all the same.
func Cases(testcases ...interface{}) *Test {
	tc := Test{}
	for i, tcase := range testcases {
		val := reflect.ValueOf(tcase)
		if val.Kind() == reflect.Invalid {
			panicF("Testcase %v is not a valid test case.", i)
		}
		// The first element determines that type of the rest of the elements.
		if tc.vType == nil {
			tc.vType = val.Type()
		} else {
			if val.Type() != tc.vType {
				panicF("Testcases should be of type %v, but element %v is of type %v.", tc.vType, i, val.Type())
			}
		}
		tc.cases = append(tc.cases, val)
	}
	return &tc
}

func runTest(fn reflect.Value, idx int, testcase reflect.Value, tp bool, r bool) bool {
	var params []reflect.Value
	if tp {
		params = append(params, reflect.ValueOf(idx))
	}
	params = append(params, testcase)
	res := fn.Call(params)
	if r {
		return res[0].Bool()
	}
	return true
}

// Run calls the given function for each test case. (Note the function may be called again with the same testcase, if the tblTest.RunOrder option is specified.)
// The function must take one of four forms.
//
//    *  `func (tc $testcase)`
//
//    *  `func (tc $testcase) bool`
//
//    *  `func (idx int, tc $testcase)`
//
//    *  `func (idx int, tc $testcase) bool`
//
func (tc *Test) Run(function interface{}) int {
	fn := reflect.ValueOf(function)
	fnType := fn.Type()

	if fnType.Kind() != reflect.Func {
		panicF("Was not provided a function.")
	}
	// Check the paramaters.
	var twoInParams bool
	var hasOutParam bool
	switch fnType.NumIn() {
	// If there is only one parameter then it should of the test case type.
	case 1:
		if fnType.In(0) != tc.vType {
			panicF("Incorrect parameter for test function given. Was given %v, expected it to be %v", fnType.In(0), tc.vType)
		}
	case 2:
		if fnType.In(0) != reflect.TypeOf(int(1)) {
			panicF("Incorrect parameter one for test function given. Was given %v, expected it to be int", fnType.In(0))
		}
		if fnType.In(1) != tc.vType {
			panicF("Incorrect parameter two for test function given. Was given %v, expected it to be %v", fnType.In(0), tc.vType)
		}
		twoInParams = true
	default:
		panicF("Incorrect number of parameters given. Expect the funtion to take one of two forms. func(idx int, testcase $T) or func(testcase $T)")
	}
	switch fnType.NumOut() {
	case 0:
	// Nothing to do.
	case 1:
		if fnType.Out(0) != reflect.TypeOf(true) {
			panicF("Expected out parameter of test function to be a boolean. Was given %v", fnType.Out(0))
		}
		hasOutParam = true
	default:
		panicF("Expected there to be not out parameters or a boolean out parameter to test function.")
	}
	if len(tc.cases) == 0 {
		return 0
	}
	// Now loop through the testcase and call the test function, check to see if we should stop or keep going.
	count := 0
	if idxs, ok := runOrder(); ok {
		for _, idx := range idxs {
			if idx < 0 || idx >= len(tc.cases) {
				continue
			}
			count++
			if !runTest(fn, idx, tc.cases[idx], twoInParams, hasOutParam) {
				break
			}
		}
		return count
	}
	if tc.InOrder {
		for idx, testcase := range tc.cases {
			count++
			if !runTest(fn, idx, testcase, twoInParams, hasOutParam) {
				break
			}
		}
		return count
	}
	list := rand.Perm(len(tc.cases))
	for _, idx := range list {
		count++
		testcase := tc.cases[idx]
		if !runTest(fn, idx, testcase, twoInParams, hasOutParam) {
			break
		}
	}
	return count
}
