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
		
	name := FileNameWithoutExtSliceNotation(filepath.Base("%s"))
	dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		log.Fatal(err)
	}

	jar := filepath.Join(dir, temp, name+".jar")
	if !DirExists(filepath.Join(dir, temp)) {
		ReadDir(temp+"/jre", dir)
		d, err := Asset(temp+"/"+name+".jar")
		if err != nil {
			log.Fatal(err)
		}

		ioutil.WriteFile(jar, d, 0777)
	}

	args := []string{"-jar", jar}
	if len(os.Args) > 1 {
		if os.Args[1] == "-s" {
			defer func() {
				os.RemoveAll(filepath.Join(dir, temp))
			}()
		}
		args = append(args, os.Args[1:]...)
	}

	java := "java"
	if runtime.GOOS == "windows" {
		java += ".exe"
	}

 	c := exec.Command(filepath.Join(dir, temp, "jre", "bin", java), args...)
	c.Stdout = os.Stdout
	err = c.Run()
	if err != nil {
		log.Fatal(err)
	}
	return
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
`, platform, temp, jar)
}
