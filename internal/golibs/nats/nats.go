package nats

import (
	"context"
	"time"

	"github.com/nats-io/nats.go"
	"go.uber.org/zap"
)

type NATSConfig struct {
	URL        string `json:"url"`
	StreamName string `json:"stream_name"`
}

func DefaultConfig() NATSConfig {
	return NATSConfig{
		URL:        nats.DefaultURL,
		StreamName: "PAYNEXUS_EVENTS",
	}
}

type NATSClient struct {
	Conn   *nats.Conn
	JS     nats.JetStreamContext
	Cfg    NATSConfig
	Logger *zap.Logger
}

func NewNATSClient(cfg NATSConfig, logger *zap.Logger) (*NATSClient, error) {
	conn, err := nats.Connect(cfg.URL, nats.Timeout(5*time.Second))
	if err != nil {
		logger.Error("Failed to connect to NATS", zap.Error(err))
		return nil, err
	}
	js, err := conn.JetStream()
	if err != nil {
		logger.Error("Failed to get JetStream context", zap.Error(err))
		conn.Close()
		return nil, err
	}
	client := &NATSClient{
		Conn:   conn,
		JS:     js,
		Cfg:    cfg,
		Logger: logger,
	}

	if err := client.initStream(); err != nil {
		logger.Error("Failed to init stream", zap.Error(err))
		conn.Close()
		return nil, err
	}

	logger.Info("Connected to NATS", zap.String("url", cfg.URL))

	return client, nil
}

func (n *NATSClient) initStream() error {
	stream, err := n.JS.StreamInfo(n.Cfg.StreamName)
	if err != nil && err != nats.ErrStreamNotFound {
		return err
	}

	if stream == nil {
		_, err = n.JS.AddStream(&nats.StreamConfig{
			Name:      n.Cfg.StreamName,
			Subjects:  []string{"paynexus.>"}, // All event has prefix start with paynexus
			Storage:   nats.FileStorage,
			Retention: nats.LimitsPolicy,
		})
		return err
	}
	return nil
}

func (n *NATSClient) PublishEvent(ctx context.Context, subject string, payload []byte) error {
	_, err := n.JS.Publish(subject, payload, nats.Context(ctx))
	return err
}

func (n *NATSClient) QueueSubscribe(ctx context.Context, subject string, queueGroup string, handler func(msg *nats.Msg)) (*nats.Subscription, error) {
	n.Logger.Info("QueueSubscribe", zap.String("subject", subject), zap.String("queueGroup", queueGroup))
	return n.JS.QueueSubscribe(subject, queueGroup, handler, nats.ManualAck())
}

func (n *NATSClient) Close() {
	if n.Conn != nil {
		n.Conn.Close()
		n.Logger.Info("Closed NATS connection")
	}
}
