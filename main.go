package main

import (
	"flag"
	"fmt"
	"github.com/platypus-platform/pp-logging"
	"github.com/platypus-platform/pp-store"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"strings"
)

type SudoersConfig struct {
	Path string
}

type Sudoers struct {
	App    string
	Owners []string
}

func main() {
	hostname, err := os.Hostname()
	if err != nil {
		logger.Fatal(err.Error())
		os.Exit(1)
	}

	var config SudoersConfig

	flag.StringVar(&config.Path, "path",
		"fake/sudoers.d", "path to sudoers.d")
	flag.Parse()

	err = pp.PollIntent(hostname, func(intent pp.IntentNode) {
		spec := []Sudoers{}

		for _, app := range intent.Apps {
			owners, err := pp.FetchOwners(app.Name)
			if err != nil {
				logger.Fatal("%s: could not fetch ownership data: %s", app.Name, err)
				continue
			}

			spec = append(spec, Sudoers{App: app.Name, Owners: owners.Users})
		}

		writeSudoers(config, spec)
	})

	if err != nil {
		logger.Fatal(err.Error())
		os.Exit(1)
	}
}

func writeSudoers(config SudoersConfig, spec []Sudoers) {
	actual := []string{}
	expected := []string{}
	files, err := ioutil.ReadDir(config.Path)
	if err != nil {
		logger.Fatal("could not read existing files: %s", err)
		return
	}
	for _, file := range files {
		if strings.HasPrefix(file.Name(), "pp-") {
			actual = append(actual, file.Name())
		}
	}

	for _, sudoers := range spec {
		appName := sudoers.App
		owners := sudoers.Owners

		if err := writeSudoersForApp(config, appName, owners); err != nil {
			logger.Fatal("%s: error writing sudoers: %s", appName, err)
		}
		expected = append(expected, "pp-"+appName)
	}

	for _, file := range subtract(actual, expected) {
		logger.Info("removing unexepected file: %s", file)

		if err := os.Remove(path.Join(config.Path, file)); err != nil {
			logger.Fatal("%s", err)
		}
	}
}

func writeSudoersForApp(config SudoersConfig, appName string, owners []string) error {
	fpath := path.Join(config.Path, "pp-"+appName)
	fcontent := fmt.Sprintf("%s ALL = (%s) ALL",
		strings.Join(owners, ", "),
		appName,
	)

	written, err := writeFileWithValidation(fpath, []byte(fcontent), 0440,
		func(tmppath string) bool {
			cmd := exec.Command("visudo", "-cf", tmppath)
			out, err := cmd.Output()
			if err != nil {
				logger.Fatal("%s: could not validate sudoers: %s", appName, err)
				logger.Fatal("%s", out)
				return false
			} else {
				return true
			}
		})

	if err != nil {
		return err
	}

	if written {
		logger.Info("%s: wrote new sudoers", appName)
	} else {
		logger.Info("%s: no change to sudoers", appName)
	}

	return nil
}

func writeFileWithValidation(
	fpath string,
	fcontent []byte,
	mod os.FileMode,
	validate func(string) bool,
) (bool, error) {
	f, err := ioutil.TempFile("", "pp-sudoers")
	if err != nil {
		return false, err
	}
	defer os.Remove(f.Name()) // This will fail in happy case, that's fine.

	if _, err := f.Write(fcontent); err != nil {
		return false, err
	}
	if err := f.Chmod(mod); err != nil {
		return false, err
	}
	if err := f.Close(); err != nil {
		return false, err
	}

	if validate(f.Name()) {
		cmd := exec.Command("cmp", "--silent", f.Name(), fpath)
		if err := cmd.Run(); err != nil {
			if err := os.Rename(f.Name(), fpath); err != nil {
				return false, err
			}
		} else {
			return false, nil
		}
	}

	return true, nil
}

func subtract(slice1 []string, slice2 []string) []string {
	diffStr := []string{}
	m := map[string]int{}

	for _, s1Val := range slice1 {
		m[s1Val] = 1
	}
	for _, s2Val := range slice2 {
		delete(m, s2Val)
	}

	for mKey, _ := range m {
		diffStr = append(diffStr, mKey)
	}

	return diffStr
}
