package consistent_hash

import (
	"fmt"
	"hash/crc32"
	"math"
	"sort"
	"strconv"
	"sync"
)

/*
	一致性哈希算法
	1. 哈希环
	2. 哈希函数
	3. 一致性哈希
简单实现逻辑：
- 为N台服务器生成互不相同的keyword（比如机器的hostname）;
- 将关键词用hash算法映射为数字，并使得该数字始终维持在一个范围，如0到2^32-1次方;
- 将hash后的数字正序排序，如此以来，N台服务器就会根据自身的keyword生成的数字，分布到一个数字形成的环上，如0到2^32-1次方首尾相连的环;
- 将需要存储的变量key同样使用hash算法计算，得到一个0到2^32-1次方范围内的数字;
- 对比变量key的数字，与N台服务器的数字值，找到第一个 >=变量key的hash数值 的服务器，即为需要存储到的机器。
*/

// 定义一个哈希函数接口
type Hasher interface {
	Hash(key []byte) uint32
}

// CRC32 哈希函数
type CRC32 struct{}

func (c *CRC32) Hash(key []byte) uint32 {
	return crc32.ChecksumIEEE(key)
}

func NewCRC32() Hasher {
	return &CRC32{}
}

const (
	DefaultReplicas = 100 // 默认虚拟节点数
	DefaultWeight   = 1   // 默认节点权重，权重值影响节点的虚拟节点数量：实际虚拟节点数 = Replicas × 权重值
)

var DefaultHasher = NewCRC32()

// Config 定义一致性哈希的配置选项
type Config struct {
	Replicas int    // 每个节点的基础虚拟节点数，最终节点的虚拟节点数量 = Replicas × 节点权重
	HashFunc Hasher // 哈希函数
}

// DefaultConfig 返回默认配置
func DefaultConfig() *Config {
	return &Config{
		Replicas: DefaultReplicas,
		HashFunc: DefaultHasher,
	}
}

type ConsistentHash struct {
	config  *Config
	hash    Hasher
	ring    []uint32            // 哈希环(记录的是hash值)
	weights map[string]int      // 节点权重
	hashMap map[uint32]string   // 记录hash环上的节点映射真实节点，方便后续查找
	members map[string]struct{} // 记录已加入的真实节点
	sync.RWMutex
}

// New 创建一个新的一致性哈希实例
func New(config *Config) *ConsistentHash {
	if config == nil {
		config = DefaultConfig()
	}
	return &ConsistentHash{
		config:  config,
		hash:    config.HashFunc,
		ring:    make([]uint32, 0),
		weights: make(map[string]int),
		hashMap: make(map[uint32]string),
		members: make(map[string]struct{}),
	}
}

func (c *ConsistentHash) SetHasher(hash Hasher) {
	if hash == nil {
		panic("hasher cannot be nil")
	}
	c.hash = hash
}

func (c *ConsistentHash) SetVirtualReplicas(replicas int) {
	c.config.Replicas = replicas
}

func (c *ConsistentHash) hashKey(key string) uint32 {
	// 为了减少内存分配过于频繁，指定固定大小的字节数组，减少内存分配带来的性能开销
	if len(key) <= 64 {
		var buf [64]byte
		copy(buf[:], key)
		// fmt.Println("hashKey", key, "=", c.hash.Hash(buf[:]))
		return c.hash.Hash(buf[:])
	}
	return c.hash.Hash([]byte(key))
}

func (c *ConsistentHash) formatKey(key string, replicas int) string {
	return key + "-" + strconv.Itoa(replicas)
}

// addNode adds a node to the hash ring with the specified weight
func (c *ConsistentHash) addNode(member string, weight int) error {
	if weight <= 0 {
		return fmt.Errorf("weight must be positive")
	}
	if _, ok := c.members[member]; ok || member == "" {
		return fmt.Errorf("member already exists or empty")
	}

	c.members[member] = struct{}{}
	c.weights[member] = weight

	// Calculate virtual nodes based on weight
	replicas := c.config.Replicas * weight
	c.ensureRingCapacity(replicas)

	// Add virtual nodes
	for i := 0; i < replicas; i++ {
		hash := c.hashKey(c.formatKey(member, i))
		c.ring = append(c.ring, hash)
		c.hashMap[hash] = member
	}

	c.sortRing()
	return nil
}

// ensureRingCapacity ensures the ring has enough capacity for new nodes
func (c *ConsistentHash) ensureRingCapacity(additionalNodes int) {
	newSize := len(c.ring) + additionalNodes
	if cap(c.ring) < newSize {
		newRing := make([]uint32, len(c.ring), newSize)
		copy(newRing, c.ring)
		c.ring = newRing
	}
}

// sortRing sorts the hash ring
func (c *ConsistentHash) sortRing() {
	sort.Slice(c.ring, func(i, j int) bool {
		return c.ring[i] < c.ring[j]
	})
}

func (c *ConsistentHash) Add(members ...string) {
	c.Lock()
	defer c.Unlock()

	for _, member := range members {
		_ = c.addNode(member, DefaultWeight)
	}
}

// AddWithWeight 添加带权重的节点
/*
weight参数指定节点的权重值，影响节点的虚拟节点数量：
	- weight必须大于0
	- 实际虚拟节点数 = 配置的基础虚拟节点数(Replicas) × weight
	- 权重越大，节点在哈希环上的虚拟节点越多，负责的哈希环范围也就越大
*/
func (c *ConsistentHash) AddWithWeight(member string, weight int) error {
	c.Lock()
	defer c.Unlock()
	return c.addNode(member, weight)
}

// findNode finds the appropriate node for the given hash
func (c *ConsistentHash) findNode(hash uint32) (int, string) {
	idx := sort.Search(len(c.ring), func(i int) bool {
		return c.ring[i] >= hash
	})
	if idx == len(c.ring) {
		idx = 0
	}
	return idx, c.hashMap[c.ring[idx]]
}

func (c *ConsistentHash) Get(key string) string {
	c.RLock()
	defer c.RUnlock()

	if len(c.ring) == 0 {
		return ""
	}
	_, node := c.findNode(c.hashKey(key))
	return node
}

func (c *ConsistentHash) GetN(key string, n int) []string {
	if n <= 0 {
		return nil
	}

	c.RLock()
	defer c.RUnlock()

	if len(c.ring) == 0 {
		return nil
	}

	idx, _ := c.findNode(c.hashKey(key))
	unique := make(map[string]struct{})
	result := make([]string, 0, n)

	for len(unique) < n && len(unique) < len(c.members) {
		member := c.hashMap[c.ring[idx]]
		if _, ok := unique[member]; !ok {
			unique[member] = struct{}{}
			result = append(result, member)
		}
		idx = (idx + 1) % len(c.ring)
	}

	return result
}

func (c *ConsistentHash) Remove(members ...string) {
	c.Lock()
	defer c.Unlock()

	for _, member := range members {
		if _, ok := c.members[member]; !ok || member == "" {
			continue
		}
		// 找到member对应的所有虚拟节点
		replicas := c.config.Replicas * c.weights[member]
		delete(c.members, member)
		delete(c.weights, member)

		for i := 0; i < replicas; i++ {
			// 计算这个member对应的hash值，然后删除
			hash := c.hashKey(c.formatKey(member, i))
			delete(c.hashMap, hash)
		}
		// 更新ring
		// 减少内存分配，直接使用原切片的分配空间
		newRing := c.ring[:0]
		//reallocate if we're holding on to too much (1/4th)
		if cap(c.ring) > len(c.hashMap)*4 {
			newRing = nil
		}
		for hash := range c.hashMap {
			newRing = append(newRing, hash)
		}
		sort.Slice(newRing, func(i, j int) bool {
			return newRing[i] < newRing[j]
		})
		c.ring = newRing
	}
}

// Stats 返回哈希环的统计信息
type Stats struct {
	TotalPhysicalNodes int                // 实际物理节点总数
	TotalHashNodes     int                // 哈希环上的节点总数（包括所有虚拟节点）
	AverageWeight      float64            // 平均权重 = 所有节点权重之和 / 节点数量
	WeightDistribution map[string]int     // 每个节点的权重分布，key为节点名，value为权重值
	LoadDistribution   map[string]float64 // 每个节点负责的哈希环占比。计算方式：
	/*
		1. 对于每个虚拟节点，计算其负责的哈希环范围（到下一个节点的距离）
		2. 将同一个物理节点的所有虚拟节点的范围相加
		3. 最终结果为该物理节点负责的哈希环百分比
	*/
}

// GetStats 获取哈希环的统计信息
// 返回的统计信息包括：
/*
	1. 物理节点数：实际添加的节点数量
	2. 哈希环节点总数：所有虚拟节点的数量之和
	3. 平均权重：所有节点的权重之和 / 节点数量
	4. 权重分布：每个节点的实际权重值
	5. 负载分布：每个节点负责的哈希环范围百分比
*/
func (c *ConsistentHash) GetStats() *Stats {
	c.RLock()
	defer c.RUnlock()

	stats := &Stats{
		TotalPhysicalNodes: len(c.members),
		TotalHashNodes:     len(c.ring),
		LoadDistribution:   make(map[string]float64),
		WeightDistribution: make(map[string]int),
	}

	// 计算权重分布
	totalWeight := 0
	stats.WeightDistribution = make(map[string]int)
	for member := range c.members {
		weight := c.weights[member] // 使用节点的实际权重
		stats.WeightDistribution[member] = weight
		totalWeight += weight
	}
	stats.AverageWeight = float64(totalWeight) / float64(len(c.members))

	if len(c.ring) == 0 {
		return stats
	}

	// 初始化每个成员的负载为0
	for member := range c.members {
		stats.LoadDistribution[member] = 0
	}

	// 计算每个成员的负载分布
	for i := 0; i < len(c.ring); i++ {
		member := c.hashMap[c.ring[i]]
		nextIdx := (i + 1) % len(c.ring)

		start := c.ring[i]
		end := c.ring[nextIdx]

		var portion uint32
		if end > start {
			portion = end - start
		} else {
			// 如果end小于start，说明跨过了0点
			portion = math.MaxUint32 - start + end
		}

		// 计算百分比
		percentage := float64(portion) / float64(math.MaxUint32) * 100
		// fmt.Printf("Debug - Range %d: member=%s, start=%d, end=%d, portion=%d, percentage=%.6f%%\n",
		// 	i, member, start, end, portion, percentage)

		stats.LoadDistribution[member] += percentage
	}

	// // 验证总和是否接近100%
	// var total float64
	// fmt.Println("\nDebug - Final Load Distribution:")
	// for member, load := range stats.LoadDistribution {
	// 	fmt.Printf("Member %s: %.6f%%\n", member, load)
	// 	total += load
	// }
	// fmt.Printf("Debug - Total Load: %.6f%%\n", total)

	// 四舍五入到四位小数
	const precision = 10000 // 10^4 表示保留4位小数
	for member := range stats.LoadDistribution {
		stats.LoadDistribution[member] = math.Round(stats.LoadDistribution[member]*precision) / precision
	}

	return stats
}

func (c *ConsistentHash) Members() []string {
	c.RLock()
	defer c.RUnlock()
	members := make([]string, 0, len(c.members))
	for member := range c.members {
		members = append(members, member)
	}
	return members
}
