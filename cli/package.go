package cli

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/mholt/archiver"
	"github.com/spf13/cobra"
	"github.com/tidwall/sjson"
)

var (
	jar      string
	out      string
	jdk      string
	platform string
	arch     string
	clean    bool
	temp     = "jutil"

	packageCmd = &cobra.Command{
		Use:   "package",
		Short: "Package commands",
		Long:  `package is used packaging jar files`,
		Run: func(cmd *cobra.Command, args []string) {
			if jar == "" || out == "" || jdk == "" {
				cmd.Usage()
				return
			}

			if clean {
				os.RemoveAll(out)
				os.RemoveAll(temp)
			}

			os.Mkdir(filepath.Join(temp), 0777)
			err := os.MkdirAll(out, 0777)
			if err != nil {
				log.Fatal("out dir err:", err)
			}

			inJar, err := os.Open(jar)
			if err != nil {
				log.Fatal("open jar err:", err)
			}
			defer inJar.Close()

			tempDir, err := filepath.Abs(temp)
			if err != nil {
				log.Fatal("abs tempdir err:", err)
			}

			outJar, err := os.Create(filepath.Join(tempDir, filepath.Base(jar)))
			if err != nil {
				log.Fatal("creating out jar failed:", err)
			}
			defer outJar.Close()

			_, err = io.Copy(outJar, inJar)
			if err != nil {
				log.Fatal("copy jar failed:", err)
			}

			err = archiver.Unarchive(jdk, tempDir)
			if err != nil {
				log.Fatal("unarchive failed:", err)
			}

			config := "{}"
			lookup := filepath.Join("bin", "java")
			if runtime.GOOS == "windows" {
				lookup += ".exe"
			}

			filepath.Walk(tempDir, func(path string, info fs.FileInfo, err error) error {
				if strings.HasSuffix(path, lookup) {
					p, _ := filepath.Rel(tempDir, filepath.Dir(path))
					config, _ = sjson.Set(config, "jre", p)
					return errors.New("")
				}
				return nil
			})

			err = ioutil.WriteFile(filepath.Join(tempDir, "jutil.json"), []byte(config), 0777)
			if err != nil {
				log.Fatal("creating config file failed", err)
			}

			bindataCmd := exec.Command("go", "install", "github.com/go-bindata/go-bindata/go-bindata@latest")
			bindataCmd.Stderr = os.Stderr
			err = bindataCmd.Run()
			if err != nil {
				log.Fatal("get go-bindata failed:", err)
			}

			gbCmd := exec.Command("go-bindata", "-o", filepath.Join(tempDir, "bindata.go"), fmt.Sprintf("%s/...", temp))
			gbCmd.Stderr = os.Stderr
			err = gbCmd.Run()
			if err != nil {
				log.Fatal("go-bindata failed:", err)
			}

			name := FileNameWithoutExtSliceNotation(filepath.Base(jar))

			maingo, err := os.Create(filepath.Join(tempDir, "main.go"))
			if err != nil {
				log.Fatal("creating main.go failed:", err)
			}
			defer maingo.Close()

			gomodCmd := exec.Command("go", "mod", "init", name)
			gomodCmd.Dir = tempDir
			gomodCmd.Stderr = os.Stderr
			err = gomodCmd.Run()
			if err != nil {
				log.Fatal("go mod init failed:", err)
			}

			gomodTidy := exec.Command("go", "mod", "tidy")
			gomodTidy.Dir = tempDir
			gomodTidy.Stderr = os.Stderr
			err = gomodTidy.Run()
			if err != nil {
				log.Fatal("go mod tidy failed:", err)
			}

			InitGo()

			_, err = io.Copy(maingo, bytes.NewBuffer([]byte(src)))
			if err != nil {
				log.Fatal("copy main.go failed:", err)
			}

			outName := name
			if platform == "windows" {
				outName += ".exe"
			}

			goCmd := exec.Command("go", "build", "-o", filepath.Join(tempDir, "..", out, outName))
			goCmd.Dir = tempDir
			goCmd.Stderr = os.Stderr
			goCmd.Env = os.Environ()
			goCmd.Env = append(goCmd.Env, fmt.Sprintf("GOOS=%s", platform))
			goCmd.Env = append(goCmd.Env, fmt.Sprintf("GOARCH=%s", arch))
			//goCmd.Env = append(goCmd.Env, fmt.Sprintf("GOPATH=%s", os.Getenv("GOPATH")))
			goCmd.Stderr = os.Stderr

			err = goCmd.Run()
			if err != nil {
				log.Fatal("go build failed:", err)
			}

			os.RemoveAll(temp)
		},
	}
)

func init() {

	packageCmd.Flags().StringVar(&jar, "jar", "", "JAR file")
	packageCmd.Flags().StringVar(&out, "out", ".", "Output directory")
	packageCmd.Flags().StringVar(&jdk, "jdk", "", "JDK path")
	packageCmd.Flags().StringVar(&platform, "platform", runtime.GOOS, "Operating system")
	packageCmd.Flags().StringVar(&arch, "arch", runtime.GOARCH, "Operating system architecture")
	packageCmd.Flags().BoolVar(&clean, "clean", false, "Clean packaging")

	rootCmd.AddCommand(packageCmd)

}

func FileNameWithoutExtSliceNotation(fileName string) string {
	return fileName[:len(fileName)-len(filepath.Ext(fileName))]
}
