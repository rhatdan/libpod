#!/usr/bin/env bats

load helpers

ALPINE="docker.io/library/alpine:latest"

@test "run a container based on local image" {
    run ${KPOD_BINARY} ${KPOD_OPTIONS} pull docker.io/library/busybox:latest
    echo "$output"
    [ "$status" -eq 0 ]
    run ${KPOD_BINARY} ${KPOD_OPTIONS} run docker.io/library/busybox:latest ls
    echo "$output"
    [ "$status" -eq 0 ]
}

@test "run a container based on a remote image" {
    run ${KPOD_BINARY} ${KPOD_OPTIONS} run ${ALPINE} ls
    echo "$output"
    [ "$status" -eq 0 ]
}

@test "run selinux test" {

    if [ ! -e /usr/sbin/selinuxenabled ] || [ ! /usr/sbin/selinuxenabled ]; then
        skip "SELinux not enabled"
    fi

    firstLabel=$(${KPOD_BINARY} ${KPOD_OPTIONS} run ${ALPINE} cat /proc/self/attr/current)
    run ${KPOD_BINARY} ${KPOD_OPTIONS} run ${ALPINE} cat /proc/self/attr/current
    echo "$output"
    [ "$status" -eq 0 ]
    [ "$output" != "${firstLabel}" ]

    run bash -c "${KPOD_BINARY} ${KPOD_OPTIONS} run -t -i --security-opt label=level:s0:c1,c2 ${ALPINE} cat /proc/self/attr/current | grep s0:c1,c2"
    echo "$output"
    [ "$status" -eq 0 ]

}

@test "run capabilities test" {

    run ${KPOD_BINARY} ${KPOD_OPTIONS} run --cap-add all ${ALPINE} cat /proc/self/status
    echo "$output"
    [ "$status" -eq 0 ]

    run ${KPOD_BINARY} ${KPOD_OPTIONS} run --cap-add sys_admin ${ALPINE} cat /proc/self/status
    echo "$output"
    [ "$status" -eq 0 ]

    run ${KPOD_BINARY} ${KPOD_OPTIONS} run --cap-drop all ${ALPINE} cat /proc/self/status
    echo "$output"
    [ "$status" -eq 0 ]

    run ${KPOD_BINARY} ${KPOD_OPTIONS} run --cap-drop setuid ${ALPINE} cat /proc/self/status
    echo "$output"
    [ "$status" -eq 0 ]

}

@test "run environment test" {

#    run bash -c "${KPOD_BINARY} ${KPOD_OPTIONS} run ${ALPINE} sh -c printenv | grep HOSTNAME"
#    echo "$output"
#    [ "$status" -eq 0 ]

    run bash -c "FOO=BAR ${KPOD_BINARY} ${KPOD_OPTIONS} run -t -i -env FOO ${ALPINE} sh -c printenv | grep FOO"
    echo "$output"
    [ "$status" -eq 0 ]

    run bash -c "${KPOD_BINARY} ${KPOD_OPTIONS} run -t -i -env FOO=BAR ${ALPINE} sh -c printenv | grep FOO"
    echo "$output"
    [ "$status" -eq 0 ]

    run bash -c "${KPOD_BINARY} ${KPOD_OPTIONS} run -t -i -env BAR ${ALPINE} sh -c printenv | grep BAR"
    echo "$output"
    [ "$status" -ne 0 ]

}
