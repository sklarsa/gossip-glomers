package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"time"

	maelstrom "github.com/jepsen-io/maelstrom/demo/go"
)

var (
	writeKey    = "writing"
	readKey     = "reading"
	registerKey = "register"
	ctx         = context.TODO()
)

func write(kv *maelstrom.KV, val int) error {
	var (
		rpcError *maelstrom.RPCError
		err      error
	)

	err = kv.Write(ctx, writeKey, true)
	if err != nil {
		return err
	}

	count, err := kv.ReadInt(ctx, registerKey)
	if err != nil {
		if errors.As(err, &rpcError) && rpcError.Code == maelstrom.KeyDoesNotExist {
			count = 0
		} else {
			return err
		}
	}

	err = kv.Write(ctx, registerKey, int(val)+count)
	if err != nil {
		return err
	}

	return kv.Write(ctx, writeKey, false)
}

func read(kv *maelstrom.KV) (int, error) {
	var err error
	err = kv.Write(ctx, readKey, time.Now())
	if err != nil {
		return 0, err
	}

	val, err := kv.ReadInt(ctx, registerKey)

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
		err = write(kv, int(val))
		for err != nil {
			err = write(kv, int(val))
		}

		body["type"] = "add_ok"
		delete(body, "delta")

		return n.Reply(msg, body)
	})

	n.Handle("read", func(msg maelstrom.Message) error {
		val, err := read(kv)
		for err != nil {
			val, err = read(kv)
		}

		body := map[string]any{
			"type":  "read_ok",
			"value": val,
		}
		return n.Reply(msg, body)

	})

	if err := n.Run(); err != nil {
		log.Fatal(err)
	}

}
