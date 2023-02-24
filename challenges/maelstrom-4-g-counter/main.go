package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"time"

	maelstrom "github.com/jepsen-io/maelstrom/demo/go"
)

var (
	ctx            = context.TODO()
	gossipInterval = 1 * time.Second
)

func readCount(kv *maelstrom.KV, nodeId string) (int, error) {
	val, err := kv.ReadInt(ctx, nodeId)
	var rpcErr *maelstrom.RPCError
	if errors.As(err, &rpcErr) && rpcErr.Code == maelstrom.KeyDoesNotExist {
		val = 0
		err = nil
	}
	return val, err
}

func writeCount(kv *maelstrom.KV, nodeId string, count int) error {
	return kv.Write(ctx, nodeId, count)
}

func writeDelta(kv *maelstrom.KV, nodeId string, delta int) error {
	count, err := readCount(kv, nodeId)
	if err != nil {
		return err
	}
	err = kv.CompareAndSwap(ctx, nodeId, count, count+delta, true)
	var rpcErr *maelstrom.RPCError
	for errors.As(err, &rpcErr) && rpcErr.Code == maelstrom.PreconditionFailed {
		count, err = readCount(kv, nodeId)
		if err != nil {
			return err
		}
		err = kv.CompareAndSwap(ctx, nodeId, count, count+delta, true)

	}
	return err
}

func main() {
	rand.Seed(time.Now().UnixNano())
	n := maelstrom.NewNode()
	kv := maelstrom.NewSeqKV(n)
	ticker := time.NewTicker(gossipInterval)

	go func() {
		for {
			<-ticker.C
			// Gossip here
			myCounts := map[string]int{}
			for _, id := range n.NodeIDs() {
				count, err := readCount(kv, id)
				if err != nil {
					panic(err)
				}
				myCounts[id] = count
			}

			for _, id := range n.NodeIDs() {
				if id == n.ID() {
					continue
				}

				body := map[string]any{
					"type":   "gossip",
					"counts": myCounts,
				}

				err := n.Send(id, body)
				if err != nil {
					panic(err)
				}
			}
		}

	}()

	n.Handle("gossip", func(msg maelstrom.Message) error {
		var body map[string]any

		if err := json.Unmarshal(msg.Body, &body); err != nil {
			return err
		}

		gossipCounts := body["counts"].(map[string]any)

		for _, id := range n.NodeIDs() {
			storeCount, err := readCount(kv, id)
			if err != nil {
				return err
			}

			gossipCount := int(gossipCounts[id].(float64))

			if gossipCount > storeCount {

				err = writeCount(kv, msg.Src, gossipCount)
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

		err = writeDelta(kv, n.ID(), int(val))
		if err != nil {
			return err
		}

		return n.Reply(msg, map[string]any{
			"type": "add_ok",
		})
	})

	n.Handle("read", func(msg maelstrom.Message) error {
		var sum int
		for _, id := range n.NodeIDs() {
			count, err := readCount(kv, id)
			if err != nil {
				return err
			}
			sum += count
		}

		return n.Reply(msg, map[string]any{
			"type":  "read_ok",
			"value": sum,
		})
	})

	if err := n.Run(); err != nil {
		ticker.Stop()
		log.Fatal(err)
	}

}
