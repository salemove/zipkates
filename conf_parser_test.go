package main

import (
	"os"
	"testing"

	. "github.com/onsi/gomega"
)

func TestLabelTagMapping(t *testing.T) {
	t.Run("Two mappings", func(t *testing.T) {
		g := NewWithT(t)

		os.Setenv("LABEL_TAG_MAPPING", `{"label_a":"tag_a", "label_b": "tag_b"}`)
		cfg, err := ParseConfigFromEnv()

		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(cfg.LabelTagMapping).To(Equal(map[string]string{
			"label_a": "tag_a",
			"label_b": "tag_b",
		}))
	})

	t.Run("Missing mapping", func(t *testing.T) {
		g := NewWithT(t)

		os.Unsetenv("LABEL_TAG_MAPPING")
		cfg, err := ParseConfigFromEnv()

		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(cfg.LabelTagMapping).To(Equal(map[string]string{"owner": "owner"}))
	})

	t.Run("Empty string", func(t *testing.T) {
		g := NewWithT(t)

		os.Setenv("LABEL_TAG_MAPPING", "")
		cfg, err := ParseConfigFromEnv()

		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(cfg.LabelTagMapping).To(Equal(map[string]string{"owner": "owner"}))
	})

	t.Run("Empty map", func(t *testing.T) {
		g := NewWithT(t)

		os.Setenv("LABEL_TAG_MAPPING", "{}")
		cfg, err := ParseConfigFromEnv()

		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(cfg.LabelTagMapping).NotTo(BeNil())
		g.Expect(len(cfg.LabelTagMapping)).To(Equal(0))
	})

	t.Run("Not an object", func(t *testing.T) {
		g := NewWithT(t)

		os.Setenv("LABEL_TAG_MAPPING", "[\"asdf\"]")
		_, err := ParseConfigFromEnv()

		g.Expect(err).To(HaveOccurred())
	})
}

func TestListenPort(t *testing.T) {
	t.Run("Not defined", func(t *testing.T) {
		g := NewWithT(t)

		os.Unsetenv("LISTEN_PORT")
		cfg, err := ParseConfigFromEnv()

		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(cfg.ListenPort).To(Equal(9411))
	})

	t.Run("Empty string", func(t *testing.T) {
		g := NewWithT(t)

		os.Setenv("LISTEN_PORT", "")
		cfg, err := ParseConfigFromEnv()

		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(cfg.ListenPort).To(Equal(9411))
	})

	t.Run("A number", func(t *testing.T) {
		g := NewWithT(t)

		os.Setenv("LISTEN_PORT", "8080")
		cfg, err := ParseConfigFromEnv()

		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(cfg.ListenPort).To(Equal(8080))
	})

	t.Run("Not a number", func(t *testing.T) {
		g := NewWithT(t)

		os.Setenv("LISTEN_PORT", "nine four one one")
		_, err := ParseConfigFromEnv()

		g.Expect(err).To(HaveOccurred())
	})
}

func TestZipkinPort(t *testing.T) {
	t.Run("Not defined", func(t *testing.T) {
		g := NewWithT(t)

		os.Unsetenv("ZIPKIN_PORT")
		cfg, err := ParseConfigFromEnv()

		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(cfg.ZipkinPort).To(Equal(9410))
	})

	t.Run("Empty string", func(t *testing.T) {
		g := NewWithT(t)

		os.Setenv("ZIPKIN_PORT", "")
		cfg, err := ParseConfigFromEnv()

		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(cfg.ZipkinPort).To(Equal(9410))
	})

	t.Run("A number", func(t *testing.T) {
		g := NewWithT(t)

		os.Setenv("ZIPKIN_PORT", "8080")
		cfg, err := ParseConfigFromEnv()

		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(cfg.ZipkinPort).To(Equal(8080))
	})

	t.Run("Not a number", func(t *testing.T) {
		g := NewWithT(t)

		os.Setenv("ZIPKIN_PORT", "nine four one one")
		_, err := ParseConfigFromEnv()

		g.Expect(err).To(HaveOccurred())
	})
}
