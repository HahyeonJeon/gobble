package engine

type resourceBudget struct {
	host hostCapacity
	cpu  float64
	mem  int64
}

func newBudget(host hostCapacity) resourceBudget {
	return resourceBudget{host: host, cpu: host.CPU, mem: host.Memory}
}

func (b resourceBudget) fits(t TaskPlan) bool {
	if b.host.CPUKnown && t.Resources.CPU > 0 && t.Resources.CPU > b.cpu {
		return false
	}
	n, ok := parseMemory(t.Resources.Memory)
	if ok && b.host.MemKnown && n > 0 && n > b.mem {
		return false
	}
	return true
}

func (b *resourceBudget) occupy(t TaskPlan) {
	if t.Resources.CPU > 0 {
		b.cpu -= t.Resources.CPU
	}
	if n, ok := parseMemory(t.Resources.Memory); ok && n > 0 {
		b.mem -= n
	}
}

func (b *resourceBudget) release(t TaskPlan) {
	if t.Resources.CPU > 0 {
		b.cpu += t.Resources.CPU
	}
	if n, ok := parseMemory(t.Resources.Memory); ok && n > 0 {
		b.mem += n
	}
}
