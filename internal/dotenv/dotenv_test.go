package dotenv

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadFillsEmptyButKeepsSetValues(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".env")
	if err := os.WriteFile(path, []byte("GOFACT_T_EMPTY=from-file\nGOFACT_T_SET=from-file\nGOFACT_T_UNSET=from-file\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GOFACT_T_EMPTY", "")
	t.Setenv("GOFACT_T_SET", "from-env")
	t.Setenv("GOFACT_T_UNSET", "")
	_ = os.Unsetenv("GOFACT_T_UNSET")

	if err := Load(path); err != nil {
		t.Fatal(err)
	}
	for key, want := range map[string]string{
		"GOFACT_T_EMPTY": "from-file",
		"GOFACT_T_SET":   "from-env",
		"GOFACT_T_UNSET": "from-file",
	} {
		if got := os.Getenv(key); got != want {
			t.Errorf("%s = %q, want %q", key, got, want)
		}
	}
}
