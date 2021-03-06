package e2e

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/fgimenez/ci-health/pkg/constants"
	"github.com/fgimenez/ci-health/pkg/runner"
	"github.com/fgimenez/ci-health/pkg/stats"
	"github.com/fgimenez/ci-health/pkg/types"
)

func TestCiHealth(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "ci-health Suite")
}

var (
	tokenPath string
)

const (
	source   = "kubevirt/kubevirt"
	dataDays = 1
)

var _ = BeforeSuite(func() {
	tokenPath = os.Getenv("GITHUB_TOKEN_PATH")
	if tokenPath == "" {
		Fail("Please specify an OAuth2 token in the env var GITHUB_TOKEN_PATH")
	}
})

var _ = Describe("ci-health stats", func() {
	It("Retrieves data from github and writes badges", func() {
		artifactsDir, err := ioutil.TempDir("", "e2e-ci-health")
		Expect(err).ToNot(HaveOccurred())

		opt := &types.Options{
			Path:      artifactsDir,
			TokenPath: tokenPath,
			Source:    source,
			DataDays:  dataDays,
			LogLevel:  "debug",
		}

		checkResults := func(results *stats.Results) {
			Expect(results.DataDays).To(Equal(dataDays))
			Expect(results.Source).To(Equal(source))

			Expect(results.Data).To(HaveLen(2))

			names := []string{constants.MergeQueueLengthName, constants.TimeToMergeName}

			parseFloat := func(value string) float64 {
				floatValue, err := strconv.ParseFloat(strings.Fields(value)[0], 64)
				Expect(err).ToNot(HaveOccurred())
				return floatValue
			}

			for _, name := range names {
				metricResults := results.Data[name]
				value := parseFloat(metricResults.Value)
				Expect(value).To(BeNumerically(">", 0))
				for _, dataPoint := range metricResults.DataPoints {
					dataPointValue := parseFloat(dataPoint.Value)
					Expect(dataPointValue).To(BeNumerically(">=", 0))
				}
			}
		}

		results, err := runner.Run(opt)
		Expect(err).ToNot(HaveOccurred())

		By("Checking returned data")
		checkResults(results)

		By("Checking badge files")
		badgeFileNames := []string{
			filepath.Join(artifactsDir, constants.MergeQueueLengthBadgeFileName),
			filepath.Join(artifactsDir, constants.TimeToMergeBadgeFileName),
		}
		for _, badgeFileName := range badgeFileNames {
			_, err := os.Stat(badgeFileName)
			Expect(err).ToNot(HaveOccurred())
		}

		By("Checking JSON file")
		jsonFileName := filepath.Join(artifactsDir, constants.JSONResultsFileName)
		data, err := ioutil.ReadFile(jsonFileName)
		Expect(err).ToNot(HaveOccurred())

		var jsonResults stats.Results
		err = json.Unmarshal(data, &jsonResults)
		Expect(err).ToNot(HaveOccurred())

		checkResults(&jsonResults)
	})
})
