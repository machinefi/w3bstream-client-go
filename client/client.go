package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/pkg/errors"
)

type (
	// Client is a client for W3bstream.
	Client struct {
		queue  chan *event
		buf    []*event
		url    string
		apiKey string
		cfg    *options
	}

	options struct {
		queueSize      uint32
		batchSize      uint32
		tickerInterval time.Duration
	}

	// Header is a header of an event.
	Header struct {
		DeviceID  string
		EventType string
		Timestamp time.Time
	}

	event struct {
		DeviceID  string `json:"device_id"`
		EventType string `json:"event_type,omitempty"`
		Payload   string `json:"payload"`
		Timestamp int64  `json:"timestamp,omitempty"`
	}
)

const (
	dataPushEventType = "DA-TA_PU-SH"
)

var (
	defaultOpts = &options{
		queueSize:      1000,
		batchSize:      100,
		tickerInterval: 200 * time.Millisecond,
	}
)

// Option is an option for NewClient.
type Option func(*options)

// WithQueueSize sets the queue size.
func WithQueueSize(val uint32) Option {
	return func(opts *options) {
		opts.queueSize = val
	}
}

// WithBatchSize sets the batch size.
func WithBatchSize(val uint32) Option {
	return func(opts *options) {
		opts.batchSize = val
	}
}

// WithInterval sets the ticker interval.
func WithInterval(val time.Duration) Option {
	return func(opts *options) {
		opts.tickerInterval = val
	}
}

// NewClient creates a new client for W3bstream.
func NewClient(url string, apiKey string, opts ...Option) *Client {
	op := defaultOpts
	for _, opt := range opts {
		opt(op)
	}
	cc := &Client{
		queue:  make(chan *event, op.queueSize),
		buf:    make([]*event, 0, op.batchSize),
		url:    url,
		apiKey: apiKey,
		cfg:    op,
	}
	go cc.worker()
	return cc
}

// Close closes the client.
func (c *Client) Close() {
	close(c.queue)
}

// PublishEventSync publishes an event synchronously.
func (c *Client) PublishEventSync(header *Header, payload []byte) (*http.Response, error) {
	if len(header.DeviceID) == 0 {
		return nil, errors.New("device_id is required")
	}
	return c.send([]*event{newEvent(header, payload)})
}

func newEvent(header *Header, payload []byte) *event {
	var t time.Time = header.Timestamp
	if t.IsZero() {
		t = time.Now()
	}
	var eventType string = "DEFAULT"
	if len(header.EventType) > 0 {
		eventType = header.EventType
	}
	return &event{
		DeviceID:  header.DeviceID,
		EventType: eventType,
		Payload:   string(payload),
		Timestamp: t.Unix(),
	}
}

// PublishEvent publishes an event asynchronously.
func (c *Client) PublishEvent(header *Header, payload []byte) error {
	if len(header.DeviceID) == 0 {
		return errors.New("device_id is required")
	}
	select {
	case c.queue <- newEvent(header, payload):
		return nil
	default:
		return errors.New("failed to send event: the queue is full")
	}
}

func (c *Client) worker() {
	ticker := time.NewTicker(c.cfg.tickerInterval)
	asyncSend := func(reqs []*event) {
		resp, err := c.send(reqs)
		if err != nil {
			log.Println("an error occurred when publishing the data to W3bstream: ", err)
			return
		}
		defer resp.Body.Close()
		if resp.StatusCode >= 400 {
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				log.Println("an error occurred when publishing the data to W3bstream: ", err)
			}
			log.Printf("an error occurred when publishing the data to W3bstream: status_code %d, body: %s", resp.StatusCode, body)
		}
	}

	for {
		select {
		case <-ticker.C:
			if len(c.buf) == 0 {
				continue
			}
			oldBuf := c.buf
			go asyncSend(oldBuf)
			c.buf = make([]*event, 0, c.cfg.batchSize)
		case ent, ok := <-c.queue:
			if !ok { // channel is closed
				return
			}
			c.buf = append(c.buf, ent)
			if len(c.buf) >= int(c.cfg.batchSize) {
				oldBuf := c.buf
				go asyncSend(oldBuf)
				c.buf = make([]*event, 0, c.cfg.batchSize)
				ticker.Reset(c.cfg.tickerInterval)
			}
		}
	}
}

func (c *Client) send(reqs []*event) (*http.Response, error) {
	encodedPayload, err := json.Marshal(reqs)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal request")
	}
	url := fmt.Sprintf(`%s?eventType=%s&timestamp=%d`, c.url, dataPushEventType, time.Now().Unix())

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(encodedPayload))
	if err != nil {
		return nil, errors.Wrap(err, "failed to create request")
	}
	req.Header.Set("Content-Type", "application/octet-stream")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	return http.DefaultClient.Do(req)
}
