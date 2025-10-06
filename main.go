package main

import (
	"fmt"
)

func main() {
	o := NewOrchestrator()
	if o == nil {
		fmt.Println("Failed to initialize orchestrator")
		return
	}
	fmt.Println("Orchestrator initialized successfully")
}
