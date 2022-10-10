package host_try

import (
	"fmt"
	"testing"
)

func TestHostTry(t *testing.T) {
		host := []string{"172.16.21.1","ntp.ntsc1.ac.cn","cn.ntp1.org.cn","cn.pool.ntp1.org","time.pool.aliyun1.com", "172.16.2.1"}
		demo := New(host, AttemptType("directConnection"))
		demo.DoTry(SetNtpTime)
		fmt.Println(demo.successHost)
		fmt.Println(demo.attemptStatus)
		fmt.Println(demo.errorHost)
		fmt.Println(demo.nowAttempt)
}

func TestNtp(t *testing.T) {
	host := "ntp.ntsc.ac.cn"
	err := SetNtpTime(host)
	if err != nil {
		t.Errorf(err.Error())
	}
}

