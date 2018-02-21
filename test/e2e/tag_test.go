package integration

import (
	"os"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Podman tag", func() {
	var (
		tempdir    string
		err        error
		podmanTest PodmanTest
	)

	BeforeEach(func() {
		tempdir, err = CreateTempDirInTempDir()
		if err != nil {
			os.Exit(1)
		}
		podmanTest = PodmanCreate(tempdir)
		podmanTest.RestoreAllArtifacts()
	})

	AfterEach(func() {
		podmanTest.Cleanup()

	})

	It("podman tag shortname:latest", func() {
		session := podmanTest.Podman([]string{"tag", ALPINE, "foobar:latest"})
		session.WaitWithDefaultTimeout()
		Expect(session.ExitCode()).To(Equal(0))

		results := podmanTest.Podman([]string{"inspect", "foobar:latest"})
		results.WaitWithDefaultTimeout()
		Expect(results.ExitCode()).To(Equal(0))
		inspectData := results.InspectImageJSON()
		Expect(StringInSlice("docker.io/library/alpine:latest", inspectData[0].RepoTags)).To(BeTrue())
		Expect(StringInSlice("foobar:latest", inspectData[0].RepoTags)).To(BeTrue())
	})

	It("podman tag shortname", func() {
		session := podmanTest.Podman([]string{"tag", ALPINE, "foobar"})
		session.WaitWithDefaultTimeout()
		Expect(session.ExitCode()).To(Equal(0))

		results := podmanTest.Podman([]string{"inspect", "foobar:latest"})
		results.WaitWithDefaultTimeout()
		Expect(results.ExitCode()).To(Equal(0))
		inspectData := results.InspectImageJSON()
		Expect(StringInSlice("docker.io/library/alpine:latest", inspectData[0].RepoTags)).To(BeTrue())
		Expect(StringInSlice("foobar:latest", inspectData[0].RepoTags)).To(BeTrue())
	})

	It("podman tag shortname:tag", func() {
		session := podmanTest.Podman([]string{"tag", ALPINE, "foobar:new"})
		session.WaitWithDefaultTimeout()
		Expect(session.ExitCode()).To(Equal(0))

		results := podmanTest.Podman([]string{"inspect", "foobar:new"})
		results.WaitWithDefaultTimeout()
		Expect(results.ExitCode()).To(Equal(0))
		inspectData := results.InspectImageJSON()
		Expect(StringInSlice("docker.io/library/alpine:latest", inspectData[0].RepoTags)).To(BeTrue())
		Expect(StringInSlice("foobar:new", inspectData[0].RepoTags)).To(BeTrue())
	})
})
