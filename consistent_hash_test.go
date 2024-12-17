package consistent_hash

import (
	"fmt"
	"math"
	"os"
	"strings"
	"testing"

	"github.com/wcharczuk/go-chart" // 引入 go-charts 包
)

/*
	为了验证新增节点的时候，是否能够正确的分配到新节点上，需要针对hasher自定义
*/

// 自定义一个可预测的哈希函数，使得我们可以控制节点在哈希环上的位置
type TestHasher struct {
	// 预定义的哈希值映射
	hashValues map[string]uint32
}

func NewTestHasher() *TestHasher {
	return &TestHasher{
		hashValues: make(map[string]uint32),
	}
}

func (h *TestHasher) Hash(key []byte) uint32 {
	// 将字节数组转换为字符串，并去除尾部的空字节
	trimmedKey := strings.TrimRight(string(key), "\x00")

	// 直接查找完整的键值
	if hash, ok := h.hashValues[trimmedKey]; ok {
		return hash
	}

	// 如果没有找到预设的哈希值，返回一个固定值
	return 0
}

func (h *TestHasher) SetHash(key string, hash uint32) {
	h.hashValues[key] = hash
}

func Test_ConsistentHash(t *testing.T) {
	// 初始化哈希函数和一致性哈希环
	hasher := NewTestHasher()
	ch := New(&Config{
		Replicas: 1,
		HashFunc: hasher,
	})

	// 1. 初始化节点配置
	ringSize := uint32(math.MaxUint32)
	interval := ringSize / 4 // 将环分成4份

	// 设置4个初始节点，均匀分布在哈希环上
	nodes := []string{"node1", "node2", "node3", "node4"}
	nodeHashes := map[string]uint32{
		"node1-0": 0,            // 0
		"node2-0": interval,     // 1/4 环
		"node3-0": interval * 2, // 2/4 环
		"node4-0": interval * 3, // 3/4 环
	}

	// 设置节点的哈希值
	fmt.Println("\nDebug - Node Hash Values:")
	for node, hash := range nodeHashes {
		fmt.Printf("%s: %d\n", node, hash)
		hasher.SetHash(node, hash)
	}

	// 添加节点到哈希环
	ch.Add(nodes...)

	// 2. 生成测试数据
	fmt.Println("\n初始数据分布:")
	testDataCount := 20
	dataInterval := ringSize / uint32(testDataCount) // 将测试数据均匀分布在环上

	testData := make([]string, 0, testDataCount)
	testDataHashes := make(map[string]uint32)
	keyToNode := make(map[string]string)
	initialDistribution := make(map[string]int)

	// 生成均匀分布的测试数据
	for i := 0; i < testDataCount; i++ {
		key := fmt.Sprintf("key-%d", i)
		keyHash := uint32(i) * dataInterval
		hasher.SetHash(key, keyHash)

		hash := hasher.Hash([]byte(key))
		node := ch.Get(key)

		fmt.Printf("hashKey %s = %d\n", key, hash)
		fmt.Printf("数据 %s (hash=%d) 被分配到节点 %s\n", key, hash, node)

		testData = append(testData, key)
		testDataHashes[key] = hash
		keyToNode[key] = node
		initialDistribution[node]++
	}

	// 打印初始状态统计信息
	stats := ch.GetStats()
	printStats(stats)
	// 生成初始数据分布饼状图
	generatePieChart(initialDistribution, "初始数据分布饼状图")

	// 3. 测试添加节点
	fmt.Println("\n添加节点 node5 后的数据分布:")
	newNode := "node5"
	// 将node5放在node1和node2之间的中点
	hasher.SetHash("node5-0", interval/2)
	ch.Add(newNode)

	// 检查数据迁移
	movedKeys := make(map[string]struct{})
	newDistribution := make(map[string]int)

	for _, key := range testData {
		newNode := ch.Get(key)
		newDistribution[newNode]++

		if oldNode := keyToNode[key]; oldNode != newNode {
			movedKeys[key] = struct{}{}
			fmt.Printf("数据 %s 从 %s 迁移到 %s\n", key, oldNode, newNode)
		}
	}

	// 打印节点数据统计
	for node, count := range newDistribution {
		fmt.Printf("节点 %s 的数据数量：%d\n", node, count)
	}

	// 重新打印所有数据的分布情况
	for _, key := range testData {
		hash := testDataHashes[key]
		node := ch.Get(key)
		fmt.Printf("hashKey %s = %d\n", key, hash)
		fmt.Printf("数据 %s (hash=%d) 被分配到节点 %s\n", key, hash, node)
	}

	// 打印添加节点后的统计信息
	stats = ch.GetStats()
	printStats(stats)
	// 生成添加节点后的饼状图
	generatePieChart(newDistribution, "添加节点后数据分布饼状图")

	// 4. 测试删除节点
	fmt.Println("\n删除节点 node2 的影响:")
	nodeToRemove := "node2"
	fmt.Printf("hashKey %s-0 = %d\n", nodeToRemove, hasher.Hash([]byte(nodeToRemove+"-0")))

	ch.Remove(nodeToRemove)
	fmt.Println("\n删除节点 node2 后的数据分布:")

	// 检查数据迁移
	finalDistribution := make(map[string]int)
	for _, key := range testData {
		hash := testDataHashes[key]
		newNode := ch.Get(key)
		finalDistribution[newNode]++
		if oldNode := keyToNode[key]; oldNode == nodeToRemove {
			fmt.Printf("数据 %s (hash=%d) 从 %s 迁移到 %s\n", key, hash, oldNode, newNode)
		}
	}

	// 打印删除节点后的统计信息
	stats = ch.GetStats()
	printStats(stats)
	// 生成删除节点后的饼状图
	generatePieChart(finalDistribution, "删除节点后数据分布饼状图")
}

func printStats(stats *Stats) {
	fmt.Printf("Total Physical Nodes: %d\n", stats.TotalPhysicalNodes)
	fmt.Printf("Total Hash Nodes: %d\n", stats.TotalHashNodes)
	fmt.Printf("Average Weight: %.2f\n", stats.AverageWeight)
	fmt.Println("Weight Distribution:")
	for node, weight := range stats.WeightDistribution {
		fmt.Printf("  %s: %d\n", node, weight)
	}
	fmt.Println("Load Distribution:")
	for node, load := range stats.LoadDistribution {
		fmt.Printf("  %s: %.2f%%\n", node, load)
	}
}

// 生成饼状图的函数
func generatePieChart(distribution map[string]int, title string) {
	pieChart := chart.PieChart{
		Title:  title,
		Width:  600,
		Height: 400,
		Values: []chart.Value{},
	}

	for node, count := range distribution {
		pieChart.Values = append(pieChart.Values, chart.Value{
			Label: fmt.Sprintf("%s: %.0f", node, float64(count)), // 显示节点名称和对应的值
			Value: float64(count),
		})
	}

	// 保存饼状图为 PNG 文件
	f, err := os.Create(fmt.Sprintf("%s.png", title))
	if err != nil {
		fmt.Printf("无法创建文件: %v\n", err)
		return
	}
	defer f.Close()
	if err := pieChart.Render(chart.PNG, f); err != nil {
		fmt.Printf("无法保存饼状图: %v\n", err)
	}
}
