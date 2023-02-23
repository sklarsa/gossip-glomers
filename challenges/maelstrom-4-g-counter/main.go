package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"

	maelstrom "github.com/jepsen-io/maelstrom/demo/go"
)

func write(kv *maelstrom.KV, key string, val int) error {
	var rpcError *maelstrom.RPCError

	count, err := kv.ReadInt(context.TODO(), key)
	if err != nil {
		if errors.As(err, &rpcError) && rpcError.Code == maelstrom.KeyDoesNotExist {
			count = 0
		} else {
			return err
		}
	}

	return kv.CompareAndSwap(context.TODO(), key, count, int(val)+count, true)
}

func read(kv *maelstrom.KV, key string) (int, error) {
	val, err := kv.ReadInt(context.TODO(), key)

	if err != nil {
		var rpcError *maelstrom.RPCError
		if errors.As(err, &rpcError) {
			if rpcError.Code == maelstrom.KeyDoesNotExist {
				return 0, nil
			}
		}
	}
	return val, err
}

func main() {
	n := maelstrom.NewNode()
	kv := maelstrom.NewSeqKV(n)

	n.Handle("add", func(msg maelstrom.Message) error {

		var (
			body map[string]any
			err  error
		)

		if err := json.Unmarshal(msg.Body, &body); err != nil {
			return err
		}

		val, ok := body["delta"].(float64)
		if !ok {
			return fmt.Errorf("cannot type assert 'delta' to float64: %v", body)
		}
		err = write(kv, n.ID(), int(val))
		for err != nil {
			err = write(kv, n.ID(), int(val))
		}

		body["type"] = "add_ok"
		delete(body, "delta")

		return n.Reply(msg, body)
	})

	n.Handle("read", func(msg maelstrom.Message) error {
		sum := 0
		for _, id := range n.NodeIDs() {
			val, err := read(kv, id)
			for err != nil {
				val, err = read(kv, id)
			}
			sum += val

		}

		body := map[string]any{
			"type":  "read_ok",
			"value": sum,
		}
		return n.Reply(msg, body)

	})

	if err := n.Run(); err != nil {
		log.Fatal(err)
	}

}
