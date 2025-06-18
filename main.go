package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math"
	"math/big"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
)

func decodeDecimal(base64Value string, scale int) float64 {
	// Decode the base64 string to bytes
	decodedBytes, err := base64.StdEncoding.DecodeString(base64Value)
	if err != nil {
		return 0
	}

	// Convert bytes to big.Int
	intVal := new(big.Int).SetBytes(decodedBytes)

	// Convert to big.Float for division
	num := new(big.Float).SetInt(intVal)

	// Calculate divisor = 10^scale using math.Pow
	divisor := math.Pow(10, float64(scale))

	// Divide and return
	result := new(big.Float).Quo(num, big.NewFloat(divisor))

	// Convert to float64
	floatVal, _ := result.Float64()
	return floatVal
}

func main() {
	topic := "pgdemo.public.products"

	consumer, err := kafka.NewConsumer(&kafka.ConfigMap{
		"bootstrap.servers": "localhost:29092",
		"group.id":          "debezium-go-consumer",
		"auto.offset.reset": "earliest", // use "latest" for latest messages
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

			if msg.Value == nil {
				fmt.Println("Skipping empty message")
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

			if op == "c" || op == "u" || op == "r" {
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
				} else if op == "r" {
					fmt.Printf("Read id: %d, name: %s, price: %.2f\n", id, name, price)
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
