package main

import (
	"encoding/json"
	"log"
	"sync"

	maelstrom "github.com/jepsen-io/maelstrom/demo/go"
)

func main() {
	n := maelstrom.NewNode()
	kv := map[float64]float64{}
	mu := &sync.Mutex{}

	n.Handle("txn", func(msg maelstrom.Message) error {
		// Unmarshal the message body as an loosely-typed map.
		var body map[string]any

		if err := json.Unmarshal(msg.Body, &body); err != nil {
			return err
		}

		reads := [][]float64{}
		writes := [][]float64{}
		txns := body["txn"].([]any)
		returnData := [][]any{}

		for _, t := range txns {
			txn := []float64{
				t.([]any)[1].(float64),
			}

			if t.([]any)[0] == "r" {
				reads = append(reads, txn)
			} else {
				txn = append(txn, t.([]any)[2].(float64))
				writes = append(writes, txn)
			}
		}

		mu.Lock()
		defer mu.Unlock()

		for _, r := range reads {
			key := r[0]
			val, ok := kv[key]
			data := []any{"r", key}
			if !ok {
				data = append(data, nil)
			} else {
				data = append(data, val)
			}
			returnData = append(returnData, data)
		}

		for _, w := range writes {
			kv[w[0]] = w[1]
			data := []any{"w", w[0], w[1]}
			returnData = append(returnData, data)
		}

		body["type"] = "txn_ok"
		body["txn"] = returnData
		delete(body, "message")

		return n.Reply(msg, body)
	})

	if err := n.Run(); err != nil {
		log.Fatal(err)
	}

}
