package cli

import (
	"io"
	"io/fs"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"

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
	temp     string

	packageCmd = &cobra.Command{
		Use:   "package",
		Short: "Package commands",
		Long:  `package is used packaging jar files`,
		Run: func(cmd *cobra.Command, args []string) {
			if jar == "" || out == "" || jdk == "" {
				cmd.Usage()
				return
			}

			temp = out
			if clean {
				err := os.RemoveAll(out)
				if err != nil {
					log.Fatal("remove out dir failed:", err)
				}
			}

			err := os.MkdirAll(out, 0777)
			if err != nil {
				log.Fatal("out dir err:", err)
			}

			tempDir, err := filepath.Abs(temp)
			if err != nil {
				log.Fatal("abs tempdir err:", err)
			}

			err = archiver.Unarchive(jdk, tempDir)
			if err != nil {
				log.Fatal("unarchive failed:", err)
			}

			err = filepath.WalkDir(tempDir, func(path string, d fs.DirEntry, err error) error {
				if err != nil {
					//return err
				}

				relPath, err := filepath.Rel(tempDir, path)
				if err != nil {
					return err
				}

				if relPath == "." {
					return nil
				}

				ok, err := regexp.MatchString("^[a-zA-Z0-9\\-\\_.]+(\\/(bin|lib)(\\/[a-zA-Z0-9\\-\\_.]+|$)+|$)$", relPath)
				if err != nil {
					return err
				}

				if !ok && d.IsDir() {
					os.RemoveAll(path)
				} else if !ok {
					os.Remove(path)
				}

				return nil
			})
			if err != nil {
				log.Fatal("walkdir failed:", err)
			}

			inJar, err := os.Open(jar)
			if err != nil {
				log.Fatal("open jar err:", err)
			}
			defer inJar.Close()

			outJar, err := os.Create(filepath.Join(tempDir, filepath.Base(jar)))
			if err != nil {
				log.Fatal("creating out jar failed:", err)
			}
			defer outJar.Close()

			_, err = io.Copy(outJar, inJar)
			if err != nil {
				log.Fatal("copy jar failed:", err)
			}

			err = generate(jar, out)
			if err != nil {
				log.Fatal("code generation failed:", err)
			}

			mod := strings.TrimSuffix(filepath.Base(jar), ".jar")

			modInitCmd := exec.Command("go", "mod", "init", mod)
			modInitCmd.Stdout = os.Stdout
			modInitCmd.Stderr = os.Stderr
			modInitCmd.Dir = out
			err = modInitCmd.Run()
			if err != nil {
				log.Fatal("go mod init failed:", err)
			}

			modTidyCmd := exec.Command("go", "mod", "tidy")
			modTidyCmd.Stdout = os.Stdout
			modTidyCmd.Stderr = os.Stderr
			modTidyCmd.Dir = out
			err = modTidyCmd.Run()
			if err != nil {
				log.Fatal("go mod tidy failed:", err)
			}

			binaryName := mod
			if runtime.GOOS == "windows" {
				binaryName += ".exe"
			}

			buildCmd := exec.Command("go", "build", "-o", binaryName)
			buildCmd.Stdout = os.Stdout
			buildCmd.Stderr = os.Stderr
			buildCmd.Dir = out
			err = buildCmd.Run()
			if err != nil {
				log.Fatal("go build failed:", err)
			}

			filepath.WalkDir(out, func(path string, d fs.DirEntry, err error) error {
				if err != nil {
					return nil
				}

				if d.IsDir() && path != out {
					os.RemoveAll(path)
				}

				if !d.IsDir() && filepath.Join(out, binaryName) != path {
					os.Remove(path)
				}

				return nil
			})

		},
	}
)

func init() {

	packageCmd.Flags().StringVar(&jar, "jar", "", "JAR file")
	packageCmd.Flags().StringVar(&out, "out", "dist", "Output directory")
	packageCmd.Flags().StringVar(&jdk, "jdk", "", "JDK path")
	packageCmd.Flags().StringVar(&platform, "platform", runtime.GOOS, "Operating system")
	packageCmd.Flags().StringVar(&arch, "arch", runtime.GOARCH, "Operating system architecture")
	packageCmd.Flags().BoolVar(&clean, "clean", false, "Clean packaging")

	rootCmd.AddCommand(packageCmd)

}

func FileNameWithoutExtSliceNotation(fileName string) string {
	return fileName[:len(fileName)-len(filepath.Ext(fileName))]
}
