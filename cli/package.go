package cli

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/mholt/archiver"
	"github.com/spf13/cobra"
)

var (
	jar      string
	out      string
	jdk      string
	platform string
	arch     string
	clean    bool
	temp     = "temp"

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
				log.Fatal(err)
			}

			inJar, err := os.Open(jar)
			if err != nil {
				log.Fatal(err)
			}
			defer inJar.Close()

			tempDir, err := filepath.Abs(temp)
			if err != nil {
				log.Fatal(err)
			}

			outJar, err := os.Create(filepath.Join(tempDir, filepath.Base(jar)))
			if err != nil {
				log.Fatal(err)
			}
			defer outJar.Close()

			_, err = io.Copy(outJar, inJar)
			if err != nil {
				log.Fatal(err)
			}

			err = archiver.Unarchive(jdk, tempDir)
			if err != nil {
				log.Fatal(err)
			}

			/*
				if strings.HasSuffix(jdk, ".tar.gz") {
					jdkFile, err := os.Open(jdk)
					if err != nil {
						log.Fatal(err)
					}

					gr, err := gzip.NewReader(jdkFile)
					if err != nil {
						log.Fatal(err)
					}

					tr := tar.NewReader(gr)
					for {
						header, err := tr.Next()
						if err == io.EOF {
							break
						}

						if header.FileInfo().IsDir() {
							continue
						}

						outPath := filepath.Join(tempDir, header.Name)
						err = os.MkdirAll(filepath.Dir(outPath), 0777)
						if err != nil {
							log.Fatal(err)
						}

						outFile, err := os.Create(outPath)
						if err != nil {
							log.Fatal(err)
						}
						defer outFile.Close()

						_, err = io.Copy(outFile, tr)
						if err != nil {
							log.Fatal(err)
						}
					}
				} else {

					r, err := zip.OpenReader(jdk)
					if err != nil {
						log.Fatal(err)
					}
					defer r.Close()

					for _, f := range r.File {
						if f.FileInfo().IsDir() {
							continue
						}

						outPath := filepath.Join(tempDir, f.Name)

						err = os.MkdirAll(filepath.Dir(outPath), 0777)
						if err != nil {
							log.Fatal(err)
						}

						outFile, err := os.Create(outPath)
						if err != nil {
							log.Fatal(err)
						}
						defer outFile.Close()

						src, err := f.Open()
						if err != nil {
							log.Fatal(err)
						}
						defer src.Close()

						_, err = io.Copy(outFile, src)
						if err != nil {
							log.Fatal(err)
						}
					}
				}
			*/

			name := FileNameWithoutExtSliceNotation(filepath.Base(jar))

			maingo, err := os.Create(filepath.Join(tempDir, "main.go"))
			if err != nil {
				log.Fatal(err)
			}
			defer maingo.Close()

			bindataCmd := exec.Command("go", "get", "-u", "github.com/go-bindata/go-bindata/...")
			err = bindataCmd.Run()
			if err != nil {
				log.Fatal(err)
			}

			gomodCmd := exec.Command("go", "mod", "init", name)
			gomodCmd.Dir = tempDir
			err = gomodCmd.Run()
			if err != nil {
				log.Fatal(err)
			}

			gbCmd := exec.Command("go-bindata", "-o", filepath.Join(tempDir, "bindata.go"), fmt.Sprintf("%s/...", temp))
			err = gbCmd.Run()
			if err != nil {
				log.Fatal(err)
			}

			InitGo()

			_, err = io.Copy(maingo, bytes.NewBuffer([]byte(src)))
			if err != nil {
				log.Fatal(err)
			}

			outName := fmt.Sprintf("%s", name)
			if platform == "windows" {
				outName += ".exe"
			}

			goCmd := exec.Command("go", "build", "-o", filepath.Join(tempDir, "..", out, outName))
			goCmd.Dir = tempDir
			goCmd.Env = os.Environ()
			goCmd.Env = append(goCmd.Env, fmt.Sprintf("GOOS=%s", platform))
			goCmd.Env = append(goCmd.Env, fmt.Sprintf("GOARCH=%s", arch))
			//goCmd.Env = append(goCmd.Env, fmt.Sprintf("GOPATH=%s", os.Getenv("GOPATH")))
			goCmd.Stderr = os.Stderr

			err = goCmd.Run()
			if err != nil {
				log.Fatal(err)
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
