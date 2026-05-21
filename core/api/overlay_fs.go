package api

import (
	"io/fs"
	"os"
)

// OverlayFS merges multiple fs.FS systems.
// Priority is given to filesystems in the order they are specified (index 0 is highest priority).
type OverlayFS struct {
	fss []fs.FS
}

// NewOverlayFS creates a new OverlayFS from the given filesystems.
// Nil filesystems are ignored.
func NewOverlayFS(fss ...fs.FS) *OverlayFS {
	var valid []fs.FS
	for _, f := range fss {
		if f != nil {
			valid = append(valid, f)
		}
	}
	return &OverlayFS{fss: valid}
}

// Open opens the named file.
func (o *OverlayFS) Open(name string) (fs.File, error) {
	var lastErr error
	for _, f := range o.fss {
		file, err := f.Open(name)
		if err == nil {
			return file, nil
		}
		// Continue search on file not found, but retain other errors
		if !os.IsNotExist(err) && err != fs.ErrNotExist {
			lastErr = err
		}
	}
	if lastErr != nil {
		return nil, lastErr
	}
	return nil, fs.ErrNotExist
}

// Stat returns the FileInfo describing the named file.
func (o *OverlayFS) Stat(name string) (fs.FileInfo, error) {
	var lastErr error
	for _, f := range o.fss {
		if statFS, ok := f.(fs.StatFS); ok {
			info, err := statFS.Stat(name)
			if err == nil {
				return info, nil
			}
			if !os.IsNotExist(err) && err != fs.ErrNotExist {
				lastErr = err
			}
		} else {
			// Fallback: open the file and stat it
			file, err := f.Open(name)
			if err == nil {
				info, err := file.Stat()
				file.Close()
				if err == nil {
					return info, nil
				}
				lastErr = err
			} else if !os.IsNotExist(err) && err != fs.ErrNotExist {
				lastErr = err
			}
		}
	}
	if lastErr != nil {
		return nil, lastErr
	}
	return nil, fs.ErrNotExist
}

// ReadDir reads the named directory and returns a list of directory entries sorted by presentation name.
func (o *OverlayFS) ReadDir(name string) ([]fs.DirEntry, error) {
	merged := make(map[string]fs.DirEntry)
	var hasDir bool
	var lastErr error

	for _, f := range o.fss {
		var entries []fs.DirEntry
		var err error
		if rdirFS, ok := f.(fs.ReadDirFS); ok {
			entries, err = rdirFS.ReadDir(name)
		} else {
			// Fallback: open and read directory if it implements ReadDir
			file, openErr := f.Open(name)
			if openErr == nil {
				if dir, ok := file.(fs.ReadDirFile); ok {
					entries, err = dir.ReadDir(-1)
				} else {
					err = &fs.PathError{Op: "readdir", Path: name, Err: fs.ErrInvalid}
				}
				file.Close()
			} else {
				err = openErr
			}
		}

		if err == nil {
			hasDir = true
			for _, entry := range entries {
				if _, exists := merged[entry.Name()]; !exists {
					merged[entry.Name()] = entry
				}
			}
		} else if !os.IsNotExist(err) && err != fs.ErrNotExist {
			lastErr = err
		}
	}

	if !hasDir {
		if lastErr != nil {
			return nil, lastErr
		}
		return nil, fs.ErrNotExist
	}

	res := make([]fs.DirEntry, 0, len(merged))
	for _, entry := range merged {
		res = append(res, entry)
	}
	return res, nil
}
