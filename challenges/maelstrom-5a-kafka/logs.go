package main

import (
	"encoding/json"
)

type LogStorage struct {
	logs    map[string][]int
	commits map[string]int

	MaxLogsToReturn int
}

func (s *LogStorage) Append(key string, msg int) int {
	_, ok := s.logs[key]
	if !ok {
		s.logs[key] = []int{msg}
		return 0
	} else {
		s.logs[key] = append(s.logs[key], msg)
		return len(s.logs[key]) - 1
	}
}

func (s *LogStorage) Poll(key string, offset int) []Log {
	msgs := []Log{}
	logs, ok := s.logs[key]
	if !ok {
		return msgs
	}

	if offset >= len(logs) {
		return msgs
	}

	for i, v := range logs[offset:] {
		if i >= s.MaxLogsToReturn {
			break
		}

		msgs = append(msgs, Log{Offset: offset + i, Msg: v})
	}

	return msgs
}

func (s *LogStorage) Commit(key string, offset int) {
	s.commits[key] = offset
}

func (s *LogStorage) GetCommitedOffset(key string) int {
	return s.commits[key]
}

func NewLogStorage() *LogStorage {
	return &LogStorage{
		logs:            map[string][]int{},
		commits:         map[string]int{},
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
