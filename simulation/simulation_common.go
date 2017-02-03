package main

import (
	"gopkg.in/dedis/onet.v1/app"
	"gopkg.in/dedis/onet.v1/log"

	"os"
	"path"
	"strings"
)

type stringSlice []string

// Len is part of sort.Interface.
func (d stringSlice) Len() int {
	return len(d)
}

// Swap is part of sort.Interface.
func (d stringSlice) Swap(i, j int) {
	d[i], d[j] = d[j], d[i]
}

// Less is part of sort.Interface. We use count as the value to sort by
func (d stringSlice) Less(i, j int) bool {
	return d[i] < d[j]
}

// CopyFiles copies the files from the service/swupdate-directory
// to the simulation-directory
func CopyFiles(dir, snapshots string, releases string) error {
	wd, err := os.Getwd()
	if err != nil {
		return err
	}
	log.Lvl2("We're in", wd)
	for _, file := range append(strings.Split(snapshots, " "),
		strings.Split(releases, " ")...) {
		dst := path.Join(dir, path.Base(file))
		if _, err := os.Stat(dst); err != nil {
			err := app.Copy(dst, "../services/debianupdate/script/"+file)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// CopyFiles copies the files from the service/swupdate-directory
// to the simulation-directory
func CopyDir(dir, snapshots string) error {
	wd, err := os.Getwd()
	if err != nil {
		return err
	}
	log.Lvl2("We're in", wd)

	releases, err := GetFileFromType(dir+"/../../script/"+snapshots,
		"Release")
	if err != nil {
		return err
	}
	packages, err := GetFileFromType(dir+"/../../script/"+snapshots,
		"Packages")
	if err != nil {
		return err
	}

	err = os.RemoveAll(dir + "/" + snapshots + "/")
	err = os.Mkdir(dir+"/"+snapshots+"/", 0777)
	if err != nil {
		return err
	}
	for _, file := range append(releases, packages...) {
		dst := path.Join(dir, snapshots, path.Base(file))
		if _, err := os.Stat(dst); err != nil {
			err := app.Copy(dst,
				dir+"/../../script/"+snapshots+"/"+file)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func GetFileFromType(dir string, filetype string) (stringSlice, error) {
	files := make([]string, 0)

	// get all the files inside the snapshots dir
	snapshots_dir, err := os.Open(dir)
	if err != nil {
		return nil, err
	}
	defer snapshots_dir.Close()

	fi, err := snapshots_dir.Stat()
	if err != nil {
		return nil, err
	}

	if fi.IsDir() {
		fis, err := snapshots_dir.Readdir(-1) // -1 == all files infos
		if err != nil {
			return nil, err
		}

		for _, fileinfos := range fis {
			if !fileinfos.IsDir() {
				name := fileinfos.Name()

				if strings.Contains(name, filetype) {
					files = append(files, fileinfos.Name())
				}
			}
		}
	}

	log.Lvl3("Files", files)
	return files, nil
}
