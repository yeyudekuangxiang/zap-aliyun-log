package logger

import (
	"encoding/base64"
	"encoding/json"
	"go.uber.org/zap/buffer"
	"go.uber.org/zap/zapcore"
	"math"
	"time"
)

var (
	bufferpool = buffer.NewPool()
)

func NewAliYunEncoder(config *EncoderConfig) *AliYunEncoder {
	return &AliYunEncoder{
		m:             make(map[string]string),
		EncoderConfig: config,
	}
}

type EncoderConfig struct {
	// Set the keys used for each log entry. If any key is empty, that portion
	// of the entry is omitted.
	MessageKey     string                                  `json:"messageKey" yaml:"messageKey"`
	LevelKey       string                                  `json:"levelKey" yaml:"levelKey"`
	TimeKey        string                                  `json:"timeKey" yaml:"timeKey"`
	NameKey        string                                  `json:"nameKey" yaml:"nameKey"`
	CallerKey      string                                  `json:"callerKey" yaml:"callerKey"`
	FunctionKey    string                                  `json:"functionKey" yaml:"functionKey"`
	StacktraceKey  string                                  `json:"stacktraceKey" yaml:"stacktraceKey"`
	EncodeLevel    func(level zapcore.Level) string        `json:"levelEncoder" yaml:"levelEncoder"`
	EncodeTime     func(t time.Time) string                `json:"timeEncoder" yaml:"timeEncoder"`
	EncodeDuration func(duration time.Duration) string     `json:"durationEncoder" yaml:"durationEncoder"`
	EncodeCaller   func(caller zapcore.EntryCaller) string `json:"callerEncoder" yaml:"callerEncoder"`
	EncodeName     func(name string) string                `json:"nameEncoder" yaml:"nameEncoder"`
}
type AliYunEncoder struct {
	*EncoderConfig
	m          map[string]string
	reflectEnc *json.Encoder
	reflectBuf *buffer.Buffer
}

var nullLiteralBytes = []byte("null")

func (a *AliYunEncoder) newJSONEncoder() zapcore.Encoder {
	ec := zapcore.EncoderConfig{
		TimeKey:       "",
		LevelKey:      "",
		NameKey:       "",
		CallerKey:     "",
		MessageKey:    "",
		StacktraceKey: "",
	}
	if a.EncodeTime != nil {
		ec.EncodeTime = func(t time.Time, encoder zapcore.PrimitiveArrayEncoder) {
			encoder.AppendString(a.EncodeTime(t))
		}
	}
	if a.EncodeDuration != nil {
		ec.EncodeDuration = func(duration time.Duration, encoder zapcore.PrimitiveArrayEncoder) {
			encoder.AppendString(a.EncodeDuration(duration))
		}
	}
	return zapcore.NewJSONEncoder(ec)
}
func (a *AliYunEncoder) MarshalLogArray(marshaler zapcore.ArrayMarshaler) (string, error) {
	en := a.newJSONEncoder()
	err := en.AddArray("", marshaler)
	if err != nil {
		return "", err
	}
	b, err := en.EncodeEntry(zapcore.Entry{}, nil)
	if err != nil {
		return "", err
	}
	if b.Len() > 5 {
		return string(b.Bytes()[4 : b.Len()-1]), nil
	}
	return b.String(), nil
}
func (a *AliYunEncoder) MarshalLogObject(marshaler zapcore.ObjectMarshaler) (string, error) {
	en := a.newJSONEncoder()
	err := en.AddObject("", marshaler)
	if err != nil {
		return "", err
	}
	b, err := en.EncodeEntry(zapcore.Entry{}, nil)
	if err != nil {
		return "", err
	}
	if b.Len() > 5 {
		return string(b.Bytes()[4 : b.Len()-1]), nil
	}
	return b.String(), nil
}
func (a *AliYunEncoder) AddArray(key string, marshaler zapcore.ArrayMarshaler) error {
	s, err := a.MarshalLogArray(marshaler)
	if err != nil {
		return err
	}
	a.AddString(key, s)
	return nil
}

func (a *AliYunEncoder) AddObject(key string, marshaler zapcore.ObjectMarshaler) error {
	s, err := a.MarshalLogObject(marshaler)
	if err != nil {
		return err
	}
	a.AddString(key, s)
	return nil
}

func (a *AliYunEncoder) AddBinary(key string, value []byte) {
	a.AddString(key, base64.StdEncoding.EncodeToString(value))
}

func (a *AliYunEncoder) AddByteString(key string, value []byte) {
	a.AddString(key, string(value))
}

func (a *AliYunEncoder) AddBool(key string, value bool) {
	if value {
		a.AddString(key, "true")
		return
	}
	a.AddString(key, "false")
}

func (a *AliYunEncoder) AddComplex128(key string, value complex128) {
	buf := bufferpool.Get()
	r, i := real(value), imag(value)
	// Because we're always in a quoted string, we can use strconv without
	// special-casing NaN and +/-Inf.
	buf.AppendFloat(r, 64)
	// If imaginary part is less than 0, minus (-) sign is added by default
	// by AppendFloat.
	if i >= 0 {
		buf.AppendByte('+')
	}
	buf.AppendFloat(i, 64)
	buf.AppendByte('i')
	a.AddString(key, buf.String())
	buf.Free()
}

func (a *AliYunEncoder) AddComplex64(key string, value complex64) {
	a.AddComplex128(key, complex128(value))
}

func (a *AliYunEncoder) AddDuration(key string, value time.Duration) {
	a.AddInt64(key, int64(value))
}

func (a *AliYunEncoder) AddFloat64(key string, value float64) {
	a.AddString(key, a.formatFloat(value, 64))
}
func (a *AliYunEncoder) formatFloat(val float64, bitSize int) string {
	buf := bufferpool.Get()
	defer buf.Free()
	switch {
	case math.IsNaN(val):
		buf.AppendString(`"NaN"`)
	case math.IsInf(val, 1):
		buf.AppendString(`"+Inf"`)
	case math.IsInf(val, -1):
		buf.AppendString(`"-Inf"`)
	default:
		buf.AppendFloat(val, bitSize)
	}
	return buf.String()
}
func (a *AliYunEncoder) AddFloat32(key string, value float32) {
	a.AddString(key, a.formatFloat(float64(value), 32))
}

func (a *AliYunEncoder) AddInt(key string, value int) {
	panic("implement me")
}

func (a *AliYunEncoder) AddInt64(key string, value int64) {
	buf := bufferpool.Get()
	defer buf.Free()
	buf.AppendInt(value)
	a.AddString(key, buf.String())
}

func (a *AliYunEncoder) AddInt32(key string, value int32) {
	a.AddInt64(key, int64(value))
}

func (a *AliYunEncoder) AddInt16(key string, value int16) {
	a.AddInt64(key, int64(value))
}

func (a *AliYunEncoder) AddInt8(key string, value int8) {
	a.AddInt64(key, int64(value))
}

func (a *AliYunEncoder) AddString(key, value string) {
	a.m[key] = value
}

func (a *AliYunEncoder) AddTime(key string, value time.Time) {
	buf := bufferpool.Get()
	defer buf.Free()
	if a.EncodeTime != nil {
		a.AddString(key, a.EncodeTime(value))
		return
	}

	buf.AppendTime(value, time.RFC3339)
	a.AddString(key, buf.String())
}

func (a *AliYunEncoder) AddUint(key string, value uint) {
	a.AddUint64(key, uint64(value))
}

func (a *AliYunEncoder) AddUint64(key string, value uint64) {
	buf := bufferpool.Get()
	defer buf.Free()
	buf.AppendUint(value)
	a.AddString(key, buf.String())
}

func (a *AliYunEncoder) AddUint32(key string, value uint32) {
	a.AddUint64(key, uint64(value))
}

func (a *AliYunEncoder) AddUint16(key string, value uint16) {
	a.AddUint64(key, uint64(value))
}

func (a *AliYunEncoder) AddUint8(key string, value uint8) {
	a.AddUint64(key, uint64(value))
}

func (a *AliYunEncoder) AddUintptr(key string, value uintptr) {
	a.AddUint64(key, uint64(value))
}
func (a *AliYunEncoder) resetReflectBuf() {
	if a.reflectBuf == nil {
		a.reflectBuf = bufferpool.Get()
		a.reflectEnc = json.NewEncoder(a.reflectBuf)

		// For consistency with our custom JSON encoder.
		a.reflectEnc.SetEscapeHTML(false)
	} else {
		a.reflectBuf.Reset()
	}
}
func (a *AliYunEncoder) encodeReflected(obj interface{}) ([]byte, error) {
	if obj == nil {
		return nullLiteralBytes, nil
	}
	a.resetReflectBuf()
	if err := a.reflectEnc.Encode(obj); err != nil {
		return nil, err
	}
	a.reflectBuf.TrimNewline()
	return a.reflectBuf.Bytes(), nil
}
func (a *AliYunEncoder) AddReflected(key string, value interface{}) error {
	valueBytes, err := a.encodeReflected(value)
	if err != nil {
		return err
	}
	a.AddString(key, string(valueBytes))
	return nil
}

func (a *AliYunEncoder) OpenNamespace(key string) {
	return
}
func (a *AliYunEncoder) clone() *AliYunEncoder {
	return &AliYunEncoder{
		EncoderConfig: a.EncoderConfig,
		m:             a.cloneMap(),
	}
}
func (a *AliYunEncoder) cloneMap() map[string]string {
	m := make(map[string]string)
	for k, v := range a.m {
		m[k] = v
	}
	return m
}
func (a *AliYunEncoder) EncodeEntry(ent zapcore.Entry, fields []zapcore.Field) (map[string]string, error) {
	final := a.clone()
	addFields(final, fields)

	if final.LevelKey != "" {
		level := ent.Level.String()
		if final.EncodeLevel != nil {
			level = final.EncodeLevel(ent.Level)
		}
		final.AddString(final.LevelKey, level)
	}
	if final.TimeKey != "" {
		final.AddTime(final.TimeKey, ent.Time)
	}
	if ent.LoggerName != "" && final.NameKey != "" {
		loggerName := ent.LoggerName
		// if no name encoder provided, fall back to FullNameEncoder for backwards
		// compatibility
		if final.EncodeName != nil {
			loggerName = final.EncodeName(loggerName)
		}
		final.AddString(final.NameKey, loggerName)
	}
	if ent.Caller.Defined {
		if final.CallerKey != "" && final.EncodeCaller != nil {
			caller := final.EncodeCaller(ent.Caller)
			final.AddString(final.CallerKey, caller)
		}
		if final.FunctionKey != "" {
			final.AddString(final.FunctionKey, ent.Caller.Function)
		}
	}
	if final.MessageKey != "" {
		final.AddString(final.MessageKey, ent.Message)
	}

	addFields(final, fields)

	if ent.Stack != "" && final.StacktraceKey != "" {
		final.AddString(final.StacktraceKey, ent.Stack)
	}

	return final.m, nil
}
