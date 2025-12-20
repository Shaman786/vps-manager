package plans

type VMPlan struct {
	Name string
	RAM  int
	CPUs int
	Disk string
}

var Available = []VMPlan{
	{"Starter", 2048, 1, "10G"},
	{"Professional", 4096, 2, "20G"},
	{"Production", 8192, 4, "40G"},
	{"Beast", 16384, 8, "80G"},
}
