package cli

import "fmt"

var (
	src string
)

func InitGo() {
	src = fmt.Sprintf(`
package main

import (
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

var (
	platform = "%s"
	temp = "%s"
)

func main() {
	var (
		err error = nil
	)

	for {
		force := err != nil
		err = run(force)
		if force && err != nil {
			log.Fatal(err)
		}
	}
}

func run(force bool) error {
	name := FileNameWithoutExtSliceNotation(filepath.Base("%s"))
	dir := TempDir()

	jar := filepath.Join(dir, temp, name+".jar")
	if force || !DirExists(filepath.Join(dir, temp)) {
		ReadDir(temp+"/jre", dir)
		d, err := Asset(temp+"/"+name+".jar")
		if err != nil {
			return err
		}

		ioutil.WriteFile(jar, d, 0777)
	}

	args := []string{"-jar", jar}
	if len(os.Args) > 1 {
		args = append(args, os.Args[1:]...)
	}

	java := "java"
	if runtime.GOOS == "windows" {
		java += ".exe"
	}

 	c := exec.Command(filepath.Join(dir, temp, "jre", "bin", java), args...)
	c.Stdout = os.Stdout
	err := c.Run()
	if err != nil {
		return err
	}

	return nil
}

func ReadDir(name, dir string) {
	os.MkdirAll(filepath.Join(dir, name), 0777)
	files, err := AssetDir(name)
	if err != nil {
		log.Fatal(err)
	}

	for _, fName := range files {
		fName = name + "/" + fName
		info, err := AssetInfo(fName)
		if err != nil || info.IsDir() {
			ReadDir(fName, dir)
			continue
		}

		d, err := Asset(fName)
		if err != nil {
			log.Fatal(err)
		}
		ioutil.WriteFile(filepath.Join(dir, fName), d, 0777)
	}
}

func FileNameWithoutExtSliceNotation(fileName string) string {
	return fileName[:len(fileName)-len(filepath.Ext(fileName))]
}


func DirExists(dir string) bool {
	fs, err := os.Stat(dir)
	return err == nil && fs.IsDir()
}

func UserHomeDir() string {
	if runtime.GOOS == "windows" {
		home := os.Getenv("HOMEDRIVE") + os.Getenv("HOMEPATH")
		if home == "" {
			home = os.Getenv("USERPROFILE")
		}
		return home
	}
	return os.Getenv("HOME")
}

func TempDir() string {
	var (
		home = UserHomeDir()
		dir  = ""
	)
	if runtime.GOOS == "windows" {
		dir = home + "\\AppData\\Local\\Temp\\Robomotion"
	} else {
		dir = "/tmp/robomotion"
	}
	return dir
}

`, platform, temp, jar)
}
