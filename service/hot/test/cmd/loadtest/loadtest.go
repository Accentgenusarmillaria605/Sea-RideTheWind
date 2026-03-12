package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"sort"
	"sync"
	"sync/atomic"
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
	broker      = flag.String("broker", "127.0.0.1:39092", "Kafka broker")
	topic       = flag.String("topic", "article-hot-events", "Kafka topic")
	duration    = flag.Int("duration", 60, "Test duration in seconds")
	concurrency = flag.Int("concurrency", 10, "Number of concurrent producers")
)

func main() {
	flag.Parse()

	var wg sync.WaitGroup
	var sent, failed atomic.Int64
	var latencies sync.Map
	stop := make(chan struct{})

	start := time.Now()

	for i := 0; i < *concurrency; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			runProducer(id, stop, &sent, &failed, &latencies)
		}(i)
	}

	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				elapsed := time.Since(start).Seconds()
				s := sent.Load()
				f := failed.Load()
				qps := float64(s) / elapsed
				fmt.Printf("[%.0fs] Sent: %d, Failed: %d, QPS: %.2f\n", elapsed, s, f, qps)
			case <-stop:
				return
			}
		}
	}()

	time.Sleep(time.Duration(*duration) * time.Second)
	close(stop)
	wg.Wait()

	elapsed := time.Since(start).Seconds()
	s := sent.Load()
	f := failed.Load()

	var allLatencies []time.Duration
	latencies.Range(func(key, value interface{}) bool {
		allLatencies = append(allLatencies, value.(time.Duration))
		return true
	})
	sort.Slice(allLatencies, func(i, j int) bool { return allLatencies[i] < allLatencies[j] })

	fmt.Printf("\n=== Final Results ===\n")
	fmt.Printf("Duration: %.2fs\n", elapsed)
	fmt.Printf("Total Sent: %d\n", s)
	fmt.Printf("Total Failed: %d\n", f)
	fmt.Printf("Average QPS: %.2f\n", float64(s)/elapsed)

	if len(allLatencies) > 0 {
		p50 := allLatencies[len(allLatencies)*50/100]
		p90 := allLatencies[len(allLatencies)*90/100]
		p95 := allLatencies[len(allLatencies)*95/100]
		p99 := allLatencies[len(allLatencies)*99/100]
		fmt.Printf("\n=== Latency Percentiles ===\n")
		fmt.Printf("p50: %v\n", p50)
		fmt.Printf("p90: %v\n", p90)
		fmt.Printf("p95: %v\n", p95)
		fmt.Printf("p99: %v\n", p99)
	}
}

func runProducer(id int, stop chan struct{}, sent, failed *atomic.Int64, latencies *sync.Map) {
	config := sarama.NewConfig()
	config.Producer.Return.Successes = true
	config.Producer.Partitioner = sarama.NewHashPartitioner

	producer, err := sarama.NewSyncProducer([]string{*broker}, config)
	if err != nil {
		log.Printf("Producer %d failed to start: %v", id, err)
		return
	}
	defer producer.Close()

	types := []string{"like", "comment", "coin", "share"}
	articles := make([]string, 100)
	for i := range articles {
		articles[i] = fmt.Sprintf("article-%d", i)
	}

	for {
		select {
		case <-stop:
			return
		default:
			event := ArticleHotEvent{
				ArticleID: articles[rand.Intn(len(articles))],
				Type:      types[rand.Intn(len(types))],
				UserID:    fmt.Sprintf("user-%d", rand.Intn(10000)),
				Timestamp: time.Now().Unix(),
			}

			data, _ := json.Marshal(event)
			msg := &sarama.ProducerMessage{
				Topic: *topic,
				Key:   sarama.StringEncoder(event.ArticleID),
				Value: sarama.ByteEncoder(data),
			}

			sendStart := time.Now()
			_, _, err := producer.SendMessage(msg)
			latency := time.Since(sendStart)

			if err != nil {
				failed.Add(1)
			} else {
				sent.Add(1)
				latencies.Store(time.Now().UnixNano(), latency)
			}
		}
	}
}
