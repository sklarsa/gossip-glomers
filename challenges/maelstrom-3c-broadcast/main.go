package main

import (
	"encoding/json"
	"log"
	"sync"
	"time"

	maelstrom "github.com/jepsen-io/maelstrom/demo/go"
)

func main() {
	n := maelstrom.NewNode()
	seenValues := map[float64]bool{}
	mu := &sync.Mutex{}
	gossipInterval := 500 * time.Millisecond
	ticker := time.NewTicker(gossipInterval)

	go func() {
		for {
			<-ticker.C
			// Gossip here
			seenValuesToSend := []float64{}
			mu.Lock()
			for v := range seenValues {
				seenValuesToSend = append(seenValuesToSend, v)
			}
			mu.Unlock()
			for _, id := range n.NodeIDs() {
				if id == n.ID() {
					continue
				}

				body := map[string]any{
					"type":       "gossip",
					"seenValues": seenValuesToSend,
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

		remoteSeenValues := body["seenValues"].([]any)

		for _, v := range remoteSeenValues {
			mu.Lock()
			if _, ok := seenValues[v.(float64)]; !ok {
				seenValues[v.(float64)] = true
			}
			mu.Unlock()
		}

		return nil

	})

	n.Handle("broadcast", func(msg maelstrom.Message) error {
		// Unmarshal the message body as an loosely-typed map.
		var body map[string]any

		if err := json.Unmarshal(msg.Body, &body); err != nil {
			return err
		}

		// Find visible nodes that are not the sender and relay the message

		val := body["message"].(float64)

		mu.Lock()
		seenValues[val] = true
		mu.Unlock()

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

		mu.Lock()
		seenValues[val] = true
		mu.Unlock()

		return n.Reply(msg, map[string]any{"replicate-ack": val})

	})

	n.Handle("read", func(msg maelstrom.Message) error {
		vals := []float64{}
		mu.Lock()
		for v := range seenValues {
			vals = append(vals, v)
		}
		mu.Unlock()
		body := map[string]any{
			"type":     "read_ok",
			"messages": vals,
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
