package cli

import "fmt"

var (
	src string
)

func InitGo() {
	src = fmt.Sprintf(`
package main

import (
	"encoding/json"
	"fmt"
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

	if force || !DirExists(filepath.Join(dir, temp)) {
		err := ReadDir(temp, dir)
		if err != nil {
			return err
		}
	}

	d, err := Asset(temp+"/jutil.json")
	if err != nil {
		return err
	}

	cfg := map[string]interface{}{}
	err = json.Unmarshal(d, &cfg)
	if err != nil {
		return err
	}

	jre, ok := cfg["jre"].(string)
	if !ok {
		return fmt.Errorf("Invalid jutil config file: jre not found")
	}

	jar := filepath.Join(dir, temp, name+".jar")
	args := []string{"-jar", jar}
	if len(os.Args) > 1 {
		args = append(args, os.Args[1:]...)
	}

	java := "java"
	if runtime.GOOS == "windows" {
		java += ".exe"
	}

 	c := exec.Command(filepath.Join(dir, temp, jre, java), args...)
	c.Stdout = os.Stdout
	err = c.Run()
	if err != nil {
		return err
	}

	return nil
}

func ReadDir(name, dir string) error {
	os.MkdirAll(filepath.Join(dir, name), 0777)
	files, err := AssetDir(name)
	if err != nil {
		return err
	}

	for _, fName := range files {
		fName = name + "/" + fName
		info, err := AssetInfo(fName)
		if err != nil || info.IsDir() {
			err = ReadDir(fName, dir)
			if err != nil {
				return err
			}
			continue
		}

		d, err := Asset(fName)
		if err != nil {
			return err
		}

		err = ioutil.WriteFile(filepath.Join(dir, fName), d, 0777)
		if err != nil {
			return err
		}
	}

	return nil
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
