package document

import (
	"log"
	"os"
	"path/filepath"
	"strings"

	"regexp"

	"github.com/fsnotify/fsnotify"
)

var defaultExcludes = []string{

	// VCS dirs
	`(^|/)\.git/`,
	`(^|/)\.hg/`,
	`(^|/)\.svn/`,

	// Vim
	`~$`,
	`\.swp$`,

	// Emacs
	`\.#`,
	`(^|/)#.*#$`,

	// OS X
	`(^|/)\.DS_Store$`,

	// node
	`(^|/)\node_modules/`,
}

var defaultExcludeMatcher multiMatcher

func init() {
	for _, pattern := range defaultExcludes {
		m := newRegexMatcher(regexp.MustCompile(pattern), true)
		defaultExcludeMatcher = append(defaultExcludeMatcher, m)
	}
}

// --------------------------------------------------------------------

const chmodMask fsnotify.Op = ^fsnotify.Op(0) ^ fsnotify.Chmod

// watch recursively watches changes in root and reports the filenames to names.
// Exclude dirs with prefix in excludes
func watch(root string, watcher *fsnotify.Watcher, names chan<- string, excludeMatcher Matcher) {

	if err := filepath.Walk(root, walker(watcher, excludeMatcher)); err != nil {
		log.Printf("Error while walking path %s: %s", root, err)
	}

	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return
			}

			stat, err := os.Stat(event.Name)
			if err != nil {
				continue
			}

			path := normalize(event.Name, stat.IsDir())

			if event.Op&chmodMask == 0 {
				continue
			}

			if event.Op&fsnotify.Chmod == fsnotify.Chmod {
				continue
			}

			// the new folder created will be watch
			if event.Op&fsnotify.Create > 0 && stat.IsDir() {
				if err := filepath.Walk(path, walker(watcher, excludeMatcher)); err != nil {
					log.Printf("Error while walking path %s: %s", path, err)
				}
			}

			names <- path

			// TODO: Cannot currently remove fsnotify watches
		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			log.Fatalf("watch file error: %s", err.Error())
		}
	}
}

// filepath.WalkFunc
func walker(watcher *fsnotify.Watcher, matcher Matcher) filepath.WalkFunc {
	return func(path string, f os.FileInfo, err error) error {
		if err != nil || !f.IsDir() {
			return nil
		}

		path = normalize(path, f.IsDir())

		ignore := true
		if !matcher.ExcludePrefix(path) {
			ignore = false
		}

		if ignore {
			return filepath.SkipDir
		}

		log.Println("watch:", path)

		if err := watcher.Add(path); err != nil {
			log.Printf("Error while watching new path %s: %s\n", path, err)
		}
		return nil
	}
}

func normalize(path string, dir bool) string {
	path = strings.TrimPrefix(path, "./")
	if dir && !strings.HasSuffix(path, "/") {
		path = path + "/"
	}
	return path
}
