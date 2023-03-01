package main

import (
	"encoding/json"
	"fmt"
	"log"

	maelstrom "github.com/jepsen-io/maelstrom/demo/go"
)

func main() {
	n := maelstrom.NewNode()
	kv := maelstrom.NewLinKV(n)
	logs := NewLogStorage(n, kv)

	n.Handle("send", func(msg maelstrom.Message) error {
		var err error
		body := map[string]any{}

		if err = json.Unmarshal(msg.Body, &body); err != nil {
			return err
		}

		key := body["key"].(string)
		val, ok := body["msg"].(float64)
		if !ok {
			return fmt.Errorf("cannot type assert 'msg' to float64: %v", body)
		}
		offset, err := logs.Append(key, int(val))
		if err != nil {
			return err
		}
		return n.Reply(msg, map[string]any{
			"type":   "send_ok",
			"offset": offset,
		})

	})

	n.Handle("poll", func(msg maelstrom.Message) error {
		var err error
		body := map[string]any{}

		if err = json.Unmarshal(msg.Body, &body); err != nil {
			return err
		}

		msgs := map[string][]Log{}

		offsets := body["offsets"].(map[string]any)
		for k, v := range offsets {
			offset := v.(float64)
			msgs[k], err = logs.Poll(k, int(offset))
			if err != nil {
				return err
			}
		}

		return n.Reply(msg, map[string]any{
			"type": "poll_ok",
			"msgs": msgs,
		})

	})

	n.Handle("commit_offsets", func(msg maelstrom.Message) error {
		var err error
		body := map[string]any{}

		if err = json.Unmarshal(msg.Body, &body); err != nil {
			return err
		}

		commit_offsets := body["offsets"].(map[string]any)
		for k, v := range commit_offsets {
			err = logs.Commit(k, int(v.(float64)))
			if err != nil {
				return err
			}
		}

		return n.Reply(msg, map[string]any{
			"type": "commit_offsets_ok",
		})
	})

	n.Handle("list_committed_offsets", func(msg maelstrom.Message) error {
		var err error
		body := map[string]any{}

		if err = json.Unmarshal(msg.Body, &body); err != nil {
			return err
		}

		committed_offsets := map[string]any{}

		keys := body["keys"].([]any)
		for _, k := range keys {
			key := k.(string)
			committed_offsets[key], err = logs.GetCommitedOffset(key)
			if err != nil {
				return err
			}
		}

		return n.Reply(msg, map[string]any{
			"type":    "list_committed_offsets_ok",
			"offsets": committed_offsets,
		})

	})

	if err := n.Run(); err != nil {
		log.Fatal(err)
	}

}
