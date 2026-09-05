package monitor

// Counts partitions known executable instances. Templates are separate;
// Reused is a subset of Succeeded, not another task state.
type Counts struct {
	Total, Succeeded, Running, Failed, Pending, Blocked int
	Skipped, Incomplete, Unknown, Unfinalized, Reused   int
	Templates, Unexpanded                               int
}

func (c *Counts) Add(t Task) {
	if t.Template {
		c.Templates++
		if !t.Expanded {
			c.Unexpanded++
		}
		return
	}
	c.Total++
	switch t.Status {
	case "succeeded":
		c.Succeeded++
		if t.Decision == "reuse" || t.Decision == "reused" {
			c.Reused++
		}
	case "running":
		c.Running++
	case "failed":
		c.Failed++
	case "not-started":
		c.Pending++
	case "blocked":
		c.Blocked++
	case "skipped":
		c.Skipped++
	case "incomplete":
		c.Incomplete++
	case "published-unfinalized":
		c.Unfinalized++
	default:
		c.Unknown++
	}
}

func (c Counts) Attention() int {
	return c.Failed + c.Blocked + c.Incomplete + c.Unknown + c.Unfinalized
}

func (c Counts) Successful() bool {
	return c.Total > 0 && c.Succeeded == c.Total && c.Unexpanded == 0
}

func (c Counts) Percent() float64 {
	if c.Total == 0 {
		return 0
	}
	return 100 * float64(c.Succeeded) / float64(c.Total)
}

func taskAttention(t Task) bool {
	if t.Template {
		return false
	}
	switch t.Status {
	case "succeeded", "running", "not-started", "skipped":
		return false
	default:
		return true
	}
}
