package mmrtesting

import (
	"context"
)

type TestCallCounter struct {
	MethodCalls map[string]int
}

type TestSendCallCounter struct {
	TestCallCounter
}

func (r *TestCallCounter) IncMethodCall(name string) int {
	if r.MethodCalls == nil {
		r.MethodCalls = make(map[string]int)
	}

	cur, ok := r.MethodCalls[name]
	if !ok {
		r.MethodCalls[name] = 1
		return 1
	}
	r.MethodCalls[name] = cur + 1
	return cur + 1
}

func (r *TestCallCounter) Reset() {
	r.MethodCalls = make(map[string]int)
}

func (r *TestCallCounter) MethodCallCount(name string) int {

	cur, ok := r.MethodCalls[name]
	if !ok {
		return 0
	}
	return cur
}

func (s *TestSendCallCounter) Open() error           { return nil }
func (s *TestSendCallCounter) Close(context.Context) {}
func (s *TestSendCallCounter) String() string        { return "testSender" }
