package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"time"

	"github.com/IBM/sarama"
)

type ArticleHotEvent struct {
	ArticleID string `json:"article_id"`
	Type      string `json:"type"`
	UserID    string `json:"user_id"`
	Timestamp int64  `json:"timestamp"`
}

var (
	broker = flag.String("broker", "127.0.0.1:39092", "Kafka broker")
	topic  = flag.String("topic", "article-hot-events", "Kafka topic")
	count  = flag.Int("count", 100, "Number of events to send")
	qps    = flag.Int("qps", 10, "Events per second")
)

func main() {
	flag.Parse()

	config := sarama.NewConfig()
	config.Producer.Return.Successes = true
	config.Producer.Partitioner = sarama.NewHashPartitioner

	producer, err := sarama.NewSyncProducer([]string{*broker}, config)
	if err != nil {
		log.Fatal(err)
	}
	defer producer.Close()

	types := []string{"like", "comment", "coin", "share"}
	articles := []string{"article-1", "article-2", "article-3", "article-4", "article-5"}

	interval := time.Second / time.Duration(*qps)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	sent := 0
	start := time.Now()

	for i := 0; i < *count; i++ {
		<-ticker.C

		event := ArticleHotEvent{
			ArticleID: articles[rand.Intn(len(articles))],
			Type:      types[rand.Intn(len(types))],
			UserID:    fmt.Sprintf("user-%d", rand.Intn(1000)),
			Timestamp: time.Now().Unix(),
		}

		data, _ := json.Marshal(event)
		msg := &sarama.ProducerMessage{
			Topic: *topic,
			Key:   sarama.StringEncoder(event.ArticleID),
			Value: sarama.ByteEncoder(data),
		}

		_, _, err := producer.SendMessage(msg)
		if err != nil {
			log.Printf("Failed to send: %v", err)
			continue
		}

		sent++
		if sent%100 == 0 {
			elapsed := time.Since(start)
			actualQPS := float64(sent) / elapsed.Seconds()
			fmt.Printf("Sent %d events, actual QPS: %.2f\n", sent, actualQPS)
		}
	}

	elapsed := time.Since(start)
	fmt.Printf("\nTotal: %d events in %.2fs, avg QPS: %.2f\n", sent, elapsed.Seconds(), float64(sent)/elapsed.Seconds())
}
