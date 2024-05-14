package snowflakeid

import (
	"errors"
	"fmt"
	"sync/atomic"
	"time"
)

const (
	// MaxSpins configures the maximum number of CAS cycles the id generator is
	// permitted. If the id generator exceeds this *and* the current sequence
	// counter is exhausted, the generator will error The generator can be
	// configured to allow *at most* this many spins. It is expect that this
	// value is used to configure the generator.
	MaxSpins = 100

	Nanos = 1e6
)

type IDState struct {
	allowSpins int
	// timeShift  & mask are not configurable, use the constant TimeBits and TimeMask instead
	workerIDMask uint64
	// maskedWorkerId is the workerId shifted into its correct bit position in our 64 bit timestamp
	maskedWorkerID uint64

	seqMask uint64
	seqBits int

	epochStartWallClock      time.Time     // will *not* include the monotonic clock reading
	generatorStart           time.Time     // Will include the monotonic clock reading
	generatorStartWallOffset time.Duration // generatorStart - epochStart, does NOT include monotonic reading

	// monotonic is our state variable which includes the timestamp and the
	// sequence number but *not* the machine id.
	//
	// ***********************************************************************
	// We strictly guarantee that `monotonic` only increases for all consumers.
	// ***********************************************************************
	//
	// Note that there is a single edge case, possible only if we are exceeding
	// the configured maximum "events per millisecond", where we error rather
	// than overflow the sequence counter into the machine field. The impact of
	// that error is simply to slow us down.
	monotonic atomic.Uint64
}

var (
	ErrWorkerBitRange    = errors.New("the bit allocation for worker id and sequence bits overflows what is reserved for our timestamp")
	ErrOverloaded        = errors.New("the id generator is over loaded for its configuration")
	ErrClockError        = errors.New("the reading from system time doesn't make any realistic sense")
	ErrSequenceViolation = errors.New("the generator produced two consecutive values that violate either the monotonic or the uniqueness promises")

	// The nanosecond unix time overflows an int64 on 2262
	// https://pkg.go.dev/time#Time.UnixNano. This is used for an error clause
	// that is essentially about catching serious clock configuration issues.
	UnixNanoEpochEndSentinel = time.Date(2261, 1, 1, 1, 1, 1, 1, time.UTC) // this is a year before the limit defined here
)

func NewIDState(cfg Config) (*IDState, error) {
	workerID, seqBits, err := workerIDSequenceBits(cfg)
	if err != nil {
		return nil, err
	}

	s := &IDState{}
	err = s.initTime(cfg.CommitmentEpoch)
	if err != nil {
		return nil, err
	}

	err = s.initState(workerID, seqBits)
	if err != nil {
		return nil, err
	}
	return s, nil
}

func IDTime(id uint64, epochStart time.Time) time.Time {

	ms := id >> TimeShift

	// Note: do not be tempted to add .UTC() here, it strips the monotonic
	// component of the time. The caller is free to do that if they wish.
	return epochStart.Add(time.Duration(ms * uint64(time.Millisecond)))
}

func EpochMS(epoch uint8) int64 {
	return int64(epoch) * ((1 << TimeBits) - 1)
}

func EpochTimeUTC(epoch uint8) time.Time {
	startMS := EpochMS(epoch)
	return time.UnixMilli(startMS).UTC()
}

// millisecondMonotonicNow returns a monotonic epoch time sample. It is based of
// a reference wall clock time read when the process initialized the IDState
func (s *IDState) millisecondMonotonicNow() uint64 {

	now := time.Now()

	// Both now & generatorStart have a monotonic sample, so Sub gives a
	// duration result which preserves that. This means NextID would not see
	// negative time adjustments. On systems that sleep, it may however see
	// clock 'pauses'. The same mechanism that makes NextID robust in the face
	// of using wall clock time also guards against this.
	epochNow := now.Sub(s.generatorStart) + s.generatorStartWallOffset
	return uint64(epochNow / time.Millisecond)
}

// NextID returns the next value in a time ordered, unique and monotonic series.
// If that property can't be assured the function will error. On error the
// caller should either sleep for millisecond or so, or error out. As the
// error condition is likely to hit due to high load, a sleep with jitter is
// considered the best approach as this will avoid thundering herd issues.
func (s *IDState) NextID() (uint64, error) {

	// We do a read/modify/write on the monotonic state variable. The
	// sync/atomic primitives guarantee the memory model for each operation. In
	// principal this is exactly like the http/azure blob storage/classic db/
	// etag mechanism: we read some value, we do some work to determine an
	// updated value, and then we only get to complete our update if the
	// original value has not changed under our feet. If it has, our
	// intermediate calculations are now invalid and we have to start again. Our
	// local state is simply the stack, and our remote is the memory location of
	// the monotonic variable.
	//
	// The hot spin is essential for the throughput of the id generator. Our
	// sequence counter can be as large as 65k depending on configuration.  That
	// puts us into sub microsecond time slices. The benchmark shows this method
	// is in the order of 10's of nanoseconds amortized. We cannot achieve that
	// throughput with locks and sleeps. This spin will only occur at times of
	// high contention so the CPU will already be loaded. Rather than spin
	// indefinitely though, we put a fixed bounds on it and error out if that is
	// exceeded. At that point, we are attempting to operate way beyond what the
	// configuration supports so erroring out will naturally throttle things
	// down.
	//
	// *************************************************************************
	// The chief concern in the logic here is to ensure the uniqueness property
	// for all consumers. The monotonic property is assured as a consequence of
	// how we do that.
	// *************************************************************************
	//
	// Of the many implementations out there, this one owes most to this source:
	//  https://github.com/influxdata/chronograf/blob/673fc1660087c0c528eb0bd5fd21ad7d589d22d7/snowflake/gen.go#L57
	//
	// Which is under GPL hence we couldn't use it.
	//
	// A second atomic based approach, with an MIT license, was also considered,
	// but it's double CAS approach seems much less 'good'
	//
	// https://github.com/godruoyi/go-snowflake/blob/master/atomic_resolver.go

	var next uint64

	// note: allowSpins == 0 is supported and simply means try once
	for i := 0; i <= s.allowSpins; i++ {

		// The following line would use wall clock time always and this would
		// produce timestamps with a smoother relationship to wall clock (ntp
		// adjusted) time. However our rule for uniqueness based on
		// monotonicity, effectively prevents reverse adjustments from
		// surfacing. So we only lose the effect of forward adjustments during
		// the process life.

		// now := uint64(time.Now().Sub(s.epochStartWallClock) / time.Millisecond)

		// Instead, we use monotonic time, which is aligned with wall clock time on process start.
		now := s.millisecondMonotonicNow()
		last := s.monotonic.Load()

		lastTime := last >> TimeShift
		lastSeq := last & s.seqMask

		switch {

		case now > lastTime:
			// Time has advanced past the millisecond of the last generated id.
			// So we use it as is, and reset the sequence. The 'reset' is
			// achieved by just ignoring the sequence bits and overwriting them
			// with the zeros we get from shifting the new time into place
			next = now << TimeShift

		// From here now is equal *or behind* the time of the last generated id.
		// Behind is possible due to clock drift, ntp adjustments (if not
		// relying on the process monotonic time).  We simply treat *behind* as
		// equivalent to the equal and bump the sequence.  If we exhaust the
		// sequence we forcibly increment the millisecond.
		//  This will cause subsequent consumers to do exactly the same, see the
		// clock time as *behind*, until the clock settles, and so they will
		// also increment the sequence or the millisecond.  Remember, we are
		// not attesting to 'millisecond precision time', we are generating a
		// time ordered unique series with a *fair* relationship to our time
		// source at millisecond 'granularity'. The key requirement is
		// **uniqueness**. monotonicity is desirable, but not critical.
		// The possibility of overlap due to process restart can be mitigated
		// with a high water tombstone as described at
		// [uniqueness tombstone](https://github.com/datatrails/epic-8120-scalable-proof-mechanisms/blob/main/forestrie-snowflakeid.md)
		// However, as the id's are assembled into trie keys which have other
		// unique properties, the tomb stone mechanism is probably overkill.

		case lastSeq == s.seqMask:
			// The sequence is exhausted, force the next millisecond and reset the sequence.

			// ** CRUCIAL ** we use the **lastTime** because it is >= now in
			// this case.

			next = (lastTime + 1) << TimeShift
		default:
			// In this case the sequence is not exhausted (and now is <=
			// lastTime). As the sequence is in the lowest order bits, simple
			// addition is all we need.
			next = last + 1
		}

		if next <= last {
			// Note on this error the caller should error out.
			return 0, fmt.Errorf("%016x:%016x %02x:%02x %d:%d:%w", last, next, lastSeq, s.seqMask, lastTime, now, ErrSequenceViolation)
		}

		if s.monotonic.CompareAndSwap(last, next) {
			// We got through the above logic without being beaten to the draw
			// by another thread so our resulting value was updated consistently
			// and has been safely written back. We are done, we have our next
			// id
			break
		}

		next = 0 // start again
	}

	if next == 0 {
		// To reach here we must be at high contention, we have failed the CAS
		// swap the allowed number of times. The caller can decide whether to
		// sleep or to error out.

		// Notice that there is nothing *perfectly safe* we can do here. Unless
		// we are successful on the CAS (above) we cannot guarantee that all
		// consumers see consistent results if we re-examine intermediate state
		// like the sequence.

		// One trick is to just do s.monotonic.Add(1), but if that overflows the
		// sequence there is a vanishingly small chance, we can duplicate an id
		// due to polluting the machine id with the overflow from the sequence.
		// As our goal is strict uniqueness, we choose to error out here. If we
		// ever get the load that trips this, the answer is horizontal scaling
		// in the first instance, and sequence counter size increase secondarily.
		// next = s.monotonic.Add(1)
		return 0, ErrOverloaded
	}
	return next | s.maskedWorkerID, nil
}

func (s *IDState) EpochStart() time.Time {
	return s.epochStartWallClock
}

func (s *IDState) initTime(epoch uint8) error {

	// https://github.com/datatrails/epic-8120-scalable-proof-mechanisms/blob/6a0385bbfd0a8cbadfd18a1e51955333cbb75271/forestrie-snowflakeid.md#datatrails-commitment-epoch

	// [datatrails commitment epoch](https://github.com/datatrails/epic-8120-scalable-proof-mechanisms/blob/main/forestrie-snowflakeid.md#datatrails-commitment-epoch)
	// The NextID method is safe regardless of whether we use wall clock time
	// (which is subject to synchronization and _does_ go backwards), or the
	// process monotonic time. We desire a roughly time ordered, unique  series.
	// We get a cleaner signal by creating the time stamp with reference to the
	// start time of the generator, so that we get a monotonic time sample.
	// Processes restart often enough that we don't need to be concerned with
	// drift against wall clock time.

	s.generatorStart = time.Now() // DONT do UTC() here, as that strips the monotonic time sample

	// The practical value of this guard is defending against clock
	// configuration issues (which may manifest during VM maintenance cycles for
	// example)
	if s.generatorStart.After(UnixNanoEpochEndSentinel) {
		return fmt.Errorf("the clock reading is close to overflowing the limit of an int64: %w", ErrClockError)
	}

	startMS := EpochMS(epoch)
	s.epochStartWallClock = time.UnixMilli(startMS).UTC()
	s.generatorStartWallOffset = s.generatorStart.Sub(s.epochStartWallClock)

	return nil
}

func (s *IDState) initState(workerID uint16, seqBits int) error {
	if seqBits > MaxWorkerBits || MaxWorkerBits-seqBits < MinWorkerBits {
		return fmt.Errorf(
			"sequence bit count %d is to large (check your CIDR config): %w",
			seqBits, ErrWorkerBitRange)
	}
	if seqBits < MinWorkerBits {
		return fmt.Errorf(
			"sequence bit count %d is to small (check your CIDR config): %w",
			seqBits, ErrWorkerBitRange)
	}

	s.workerIDMask = ((1 << (MaxWorkerBits - seqBits)) - 1) << seqBits
	s.maskedWorkerID = uint64(workerID) << seqBits
	s.seqMask = (1 << seqBits) - 1
	s.seqBits = seqBits
	s.monotonic.Store(0)
	return nil
}
