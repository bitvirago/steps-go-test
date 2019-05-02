package main

import (
	"fmt"
	"github.com/bitrise-io/go-utils/command"
	"github.com/bitrise-io/go-utils/log"
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

func getDeployDir() (string, error) {
	deployDir := os.Getenv("BITRISE_DEPLOY_DIR")
	if deployDir == "" {
		return "", fmt.Errorf("BITRISE_DEPLOY_DIR env not set")
	}
	if err := os.MkdirAll(deployDir, 0777); err != nil {
		return "", fmt.Errorf("failed to create BITRISE_DEPLOY_DIR: %s", err)
	}

	return deployDir, nil
}

func createPackageCodeCoverageFile() (string, error) {
	deployDir, err := getDeployDir()
	if err != nil {
		return "", err
	}

	pth := filepath.Join(deployDir, "cover_profile.out")
	if _, err := os.Create(pth); err != nil {
		return "", err
	}

	return pth, nil
}

func createCoverage(packageCodeCoveragePth, packages string) {
	deployDir, err := getDeployDir()
	if err != nil {
		failf("cannot create deploy dir", err)
	}

	cmd := command.NewWithStandardOuts("bash", "-c",
		fmt.Sprintf("go test -v -race -coverprofile=%s -covermode=atomic %s | tee %s",
			packageCodeCoveragePth,
			packages,
			filepath.Join(deployDir, "output.log"),
		),
	)
	log.Printf("$ %s", cmd.PrintableCommandArgs())

	if err := cmd.Run(); err != nil {
		failf("go test failed: %s", err)
	}

	if err := tools.ExportEnvironmentWithEnvman("GO_CODE_COVERAGE_REPORT_PATH", packageCodeCoveragePth); err != nil {
		failf("Failed to export GO_CODE_COVERAGE_REPORT_PATH=%s", packageCodeCoveragePth)
	}

	log.Donef("\ncode coverage is available at: GO_CODE_COVERAGE_REPORT_PATH=%s", packageCodeCoveragePth)
}

func createHtmlCoverage(packageCodeCoveragePth string) {
	deployDir, err := getDeployDir()
	if err != nil {
		failf("cannot create deploy dir", err)
	}

	htmlTempFile := filepath.Join(deployDir, "cover_profile.html")
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

func createJUnitCoverage() {
	deployDir, err := getDeployDir()
	if err != nil {
		failf("cannot create deploy dir", err)
	}

	jUnitFile := filepath.Join(deployDir, "cover_profile.xml")
	cmd := command.NewWithStandardOuts("bash", "-c", fmt.Sprintf("cat %s | go-junit-report > %s", filepath.Join(deployDir, "output.log"), jUnitFile))

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

	packageCodeCoveragePth, err := createPackageCodeCoverageFile()
	if err != nil {
		failf(err.Error())
	}

	createCoverage(packageCodeCoveragePth, packages)
	createHtmlCoverage(packageCodeCoveragePth)
	createJUnitCoverage()
}
