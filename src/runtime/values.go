package runtime

const (
	NoneValue = iota
	NumberValue
	BooleanValue
)

var ValueNames = map[int]string{
	NoneValue:    "None",
	NumberValue:  "Number",
	BooleanValue: "Boolean",
}

type RuntimeValue struct {
	Type     int         `json:"type"`
	Name     string      `json:"name"`
	Value    interface{} `json:"value"`
	Constant bool
}
