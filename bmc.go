package main

import (
	"bufio"
	"bytes"
	"fmt"
	"github.com/platinasystems/test"
	"net"
	"os/exec"
	"regexp"
	"strconv"
	"testing"
)

func bmcTest(t *testing.T) {
	test.SkipIfDryRun(t)
	assert := test.Assert{t}

	if *test.VVV {
		t.Log("checking redis-cli")
	}
	assert.Program("redis-cli")

	ll := bmcGetIpv6LinkLocal(t)

	fields := bmcGetFields(t, ll)
	fields.TestFloatValue(t, "vmon.1v0.tha.units.V", 0.9, 1.1)
	fields.TestFloatValue(t, "vmon.3v3.sys.units.V", 3.2, 3.4)
	fields.TestFloatValue(t, "vmon.3v3.bmc.units.V", 3.2, 3.4)
	fields.TestFloatValue(t, "vmon.5v.sb.units.V", 4.9, 5.1)
	fields.TestFloatValue(t, "psu1.temp2.units.C", 20.0, 55.0)
	fields.TestFloatValue(t, "host.temp.target.units.C", 20.0, 75.0)
}

func bmcGetIpv6LinkLocal(t *testing.T) (ip net.IP) {
	t.Helper()
	assert := test.Assert{t}

	out, err := exec.Command(*Goes, "mac-ll").Output()
	if err != nil {
		assert.Nil(fmt.Errorf("Unable to run mac-ll"))
	}

	re := regexp.MustCompile(`(?m)link-local:\s*([:0-9a-fA-F]+)`)
	sm := re.FindAllSubmatch(out, -1)
	if len(sm) > 0 {
		if len(sm[0]) > 1 {
			ip = net.ParseIP(string(sm[0][1]))
		}
	}

	if ip == nil {
		assert.Nil(fmt.Errorf("Invalid link-local address"))
	}

	return

}

func bmcGetFields(t *testing.T, ip net.IP) (fields bmcFields) {
	t.Helper()
	assert := test.Assert{t}

	out, err := exec.Command("/usr/bin/redis-cli", "--raw",
		"-h", ip.String()+"%eth0",
		"hgetall", "platina-mk1-bmc").Output()
	if err != nil {
		assert.Nil(fmt.Errorf("Unable to run redis-cli"))
	}

	fields = make(bmcFields)

	scanner := bufio.NewScanner(bytes.NewReader(out))
	for scanner.Scan() {
		key := scanner.Text()
		if _, found := fields[key]; found {
			assert.Nil(fmt.Errorf("hgetall returned duplicate field: %s", key))
		}

		scanner.Scan()
		fields[key] = scanner.Text()
	}
	if scanner.Err() != nil {
		assert.Nil(fmt.Errorf("Incomplete hgetall"))
	}
	return
}

type bmcFields map[string]string

func (fields bmcFields) TestFloatValue(t *testing.T, name string, min, max float32) {
	assert := test.Assert{t}
	v, ok := fields[name]
	if !ok {
		assert.Nil(fmt.Errorf("BMC field %s not found.", name))
	}

	value64, err := strconv.ParseFloat(v, 32)
	if err != nil {
		assert.Nil(fmt.Errorf("BMC field value invalid (%s).", v))
	}

	value := float32(value64)
	if value < min || value > max {
		assert.Nil(fmt.Errorf("BMC field %s:%6.3f out of range (%6.3f,%6.3f)",
			name, value, min, max))
	}
}
