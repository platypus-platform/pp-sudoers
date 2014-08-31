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
		for _, app := range intent.Apps {
			owners, err := pp.FetchOwners(app.Name)
			if err != nil {
				logger.Fatal("%s: could not fetch ownership data: %s", app.Name, err)
			}

			if err := writeSudoers(config, app.Name, owners.Users); err != nil {
				logger.Fatal("%s: error writing sudoers: %s", app.Name, err)
			}
		}
		// TODO: Remove old definitions
	})

	if err != nil {
		logger.Fatal(err.Error())
		os.Exit(1)
	}
}

func writeSudoers(config SudoersConfig, appName string, owners []string) error {
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
