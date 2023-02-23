package main

import (
	"encoding/json"
	"fmt"
	"log"

	maelstrom "github.com/jepsen-io/maelstrom/demo/go"
)

func getVisibleNodeIds(n *maelstrom.Node, topology map[string]any) []string {
	nodes := topology[n.ID()].([]any)
	retVal := []string{}
	for _, n := range nodes {
		retVal = append(retVal, n.(string))
	}
	return retVal

}

func main() {
	n := maelstrom.NewNode()
	seenValues := []float64{}
	topology := map[string]any{}

	n.Handle("broadcast", func(msg maelstrom.Message) error {
		// Unmarshal the message body as an loosely-typed map.
		var body map[string]any

		if err := json.Unmarshal(msg.Body, &body); err != nil {
			return err
		}

		// Find visible nodes that are not the sender and relay the message
		visibleNodes := getVisibleNodeIds(n, topology)
		println(visibleNodes)
		for _, id := range visibleNodes {
			if id != msg.Src {
				err := n.Send(id, body)
				if err != nil {
					return err
				}
			}
		}

		val, ok := body["message"].(float64)
		if !ok {
			return fmt.Errorf("cannot type assert 'message' to float64: %v", body)
		}

		seenValues = append(seenValues, val)

		body["type"] = "broadcast_ok"
		delete(body, "message")

		return n.Reply(msg, body)
	})

	n.Handle("read", func(msg maelstrom.Message) error {
		body := map[string]any{
			"type":     "read_ok",
			"messages": seenValues,
		}
		return n.Reply(msg, body)
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
		log.Fatal(err)
	}

}
