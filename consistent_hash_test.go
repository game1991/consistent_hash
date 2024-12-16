package consistent_hash

import (
	"fmt"
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
	// 如果开头是node1，返回0
	// 如果开头是node2，返回100
	// 如果开头是node3，返回200
	// 如果开头是node4，返回300
	// 定义节点前缀到哈希值的映射(由于key在经过hashKey之后会出现空byte占位，所以使用特殊匹配来进行赋值)
	trimmedKey := strings.TrimRight(string(key), "\x00")
	// 装载node时摘取-之前的字符串
	if strings.HasPrefix(trimmedKey, "node") { // 如果是node开头，返回0
		split := strings.SplitN(trimmedKey, "-", 2)
		trimmedKey = split[0]
	}

	if hash, ok := h.hashValues[trimmedKey]; ok {
		return hash
	}

	// 默认返回0
	return 0
}

func (h *TestHasher) SetHash(key string, hash uint32) {
	h.hashValues[key] = hash
}

func Test_ConsistentHash(t *testing.T) {
	// 创建自定义哈希函数
	hasher := NewTestHasher()

	// 创建一致性哈希实例
	ch := New(&Config{
		Replicas: 1, // 为了简化测试，每个节点只有一个虚拟节点
		HashFunc: hasher,
	})

	// 设置节点的哈希值，使其在哈希环上均匀分布
	// node1: 0
	// node2: 100
	// node3: 200
	// node4: 300
	nodes := []string{"node1", "node2", "node3", "node4"}
	nodeHashes := map[string]uint32{
		"node1": 0,
		"node2": 100,
		"node3": 200,
		"node4": 300,
	}
	for node, hash := range nodeHashes {
		hasher.SetHash(node, hash)
	}

	// 添加节点
	ch.Add(nodes...)

	// 生成测试数据并设置其哈希值
	testData := make([]string, 0)
	testDataHashes := make(map[string]uint32)

	// 创建测试数据，使其分布在不同的区间
	for i := 0; i < 400; i += 20 {
		key := fmt.Sprintf("key-%d", i)
		testData = append(testData, key)
		testDataHashes[key] = uint32(i)
		hasher.SetHash(key, uint32(i))
	}

	// 记录初始分布
	initialDistribution := make(map[string]int)
	keyToNode := make(map[string]string)

	fmt.Println("\n初始数据分布:")
	for _, key := range testData {
		node := ch.Get(key)
		initialDistribution[node]++
		keyToNode[key] = node
		fmt.Printf("数据 %s (hash=%d) 被分配到节点 %s\n", key, testDataHashes[key], node)
	}
	// 创建并保存初始饼状图
	generatePieChart(initialDistribution, testData, "初始数据分布饼状图")

	// 添加新节点 node5，位于 node1 和 node2 之间
	newNode := "node5"
	hasher.SetHash("node5", 50)
	ch.Add(newNode)

	// 检查数据迁移
	fmt.Println("\n添加节点 node5 后的数据分布:")
	migrations := make(map[string]map[string]int)
	newDistribution := make(map[string]int)

	for _, key := range testData {
		newNode := ch.Get(key)
		oldNode := keyToNode[key]
		newDistribution[newNode]++

		if oldNode != newNode {
			if migrations[oldNode] == nil {
				migrations[oldNode] = make(map[string]int)
			}
			migrations[oldNode][newNode]++
			fmt.Printf("数据 %s 从 %s 迁移到 %s\n", key, oldNode, newNode)
		}
	}

	// 验证数据迁移是否符合一致性哈希的特性
	// 对于每个节点，检查其数据是否只迁移到了顺时针方向的下一个节点
	for oldNode, targetNodes := range migrations {
		if len(targetNodes) > 1 {
			t.Errorf("节点 %s 的数据迁移到了多个节点: %v", oldNode, targetNodes)
		}
		// 检查是否只迁移到了新节点
		for targetNode := range targetNodes {
			if targetNode != newNode {
				t.Errorf("数据从 %s 迁移到了错误的节点 %s，应该迁移到 %s", oldNode, targetNode, newNode)
			}
		}
	}
	// 打印新的数据分布
	for node, count := range newDistribution {
		fmt.Printf("节点 %s 的数据数量：%d\n", node, count)
	}
	// 打印当前的哈希环上的节点和数据分布
	for _, key := range testData {
		node := ch.Get(key)
		fmt.Printf("数据 %s (hash=%d) 被分配到节点 %s\n", key, testDataHashes[key], node)
		// 更新keyToNode
		keyToNode[key] = node
	}

	// 生成并保存添加节点后的饼状图
	generatePieChart(newDistribution, testData, "添加节点后数据分布饼状图")

	// 测试删除节点
	fmt.Println("\n删除节点 node2 的影响:")
	nodeToRemove := "node2"
	// 删除节点
	ch.Remove(nodeToRemove)
	newDistribution = make(map[string]int)
	// 验证数据迁移
	fmt.Println("\n删除节点 node2 后的数据分布:")
	migrations = make(map[string]map[string]int)

	for _, key := range testData {
		newNode := ch.Get(key)
		oldNode := keyToNode[key]
		newDistribution[newNode]++
		if oldNode != newNode {
			if migrations[oldNode] == nil {
				migrations[oldNode] = make(map[string]int)
			}
			migrations[oldNode][newNode]++
			fmt.Printf("数据 %s 从 %s 迁移到 %s\n", key, oldNode, newNode)
		}
	}

	// 验证数据迁移是否符合一致性哈希的特性
	// 对于每个节点，检查其数据是否只迁移到了顺时针方向的下一个节点
	nextNode := "node3"
	for oldNode, targetNodes := range migrations {
		if len(targetNodes) > 1 {
			t.Errorf("节点 %s 的数据迁移到了多个节点: %v", oldNode, targetNodes)
		}
		// 检查是否只迁移到了新节点
		for targetNode := range targetNodes {
			if targetNode != nextNode {
				t.Errorf("数据从 %s 迁移到了错误的节点 %s，应该迁移到 %s", oldNode, targetNode, nextNode)
			}
		}
	}
	// 生成并保存删除节点后的饼状图
	generatePieChart(newDistribution, testData, "删除节点后数据分布饼状图")
}

// 生成饼状图的函数
func generatePieChart(distribution map[string]int, testData []string, title string) {
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
