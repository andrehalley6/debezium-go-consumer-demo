package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
)

func decodeDecimal(base64Str string, scale int) float64 {
	bytes, err := base64.StdEncoding.DecodeString(base64Str)
	if err != nil {
		return 0
	}

	// Reverse the byte slice to match the Java BigInteger behavior
	for i, j := 0, len(bytes)-1; i < j; i, j = i+1, j-1 {
		bytes[i], bytes[j] = bytes[j], bytes[i]
	}

	bi := new(big.Int).SetBytes(bytes)
	val := new(big.Float).SetInt(bi)

	scaleFactor := new(big.Float).SetFloat64(float64(1))
	for i := 0; i < scale; i++ {
		scaleFactor.Mul(scaleFactor, big.NewFloat(10))
	}

	val.Quo(val, scaleFactor)
	f64, _ := val.Float64()
	return f64
}

func main() {
	topic := "pgdemo.public.products"

	consumer, err := kafka.NewConsumer(&kafka.ConfigMap{
		"bootstrap.servers": "localhost:29092",
		"group.id":          "debezium-go-consumer",
		"auto.offset.reset": "earliest",
	})
	if err != nil {
		panic(err)
	}
	defer consumer.Close()

	err = consumer.SubscribeTopics([]string{topic}, nil)
	if err != nil {
		panic(err)
	}

	fmt.Printf("Listening to topic: %s\n", topic)
	fmt.Println("Press Ctrl+C to exit.")

	sigchan := make(chan os.Signal, 1)
	signal.Notify(sigchan, syscall.SIGINT, syscall.SIGTERM)

	run := true
	for run {
		select {
		case sig := <-sigchan:
			fmt.Printf("Caught signal %v: terminating\n", sig)
			run = false
		default:
			msg, err := consumer.ReadMessage(100 * time.Millisecond)
			if err != nil {
				continue
			}

			fmt.Println("Raw message:")
			fmt.Println(string(msg.Value))
			fmt.Println("--------------------------------------------------------------------------------")

			var root map[string]interface{}
			err = json.Unmarshal(msg.Value, &root)
			if err != nil {
				fmt.Println("Invalid JSON:", err)
				continue
			}

			payload := root["payload"].(map[string]interface{})
			op := payload["op"].(string)

			if op == "c" || op == "u" {
				after := payload["after"].(map[string]interface{})
				id := int(after["id"].(float64))
				name := after["name"].(string)
				priceMap := after["price"].(map[string]interface{})
				value := priceMap["value"].(string)
				scale := int(priceMap["scale"].(float64))
				price := decodeDecimal(value, scale)

				if op == "c" {
					fmt.Printf("Created id: %d, name: %s, price: %.2f\n", id, name, price)
				} else if op == "u" {
					fmt.Printf("Updated id: %d, name: %s, price: %.2f\n", id, name, price)
				}
			} else if op == "d" {
				before := payload["before"].(map[string]interface{})
				id := int(before["id"].(float64))
				fmt.Printf("Deleted id: %d\n", id)
			}

			fmt.Println("--------------------------------------------------------------------------------")
		}
	}

	fmt.Println("Closing consumer")
}
