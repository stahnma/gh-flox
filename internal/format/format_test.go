package format

import (
	"bytes"
	"strings"
	"testing"
)

func TestWriteJSON_Normal(t *testing.T) {
	var buf bytes.Buffer
	data := map[string]int{"stars": 42}

	if err := WriteJSON(&buf, data, false); err != nil {
		t.Fatal(err)
	}

	out := buf.String()
	if !strings.Contains(out, `"stars": 42`) {
		t.Errorf("expected JSON with stars, got:\n%s", out)
	}
	if strings.Contains(out, "```") {
		t.Error("normal mode should not have backticks")
	}
}

func TestWriteJSON_SlackMode(t *testing.T) {
	var buf bytes.Buffer
	data := map[string]int{"stars": 42}

	if err := WriteJSON(&buf, data, true); err != nil {
		t.Fatal(err)
	}

	out := buf.String()
	if !strings.HasPrefix(out, "```\n") {
		t.Errorf("slack mode should start with ```, got:\n%s", out)
	}
	if !strings.HasSuffix(out, "```\n") {
		t.Errorf("slack mode should end with ```, got:\n%s", out)
	}
	if !strings.Contains(out, `"stars": 42`) {
		t.Errorf("expected JSON content, got:\n%s", out)
	}
}

func TestWriteJSON_Struct(t *testing.T) {
	var buf bytes.Buffer
	type item struct {
		Name string `json:"name"`
	}
	data := []item{{Name: "test"}}

	if err := WriteJSON(&buf, data, false); err != nil {
		t.Fatal(err)
	}

	out := buf.String()
	if !strings.Contains(out, `"name": "test"`) {
		t.Errorf("expected struct JSON, got:\n%s", out)
	}
}
