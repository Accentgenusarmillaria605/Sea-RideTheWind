package heavykeeper

import (
	"fmt"
	"testing"
)

func TestHeavyKeeper_BasicAdd(t *testing.T) {
	hk := New(DefaultConfig())

	// 模拟高频文章
	for i := 0; i < 1000; i++ {
		hk.Add("article-hot", 1)
	}
	for i := 0; i < 100; i++ {
		hk.Add("article-warm", 1)
	}
	for i := 0; i < 10; i++ {
		hk.Add("article-cold", 1)
	}

	topK := hk.TopK()
	if len(topK) == 0 {
		t.Fatal("TopK should not be empty")
	}
	if topK[0].Key != "article-hot" {
		t.Errorf("expected top item to be article-hot, got %s", topK[0].Key)
	}
}

func TestHeavyKeeper_WeightedAdd(t *testing.T) {
	hk := New(DefaultConfig())

	// 文章A：100次点赞(权重1) + 20次评论(权重5) + 5次投币(权重10)
	// 期望热度 = 100*1 + 20*5 + 5*10 = 250
	for i := 0; i < 100; i++ {
		hk.Add("article-A", 1)
	}
	for i := 0; i < 20; i++ {
		hk.Add("article-A", 5)
	}
	for i := 0; i < 5; i++ {
		hk.Add("article-A", 10)
	}

	// 文章B：200次点赞(权重1)
	// 期望热度 = 200
	for i := 0; i < 200; i++ {
		hk.Add("article-B", 1)
	}

	topK := hk.TopK()
	if len(topK) < 2 {
		t.Fatal("TopK should have at least 2 items")
	}
	// article-A 热度更高，应排在前面
	if topK[0].Key != "article-A" {
		t.Errorf("expected article-A to be top, got %s (count=%d)", topK[0].Key, topK[0].Count)
	}
}

func TestHeavyKeeper_TopKCapacity(t *testing.T) {
	cfg := DefaultConfig()
	cfg.TopK = 10
	hk := New(cfg)

	// 插入 50 个不同的文章
	for i := 0; i < 50; i++ {
		key := fmt.Sprintf("article-%d", i)
		for j := 0; j < (i + 1); j++ {
			hk.Add(key, 1)
		}
	}

	topK := hk.TopK()
	if len(topK) > 10 {
		t.Errorf("TopK length should be <= 10, got %d", len(topK))
	}
}

func TestHeavyKeeper_ConflictDecay(t *testing.T) {
	// 验证冲突衰减机制：高频 item 应稳定占据 Top-K，不会被大量低频 item 冲掉
	cfg := Config{Width: 100, Depth: 3, Decay: 0.9, TopK: 5}
	hk := New(cfg)

	// 先建立一个高频 item
	for i := 0; i < 500; i++ {
		hk.Add("hot-article", 10)
	}

	// 用大量不同的低频 item 制造冲突
	for i := 0; i < 1000; i++ {
		key := fmt.Sprintf("noise-%d", i)
		hk.Add(key, 1)
	}

	// 高频 item 应仍在 Top-K 中（HeavyKeeper 的核心保证）
	topK := hk.TopK()
	found := false
	for _, item := range topK {
		if item.Key == "hot-article" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("hot-article should survive conflict decay, but not found in TopK: %+v", topK)
	}
}