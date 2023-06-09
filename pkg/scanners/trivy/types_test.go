package main

import (
	"testing"

	"github.com/aquasecurity/trivy/pkg/fanal/analyzer"
	"github.com/stretchr/testify/assert"
)

func TestAppendDisabledAnalyzers(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		disabled := appendDisabledAnalyzers()
		assert.Empty(t, disabled)
	})

	t.Run("single", func(t *testing.T) {
		disabled := appendDisabledAnalyzers(analyzer.TypeLockfiles)
		assert.Equal(t, analyzer.TypeLockfiles, disabled)
	})

	t.Run("multiple", func(t *testing.T) {
		disabled := appendDisabledAnalyzers(analyzer.TypeLockfiles, analyzer.TypeConfigFiles)
		assert.Equal(t, []analyzer.Type{"bundler", "npm", "yarn", "pnpm", "pip", "pipenv", "poetry", "gomod", "pom", "conan-lock", "gradle-lockfile", "yaml", "json", "dockerfile", "terraform", "cloudFormation", "helm"}, disabled)
	})
}
