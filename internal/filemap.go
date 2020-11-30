package internal

import (
	"path"
	"path/filepath"
)

type FileMap struct {
	includes *baseAndAbs
	excludes *baseAndAbs
}

func NewFileMap(root string, includes []string, excludes []string) *FileMap {
	return &FileMap{
		includes: newBaseAndAbs(root, includes),
		excludes: newBaseAndAbs(root, excludes),
	}
}

func (x *FileMap) ToInclude(p string) bool {
	base := path.Base(p)
	abs, err := filepath.Abs(p)
	switch {
	case err != nil:
		return false
	case x.includes.abses.match(abs):
		return true
	case x.excludes.abses.match(abs):
		return false
	default:
		return x.includes.bases.match(base) && !x.excludes.bases.match(base)
	}
}

func (x *FileMap) ExplicitlyExcluded(p string) bool {
	abs, err := filepath.Abs(p)
	return err != nil || x.excludes.abses.match(abs)
}

type pathsMatcher []string

func (x pathsMatcher) match(path string) bool {
	for _, y := range x {
		if m, _ := filepath.Match(y, path); m {
			return true
		}
	}
	return false
}

type baseAndAbs struct {
	bases pathsMatcher
	abses pathsMatcher
}

func newBaseAndAbs(root string, l []string) *baseAndAbs {
	r := &baseAndAbs{
		bases: make(pathsMatcher, 0),
		abses: make(pathsMatcher, 0),
	}
	for _, x := range l {
		if filepath.Base(x) == x {
			r.bases = append(r.bases, x)
		} else {
			if !path.IsAbs(x) {
				x = path.Join(root, x)
			}
			r.abses = append(r.abses, path.Clean(x))
		}
	}

	return r
}
