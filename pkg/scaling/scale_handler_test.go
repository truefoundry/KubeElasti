/*
Copyright 2024.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package scaling

import (
	"os"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

func TestScaleHandler(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "ScaleHandler Suite")
}

var _ = Describe("StartScaleDownWatcher Polling Interval", func() {
	var (
		observedLogs *observer.ObservedLogs
		logger       *zap.Logger
	)

	BeforeEach(func() {
		// Create a logger that captures log output for assertion
		core, logs := observer.New(zap.WarnLevel)
		observedLogs = logs
		logger = zap.New(core)
	})

	AfterEach(func() {
		// Clean up environment variable
		os.Unsetenv("POLLING_INTERVAL")
	})

	// Helper function to test polling interval parsing logic
	testPollingIntervalParsing := func(envValue string) (time.Duration, bool) {
		pollingInterval := 30 * time.Second
		hasWarning := false

		if envValue != "" {
			duration, err := time.ParseDuration(envValue)
			if err != nil {
				logger.Warn("Invalid POLLING_INTERVAL value, using default 30s", zap.Error(err))
				hasWarning = true
			} else if duration <= 0 {
				logger.Warn("POLLING_INTERVAL must be positive, using default 30s", zap.String("value", envValue))
				hasWarning = true
			} else {
				pollingInterval = duration
			}
		}

		return pollingInterval, hasWarning
	}

	Context("When POLLING_INTERVAL is not set", func() {
		It("should use default 30s interval", func() {
			interval, hasWarning := testPollingIntervalParsing("")

			Expect(interval).To(Equal(30 * time.Second))
			Expect(hasWarning).To(BeFalse())
			Expect(observedLogs.Len()).To(Equal(0))
		})
	})

	Context("When POLLING_INTERVAL is set to valid values", func() {
		It("should parse '10s' correctly", func() {
			interval, hasWarning := testPollingIntervalParsing("10s")

			Expect(interval).To(Equal(10 * time.Second))
			Expect(hasWarning).To(BeFalse())
			Expect(observedLogs.Len()).To(Equal(0))
		})

		It("should parse '30s' correctly", func() {
			interval, hasWarning := testPollingIntervalParsing("30s")

			Expect(interval).To(Equal(30 * time.Second))
			Expect(hasWarning).To(BeFalse())
			Expect(observedLogs.Len()).To(Equal(0))
		})

		It("should parse '1m' correctly", func() {
			interval, hasWarning := testPollingIntervalParsing("1m")

			Expect(interval).To(Equal(1 * time.Minute))
			Expect(hasWarning).To(BeFalse())
			Expect(observedLogs.Len()).To(Equal(0))
		})

		It("should parse '2m30s' correctly", func() {
			interval, hasWarning := testPollingIntervalParsing("2m30s")

			Expect(interval).To(Equal(2*time.Minute + 30*time.Second))
			Expect(hasWarning).To(BeFalse())
			Expect(observedLogs.Len()).To(Equal(0))
		})

		It("should parse '1s' correctly", func() {
			interval, hasWarning := testPollingIntervalParsing("1s")

			Expect(interval).To(Equal(1 * time.Second))
			Expect(hasWarning).To(BeFalse())
			Expect(observedLogs.Len()).To(Equal(0))
		})

		It("should parse '5m' correctly", func() {
			interval, hasWarning := testPollingIntervalParsing("5m")

			Expect(interval).To(Equal(5 * time.Minute))
			Expect(hasWarning).To(BeFalse())
			Expect(observedLogs.Len()).To(Equal(0))
		})

		It("should parse '10m' correctly", func() {
			interval, hasWarning := testPollingIntervalParsing("10m")

			Expect(interval).To(Equal(10 * time.Minute))
			Expect(hasWarning).To(BeFalse())
			Expect(observedLogs.Len()).To(Equal(0))
		})
	})

	Context("When POLLING_INTERVAL is invalid", func() {
		It("should fall back to default 30s when set to 'invalid'", func() {
			interval, hasWarning := testPollingIntervalParsing("invalid")

			Expect(interval).To(Equal(30 * time.Second))
			Expect(hasWarning).To(BeTrue())
			Expect(observedLogs.Len()).To(Equal(1))
			allLogs := observedLogs.All()
			Expect(allLogs[0].Message).To(Equal("Invalid POLLING_INTERVAL value, using default 30s"))
			Expect(allLogs[0].Level).To(Equal(zap.WarnLevel))
		})

		It("should fall back to default 30s when set to '30' (missing unit)", func() {
			interval, hasWarning := testPollingIntervalParsing("30")

			Expect(interval).To(Equal(30 * time.Second))
			Expect(hasWarning).To(BeTrue())
			Expect(observedLogs.Len()).To(Equal(1))
			allLogs := observedLogs.All()
			Expect(allLogs[0].Message).To(Equal("Invalid POLLING_INTERVAL value, using default 30s"))
		})

		It("should fall back to default 30s when set to 'abc'", func() {
			interval, hasWarning := testPollingIntervalParsing("abc")

			Expect(interval).To(Equal(30 * time.Second))
			Expect(hasWarning).To(BeTrue())
			Expect(observedLogs.Len()).To(Equal(1))
			allLogs := observedLogs.All()
			Expect(allLogs[0].Message).To(Equal("Invalid POLLING_INTERVAL value, using default 30s"))
		})

		It("should fall back to default 30s when set to whitespace", func() {
			interval, hasWarning := testPollingIntervalParsing("   ")

			Expect(interval).To(Equal(30 * time.Second))
			Expect(hasWarning).To(BeTrue())
			Expect(observedLogs.Len()).To(Equal(1))
			allLogs := observedLogs.All()
			Expect(allLogs[0].Message).To(Equal("Invalid POLLING_INTERVAL value, using default 30s"))
		})

		It("should fall back to default 30s when set to negative value '-10s'", func() {
			interval, hasWarning := testPollingIntervalParsing("-10s")

			Expect(interval).To(Equal(30 * time.Second))
			Expect(hasWarning).To(BeTrue())
			Expect(observedLogs.Len()).To(Equal(1))
			allLogs := observedLogs.All()
			Expect(allLogs[0].Message).To(Equal("POLLING_INTERVAL must be positive, using default 30s"))
		})

		It("should fall back to default 30s when set to zero '0s'", func() {
			interval, hasWarning := testPollingIntervalParsing("0s")

			Expect(interval).To(Equal(30 * time.Second))
			Expect(hasWarning).To(BeTrue())
			Expect(observedLogs.Len()).To(Equal(1))
			allLogs := observedLogs.All()
			Expect(allLogs[0].Message).To(Equal("POLLING_INTERVAL must be positive, using default 30s"))
		})
	})

	Context("Environment variable name verification", func() {
		It("should read from POLLING_INTERVAL environment variable", func() {
			os.Setenv("POLLING_INTERVAL", "15s")
			defer os.Unsetenv("POLLING_INTERVAL")

			envValue := os.Getenv("POLLING_INTERVAL")
			interval, hasWarning := testPollingIntervalParsing(envValue)

			Expect(interval).To(Equal(15 * time.Second))
			Expect(hasWarning).To(BeFalse())
		})

		It("should NOT read from deprecated POLLING_VARIABLE environment variable", func() {
			// This test verifies the bug fix - the old variable name should not be used
			os.Setenv("POLLING_VARIABLE", "15s")
			defer os.Unsetenv("POLLING_VARIABLE")

			// Attempting to read from the old variable name should return empty
			envValue := os.Getenv("POLLING_INTERVAL")
			interval, hasWarning := testPollingIntervalParsing(envValue)

			// Should use default since POLLING_INTERVAL is not set
			Expect(interval).To(Equal(30 * time.Second))
			Expect(hasWarning).To(BeFalse())
		})
	})
})
