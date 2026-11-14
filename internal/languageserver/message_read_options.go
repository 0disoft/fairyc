package languageserver

import (
	"context"
	"fmt"
	"io"
)

// Nil fields preserve the existing reader, ReadFull and background context.
// Cancellation is cooperative; ServeContext closes owned blocking inputs on
// shutdown. An arbitrary injected reader must unblock its own pending Read.
type messageReadOptions struct {
	Context  context.Context
	ReadFull func(context.Context, io.Reader, []byte) (int, error)
}

func resolveMessageReadOptions(options []messageReadOptions) (messageReadOptions, error) {
	if len(options) > 1 {
		return messageReadOptions{}, fmt.Errorf("at most one LSP read options value is supported")
	}
	var policy messageReadOptions
	if len(options) == 1 {
		policy = options[0]
	}
	if policy.Context == nil {
		policy.Context = context.Background()
	}
	if err := policy.Context.Err(); err != nil {
		return policy, err
	}
	if policy.ReadFull == nil {
		policy.ReadFull = func(_ context.Context, reader io.Reader, buffer []byte) (int, error) {
			return io.ReadFull(reader, buffer)
		}
	}
	return policy, nil
}

const messageReadChunkBytes = 32 << 10
const messageMaxEmptyReads = 100

type messageBodyReader struct {
	ctx        context.Context
	reader     io.Reader
	emptyReads int
}

func (reader *messageBodyReader) Read(buffer []byte) (int, error) {
	if err := reader.ctx.Err(); err != nil {
		return 0, err
	}
	count, err := reader.reader.Read(buffer[:min(len(buffer), messageReadChunkBytes)])
	if canceled := reader.ctx.Err(); canceled != nil {
		return count, canceled
	}
	if count == 0 && err == nil {
		reader.emptyReads++
		if reader.emptyReads >= messageMaxEmptyReads {
			return 0, io.ErrNoProgress
		}
	} else {
		reader.emptyReads = 0
	}
	return count, err
}
