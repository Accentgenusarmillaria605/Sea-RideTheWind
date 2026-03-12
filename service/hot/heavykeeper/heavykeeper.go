package heavykeeper

import (
	"container/heap"
	"hash/fnv"
	"math"
	"math/rand"
	"sort"
	"sync"
)

// Item 表示 Top-K 中的一个元素
type Item struct {
	Key   string
	Count uint32
}

// bucket 是哈希表中的单个桶
type bucket struct {
	fingerprint uint32
	count       uint32
}

// HeavyKeeper 实现带衰减的 Top-K 计数算法
type HeavyKeeper struct {
	width int
	depth int
	decay float64
	table [][]bucket
	k     int
	mu    sync.Mutex

	// Top-K 维护：使用 map + min-heap
	topKMap  map[string]uint32 // key -> count
	topKHeap *minHeap
}

// Config 是 HeavyKeeper 的配置参数
type Config struct {
	Width int     `json:"width" yaml:"Width"` // 每行桶数（推荐 1000）
	Depth int     `json:"depth" yaml:"Depth"` // 行数/哈希函数数量（推荐 5）
	Decay float64 `json:"decay" yaml:"Decay"` // 衰减概率基数（推荐 0.9）
	TopK  int     `json:"topk"  yaml:"TopK"`  // 保留 Top-K 数量（推荐 100）
}

// DefaultConfig 返回默认配置
func DefaultConfig() Config {
	return Config{
		Width: 1000,
		Depth: 5,
		Decay: 0.9,
		TopK:  100,
	}
}

// New 创建 HeavyKeeper 实例
func New(cfg Config) *HeavyKeeper {
	table := make([][]bucket, cfg.Depth)
	for i := range table {
		table[i] = make([]bucket, cfg.Width)
	}
	h := &minHeap{}
	heap.Init(h)

	return &HeavyKeeper{
		width:    cfg.Width,
		depth:    cfg.Depth,
		decay:    cfg.Decay,
		table:    table,
		k:        cfg.TopK,
		topKMap:  make(map[string]uint32),
		topKHeap: h,
	}
}

// Add 添加一个 item 并指定权重（点赞=1, 评论=5, 投币=10）
func (hk *HeavyKeeper) Add(key string, weight uint32) {
	hk.mu.Lock()
	defer hk.mu.Unlock()

	fp := hk.fingerprint(key)
	var maxCount uint32

	for row := 0; row < hk.depth; row++ {
		col := hk.hash(key, row) % hk.width
		b := &hk.table[row][col]

		if b.count == 0 {
			// 空桶：占用
			b.fingerprint = fp
			b.count = weight
		} else if b.fingerprint == fp {
			// 指纹匹配：累加权重
			b.count += weight
		} else {
			// 冲突：以概率 decay^count 衰减
			decayProb := math.Pow(hk.decay, float64(b.count))
			if rand.Float64() < decayProb {
				if b.count <= weight {
					b.count = weight
					b.fingerprint = fp
				} else {
					b.count -= weight
				}
			}
		}

		if b.fingerprint == fp && b.count > maxCount {
			maxCount = b.count
		}
	}

	// 更新 Top-K
	hk.updateTopK(key, maxCount)
}

// TopK 返回当前 Top-K 列表（按 count 降序）
func (hk *HeavyKeeper) TopK() []Item {
	hk.mu.Lock()
	defer hk.mu.Unlock()

	items := make([]Item, 0, len(hk.topKMap))
	for key, count := range hk.topKMap {
		items = append(items, Item{Key: key, Count: count})
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].Count > items[j].Count
	})
	return items
}

// updateTopK 维护内存中的 Top-K min-heap
func (hk *HeavyKeeper) updateTopK(key string, count uint32) {
	if count == 0 {
		return
	}

	// 已在 Top-K 中：更新 count
	if _, exists := hk.topKMap[key]; exists {
		hk.topKMap[key] = count
		hk.rebuildHeap()
		return
	}

	// Top-K 未满：直接加入
	if len(hk.topKMap) < hk.k {
		hk.topKMap[key] = count
		heap.Push(hk.topKHeap, heapItem{key: key, count: count})
		return
	}

	// Top-K 已满：与堆顶比较
	if hk.topKHeap.Len() > 0 {
		minItem := (*hk.topKHeap)[0]
		if count > minItem.count {
			// 淘汰堆顶，加入新元素
			delete(hk.topKMap, minItem.key)
			heap.Pop(hk.topKHeap)
			hk.topKMap[key] = count
			heap.Push(hk.topKHeap, heapItem{key: key, count: count})
		}
	}
}

// rebuildHeap 重建最小堆（更新 count 后需要）
// 注：当前 TopK 默认 100，重建开销极低（微秒级）。
// 若 TopK 扩大到万级，可引入 key→heapIndex 映射 + heap.Fix 优化为 O(log n)。
func (hk *HeavyKeeper) rebuildHeap() {
	h := &minHeap{}
	for key, count := range hk.topKMap {
		*h = append(*h, heapItem{key: key, count: count})
	}
	heap.Init(h)
	hk.topKHeap = h
}

// fingerprint 生成 key 的 32 位指纹
func (hk *HeavyKeeper) fingerprint(key string) uint32 {
	h := fnv.New32a()
	h.Write([]byte(key))
	return h.Sum32()
}

// hash 使用第 seed 个哈希函数计算 key 的桶索引
func (hk *HeavyKeeper) hash(key string, seed int) int {
	h := fnv.New64a()
	h.Write([]byte(key))
	h.Write([]byte{byte(seed)})
	return int(h.Sum64() % uint64(hk.width))
}

// ==================== Min-Heap ====================

type heapItem struct {
	key   string
	count uint32
}

type minHeap []heapItem

func (h minHeap) Len() int            { return len(h) }
func (h minHeap) Less(i, j int) bool  { return h[i].count < h[j].count }
func (h minHeap) Swap(i, j int)       { h[i], h[j] = h[j], h[i] }
func (h *minHeap) Push(x interface{}) { *h = append(*h, x.(heapItem)) }
func (h *minHeap) Pop() interface{} {
	old := *h
	n := len(old)
	item := old[n-1]
	*h = old[:n-1]
	return item
}