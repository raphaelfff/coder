package provisionerdserver

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type fakeNow struct {
	n []time.Time
}

func (f *fakeNow) now() time.Time {
	n := f.n[0]
	f.n = f.n[1:]
	return n
}

func TestDebouncer(t *testing.T) {
	t.Parallel()
	uut := NewDebouncer(time.Second * 5)
	fn := &fakeNow{}
	uut.now = fn.now

	// before we Signal, any time will not debounce, and we don't even call now()
	require.False(t, uut.Debounce())
	require.False(t, uut.Debounce())

	fn.n = append(fn.n, time.Unix(1693390000, 0))
	fn.n = append(fn.n, time.Unix(1693390001, 0))
	fn.n = append(fn.n, time.Unix(1693390002, 0))
	fn.n = append(fn.n, time.Unix(1693390005, 1))
	fn.n = append(fn.n, time.Unix(1693390005, 2))
	uut.Signal()
	require.True(t, uut.Debounce())
	require.True(t, uut.Debounce())
	require.False(t, uut.Debounce())
	require.False(t, uut.Debounce())
}
