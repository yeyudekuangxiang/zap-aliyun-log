package logger

import (
	sls "github.com/aliyun/aliyun-log-go-sdk"
	"github.com/aliyun/aliyun-log-go-sdk/producer"
	"github.com/gogo/protobuf/proto"
	"go.uber.org/zap/zapcore"
	"time"
)

type ProducerConfig struct {
	ProjectName string
	LogStore    string
	Topic       string
	Source      string
	zapcore.LevelEnabler
}

func NewAliYunCore(enc *AliYunEncoder, producer *producer.Producer, config ProducerConfig) *AliYunCore {
	return &AliYunCore{
		enc:            enc,
		producer:       producer,
		ProducerConfig: config,
	}
}

type AliYunCore struct {
	ProducerConfig
	enc      *AliYunEncoder
	producer *producer.Producer
	client   sls.ClientInterface
}

func (a *AliYunCore) With(fields []zapcore.Field) zapcore.Core {
	clone := a.clone()
	addFields(clone.enc, fields)
	return clone
}
func addFields(enc zapcore.ObjectEncoder, fields []zapcore.Field) {
	for i := range fields {
		fields[i].AddTo(enc)
	}
}
func (a *AliYunCore) Check(ent zapcore.Entry, ce *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	if a.Enabled(ent.Level) {
		return ce.AddCore(ent, a)
	}
	return ce
}

func (a *AliYunCore) Write(entry zapcore.Entry, fields []zapcore.Field) error {
	m, err := a.enc.EncodeEntry(entry, fields)
	if err != nil {
		return err
	}
	var content []*sls.LogContent

	for key, value := range m {
		content = append(content, &sls.LogContent{
			Key:   proto.String(key),
			Value: proto.String(value),
		})
	}

	log := &sls.Log{
		Time:     proto.Uint32(uint32(time.Now().Unix())),
		Contents: content,
	}

	return a.producer.SendLog(a.ProjectName, a.LogStore, a.Topic, a.Source, log)

}

func (a *AliYunCore) Sync() error {
	return a.producer.Close(30)
}
func (a *AliYunCore) clone() *AliYunCore {
	return &AliYunCore{
		ProducerConfig: a.ProducerConfig,
		client:         a.client,
		enc:            a.enc.clone(),
	}
}
