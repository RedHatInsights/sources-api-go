package fixtures

type OffsetLimit struct {
	Limit  int
	Offset int
}

var TestDataOffsetLimit = []OffsetLimit{
	{
		Limit:  10,
		Offset: 0,
	},
	{
		Limit:  10,
		Offset: 1,
	},
	{
		Limit:  10,
		Offset: 100,
	},
	{
		Limit:  1,
		Offset: 0,
	},
	{
		Limit:  1,
		Offset: 1,
	},
	{
		Limit:  1,
		Offset: 100,
	},
}
