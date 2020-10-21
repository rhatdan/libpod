package integration

import (
	"fmt"
	"os"
	"strings"

	. "github.com/containers/podman/v2/test/utils"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/uber/jaeger-client-go/utils"
)

var _ = Describe("Podman run networking", func() {
	var (
		tempdir     string
		err         error
		podmanTest  *PodmanTestIntegration
		hostname, _ = os.Hostname()
	)

	BeforeEach(func() {
		tempdir, err = CreateTempDirInTempDir()
		if err != nil {
			os.Exit(1)
		}
		podmanTest = PodmanTestCreate(tempdir)
		podmanTest.Setup()
		podmanTest.SeedImages()
	})

	AfterEach(func() {
		podmanTest.Cleanup()
		f := CurrentGinkgoTestDescription()
		processTestResult(f)

	})

	It("podman run network connection with default bridge", func() {
		session := podmanTest.Podman([]string{"run", "-dt", ALPINE, "wget", "www.podman.io"})
		session.Wait(90)
		Expect(session.ExitCode()).To(Equal(0))
	})

	It("podman run network connection with host", func() {
		session := podmanTest.Podman([]string{"run", "-dt", "--network", "host", ALPINE, "wget", "www.podman.io"})
		session.Wait(90)
		Expect(session.ExitCode()).To(Equal(0))
	})

	It("podman run network connection with loopback", func() {
		session := podmanTest.Podman([]string{"run", "-dt", "--network", "host", ALPINE, "wget", "www.podman.io"})
		session.Wait(90)
		Expect(session.ExitCode()).To(Equal(0))
	})

	It("podman run network expose port 222", func() {
		SkipIfRootless("iptables is not supported for rootless users")
		session := podmanTest.Podman([]string{"run", "-dt", "--expose", "222-223", "-P", ALPINE, "/bin/sh"})
		session.Wait(30)
		Expect(session.ExitCode()).To(Equal(0))
		results := SystemExec("iptables", []string{"-t", "nat", "-L"})
		Expect(results.ExitCode()).To(Equal(0))
		Expect(results.OutputToString()).To(ContainSubstring("222"))
		Expect(results.OutputToString()).To(ContainSubstring("223"))
	})

	It("podman run -p 80", func() {
		name := "testctr"
		session := podmanTest.Podman([]string{"create", "-t", "-p", "80", "--name", name, ALPINE, "/bin/sh"})
		session.WaitWithDefaultTimeout()
		inspectOut := podmanTest.InspectContainer(name)
		Expect(len(inspectOut)).To(Equal(1))
		Expect(len(inspectOut[0].NetworkSettings.Ports)).To(Equal(1))
		Expect(len(inspectOut[0].NetworkSettings.Ports["80/tcp"])).To(Equal(1))
		Expect(inspectOut[0].NetworkSettings.Ports["80/tcp"][0].HostPort).To(Not(Equal("80")))
		Expect(inspectOut[0].NetworkSettings.Ports["80/tcp"][0].HostIP).To(Equal(""))
	})

	It("podman run -p 8080:80", func() {
		name := "testctr"
		session := podmanTest.Podman([]string{"create", "-t", "-p", "8080:80", "--name", name, ALPINE, "/bin/sh"})
		session.WaitWithDefaultTimeout()
		inspectOut := podmanTest.InspectContainer(name)
		Expect(len(inspectOut)).To(Equal(1))
		Expect(len(inspectOut[0].NetworkSettings.Ports)).To(Equal(1))
		Expect(len(inspectOut[0].NetworkSettings.Ports["80/tcp"])).To(Equal(1))
		Expect(inspectOut[0].NetworkSettings.Ports["80/tcp"][0].HostPort).To(Equal("8080"))
		Expect(inspectOut[0].NetworkSettings.Ports["80/tcp"][0].HostIP).To(Equal(""))
	})

	It("podman run -p 8080:80/TCP", func() {
		name := "testctr"
		// "TCP" in upper characters
		session := podmanTest.Podman([]string{"create", "-t", "-p", "8080:80/TCP", "--name", name, ALPINE, "/bin/sh"})
		session.WaitWithDefaultTimeout()
		inspectOut := podmanTest.InspectContainer(name)
		Expect(len(inspectOut)).To(Equal(1))
		Expect(len(inspectOut[0].NetworkSettings.Ports)).To(Equal(1))
		// "tcp" in lower characters
		Expect(len(inspectOut[0].NetworkSettings.Ports["80/tcp"])).To(Equal(1))
		Expect(inspectOut[0].NetworkSettings.Ports["80/tcp"][0].HostPort).To(Equal("8080"))
		Expect(inspectOut[0].NetworkSettings.Ports["80/tcp"][0].HostIP).To(Equal(""))
	})

	It("podman run -p 80/udp", func() {
		name := "testctr"
		session := podmanTest.Podman([]string{"create", "-t", "-p", "80/udp", "--name", name, ALPINE, "/bin/sh"})
		session.WaitWithDefaultTimeout()
		inspectOut := podmanTest.InspectContainer(name)
		Expect(len(inspectOut)).To(Equal(1))
		Expect(len(inspectOut[0].NetworkSettings.Ports)).To(Equal(1))
		Expect(len(inspectOut[0].NetworkSettings.Ports["80/udp"])).To(Equal(1))
		Expect(inspectOut[0].NetworkSettings.Ports["80/udp"][0].HostPort).To(Not(Equal("80")))
		Expect(inspectOut[0].NetworkSettings.Ports["80/udp"][0].HostIP).To(Equal(""))
	})

	It("podman run -p 127.0.0.1:8080:80", func() {
		name := "testctr"
		session := podmanTest.Podman([]string{"create", "-t", "-p", "127.0.0.1:8080:80", "--name", name, ALPINE, "/bin/sh"})
		session.WaitWithDefaultTimeout()
		inspectOut := podmanTest.InspectContainer(name)
		Expect(len(inspectOut)).To(Equal(1))
		Expect(len(inspectOut[0].NetworkSettings.Ports)).To(Equal(1))
		Expect(len(inspectOut[0].NetworkSettings.Ports["80/tcp"])).To(Equal(1))
		Expect(inspectOut[0].NetworkSettings.Ports["80/tcp"][0].HostPort).To(Equal("8080"))
		Expect(inspectOut[0].NetworkSettings.Ports["80/tcp"][0].HostIP).To(Equal("127.0.0.1"))
	})

	It("podman run -p 127.0.0.1:8080:80/udp", func() {
		name := "testctr"
		session := podmanTest.Podman([]string{"create", "-t", "-p", "127.0.0.1:8080:80/udp", "--name", name, ALPINE, "/bin/sh"})
		session.WaitWithDefaultTimeout()
		inspectOut := podmanTest.InspectContainer(name)
		Expect(len(inspectOut)).To(Equal(1))
		Expect(len(inspectOut[0].NetworkSettings.Ports)).To(Equal(1))
		Expect(len(inspectOut[0].NetworkSettings.Ports["80/udp"])).To(Equal(1))
		Expect(inspectOut[0].NetworkSettings.Ports["80/udp"][0].HostPort).To(Equal("8080"))
		Expect(inspectOut[0].NetworkSettings.Ports["80/udp"][0].HostIP).To(Equal("127.0.0.1"))
	})

	It("podman run -p [::1]:8080:80/udp", func() {
		name := "testctr"
		session := podmanTest.Podman([]string{"create", "-t", "-p", "[::1]:8080:80/udp", "--name", name, ALPINE, "/bin/sh"})
		session.WaitWithDefaultTimeout()
		inspectOut := podmanTest.InspectContainer(name)
		Expect(len(inspectOut)).To(Equal(1))
		Expect(len(inspectOut[0].NetworkSettings.Ports)).To(Equal(1))
		Expect(len(inspectOut[0].NetworkSettings.Ports["80/udp"])).To(Equal(1))
		Expect(inspectOut[0].NetworkSettings.Ports["80/udp"][0].HostPort).To(Equal("8080"))
		Expect(inspectOut[0].NetworkSettings.Ports["80/udp"][0].HostIP).To(Equal("::1"))
	})

	It("podman run -p [::1]:8080:80/tcp", func() {
		name := "testctr"
		session := podmanTest.Podman([]string{"create", "-t", "-p", "[::1]:8080:80/tcp", "--name", name, ALPINE, "/bin/sh"})
		session.WaitWithDefaultTimeout()
		inspectOut := podmanTest.InspectContainer(name)
		Expect(len(inspectOut)).To(Equal(1))
		Expect(len(inspectOut[0].NetworkSettings.Ports)).To(Equal(1))
		Expect(len(inspectOut[0].NetworkSettings.Ports["80/tcp"])).To(Equal(1))
		Expect(inspectOut[0].NetworkSettings.Ports["80/tcp"][0].HostPort).To(Equal("8080"))
		Expect(inspectOut[0].NetworkSettings.Ports["80/tcp"][0].HostIP).To(Equal("::1"))
	})

	It("podman run --expose 80 -P", func() {
		name := "testctr"
		session := podmanTest.Podman([]string{"create", "-t", "--expose", "80", "-P", "--name", name, ALPINE, "/bin/sh"})
		session.WaitWithDefaultTimeout()
		inspectOut := podmanTest.InspectContainer(name)
		Expect(len(inspectOut)).To(Equal(1))
		Expect(len(inspectOut[0].NetworkSettings.Ports)).To(Equal(1))
		Expect(len(inspectOut[0].NetworkSettings.Ports["80/tcp"])).To(Equal(1))
		Expect(inspectOut[0].NetworkSettings.Ports["80/tcp"][0].HostPort).To(Not(Equal("0")))
		Expect(inspectOut[0].NetworkSettings.Ports["80/tcp"][0].HostIP).To(Equal(""))
	})

	It("podman run --expose 80/udp -P", func() {
		name := "testctr"
		session := podmanTest.Podman([]string{"create", "-t", "--expose", "80/udp", "-P", "--name", name, ALPINE, "/bin/sh"})
		session.WaitWithDefaultTimeout()
		inspectOut := podmanTest.InspectContainer(name)
		Expect(len(inspectOut)).To(Equal(1))
		Expect(len(inspectOut[0].NetworkSettings.Ports)).To(Equal(1))
		Expect(len(inspectOut[0].NetworkSettings.Ports["80/udp"])).To(Equal(1))
		Expect(inspectOut[0].NetworkSettings.Ports["80/udp"][0].HostPort).To(Not(Equal("0")))
		Expect(inspectOut[0].NetworkSettings.Ports["80/udp"][0].HostIP).To(Equal(""))
	})

	It("podman run --expose 80 -p 80", func() {
		name := "testctr"
		session := podmanTest.Podman([]string{"create", "-t", "--expose", "80", "-p", "80", "--name", name, ALPINE, "/bin/sh"})
		session.WaitWithDefaultTimeout()
		inspectOut := podmanTest.InspectContainer(name)
		Expect(len(inspectOut)).To(Equal(1))
		Expect(len(inspectOut[0].NetworkSettings.Ports)).To(Equal(1))
		Expect(len(inspectOut[0].NetworkSettings.Ports["80/tcp"])).To(Equal(1))
		Expect(inspectOut[0].NetworkSettings.Ports["80/tcp"][0].HostPort).To(Not(Equal("80")))
		Expect(inspectOut[0].NetworkSettings.Ports["80/tcp"][0].HostIP).To(Equal(""))
	})

	It("podman run -p 127.0.0.1::8080/udp", func() {
		name := "testctr"
		session := podmanTest.Podman([]string{"create", "-t", "-p", "127.0.0.1::8080/udp", "--name", name, ALPINE, "/bin/sh"})
		session.WaitWithDefaultTimeout()
		inspectOut := podmanTest.InspectContainer(name)
		Expect(len(inspectOut)).To(Equal(1))
		Expect(len(inspectOut[0].NetworkSettings.Ports)).To(Equal(1))
		Expect(len(inspectOut[0].NetworkSettings.Ports["8080/udp"])).To(Equal(1))
		Expect(inspectOut[0].NetworkSettings.Ports["8080/udp"][0].HostPort).To(Not(Equal("8080")))
		Expect(inspectOut[0].NetworkSettings.Ports["8080/udp"][0].HostIP).To(Equal("127.0.0.1"))
	})

	It("podman run -p :8080", func() {
		name := "testctr"
		session := podmanTest.Podman([]string{"create", "-t", "-p", ":8080", "--name", name, ALPINE, "/bin/sh"})
		session.WaitWithDefaultTimeout()
		inspectOut := podmanTest.InspectContainer(name)
		Expect(len(inspectOut)).To(Equal(1))
		Expect(len(inspectOut[0].NetworkSettings.Ports)).To(Equal(1))
		Expect(len(inspectOut[0].NetworkSettings.Ports["8080/tcp"])).To(Equal(1))
		Expect(inspectOut[0].NetworkSettings.Ports["8080/tcp"][0].HostPort).To(Not(Equal("8080")))
		Expect(inspectOut[0].NetworkSettings.Ports["8080/tcp"][0].HostIP).To(Equal(""))
	})

	It("podman run -p 8080:8080 -p 8081:8080", func() {
		name := "testctr"
		session := podmanTest.Podman([]string{"create", "-t", "-p", "4000:8080", "-p", "8000:8080", "--name", name, ALPINE, "/bin/sh"})
		session.WaitWithDefaultTimeout()
		inspectOut := podmanTest.InspectContainer(name)
		Expect(len(inspectOut)).To(Equal(1))
		Expect(len(inspectOut[0].NetworkSettings.Ports)).To(Equal(1))
		Expect(len(inspectOut[0].NetworkSettings.Ports["8080/tcp"])).To(Equal(2))

		hp1 := inspectOut[0].NetworkSettings.Ports["8080/tcp"][0].HostPort
		hp2 := inspectOut[0].NetworkSettings.Ports["8080/tcp"][1].HostPort

		// We can't guarantee order
		Expect((hp1 == "4000" && hp2 == "8000") || (hp1 == "8000" && hp2 == "4000")).To(BeTrue())
	})

	It("podman run -p 0.0.0.0:8080:80", func() {
		name := "testctr"
		session := podmanTest.Podman([]string{"create", "-t", "-p", "0.0.0.0:8080:80", "--name", name, ALPINE, "/bin/sh"})
		session.WaitWithDefaultTimeout()
		inspectOut := podmanTest.InspectContainer(name)
		Expect(len(inspectOut)).To(Equal(1))
		Expect(len(inspectOut[0].NetworkSettings.Ports)).To(Equal(1))
		Expect(len(inspectOut[0].NetworkSettings.Ports["80/tcp"])).To(Equal(1))
		Expect(inspectOut[0].NetworkSettings.Ports["80/tcp"][0].HostPort).To(Equal("8080"))
		Expect(inspectOut[0].NetworkSettings.Ports["80/tcp"][0].HostIP).To(Equal(""))
	})

	It("podman run network expose host port 80 to container port 8000", func() {
		SkipIfRootless("iptables is not supported for rootless users")
		session := podmanTest.Podman([]string{"run", "-dt", "-p", "80:8000", ALPINE, "/bin/sh"})
		session.Wait(30)
		Expect(session.ExitCode()).To(Equal(0))
		results := SystemExec("iptables", []string{"-t", "nat", "-L"})
		Expect(results.ExitCode()).To(Equal(0))
		Expect(results.OutputToString()).To(ContainSubstring("8000"))

		ncBusy := SystemExec("nc", []string{"-l", "-p", "80"})
		Expect(ncBusy).To(ExitWithError())
	})

	It("podman run network expose host port 8081 to container port 8000 using rootlesskit port handler", func() {
		session := podmanTest.Podman([]string{"run", "--network", "slirp4netns:port_handler=rootlesskit", "-dt", "-p", "8081:8000", ALPINE, "/bin/sh"})
		session.Wait(30)
		Expect(session.ExitCode()).To(Equal(0))

		ncBusy := SystemExec("nc", []string{"-l", "-p", "8081"})
		Expect(ncBusy).To(ExitWithError())
	})

	It("podman run network expose host port 8082 to container port 8000 using slirp4netns port handler", func() {
		session := podmanTest.Podman([]string{"run", "--network", "slirp4netns:port_handler=slirp4netns", "-dt", "-p", "8082:8000", ALPINE, "/bin/sh"})
		session.Wait(30)
		Expect(session.ExitCode()).To(Equal(0))
		ncBusy := SystemExec("nc", []string{"-l", "-p", "8082"})
		Expect(ncBusy).To(ExitWithError())
	})

	It("podman run network expose host port 8080 to container port 8000 using invalid port handler", func() {
		session := podmanTest.Podman([]string{"run", "--network", "slirp4netns:port_handler=invalid", "-dt", "-p", "8080:8000", ALPINE, "/bin/sh"})
		session.Wait(30)
		Expect(session.ExitCode()).To(Not(Equal(0)))
	})

	It("podman run slirp4netns network with host loopback", func() {
		session := podmanTest.Podman([]string{"run", "--network", "slirp4netns:allow_host_loopback=true", ALPINE, "ping", "-c1", "10.0.2.2"})
		session.Wait(30)
		Expect(session.ExitCode()).To(Equal(0))
	})

	It("podman run slirp4netns network with different cidr", func() {
		slirp4netnsHelp := SystemExec("slirp4netns", []string{"--help"})
		Expect(slirp4netnsHelp.ExitCode()).To(Equal(0))

		networkConfiguration := "slirp4netns:cidr=192.168.0.0/24,allow_host_loopback=true"
		session := podmanTest.Podman([]string{"run", "--network", networkConfiguration, ALPINE, "ping", "-c1", "192.168.0.2"})
		session.Wait(30)

		if strings.Contains(slirp4netnsHelp.OutputToString(), "cidr") {
			Expect(session.ExitCode()).To(Equal(0))
		} else {
			Expect(session.ExitCode()).ToNot(Equal(0))
			Expect(session.ErrorToString()).To(ContainSubstring("cidr not supported"))
		}
	})

	It("podman run network bind to 127.0.0.1", func() {
		slirp4netnsHelp := SystemExec("slirp4netns", []string{"--help"})
		Expect(slirp4netnsHelp.ExitCode()).To(Equal(0))
		networkConfiguration := "slirp4netns:outbound_addr=127.0.0.1,allow_host_loopback=true"

		if strings.Contains(slirp4netnsHelp.OutputToString(), "outbound-addr") {
			ncListener := StartSystemExec("nc", []string{"-v", "-n", "-l", "-p", "8083"})
			session := podmanTest.Podman([]string{"run", "--network", networkConfiguration, "-dt", ALPINE, "nc", "-w", "2", "10.0.2.2", "8083"})
			session.Wait(30)
			ncListener.Wait(30)

			Expect(session.ExitCode()).To(Equal(0))
			Expect(ncListener.ExitCode()).To(Equal(0))
			Expect(ncListener.ErrorToString()).To(ContainSubstring("127.0.0.1"))
		} else {
			session := podmanTest.Podman([]string{"run", "--network", networkConfiguration, "-dt", ALPINE, "nc", "-w", "2", "10.0.2.2", "8083"})
			session.Wait(30)
			Expect(session.ExitCode()).ToNot(Equal(0))
			Expect(session.ErrorToString()).To(ContainSubstring("outbound_addr not supported"))
		}
	})

	It("podman run network bind to HostIP", func() {
		ip, err := utils.HostIP()
		Expect(err).To(BeNil())

		slirp4netnsHelp := SystemExec("slirp4netns", []string{"--help"})
		Expect(slirp4netnsHelp.ExitCode()).To(Equal(0))
		networkConfiguration := fmt.Sprintf("slirp4netns:outbound_addr=%s,allow_host_loopback=true", ip.String())

		if strings.Contains(slirp4netnsHelp.OutputToString(), "outbound-addr") {
			ncListener := StartSystemExec("nc", []string{"-v", "-n", "-l", "-p", "8084"})
			session := podmanTest.Podman([]string{"run", "--network", networkConfiguration, "-dt", ALPINE, "nc", "-w", "2", "10.0.2.2", "8084"})
			session.Wait(30)
			ncListener.Wait(30)

			Expect(session.ExitCode()).To(Equal(0))
			Expect(ncListener.ExitCode()).To(Equal(0))
			Expect(ncListener.ErrorToString()).To(ContainSubstring(ip.String()))
		} else {
			session := podmanTest.Podman([]string{"run", "--network", networkConfiguration, "-dt", ALPINE, "nc", "-w", "2", "10.0.2.2", "8084"})
			session.Wait(30)
			Expect(session.ExitCode()).ToNot(Equal(0))
			Expect(session.ErrorToString()).To(ContainSubstring("outbound_addr not supported"))
		}
	})

	It("podman run network expose ports in image metadata", func() {
		session := podmanTest.Podman([]string{"create", "--name", "test", "-t", "-P", nginx})
		session.Wait(90)
		Expect(session.ExitCode()).To(Equal(0))
		results := podmanTest.Podman([]string{"inspect", "test"})
		results.Wait(30)
		Expect(results.ExitCode()).To(Equal(0))
		Expect(results.OutputToString()).To(ContainSubstring(`"80/tcp":`))
	})

	It("podman run network expose duplicate host port results in error", func() {
		SkipIfRootless("FIXME we should be able to run this test in rootless mode with different ports")

		session := podmanTest.Podman([]string{"run", "--name", "test", "-dt", "-p", "80", ALPINE, "/bin/sh"})
		session.WaitWithDefaultTimeout()
		Expect(session.ExitCode()).To(Equal(0))

		inspect := podmanTest.Podman([]string{"inspect", "test"})
		inspect.WaitWithDefaultTimeout()
		Expect(inspect.ExitCode()).To(Equal(0))

		containerConfig := inspect.InspectContainerToJSON()
		Expect(containerConfig[0].NetworkSettings.Ports).To(Not(BeNil()))
		Expect(containerConfig[0].NetworkSettings.Ports["80/tcp"]).To(Not(BeNil()))
		Expect(containerConfig[0].NetworkSettings.Ports["80/tcp"][0].HostPort).ToNot(Equal(80))
	})

	It("podman run hostname test", func() {
		session := podmanTest.Podman([]string{"run", "--rm", ALPINE, "printenv", "HOSTNAME"})
		session.WaitWithDefaultTimeout()
		Expect(session.ExitCode()).To(Equal(0))
		match, _ := session.GrepString(hostname)
		Expect(match).Should(BeFalse())
	})

	It("podman run --net host hostname test", func() {
		session := podmanTest.Podman([]string{"run", "--rm", "--net", "host", ALPINE, "printenv", "HOSTNAME"})
		session.WaitWithDefaultTimeout()
		Expect(session.ExitCode()).To(Equal(0))
		match, _ := session.GrepString(hostname)
		Expect(match).Should(BeTrue())
	})
	It("podman run --net host --uts host hostname test", func() {
		session := podmanTest.Podman([]string{"run", "--rm", "--net", "host", "--uts", "host", ALPINE, "printenv", "HOSTNAME"})
		session.WaitWithDefaultTimeout()
		Expect(session.ExitCode()).To(Equal(0))
		match, _ := session.GrepString(hostname)
		Expect(match).Should(BeTrue())
	})
	It("podman run --uts host hostname test", func() {
		session := podmanTest.Podman([]string{"run", "--rm", "--uts", "host", ALPINE, "printenv", "HOSTNAME"})
		session.WaitWithDefaultTimeout()
		Expect(session.ExitCode()).To(Equal(0))
		match, _ := session.GrepString(hostname)
		Expect(match).Should(BeTrue())
	})

	It("podman run --net host --hostname ... hostname test", func() {
		session := podmanTest.Podman([]string{"run", "--rm", "--net", "host", "--hostname", "foobar", ALPINE, "printenv", "HOSTNAME"})
		session.WaitWithDefaultTimeout()
		Expect(session.ExitCode()).To(Equal(0))
		match, _ := session.GrepString("foobar")
		Expect(match).Should(BeTrue())
	})

	It("podman run --hostname ... hostname test", func() {
		session := podmanTest.Podman([]string{"run", "--rm", "--hostname", "foobar", ALPINE, "printenv", "HOSTNAME"})
		session.WaitWithDefaultTimeout()
		Expect(session.ExitCode()).To(Equal(0))
		match, _ := session.GrepString("foobar")
		Expect(match).Should(BeTrue())
	})

	It("podman run --net container: and --uts container:", func() {
		ctrName := "ctrToJoin"
		ctr1 := podmanTest.RunTopContainer(ctrName)
		ctr1.WaitWithDefaultTimeout()
		Expect(ctr1.ExitCode()).To(Equal(0))

		ctr2 := podmanTest.Podman([]string{"run", "-d", "--net=container:" + ctrName, "--uts=container:" + ctrName, ALPINE, "true"})
		ctr2.WaitWithDefaultTimeout()
		Expect(ctr2.ExitCode()).To(Equal(0))
	})

	It("podman run --net container: copies hosts and resolv", func() {
		ctrName := "ctr1"
		ctr1 := podmanTest.RunTopContainer(ctrName)
		ctr1.WaitWithDefaultTimeout()
		Expect(ctr1.ExitCode()).To(Equal(0))

		// Exec in and modify /etc/resolv.conf and /etc/hosts
		exec1 := podmanTest.Podman([]string{"exec", ctrName, "sh", "-c", "echo nameserver 192.0.2.1 > /etc/resolv.conf"})
		exec1.WaitWithDefaultTimeout()
		Expect(exec1.ExitCode()).To(Equal(0))

		exec2 := podmanTest.Podman([]string{"exec", ctrName, "sh", "-c", "echo 192.0.2.2 test1 > /etc/hosts"})
		exec2.WaitWithDefaultTimeout()
		Expect(exec2.ExitCode()).To(Equal(0))

		ctrName2 := "ctr2"
		ctr2 := podmanTest.Podman([]string{"run", "-d", "--net=container:" + ctrName, "--name", ctrName2, ALPINE, "top"})
		ctr2.WaitWithDefaultTimeout()
		Expect(ctr2.ExitCode()).To(Equal(0))

		exec3 := podmanTest.Podman([]string{"exec", "-i", ctrName2, "cat", "/etc/resolv.conf"})
		exec3.WaitWithDefaultTimeout()
		Expect(exec3.ExitCode()).To(Equal(0))
		Expect(exec3.OutputToString()).To(ContainSubstring("nameserver 192.0.2.1"))

		exec4 := podmanTest.Podman([]string{"exec", "-i", ctrName2, "cat", "/etc/hosts"})
		exec4.WaitWithDefaultTimeout()
		Expect(exec4.ExitCode()).To(Equal(0))
		Expect(exec4.OutputToString()).To(ContainSubstring("192.0.2.2 test1"))
	})

	It("podman run /etc/hosts contains --hostname", func() {
		session := podmanTest.Podman([]string{"run", "--rm", "--hostname", "foohostname", ALPINE, "grep", "foohostname", "/etc/hosts"})
		session.WaitWithDefaultTimeout()
		Expect(session.ExitCode()).To(Equal(0))
	})

	It("podman run --uidmap /etc/hosts contains --hostname", func() {
		SkipIfRootless("uidmap population of cninetworks not supported for rootless users")
		session := podmanTest.Podman([]string{"run", "--uidmap", "0:100000:1000", "--rm", "--hostname", "foohostname", ALPINE, "grep", "foohostname", "/etc/hosts"})
		session.WaitWithDefaultTimeout()
		Expect(session.ExitCode()).To(Equal(0))

		session = podmanTest.Podman([]string{"run", "--uidmap", "0:100000:1000", "--rm", "--hostname", "foohostname", "-v", "/etc/hosts:/etc/hosts", ALPINE, "grep", "foohostname", "/etc/hosts"})
		session.WaitWithDefaultTimeout()
		Expect(session.ExitCode()).To(Equal(1))
	})

	It("podman run network in user created network namespace", func() {
		SkipIfRootless("ip netns is not supported for rootless users")
		if Containerized() {
			Skip("Cannot be run within a container.")
		}
		addXXX := SystemExec("ip", []string{"netns", "add", "xxx"})
		Expect(addXXX.ExitCode()).To(Equal(0))
		defer func() {
			delXXX := SystemExec("ip", []string{"netns", "delete", "xxx"})
			Expect(delXXX.ExitCode()).To(Equal(0))
		}()

		session := podmanTest.Podman([]string{"run", "-dt", "--net", "ns:/run/netns/xxx", ALPINE, "wget", "www.podman.io"})
		session.Wait(90)
		Expect(session.ExitCode()).To(Equal(0))
	})

	It("podman run n user created network namespace with resolv.conf", func() {
		SkipIfRootless("ip netns is not supported for rootless users")
		if Containerized() {
			Skip("Cannot be run within a container.")
		}
		addXXX2 := SystemExec("ip", []string{"netns", "add", "xxx2"})
		Expect(addXXX2.ExitCode()).To(Equal(0))
		defer func() {
			delXXX2 := SystemExec("ip", []string{"netns", "delete", "xxx2"})
			Expect(delXXX2.ExitCode()).To(Equal(0))
		}()

		mdXXX2 := SystemExec("mkdir", []string{"-p", "/etc/netns/xxx2"})
		Expect(mdXXX2.ExitCode()).To(Equal(0))
		defer os.RemoveAll("/etc/netns/xxx2")

		nsXXX2 := SystemExec("bash", []string{"-c", "echo nameserver 11.11.11.11 > /etc/netns/xxx2/resolv.conf"})
		Expect(nsXXX2.ExitCode()).To(Equal(0))

		session := podmanTest.Podman([]string{"run", "--net", "ns:/run/netns/xxx2", ALPINE, "cat", "/etc/resolv.conf"})
		session.Wait(90)
		Expect(session.ExitCode()).To(Equal(0))
		Expect(session.OutputToString()).To(ContainSubstring("11.11.11.11"))
	})

	It("podman run network in bogus user created network namespace", func() {
		session := podmanTest.Podman([]string{"run", "-dt", "--net", "ns:/run/netns/xxy", ALPINE, "wget", "www.podman.io"})
		session.Wait(90)
		Expect(session).To(ExitWithError())
		Expect(session.ErrorToString()).To(ContainSubstring("stat /run/netns/xxy: no such file or directory"))
	})

	It("podman run in custom CNI network with --static-ip", func() {
		SkipIfRootless("Rootless mode does not support --ip")
		netName := "podmantestnetwork"
		ipAddr := "10.25.30.128"
		create := podmanTest.Podman([]string{"network", "create", "--subnet", "10.25.30.0/24", netName})
		create.WaitWithDefaultTimeout()
		Expect(create.ExitCode()).To(BeZero())
		defer podmanTest.removeCNINetwork(netName)

		run := podmanTest.Podman([]string{"run", "-t", "-i", "--rm", "--net", netName, "--ip", ipAddr, ALPINE, "ip", "addr"})
		run.WaitWithDefaultTimeout()
		Expect(run.ExitCode()).To(BeZero())
		Expect(run.OutputToString()).To(ContainSubstring(ipAddr))
	})

	It("podman run with new:pod and static-ip", func() {
		SkipIfRootless("Rootless does not support --ip")
		netName := "podmantestnetwork2"
		ipAddr := "10.25.40.128"
		podname := "testpod"
		create := podmanTest.Podman([]string{"network", "create", "--subnet", "10.25.40.0/24", netName})
		create.WaitWithDefaultTimeout()
		Expect(create.ExitCode()).To(BeZero())
		defer podmanTest.removeCNINetwork(netName)

		run := podmanTest.Podman([]string{"run", "-t", "-i", "--rm", "--pod", "new:" + podname, "--net", netName, "--ip", ipAddr, ALPINE, "ip", "addr"})
		run.WaitWithDefaultTimeout()
		Expect(run.ExitCode()).To(BeZero())
		Expect(run.OutputToString()).To(ContainSubstring(ipAddr))

		podrm := podmanTest.Podman([]string{"pod", "rm", "-f", podname})
		podrm.WaitWithDefaultTimeout()
		Expect(podrm.ExitCode()).To(BeZero())
	})

	It("podman run net=host adds entry to /etc/hosts", func() {
		run := podmanTest.Podman([]string{"run", "--net=host", ALPINE, "cat", "/etc/hosts"})
		run.WaitWithDefaultTimeout()
		Expect(run.ExitCode()).To(BeZero())
		Expect(strings.Contains(run.OutputToString(), "127.0.1.1")).To(BeTrue())
	})

	It("podman run with --net=host and --hostname sets correct hostname", func() {
		hostname := "testctr"
		run := podmanTest.Podman([]string{"run", "--net=host", "--hostname", hostname, ALPINE, "hostname"})
		run.WaitWithDefaultTimeout()
		Expect(run.ExitCode()).To(BeZero())
		Expect(strings.Contains(run.OutputToString(), "testctr")).To(BeTrue())
	})
})
