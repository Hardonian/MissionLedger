package main

import (
	"encoding/json"
	"fmt"
)

func main() {
	payload := map[string]interface{}{
		"tenant_id":       "demo-tenant",
		"objective":       "Investigate a production error without leaking secrets or overspending budget",
		"requested_tools": []string{"read_file", "terminal"},
		"budget_usd":      1.25,
		"created_by":      "founder-demo",
	}

	bytes, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		panic(err)
	}

	fmt.Println(string(bytes))
}
