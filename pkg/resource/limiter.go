package resource

type LimitedResource interface {
	Acquire(tenant string) bool
	SetInitialCpuTime(cpuTime int)
}

type Permit interface {
	Release()
}
