package snowflakeid

type Config struct {
	// CommitmentEpoch determines our reference zero time with respect to unix
	// time It is expected that the service code defines this as a constant.
	// Note that this is a uint8 so that CommitmentEpoch * 2^TimeBits can not
	// overflow an int64. Each epoch is ~34 years and the correct current
	// configuration is 1, service code should make that a constant.
	CommitmentEpoch uint8

	// WorkerCIDR ensures two pods can't generate the same snowflake id by
	// selecting bits from the pods private ip address.
	WorkerCIDR string

	// PodIP is the workload private ip address obtained via the Kubernetes
	PodIP string

	// AllowSpins should typically be set to the constant MaxSpins. If you are
	// un-familiar with how the generator works, just set this to MaxSpins.
	// Setting it to zero is supported, and effectively will cause the generator
	// to error when there is high contention. We do not support an infinite
	// number of spins, and for that reason we use a narrow type
	AllowSpins uint8
}

const (
	// TimeBits is the number of bits in the timestamp reserved for time. Our
	// timestamp has millisecond precision and this setting gives us an epoch
	// of 34 years. As our reference time is the unix time, we are in epoch 1
	// and the next epoch is ~2038 (which is when people with 32 bit time values
	// will be dealing with the unix epoc anyway)
	//
	// Notice that this setting is not configurable, though it is plausible we
	// could change it in the future. There is nothing 'impossible' about that
	// but it would be a very significant ask.
	TimeBits  = 40
	TimeShift = 64 - 40

	TimeMask uint64 = ((1 << TimeBits) - 1) << TimeShift
)
