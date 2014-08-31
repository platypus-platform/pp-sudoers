package main

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/platypus-platform/pp-logging"

	"bytes"
	"io/ioutil"
	"os"
	"path"
)

var _ = Describe("Writing sudoers", func() {
	It("Writes a new sudoers file for app", func() {
		basedir := tempDir()
		defer os.RemoveAll(basedir)

		config := SudoersConfig{Path: basedir}

		err := writeSudoers(config, "testapp", []string{"xavier, donalias"})

		expectedFile := path.Join(basedir, "pp-testapp")

		contents, err := ioutil.ReadFile(expectedFile)

		Expect(err).To(BeNil())
		Expect(string(contents)).To(Equal("xavier, donalias ALL = (testapp) ALL"))
	})

	It("Does not write invalid sudoers", func() {
		var buf bytes.Buffer
		logger.SetOut(&buf)
		defer logger.SetOut(logger.DefaultOut())

		basedir := tempDir()
		defer os.RemoveAll(basedir)

		config := SudoersConfig{Path: basedir}

		err := writeSudoers(config, "testapp", []string{"$%$%$((%$"})

		Expect(err).To(BeNil())

		expectedFile := path.Join(basedir, "pp-testapp")

		if _, err := os.Stat(expectedFile); !os.IsNotExist(err) {
			Fail("File was written")
		}

		Expect(buf.String()).To(ContainSubstring("could not validate sudoers"))
		Expect(buf.String()).To(ContainSubstring("testapp"))
	})

	It("Does not update sudoers if it hasn't changed", func() {
		var buf bytes.Buffer
		logger.SetOut(&buf)
		defer logger.SetOut(logger.DefaultOut())

		basedir := tempDir()
		defer os.RemoveAll(basedir)

		config := SudoersConfig{Path: basedir}

		expectedFile := path.Join(basedir, "pp-testapp")
		ioutil.WriteFile(expectedFile, []byte("xavier ALL = (testapp) ALL"), 0440)

		s1, _ := os.Stat(expectedFile)

		err := writeSudoers(config, "testapp", []string{"xavier"})

		s2, _ := os.Stat(expectedFile)

		Expect(err).To(BeNil())
		Expect(s2).To(Equal(s1))
	})

	It("Updates existing sudoers if it has changed", func() {
		var buf bytes.Buffer
		logger.SetOut(&buf)
		defer logger.SetOut(logger.DefaultOut())

		basedir := tempDir()
		defer os.RemoveAll(basedir)

		config := SudoersConfig{Path: basedir}

		expectedFile := path.Join(basedir, "pp-testapp")
		ioutil.WriteFile(expectedFile, []byte("don ALL = (testapp) ALL"), 0440)

		err := writeSudoers(config, "testapp", []string{"xavier"})

		contents, err := ioutil.ReadFile(expectedFile)

		Expect(err).To(BeNil())
		Expect(string(contents)).To(Equal("xavier ALL = (testapp) ALL"))

	})
})

func tempDir() string {
	dir, err := ioutil.TempDir("", "preparer-test")
	if err != nil {
		panic(err)
	}
	return dir
}
