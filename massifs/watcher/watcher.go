package watcher

// Watch for log changes, relying on the blob last idtimestamps to do so
// efficiently.

import (
	"fmt"

	"time"

	"github.com/datatrails/go-datatrails-merklelog/massifs"
	"github.com/datatrails/go-datatrails-merklelog/massifs/snowflakeid"
	// "github.com/datatrails/go-datatrails-common/azblob"
)

const (
	currentEpoch    = uint8(1) // good until the end of the first unix epoch
	DefaultInterval = time.Second * 1
	// Azure promises tag index consistency within "seconds"
	DefaultHorizon = time.Minute
)

type WatchConfig struct {
	Since         time.Time
	IDSince       string
	Horizon       time.Duration
	Interval      time.Duration
	IntervalCount int
}

type Watcher struct {
	Cfg WatchConfig
	// these are just for reporting for now
	LastSince   time.Time
	LastIDSince string
}

func (w *Watcher) FirstFilter() string {
	w.LastSince = w.Cfg.Since
	w.LastIDSince = w.Cfg.IDSince
	return fmt.Sprintf(`"lastid">='%s'`, w.Cfg.IDSince)
}

func (w *Watcher) NextFilter() string {
	if w.Cfg.Horizon == 0 {
		return w.FirstFilter()
	}
	w.LastSince = time.Now().Add(-w.Cfg.Horizon)
	w.LastIDSince = massifs.IDTimeHex(w.LastSince)
	return fmt.Sprintf(`"lastid">='%s'`, w.LastIDSince)
}

func NewWatcher(cfg WatchConfig) (Watcher, error) {
	if err := ConfigDefaults(&cfg); err != nil {
		return Watcher{}, err
	}
	return Watcher{Cfg: cfg}, nil
}

func (w Watcher) ConfigString() string {
	return fmt.Sprintf("Since: %s, Interval: %v, IDSince: %s", w.Cfg.Since.UTC().Format(time.RFC3339), w.Cfg.Interval, w.Cfg.IDSince)
}

func ConfigDefaults(cfg *WatchConfig) error {
	if cfg.Since.Equal(time.Time{}) && cfg.IDSince == "" && cfg.Horizon == 0 {
		return fmt.Errorf("provide horizon on its own or either of the since parameters.")
	}
	// If horizon is provided, the since values are derived

	if cfg.Interval == 0 {
		cfg.Interval = DefaultInterval
	}

	var sinceUnset bool
	if cfg.Horizon == 0 {
		// temporarily force a horizon
		cfg.Horizon = time.Second * 30
	}

	// since defaults to now (but will get trumped by horizon if that was provided)
	if cfg.Since.Equal(time.Time{}) {
		cfg.Since = time.Now()
		sinceUnset = true
	}
	// explicit horizon trumps since, and the default horizon trumps the default since
	if cfg.Horizon > 0 && sinceUnset {
		cfg.Since = time.Now().Add(-cfg.Horizon)
	}
	if cfg.IDSince == "" {
		cfg.IDSince = massifs.IDTimeHex(cfg.Since)
	} else {
		id, epoch, err := massifs.SplitIDTimestampHex(cfg.IDSince)
		if err != nil {
			return err
		}
		// set since from the provided idsince so we can report in human
		cfg.Since = snowflakeid.IDTime(id, snowflakeid.EpochTimeUTC(epoch))
	}
	return nil
}
