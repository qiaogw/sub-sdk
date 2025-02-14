package pathx

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetGitHome(t *testing.T) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return
	}
	actual, err := GetGitHome()
	if err != nil {
		return
	}

	expected := filepath.Join(homeDir, goctlDir, gitDir)
	assert.Equal(t, expected, actual)
}

func TestGetGoctlHome(t *testing.T) {
	t.Run("goctl_is_file", func(t *testing.T) {
		tmpFile := filepath.Join(t.TempDir(), "a.tmp")
		backupTempFile := tmpFile + ".old"
		err := os.WriteFile(tmpFile, nil, 0o666)
		if err != nil {
			return
		}
		RegisterGoctlHome(tmpFile)
		home, err := GetGoctlHome()
		if err != nil {
			return
		}
		info, err := os.Stat(home)
		assert.Nil(t, err)
		assert.True(t, info.IsDir())

		_, err = os.Stat(backupTempFile)
		assert.Nil(t, err)
	})

	t.Run("goctl_is_dir", func(t *testing.T) {
		RegisterGoctlHome("")
		dir := t.TempDir()
		RegisterGoctlHome(dir)
		home, err := GetGoctlHome()
		assert.Nil(t, err)
		assert.Equal(t, dir, home)
	})
}
func TestRenameFilesWithPrefixAndSuffix(t *testing.T) {

	wd, err := os.Getwd()
	if err != nil {
		assert.NoError(t, err)
	}
	parentDir := filepath.Dir(wd)
	fmt.Println("Parent directory:", parentDir)
	homeDir := filepath.Join(parentDir, "templatex", "template2", "admin", "logic")

	prefix := "admin"
	suffix := ""
	err = RenameFilesWithPrefixAndSuffix(homeDir, prefix, suffix)
	if err != nil {
		assert.NoError(t, err)
	}

}
