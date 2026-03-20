package cmd

import (
	"encoding/json"
	"fmt"
	"os"
)

func printJSON(v any) {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	enc.Encode(v)
}

func printJSONError(msg string) {
	fmt.Fprintf(os.Stderr, `{"error":%q}`+"\n", msg)
}
