package logger

type NoopLogger struct{}

func (n *NoopLogger) Init(path string)            {}
func (n *NoopLogger) InitMultiWriter(path string) {}

func (n *NoopLogger) Info(msg string)  {}
func (n *NoopLogger) Warn(msg string)  {}
func (n *NoopLogger) Error(msg string) {}
func (n *NoopLogger) Fatal(msg string) {}
func (n *NoopLogger) Debug(msg string) {}
func (n *NoopLogger) Panic(msg string) {}

func (n *NoopLogger) WithStr(key, value string) Logger       { return &NoopLogger{} }
func (n *NoopLogger) WithBool(key string, value bool) Logger { return &NoopLogger{} }
func (n *NoopLogger) WithInt(key string, value int) Logger   { return &NoopLogger{} }
func (n *NoopLogger) WithAny(key string, value any) Logger   { return &NoopLogger{} }
