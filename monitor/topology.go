package monitor

import "fmt"

func taskRanks(snapshot Snapshot) (map[string]int, error) {
	degree := map[string]int{}
	order := []string{}
	for _, task := range snapshot.Tasks {
		if _, exists := degree[task.TaskID]; !exists {
			degree[task.TaskID] = 0
			order = append(order, task.TaskID)
		}
	}
	next := map[string][]string{}
	seen := map[Edge]bool{}
	for _, edge := range snapshot.Edges {
		if _, ok := degree[edge.From]; !ok {
			return nil, fmt.Errorf("DAG references missing task %q", edge.From)
		}
		if _, ok := degree[edge.To]; !ok {
			return nil, fmt.Errorf("DAG references missing task %q", edge.To)
		}
		if !seen[edge] {
			next[edge.From] = append(next[edge.From], edge.To)
			degree[edge.To]++
			seen[edge] = true
		}
	}
	queue := []string{}
	for _, id := range order {
		if degree[id] == 0 {
			queue = append(queue, id)
		}
	}
	rank := map[string]int{}
	processed := 0
	for head := 0; head < len(queue); head++ {
		id := queue[head]
		processed++
		for _, target := range next[id] {
			rank[target] = max(rank[target], rank[id]+1)
			degree[target]--
			if degree[target] == 0 {
				queue = append(queue, target)
			}
		}
	}
	if processed != len(degree) {
		return nil, fmt.Errorf("monitor DAG contains a cycle")
	}
	return rank, nil
}
