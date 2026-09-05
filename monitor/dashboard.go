package monitor

import (
	"fmt"
	"slices"
	"sort"
	"strings"

	"github.com/HahyeonJeon/gobble"
)

// Dashboard is an indexed presentation of a snapshot. Indices in Stages and
// Attention address Snapshot.Tasks. Rebuild it once per refresh, not per frame.
type Dashboard struct {
	Snapshot  Snapshot
	Total     Counts
	Shared    Counts
	Stages    []Stage
	Edges     []Edge
	Samples   []Sample
	Attention []int
	byTask    map[string]int
	bySample  map[string][]int
}

type Stage struct {
	ID, Name, Scope string
	Rank            int
	Tasks           []int
	Counts          Counts
}

type Sample struct {
	ID     string
	Counts Counts
}

// Build aggregates a control snapshot. Grouping includes topological rank so
// contracting repeated commands cannot introduce a cycle or hide a dependency.
// Repeated instances of an authored task keep their common graph position.
func Build(snapshot Snapshot) (*Dashboard, error) {
	d := &Dashboard{Snapshot: snapshot, byTask: map[string]int{}, bySample: map[string][]int{}}
	ranks, err := taskRanks(snapshot)
	if err != nil {
		return nil, err
	}
	groups := map[string]int{}
	stageForTask := map[string]string{}
	for i, task := range snapshot.Tasks {
		if _, exists := d.byTask[task.Identity]; exists {
			return nil, fmt.Errorf("duplicate monitor identity %q", task.Identity)
		}
		d.byTask[task.Identity] = i
		d.Total.Add(task)
		owners := uniqueSamples(task.Display.Samples)
		if len(owners) == 0 {
			d.Shared.Add(task)
		}
		for _, owner := range owners {
			d.bySample[owner] = append(d.bySample[owner], i)
		}
		if taskAttention(task) {
			d.Attention = append(d.Attention, i)
		}
		name := task.Display.Stage
		if name == "" {
			name = task.Name
		}
		if name == "" {
			name = task.TaskID
		}
		scope := task.Display.Scope
		if scope == "" && len(owners) > 0 {
			scope = gobble.DisplaySample
		}
		rank := ranks[task.TaskID]
		id := fmt.Sprintf("%d/%d:%s/%d:%s", rank, len(scope), scope, len(name), name)
		pos, exists := groups[id]
		if !exists {
			pos = len(d.Stages)
			groups[id] = pos
			d.Stages = append(d.Stages, Stage{ID: id, Name: name, Scope: scope, Rank: rank})
		}
		stage := &d.Stages[pos]
		stage.Tasks = append(stage.Tasks, i)
		stage.Counts.Add(task)
		stageForTask[task.TaskID] = id
	}
	for owner, indices := range d.bySample {
		var count Counts
		for _, i := range indices {
			count.Add(snapshot.Tasks[i])
		}
		d.Samples = append(d.Samples, Sample{ID: owner, Counts: count})
	}
	sort.Slice(d.Samples, func(i, j int) bool { return d.Samples[i].ID < d.Samples[j].ID })
	sort.SliceStable(d.Stages, func(i, j int) bool { return d.Stages[i].Rank < d.Stages[j].Rank })
	sort.SliceStable(d.Attention, func(i, j int) bool {
		return attentionOrder(snapshot.Tasks[d.Attention[i]].Status) < attentionOrder(snapshot.Tasks[d.Attention[j]].Status)
	})
	seen := map[Edge]bool{}
	for _, edge := range snapshot.Edges {
		pair := Edge{From: stageForTask[edge.From], To: stageForTask[edge.To]}
		if pair.From != pair.To && !seen[pair] {
			d.Edges = append(d.Edges, pair)
			seen[pair] = true
		}
	}
	return d, nil
}

func uniqueSamples(samples []string) []string {
	result := []string{}
	for _, id := range samples {
		if id != "" && !slices.Contains(result, id) {
			result = append(result, id)
		}
	}
	return result
}

func attentionOrder(status string) int {
	switch status {
	case "failed":
		return 0
	case "unknown", "unknown-backend":
		return 1
	case "incomplete", "published-unfinalized":
		return 2
	default:
		return 3
	}
}

// SearchSamples searches labels; selecting a returned ID always uses exact
// membership. No task-name substring is treated as sample identity.
func (d *Dashboard) SearchSamples(query string) []Sample {
	query = strings.ToLower(strings.TrimSpace(query))
	result := []Sample{}
	for _, sample := range d.Samples {
		if strings.Contains(strings.ToLower(sample.ID), query) {
			result = append(result, sample)
		}
	}
	return result
}

func (d *Dashboard) SampleTasks(id string) []int {
	return append([]int(nil), d.bySample[id]...)
}

func (d *Dashboard) Task(identity string) (Task, bool) {
	i, exists := d.byTask[identity]
	if !exists {
		return Task{}, false
	}
	return d.Snapshot.Tasks[i], true
}

// StageTasks scopes sample-owned work but retains shared/unassigned stages as
// dependency context. Their counts never enter Sample.Counts.
func (d *Dashboard) StageTasks(stage Stage, sample string) []int {
	result := []int{}
	for _, i := range stage.Tasks {
		task := d.Snapshot.Tasks[i]
		if sample == "" || len(task.Display.Samples) == 0 || slices.Contains(task.Display.Samples, sample) {
			result = append(result, i)
		}
	}
	return result
}

func (d *Dashboard) Count(indices []int) Counts {
	var result Counts
	for _, i := range indices {
		result.Add(d.Snapshot.Tasks[i])
	}
	return result
}

func (d *Dashboard) Neighbors(stage string) (up, down []string) {
	names := make(map[string]string, len(d.Stages))
	for _, s := range d.Stages {
		names[s.ID] = s.Name
	}
	for _, edge := range d.Edges {
		if edge.To == stage {
			up = append(up, names[edge.From])
		}
		if edge.From == stage {
			down = append(down, names[edge.To])
		}
	}
	return up, down
}
