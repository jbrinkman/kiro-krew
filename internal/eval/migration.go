package eval

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// migrateExistingDirectories renames old format directories to new timestamped format
func migrateExistingDirectories() error {
	resultsDir := filepath.Join(".kiro-krew", "evals", "results")
	
	entries, err := os.ReadDir(resultsDir)
	if err != nil {
		// Results directory doesn't exist yet, nothing to migrate
		return nil
	}
	
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		
		dirName := entry.Name()
		hash := parseDirectoryName(dirName)
		
		// Skip if already in new format or not a valid hash
		if hash == "" || hash != dirName {
			continue
		}
		
		// Get commit timestamp and convert to our format
		timestamp, err := getCommitTimestamp(hash)
		if err != nil {
			// Skip directories where we can't get commit timestamp
			fmt.Printf("Warning: skipping migration of %s: %v\n", dirName, err)
			continue
		}
		
		// Convert timestamp to our format
		t := time.Unix(timestamp, 0).UTC()
		timestampPrefix := t.Format("010215-150405")
		
		oldPath := filepath.Join(resultsDir, dirName)
		newDirName := timestampPrefix + "-" + hash
		newPath := filepath.Join(resultsDir, newDirName)
		
		// Rename directory
		if err := os.Rename(oldPath, newPath); err != nil {
			fmt.Printf("Warning: failed to migrate %s: %v\n", dirName, err)
			continue
		}
		
		fmt.Printf("Migrated %s → %s\n", dirName, newDirName)
	}
	
	return nil
}