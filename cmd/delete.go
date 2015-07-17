package cmd

import (
	"errors"
	"github.com/Masterminds/cookoo"
	"os"
	"path/filepath"
	"strings"
)

// DeleteUnusedPackages removes packages from vendor/ that no longer used.
func DeleteUnusedPackages(c cookoo.Context, p *cookoo.Params) (interface{}, cookoo.Interrupt) {
	// Conditional opt-out to keep the unused dependencies.
	optOut := p.Get("optOut", false).(bool)
	if optOut == true {
		return nil, nil
	}

	vpath, err := VendorPath(c)
	if err != nil {
		return nil, err
	}
	if vpath == "" {
		return false, errors.New("Vendor not set")
	}

	// Build directory tree of what to keep.
	cfg := p.Get("conf", nil).(*Config)
	var pkgList []string
	for _, dep := range cfg.Imports {
		pkgList = append(pkgList, dep.Name)
	}

	// Callback function for filepath.Walk to delete packages not in yaml file.
	var searchPath string

	var markForDelete []string

	fn := func(path string, info os.FileInfo, err error) error {
		// Bubble up the error
		if err != nil {
			return err
		}

		if info.IsDir() == false || path == searchPath || path == vpath {
			return nil
		}

		// fmt.Println(path)
		// fmt.Println(info.Name())

		localPath := strings.TrimPrefix(path, searchPath)

		keep := false

		// First check if the path has a prefix that's a specific package. If
		// so we keep it to keep the package.
		for _, name := range pkgList {
			if strings.HasPrefix(localPath, name) {
				keep = true
			}
		}

		// If a package is, for example, github.com/Masterminds/glide the
		// previous look will not mark the directories github.com or
		// github.com/Masterminds to keep. Here we see if these names prefix
		// and packages we know about to mark as keepers.
		if keep == false {
			for _, name := range pkgList {
				if strings.HasPrefix(name, localPath) {
					keep = true
				}
			}
		}

		// Remove the directory if we are not keeping it.
		if keep == false {
			// Mark for deletion
			markForDelete = append(markForDelete, path)
		}

		return nil
	}

	// Walk src directories (only 1 level deep)
	searchPath = vpath
	err = filepath.Walk(searchPath, fn)
	if err != nil {
		return false, err
	}

	for _, path := range markForDelete {
		localPath := strings.TrimPrefix(path, searchPath)
		Info("Removing unused package: %s\n", localPath)
		rerr := os.RemoveAll(path)
		if rerr != nil {
			return false, rerr
		}
	}

	return nil, nil
}
