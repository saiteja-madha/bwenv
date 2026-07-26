package output

import (
	"bytes"
	"reflect"
	"strings"
	"testing"

	"bwenv/internal/bws"
	"bwenv/internal/environment"

	"github.com/joho/godotenv"
)

func outputFixture() []environment.Entry {
	return []environment.Entry{
		{Secret: bws.Secret{ID: "1", Key: "EMPTY", Value: ""}, Source: "app"},
		{Secret: bws.Secret{ID: "2", Key: "TOKEN", Value: "a=b\nline"}, Source: "shared"},
	}
}

func TestRenderAllFormats(t *testing.T) {
	for _, format := range []string{"json", "yaml", "env", "table", "tsv", "none"} {
		t.Run(format, func(t *testing.T) {
			var buffer bytes.Buffer
			if err := RenderEntries(&buffer, outputFixture(), format, false, "no"); err != nil {
				t.Fatal(err)
			}
			if format != "none" && buffer.Len() == 0 {
				t.Fatalf("%s output is empty", format)
			}
		})
	}
}

func TestEnvOutputQuotesLosslessly(t *testing.T) {
	var buffer bytes.Buffer
	if err := RenderEntries(&buffer, outputFixture(), "env", false, "no"); err != nil {
		t.Fatal(err)
	}
	want := "EMPTY=\"\"\nTOKEN=\"a=b\\nline\"\n"
	if buffer.String() != want {
		t.Fatalf("env output = %q, want %q", buffer.String(), want)
	}
}

func TestEnvOutputRoundTripsSpecialValues(t *testing.T) {
	want := map[string]string{
		"DOLLAR":    "$HOME ${TOKEN}",
		"EMPTY":     "",
		"MULTILINE": "first\r\nsecond",
		"PUNCT":     "quote=\" slash=\\ bang=! tick=`",
		"UNICODE":   "héllo 日本語",
	}
	entries := make([]environment.Entry, 0, len(want))
	for key, value := range want {
		entries = append(entries, environment.Entry{
			Secret: bws.Secret{Key: key, Value: value},
			Source: "app",
		})
	}
	var buffer bytes.Buffer
	if err := RenderEntries(&buffer, entries, "env", false, "no"); err != nil {
		t.Fatal(err)
	}
	got, err := godotenv.Unmarshal(buffer.String())
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("dotenv round trip = %#v, want %#v; output: %q", got, want, buffer.String())
	}
}

func TestJSONUsesNormalizedKeyAndSource(t *testing.T) {
	var buffer bytes.Buffer
	if err := RenderEntries(&buffer, outputFixture()[:1], "json", true, "no"); err != nil {
		t.Fatal(err)
	}
	for _, fragment := range []string{`"key": "EMPTY"`, `"source": "app"`} {
		if !strings.Contains(buffer.String(), fragment) {
			t.Fatalf("JSON missing %s: %s", fragment, buffer.String())
		}
	}
}

func TestColorYesAddsANSI(t *testing.T) {
	var buffer bytes.Buffer
	if err := RenderEntries(&buffer, outputFixture()[:1], "env", false, "yes"); err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(buffer.String(), "\x1b[36m") || !strings.HasSuffix(buffer.String(), "\x1b[0m") {
		t.Fatalf("color output missing ANSI wrapper: %q", buffer.String())
	}
}
