package main

import (
	"encoding/json"
	"log"
	"time"

	maelstrom "github.com/jepsen-io/maelstrom/demo/go"
)

func main() {
	n := maelstrom.NewNode()
	seenValues := []float64{}

	n.Handle("broadcast", func(msg maelstrom.Message) error {
		// Unmarshal the message body as an loosely-typed map.
		var body map[string]any

		if err := json.Unmarshal(msg.Body, &body); err != nil {
			return err
		}

		go func() {
			acks := map[string]bool{}
			var allAcked bool

			for _, id := range n.NodeIDs() {
				if id != n.ID() {
					acks[id] = false
				}
			}

			for !allAcked {
				replicationBody := map[string]any{
					"type":    "ack",
					"message": body["message"],
				}

				replicationBody["type"] = "replicate"
				for id, acked := range acks {
					if !acked {
						err := n.Send(id, replicationBody)
						if err == nil {
							acks[id] = true
						}
					}
				}

				allAcked = true
				for _, acked := range acks {
					if !acked {
						allAcked = false
					}
				}

				time.Sleep(100 * time.Millisecond)
			}

		}()

		// Find visible nodes that are not the sender and relay the message

		val := body["message"].(float64)

		seenValues = append(seenValues, val)

		body["type"] = "broadcast_ok"
		delete(body, "message")

		return n.Reply(msg, body)
	})

	n.Handle("replicate", func(msg maelstrom.Message) error {
		var body map[string]any

		if err := json.Unmarshal(msg.Body, &body); err != nil {
			return err
		}

		val := body["message"].(float64)
		seenValues = append(seenValues, val)

		return nil

	})

	n.Handle("read", func(msg maelstrom.Message) error {
		body := map[string]any{
			"type":     "read_ok",
			"messages": seenValues,
		}
		return n.Reply(msg, body)
	})

	n.Handle("topology", func(msg maelstrom.Message) error {
		return n.Reply(msg, map[string]any{
			"type": "topology_ok",
		})
	})

	if err := n.Run(); err != nil {
		log.Fatal(err)
	}

}
