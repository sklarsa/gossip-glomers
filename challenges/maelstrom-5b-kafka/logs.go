package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	maelstrom "github.com/jepsen-io/maelstrom/demo/go"
)

func commitKey(key string) string {
	return fmt.Sprintf("commit-%s", key)
}

func logKey(key string) string {
	return fmt.Sprintf("logs-%s", key)
}

func lockKey(key string) string {
	return fmt.Sprintf("lock-%s", key)
}

type LogStorage struct {
	n   *maelstrom.Node
	kv  *maelstrom.KV
	ctx context.Context

	MaxLogsToReturn int
}

func (s *LogStorage) requestLock(key string) error {
	err := s.kv.CompareAndSwap(s.ctx, lockKey(key), 0, 1, true)
	var rpcError *maelstrom.RPCError
	for errors.As(err, &rpcError) && rpcError.Code == maelstrom.PreconditionFailed {
		err = s.kv.CompareAndSwap(s.ctx, lockKey(key), 0, 1, true)
	}
	return err
}

func (s *LogStorage) releaseLock(key string) {
	err := s.kv.Write(s.ctx, lockKey(key), 0)
	if err != nil {
		panic(err)
	}
}

func (s *LogStorage) Append(key string, msg int) (int, error) {
	var (
		offset   int
		rpcError *maelstrom.RPCError
		logs     []any
	)

	err := s.requestLock(key)
	defer s.releaseLock(key)

	if err != nil {
		return 0, err
	}

	logData, err := s.kv.Read(s.ctx, logKey(key))
	if errors.As(err, &rpcError) && rpcError.Code == maelstrom.KeyDoesNotExist {
		logs = []any{}
	} else if err != nil {
		return offset, err
	}

	// if logs are nil then logData exists
	if logs == nil {
		logs = logData.([]any)
	}

	logs = append(logs, msg)

	err = s.kv.Write(s.ctx, logKey(key), logs)
	if err != nil {
		return 0, err
	}
	return len(logs) - 1, nil
}

func (s *LogStorage) Poll(key string, offset int) ([]Log, error) {
	msgs := []Log{}
	err := s.requestLock(key)
	if err != nil {
		return msgs, err
	}
	defer s.releaseLock(key)
	logData, err := s.kv.Read(s.ctx, logKey(key))
	var rpcError *maelstrom.RPCError
	if errors.As(err, &rpcError) && rpcError.Code == maelstrom.KeyDoesNotExist {
		return msgs, nil
	}

	if err != nil {
		return msgs, err
	}

	logs := logData.([]any)

	if offset >= len(logs) {
		return msgs, nil
	}

	for i, v := range logs[offset:] {
		if i >= s.MaxLogsToReturn {
			break
		}

		msgs = append(msgs, Log{Offset: offset + i, Msg: int(v.(float64))})
	}

	return msgs, nil
}

func (s *LogStorage) Commit(key string, offset int) error {
	err := s.requestLock(key)
	if err != nil {
		return err
	}
	defer s.releaseLock(key)
	return s.kv.Write(s.ctx, key, offset)
}

func (s *LogStorage) GetCommitedOffset(key string) (int, error) {
	var rpcError *maelstrom.RPCError
	err := s.requestLock(key)
	defer s.releaseLock(key)
	if err != nil {
		return 0, err
	}
	val, err := s.kv.ReadInt(s.ctx, commitKey(key))
	if errors.As(err, &rpcError) && rpcError.Code == maelstrom.KeyDoesNotExist {
		return 0, nil
	}
	return val, err
}

func NewLogStorage(n *maelstrom.Node, kv *maelstrom.KV) *LogStorage {
	return &LogStorage{
		n:   n,
		kv:  kv,
		ctx: context.TODO(),

		MaxLogsToReturn: 10,
	}
}

type Log struct {
	Offset int
	Msg    int
}

func (l *Log) UnmarshalJSON(data []byte) error {
	vals := []int{}
	err := json.Unmarshal(data, &vals)
	if err != nil {
		return err
	}

	l.Offset = vals[0]
	l.Msg = vals[1]

	return nil

}

func (l Log) MarshalJSON() ([]byte, error) {
	vals := []int{l.Offset, l.Msg}
	return json.Marshal(vals)

}
