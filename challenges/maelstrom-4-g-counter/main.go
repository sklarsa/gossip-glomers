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
	gossipInterval = 1 * time.Second
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

func readCount(thisNodeId string, kv *maelstrom.KV, targetNodeId string) (int, error) {
	val, err := kv.ReadInt(ctx, fmt.Sprintf("%s-%s", thisNodeId, targetNodeId))
	var rpcErr *maelstrom.RPCError
	if errors.As(err, &rpcErr) && rpcErr.Code == maelstrom.KeyDoesNotExist {
		val = 0
		err = nil
	}
	return val, err
}

func writeCount(n *maelstrom.Node, kv *maelstrom.KV, nodeId string, count int) error {
	return kv.Write(ctx, fmt.Sprintf("%s-%s", n.ID(), nodeId), count)
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
			count, err := readCount(n.ID(), kv, id)
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

		nodeId := msg.Src
		gossipCount := int(body["count"].(float64))

		storeCount, err := readCount(n.ID(), kv, nodeId)
		if err != nil {
			return err
		}

		if gossipCount > storeCount {
			err = writeCount(n, kv, nodeId, gossipCount)
			if err != nil {
				return err
			}

		}

		return nil

	})

	n.Handle("add", func(msg maelstrom.Message) error {

		var body map[string]any

		if err := json.Unmarshal(msg.Body, &body); err != nil {
			return err
		}

		val, ok := body["delta"].(float64)
		if !ok {
			return fmt.Errorf("cannot type assert 'delta' to float64: %v", body)
		}

		count, err := readCount(n.ID(), kv, n.ID())
		if err != nil {
			return err
		}

		err = writeCount(n, kv, n.ID(), count+int(val))
		if err != nil {
			return err
		}

		return n.Reply(msg, map[string]any{
			"type": "add_ok",
		})
	})

	n.Handle("read", func(msg maelstrom.Message) error {
		sum := 0
		for _, id := range n.NodeIDs() {
			for _, id2 := range n.NodeIDs() {
				count, err := readCount(id, kv, id2)
				if err != nil {
					return err
				}
				sum += count
			}

		}

		return n.Reply(msg, map[string]any{
			"type":  "read_ok",
			"value": sum,
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
