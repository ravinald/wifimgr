// Package durfmt renders durations for humans. Two questions get asked of a
// duration and they want different answers: how long an operation took
// (Elapsed) and how long ago something happened (Age). Elapsed needs
// sub-second resolution and rarely exceeds minutes; Age needs days and would
// only be made noisy by milliseconds. One formatter serving both produces
// "0s" for a fast refresh or "269559ms" for a stale cache.
//
// Machine-read fields — JSON stats, cache metadata, debug logs — keep their raw
// units and do not belong here.
package durfmt

import (
	"fmt"
	"time"
)

// Elapsed renders how long an operation took: exact milliseconds below a
// second, tenths below a minute, whole seconds above. The bands keep the field
// short and its width stable, which matters in a repainted progress board where
// a jittering column reads as a redraw bug.
func Elapsed(d time.Duration) string {
	if d < 0 {
		d = 0
	}
	switch {
	case d < time.Second:
		return fmt.Sprintf("%dms", d.Milliseconds())
	case d < time.Minute:
		return d.Round(100 * time.Millisecond).String()
	default:
		return d.Round(time.Second).String()
	}
}

// Age renders how long ago something happened, to two units of d/h/m/s. Two
// units is the point: "3d4h" answers the staleness question a cache footer or a
// last-seen column is asking, where "3d4h17m52s" only buries it.
func Age(d time.Duration) string {
	if d < 0 {
		d = 0
	}
	switch {
	case d < time.Minute:
		return fmt.Sprintf("%ds", int(d.Seconds()))
	case d < time.Hour:
		return twoUnit(int(d/time.Minute), int(d/time.Second)%60, "m", "s")
	case d < 24*time.Hour:
		return twoUnit(int(d/time.Hour), int(d/time.Minute)%60, "h", "m")
	default:
		return twoUnit(int(d/(24*time.Hour)), int(d/time.Hour)%24, "d", "h")
	}
}

// AgeSince is Age of time.Since(t), with the guards a timestamp needs. A zero
// time means never recorded and a future one means vendor-side clock skew;
// neither is an age, and both render as an em dash rather than "0s", which
// would read as "just now".
func AgeSince(t time.Time) string {
	if t.IsZero() {
		return "—"
	}
	d := time.Since(t)
	if d < 0 {
		return "—"
	}
	return Age(d)
}

func twoUnit(major, minor int, majorUnit, minorUnit string) string {
	if minor == 0 {
		return fmt.Sprintf("%d%s", major, majorUnit)
	}
	return fmt.Sprintf("%d%s%d%s", major, majorUnit, minor, minorUnit)
}
