package format

import (
	"encoding/json"
	"fmt"
	"io"
)

// WriteJSON writes formatted JSON to w, optionally wrapped in a slack code block.
func WriteJSON(w io.Writer, v any, slackMode bool) error {
	output, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	if slackMode {
		fmt.Fprintln(w, "```")
	}
	fmt.Fprintln(w, string(output))
	if slackMode {
		fmt.Fprintln(w, "```")
	}
	return nil
}
