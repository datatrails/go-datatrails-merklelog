package snowflakeid

import (
	"errors"
	"fmt"
	"sync/atomic"
	"testing"
)

// Benchmark_NextIDStressTest stresses the id generator has hard as the host CPU
// will allow. This is *way* beyond what we expect service code to be able to
// achieve.
func Benchmark_NextIDStressTest(b *testing.B) {

	const msgDuplicationViolation = "*UV*"    // "uniqueness violation: two consecutive ids the same"
	const msgMonotonicViolation = "*MV*"      // "monotonic violation: new id less than previous"
	const msgMonotonicTimeViolation = "*MVt*" // "monotonic violation: new id timestamp less than previous"

	var errCount atomic.Int32
	var idCount atomic.Int32
	var sharedLast atomic.Uint64

	type sample struct {
		id  uint64
		iid int32
	}

	idc := make(chan sample, 1000*10)

	cfg := Config{
		CommitmentEpoch: 1,
		WorkerCIDR:      "0.0.0.0/16",
		PodIP:           "10.0.0.1",
		AllowSpins:      100,
	}
	s, err := NewIDState(cfg)
	if err != nil {
		b.Fatalf("initializing benchmark: %v", err)
	}

	go func(c chan sample) {
		// This is all about showing we don't get historic duplication. Note
		// that the order of id's on the queue can not be guaranteed to be
		// the order of creation, as to post to the queue goes via mutex
		// acquisition

		var i int32
		all := make(map[uint64][2]int32)

		for it := range c {
			id := it.id
			iid := it.iid

			if iprev, ok := all[id]; ok {
				fmt.Printf("all:[%d:%d:%d]p %s: id %016x\n", i, iprev, iid, msgDuplicationViolation, id)
			}
			all[id] = [2]int32{iid, i}
			i += 1
		}
	}(idc)

	b.RunParallel(func(pb *testing.PB) {

		// We can check that *each go-routine* sees a consistent series of ids
		//
		// But we can't check id monotonicity across go-routines without using
		// a lock. Channels internally use locks. And if we take the time to
		// acquire the lock on send, we will not be able to saturate the id
		// generator. Notice that if checkAll is true, the error rate drops

		var last uint64
		var lastShared uint64
		var iidLast int32

		for pb.Next() {

			// It is not possible to check the shared state perfectly. When
			// checkShared is enabled this test *will* perceive violations. That
			// we check the shared state at all is by  way of demonstrating that
			// this test *can* detect these issues.
			const checkShared = false

			// The local check is sufficient to catch violations. Change NextID
			// to use now instead of lastTime to deal with the <= case and see
			// what happens.
			const checkLocal = true

			// checkAll sends the id's to a chanel, and as this causes mutex
			// acquisition, it means we won't stress the generator as hard, but
			// we do guarantee we catch any uniqueness violation. the errcount
			// of the sessions is correspondingly much lower, and the
			// benchmarked cost is 4-6x slower. With this check disabled, the
			// check for uniqueness against the generator local last id is
			// actually more than adequate to catch uniqueness problems in
			// practice.
			const checkAll = false

			idCount.Load()
			id, err := s.NextID()
			if err != nil {
				if !errors.Is(err, ErrOverloaded) {
					fmt.Printf("%v\n", err)
				}
				errCount.Add(1)
				continue
			}
			// idid is just for telemetry, there is an irreconcilable race that prevents this from being perfect
			iid := idCount.Add(1)
			if checkAll {
				idc <- sample{id, iid}
			}

			// !!!This races, if another routine calls NextID after us but gets to the Swap *before* us, the id's will be out of order
			lastShared = sharedLast.Swap(id)

			// Catch to sequential id's that collide. Of course this doesn't catch
			// non sequential duplication
			if checkLocal && id == last {
				fmt.Printf("[%d:%d]  %s: id %016x\n", iidLast, iid, msgDuplicationViolation, id)
			}
			if checkShared && id == lastShared {
				fmt.Printf("[%d:%d]s %s: id %016x\n", iidLast, iid, msgMonotonicViolation, id)
			}
			if checkLocal && id < last {
				fmt.Printf(
					"[%d:%d]  %s last:id %016x:%016x %016x:%016x %02x:%02x %s\n", iidLast, iid, msgMonotonicViolation,
					last, id, last&TimeMask, id&TimeMask, last&s.seqMask, id&s.seqMask, IDTime(id, s.EpochStart()).String())
			}
			if checkShared && id < lastShared {
				fmt.Printf(
					"[%d:%d]s %s last:id %016x:%016x %016x:%016x %02x:%02x %s\n", iidLast, iid, msgMonotonicViolation,
					lastShared, id, lastShared&TimeMask, id&TimeMask, lastShared&s.seqMask, id&s.seqMask, IDTime(id, s.EpochStart()).String())
			}

			// Monotonicity of the time field should hold too, even when we force
			// increment the millisecond, it can only be larger than any previous
			// value seen
			if checkLocal && IDTime(last, s.EpochStart()).After(IDTime(id, s.EpochStart())) {
				fmt.Printf("[%d:%d]  %s: last:id %016x:%016x\n", iidLast, iid, msgMonotonicTimeViolation, last, id)
			}
			if checkShared && IDTime(lastShared, s.EpochStart()).After(IDTime(id, s.EpochStart())) {
				fmt.Printf("[%d:%d]s %s: last:id %016x:%016x\n", iidLast, iid, msgMonotonicTimeViolation, lastShared, id)
			}

			last = id
			iidLast = iid
		}
	})

	close(idc)
	ic, ec := idCount.Load(), errCount.Load()

	fmt.Printf("idcount: %d, errcount: %d\n", ic, ec)

}

func TestIDState_initState(t *testing.T) {
	type fields struct {
		timeMask       uint64
		workerIDMask   uint64
		maskedWorkerID uint64
		seqMask        uint64
		seqBits        int
	}
	type args struct {
		workerID uint16
		seqBits  int
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{"sequence bits maxed", fields{}, args{seqBits: 16}, false},
		{"sequence bits greater than worker bits", fields{}, args{seqBits: 25}, true},
		{"sequence bits greater than max sequence bits", fields{}, args{seqBits: 17}, true},
		{"sequence bits to small", fields{}, args{seqBits: 7}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &IDState{
				workerIDMask:   tt.fields.workerIDMask,
				maskedWorkerID: tt.fields.maskedWorkerID,
				seqMask:        tt.fields.seqMask,
				seqBits:        tt.fields.seqBits,
			}
			if err := s.initState(tt.args.workerID, tt.args.seqBits); (err != nil) != tt.wantErr {
				t.Errorf("IDState.initState() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
