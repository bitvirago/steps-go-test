package main

import (
	"fmt"
	"github.com/bitrise-io/go-utils/command"
	"github.com/bitrise-io/go-utils/log"
	"github.com/bitrise-io/go-utils/pathutil"
	"github.com/bitrise-tools/go-steputils/tools"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const tempDir = "go-test-coverage"

func failf(format string, args ...interface{}) {
	log.Errorf(format, args...)
	os.Exit(1)
}

func installedInPath(name string) bool {
	cmd := exec.Command("which", name)
	outBytes, err := cmd.Output()
	return err == nil && strings.TrimSpace(string(outBytes)) != ""
}

func getTempDirName() (string, error) {
	tmpDir, err := pathutil.NormalizedOSTempDirPath(tempDir)
	if err != nil {
		return "", fmt.Errorf("failed to create tmp dir for code coverage reports: %s", err)
	}

	return tmpDir, nil
}

func createPackageCodeCoverageFile(tmpDir string) (string, error) {
	pth := filepath.Join(tmpDir, "cover_profile.out")
	if _, err := os.Create(pth); err != nil {
		return "", err
	}

	return pth, nil
}

func createCoverage(packageCodeCoveragePth, packages string) {
	cmd := command.NewWithStandardOuts("go", "test", "-v", "-race", "-coverprofile="+packageCodeCoveragePth, "-covermode=atomic", packages)

	log.Printf("$ %s", cmd.PrintableCommandArgs())

	if err := cmd.Run(); err != nil {
		failf("go test failed: %s", err)
	}

	if err := tools.ExportEnvironmentWithEnvman("GO_CODE_COVERAGE_REPORT_PATH", packageCodeCoveragePth); err != nil {
		failf("Failed to export GO_CODE_COVERAGE_REPORT_PATH=%s", packageCodeCoveragePth)
	}

	log.Donef("\ncode coverage is available at: GO_CODE_COVERAGE_REPORT_PATH=%s", packageCodeCoveragePth)
}

func createHtmlCoverage(packageCodeCoveragePth, tmpDir string) {
	htmlTempFile := filepath.Join(tmpDir, "cover_profile.html")
	cmd := command.NewWithStandardOuts("go", "tool", "cover", "-html="+packageCodeCoveragePth, "-o", htmlTempFile)

	log.Printf("$ %s", cmd.PrintableCommandArgs())

	if err := cmd.Run(); err != nil {
		failf("go test failed: %s", err)
	}

	if err := tools.ExportEnvironmentWithEnvman("GO_CODE_COVERAGE_HTML_REPORT_PATH", htmlTempFile); err != nil {
		failf("Failed to export GO_CODE_COVERAGE_HTML_REPORT_PATH=%s", htmlTempFile)
	}

	log.Donef("\ncode coverage is available at: GO_CODE_COVERAGE_HTML_REPORT_PATH=%s", htmlTempFile)
}

func createJUnitCoverage(packageCodeCoveragePth, tmpDir string) {
	jUnitFile := filepath.Join(tmpDir, "cover_profile.xml")
	cmd := command.NewWithStandardOuts("bash", "-c", fmt.Sprintf("cat %s | go-junit-report > %s", packageCodeCoveragePth, jUnitFile))

	log.Printf("$ %s", cmd.PrintableCommandArgs())

	if err := cmd.Run(); err != nil {
		failf("go test failed: %s", err)
	}

	if err := tools.ExportEnvironmentWithEnvman("GO_CODE_COVERAGE_JUNIT_REPORT_PATH", jUnitFile); err != nil {
		failf("Failed to export GO_CODE_COVERAGE_JUNIT_REPORT_PATH=%s", jUnitFile)
	}

	log.Donef("\ncode coverage is available at: GO_CODE_COVERAGE_JUNIT_REPORT_PATH=%s", jUnitFile)
}

func main() {
	packages := os.Getenv("packages")

	log.Infof("Configs:")
	log.Printf("- packages: %s", packages)

	if packages == "" {
		failf("Required input not defined: packages")
	}

	if !installedInPath("go-junit-report") {
		cmd := command.New("go", "get", "-u", "github.com/jstemmer/go-junit-report")

		log.Infof("\nInstalling go-junit-report")
		log.Donef("$ %s", cmd.PrintableCommandArgs())

		if out, err := cmd.RunAndReturnTrimmedCombinedOutput(); err != nil {
			failf("failed to install go-junit-report: %s", out)
		}
	}

	log.Infof("\nRunning go test...")

	tmpDir, err := getTempDirName()
	if err != nil {
		failf("failed to create temp dir")
	}

	packageCodeCoveragePth, err := createPackageCodeCoverageFile(tmpDir)
	if err != nil {
		failf(err.Error())
	}

	createCoverage(packageCodeCoveragePth, packages)
	createHtmlCoverage(packageCodeCoveragePth, tmpDir)
	createJUnitCoverage(packageCodeCoveragePth, tmpDir)
}
