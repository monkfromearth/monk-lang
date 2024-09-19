package utils

import (
	"encoding/json"
	"fmt"
)

func JSONStringify(value interface{}) string {
	data, err := json.MarshalIndent(value, "", "  ")

	if err != nil {
		fmt.Printf("Error marshaling JSON: %v\n", err)
		return ""
	}

	return string(data)
}

func PrettyPrint(value interface{}) {
	data := JSONStringify(value)

	fmt.Println(data)
}
