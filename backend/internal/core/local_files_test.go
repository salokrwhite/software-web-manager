package core

import (
	"testing"
)

func TestNormalizeLocalFilePath(t *testing.T) {
	t.Parallel()

	got := normalizeLocalFilePath(`app\release\win\setup.exe`)
	want := "app/release/win/setup.exe"
	if got != want {
		t.Fatalf("normalizeLocalFilePath() = %q, want %q", got, want)
	}
}

func TestNormalizeLocalFilePathRejectsTraversal(t *testing.T) {
	t.Parallel()

	if got := normalizeLocalFilePath(`app\release\..\release\win\setup.exe`); got != "" {
		t.Fatalf("normalizeLocalFilePath() = %q, want empty for traversal", got)
	}
}

func TestResolveLocalStoragePathRejectsTraversal(t *testing.T) {
	t.Parallel()

	if _, err := resolveLocalStoragePath(t.TempDir(), `..\..\windows\system32\cmd.exe`); err == nil {
		t.Fatal("resolveLocalStoragePath() expected traversal rejection")
	}
}

func TestSanitizeUploadedFilename(t *testing.T) {
	t.Parallel()

	got := SanitizeUploadedFilename(`..\..\payload\installer.exe`)
	if got != "installer.exe" {
		t.Fatalf("SanitizeUploadedFilename() = %q, want installer.exe", got)
	}
}
