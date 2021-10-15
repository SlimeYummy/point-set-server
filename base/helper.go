package base

import "time"

func Frame2Duration(frame uint32) time.Duration {
	ms := int64(frame) * 100 * int64(time.Millisecond)
	return time.Duration(ms)
}

func Duration2Frame(dura time.Duration) int64 {
	return int64(dura.Milliseconds()) / 100
}
