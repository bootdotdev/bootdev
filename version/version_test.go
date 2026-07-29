package version

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

func TestGetLatestVersionRespectsGoProxyOff(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("test uses a POSIX executable script")
	}

	dir := t.TempDir()
	fakeGo := filepath.Join(dir, "go")
	script := `#!/bin/sh
if [ "$GOPROXY" != "off" ]; then
	echo "unexpected GOPROXY: $GOPROXY" >&2
	exit 1
fi
printf '{"Version":"v1.2.3"}'
`
	if err := os.WriteFile(fakeGo, []byte(script), 0o755); err != nil {
		t.Fatalf("create fake go command: %v", err)
	}
	t.Setenv("PATH", dir)
	t.Setenv("GOPROXY", "off")
	t.Setenv("GOPRIVATE", "")
	t.Setenv("GONOPROXY", "none")

	latest, err := getLatestVersionWithTimeout(time.Second)
	if err != nil {
		t.Fatalf("get latest version: %v", err)
	}
	if latest != "v1.2.3" {
		t.Fatalf("latest version = %q, want v1.2.3", latest)
	}
}

func TestGetLatestVersionHasOverallTimeout(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("test uses a POSIX executable script")
	}

	dir := t.TempDir()
	fakeGo := filepath.Join(dir, "go")
	if err := os.WriteFile(fakeGo, []byte("#!/bin/sh\nexec /bin/sleep 5\n"), 0o755); err != nil {
		t.Fatalf("create fake go command: %v", err)
	}
	t.Setenv("PATH", dir)

	start := time.Now()
	_, err := getLatestVersionWithTimeout(25 * time.Millisecond)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("error = %v, want context deadline exceeded", err)
	}
	if elapsed := time.Since(start); elapsed > time.Second {
		t.Fatalf("version lookup exceeded timeout allowance: %s", elapsed)
	}
}
