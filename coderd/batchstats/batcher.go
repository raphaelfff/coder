package batchstats

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"golang.org/x/xerrors"

	"cdr.dev/slog"
	"cdr.dev/slog/sloggers/sloghuman"

	"github.com/coder/coder/coderd/database"
	"github.com/coder/coder/coderd/database/dbauthz"
	"github.com/coder/coder/codersdk/agentsdk"
)

const (
	defaultBufferSize    = 1024
	defaultFlushInterval = time.Second
)

// Batcher holds a buffer of agent stats and periodically flushes them to
// its configured store. It also updates the workspace's last used time.
type Batcher struct {
	store database.Store
	log   slog.Logger

	buf       chan database.InsertWorkspaceAgentStatParams
	batchSize int

	// tickCh is used to periodically flush the buffer.
	tickCh   <-chan time.Time
	ticker   *time.Ticker
	interval time.Duration
	// flushLever is used to signal the flusher to flush the buffer immediately.
	flushLever chan struct{}
	// flushed is used during testing to signal that a flush has completed.
	flushed  chan<- int
	flushing atomic.Bool
}

// Option is a functional option for configuring a Batcher.
type Option func(b *Batcher)

// WithStore sets the store to use for storing stats.
func WithStore(store database.Store) Option {
	return func(b *Batcher) {
		b.store = store
	}
}

// WithBatchSize sets the number of stats to store in a batch.
func WithBatchSize(size int) Option {
	return func(b *Batcher) {
		b.batchSize = size
	}
}

// WithInterval sets the interval for flushes.
func WithInterval(d time.Duration) Option {
	return func(b *Batcher) {
		b.interval = d
	}
}

// WithLogger sets the logger to use for logging.
func WithLogger(log slog.Logger) Option {
	return func(b *Batcher) {
		b.log = log
	}
}

// New creates a new Batcher and starts it.
func New(ctx context.Context, opts ...Option) (*Batcher, func(), error) {
	b := &Batcher{}
	b.log = slog.Make(sloghuman.Sink(os.Stderr))
	b.flushLever = make(chan struct{}, 1) // Buffered so that it doesn't block.
	for _, opt := range opts {
		opt(b)
	}

	if b.store == nil {
		return nil, nil, xerrors.Errorf("no store configured for batcher")
	}

	if b.interval == 0 {
		b.interval = defaultFlushInterval
	}

	if b.batchSize == 0 {
		b.batchSize = defaultBufferSize
	}

	if b.tickCh == nil {
		b.ticker = time.NewTicker(b.interval)
		b.tickCh = b.ticker.C
	}

	cancelCtx, cancelFunc := context.WithCancel(ctx)
	done := make(chan struct{})
	go func() {
		b.run(cancelCtx)
		close(done)
	}()

	closer := func() {
		cancelFunc()
		if b.ticker != nil {
			b.ticker.Stop()
		}
		<-done
	}

	b.buf = make(chan database.InsertWorkspaceAgentStatParams, b.batchSize)

	return b, closer, nil
}

// Add adds a stat to the batcher for the given workspace and agent.
func (b *Batcher) Add(
	agentID uuid.UUID,
	templateID uuid.UUID,
	userID uuid.UUID,
	workspaceID uuid.UUID,
	st agentsdk.Stats,
) error {
	var p database.InsertWorkspaceAgentStatParams
	p.ID = uuid.New()
	p.CreatedAt = database.Now()
	p.AgentID = agentID
	p.UserID = userID
	p.TemplateID = templateID
	p.WorkspaceID = workspaceID
	// Convert the map[string]int to a json.RawMessage for Postgres.
	cbp, err := json.Marshal(st.ConnectionsByProto)
	if err != nil {
		b.log.Warn(context.Background(), "failed to marshal connections by proto")
		cbp = json.RawMessage(`{}`)
	}
	p.ConnectionsByProto = cbp
	p.ConnectionCount = st.ConnectionCount
	p.RxPackets = st.RxPackets
	p.RxBytes = st.RxBytes
	p.TxPackets = st.TxPackets
	p.TxBytes = st.TxBytes
	p.SessionCountVSCode = st.SessionCountVSCode
	p.SessionCountJetBrains = st.SessionCountJetBrains
	p.SessionCountReconnectingPTY = st.SessionCountReconnectingPTY
	p.SessionCountSSH = st.SessionCountSSH
	p.ConnectionMedianLatencyMS = st.ConnectionMedianLatencyMS

	b.buf <- p

	// If the buffer is over 80% full, signal the flusher to flush immediately.
	// We want to trigger flushes early to reduce the likelihood of
	// accidentally growing the buffer over batchSize.
	filled := float64(len(b.buf)) / float64(b.batchSize)
	if filled >= 0.8 && !b.flushing.Load() {
		b.log.Debug(context.Background(), "triggering flush", slog.F("id", p.ID))
		b.flushLever <- struct{}{}
		b.flushing.Store(true)
	}
	return nil
}

// Run runs the batcher.
func (b *Batcher) run(ctx context.Context) {
	// nolint:gocritic // This is only ever used for one thing - inserting agent stats.
	authCtx := dbauthz.AsSystemRestricted(ctx)
	for {
		select {
		case <-b.tickCh:
			b.flush(authCtx, false, "scheduled")
		case <-b.flushLever:
			// If the flush lever is depressed, flush the buffer immediately.
			b.flush(authCtx, true, "reaching capacity")
		case <-ctx.Done():
			b.log.Warn(ctx, "context done, flushing before exit")
			b.flush(authCtx, true, "exit")
			return
		}
	}
}

// flush flushes the batcher's buffer.
func (b *Batcher) flush(ctx context.Context, forced bool, reason string) {
	count := len(b.buf)
	start := time.Now()
	defer func() {
		// Notify that a flush has completed. This only happens in tests.
		if b.flushed != nil {
			select {
			case <-ctx.Done():
				close(b.flushed)
			default:
				b.flushed <- count
			}
		}
		if count > 0 {
			elapsed := time.Since(start)
			b.log.Debug(ctx, "flush complete",
				slog.F("count", count),
				slog.F("elapsed", elapsed),
				slog.F("forced", forced),
				slog.F("reason", reason),
			)
		}
		b.flushing.Store(false)
	}()

	if count == 0 {
		return
	}

	var batch database.InsertWorkspaceAgentStatsParams
	var rawMessages [][]byte
	// only read a limited number of messages from buf
	for i := 0; i < count; i++ {
		p, ok := <-b.buf
		if !ok {
			break // closed?
		}
		batch.ID = append(batch.ID, p.ID)
		batch.AgentID = append(batch.AgentID, p.AgentID)
		batch.UserID = append(batch.UserID, p.UserID)
		batch.TemplateID = append(batch.TemplateID, p.TemplateID)
		batch.WorkspaceID = append(batch.WorkspaceID, p.WorkspaceID)
		batch.CreatedAt = append(batch.CreatedAt, p.CreatedAt)
		batch.ConnectionCount = append(batch.ConnectionCount, p.ConnectionCount)
		batch.ConnectionMedianLatencyMS = append(batch.ConnectionMedianLatencyMS, p.ConnectionMedianLatencyMS)
		batch.RxBytes = append(batch.RxBytes, p.RxBytes)
		batch.RxPackets = append(batch.RxPackets, p.RxPackets)
		batch.TxBytes = append(batch.TxBytes, p.TxBytes)
		batch.TxPackets = append(batch.TxPackets, p.TxPackets)
		batch.SessionCountJetBrains = append(batch.SessionCountJetBrains, p.SessionCountJetBrains)
		batch.SessionCountReconnectingPTY = append(batch.SessionCountReconnectingPTY, p.SessionCountReconnectingPTY)
		batch.SessionCountSSH = append(batch.SessionCountSSH, p.SessionCountSSH)
		batch.SessionCountVSCode = append(batch.SessionCountVSCode, p.SessionCountVSCode)
		rawMessages = append(rawMessages, p.ConnectionsByProto)
	}

	// Join the raw message into a single array
	var buf bytes.Buffer
	_, _ = buf.WriteRune('[')
	_, _ = buf.Write(bytes.Join(rawMessages, []byte{','}))
	_, _ = buf.WriteRune(']')
	batch.ConnectionsByProto = buf.Bytes()

	// marshal connections by proto
	err := b.store.InsertWorkspaceAgentStats(ctx, batch)
	elapsed := time.Since(start)
	if err != nil {
		b.log.Error(ctx, "error inserting workspace agent stats", slog.Error(err), slog.F("elapsed", elapsed))
		return
	}
}
