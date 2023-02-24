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
	ctx            = context.TODO()
	gossipInterval = 500 * time.Second
	key            = "count"
)

func getVisibleNodeIds(n *maelstrom.Node, topology map[string]any) []string {
	retVal := []string{}
	nodes, ok := topology[n.ID()]
	if !ok {
		return retVal
	}

	for _, n := range nodes.([]any) {
		retVal = append(retVal, n.(string))
	}
	return retVal
}

func readCount(kv *maelstrom.KV) (int, error) {
	val, err := kv.ReadInt(ctx, key)
	var rpcErr *maelstrom.RPCError
	if errors.As(err, &rpcErr) && rpcErr.Code == maelstrom.KeyDoesNotExist {
		val = 0
		err = nil
	}
	return val, err
}

func writeCount(kv *maelstrom.KV, count int) error {
	return kv.Write(ctx, key, count)
}

func writeDelta(kv *maelstrom.KV, delta int) error {
	count, err := readCount(kv)
	if err != nil {
		return err
	}
	err = kv.CompareAndSwap(ctx, key, count, count+delta, true)
	var rpcErr *maelstrom.RPCError
	for errors.As(err, &rpcErr) && rpcErr.Code == maelstrom.PreconditionFailed {
		count, err = readCount(kv)
		if err != nil {
			return err
		}
		err = kv.CompareAndSwap(ctx, key, count, count+delta, true)

	}
	return err
}

func main() {
	n := maelstrom.NewNode()
	kv := maelstrom.NewSeqKV(n)
	ticker := time.NewTicker(gossipInterval)
	topology := map[string]any{}

	go func() {
		// Gossip here
		if topology == nil {
			return
		}

		visibleNodes := getVisibleNodeIds(n, topology)
		for _, id := range visibleNodes {
			count, err := readCount(kv)
			if err != nil {
				panic(err)
			}
			body := map[string]any{
				"type":  "gossip",
				"count": count,
			}

			err = n.Send(id, body)
			if err != nil {
				panic(err)
			}
		}

	}()

	n.Handle("gossip", func(msg maelstrom.Message) error {
		var body map[string]any

		if err := json.Unmarshal(msg.Body, &body); err != nil {
			return err
		}

		gossipCount := int(body["count"].(float64))

		storeCount, err := readCount(kv)
		if err != nil {
			return err
		}

		if gossipCount > storeCount {
			err = writeCount(kv, gossipCount)
			if err != nil {
				return err
			}
		}

		newCount, err := readCount(kv)
		if err != nil {
			return err
		}
		body["count"] = newCount

		visibleNodes := getVisibleNodeIds(n, topology)
		for _, id := range visibleNodes {
			if id != msg.Src {
				err := n.Send(id, body)
				if err != nil {
					return err
				}
			}
		}

		return nil

	})

	n.Handle("add", func(msg maelstrom.Message) error {

		var (
			body map[string]any
			err  error
		)

		if err = json.Unmarshal(msg.Body, &body); err != nil {
			return err
		}

		val, ok := body["delta"].(float64)
		if !ok {
			return fmt.Errorf("cannot type assert 'delta' to float64: %v", body)
		}

		err = writeDelta(kv, int(val))
		if err != nil {
			return err
		}

		return n.Reply(msg, map[string]any{
			"type": "add_ok",
		})
	})

	n.Handle("read", func(msg maelstrom.Message) error {
		count, err := readCount(kv)
		if err != nil {
			return err
		}

		return n.Reply(msg, map[string]any{
			"type":  "read_ok",
			"value": count,
		})
	})

	n.Handle("topology", func(msg maelstrom.Message) error {
		var body map[string]any

		if err := json.Unmarshal(msg.Body, &body); err != nil {
			return err
		}

		topology = body["topology"].(map[string]any)

		return n.Reply(msg, map[string]any{
			"type": "topology_ok",
		})
	})

	if err := n.Run(); err != nil {
		ticker.Stop()
		log.Fatal(err)
	}

}
